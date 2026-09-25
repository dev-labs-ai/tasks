# Tech Spec --- Todo List MVP

**Status:** Draft
**Last updated:** 2026-09-25

Inputs:

-   [prd.md](prd.md) --- what the MVP does and why.
-   [web-system-stack-specification.md](web-system-stack-specification.md)
    --- the stack and the rules every project follows.

This document defines how the MVP is built. It fills the gaps the stack
specification leaves open; where it departs from the specification, the
departure is marked **Departure** and justified.

------------------------------------------------------------------------

## 1. Key decisions

| Area              | Decision                                                        | Why                                                                                              |
|-------------------|-----------------------------------------------------------------|--------------------------------------------------------------------------------------------------|
| Rendering         | Nuxt as a client-rendered SPA (`ssr: false`)                    | Every screen but login and register is behind authentication; no SEO need; no cookie forwarding during SSR. |
| Topology          | Frontend and API served from one origin; `/api/*` goes to Go    | The session cookie stays first-party; no CORS; `SameSite=Lax` works.                              |
| Authentication    | Server-side sessions stored in PostgreSQL, opaque token in an `HttpOnly` cookie | FR-3 needs logout to end the session immediately; deleting a row does that. Tokens with refresh would need a denylist. No Redis (spec section 18). |
| Password hashing  | Argon2id, m=19 MiB, t=2, p=1                                    | Minimum configuration recommended by the OWASP Password Storage Cheat Sheet.                     |
| Login limit       | 5 attempts per email per 15-minute window, counted in PostgreSQL | Stops online password guessing. An atomic upsert keeps the limit exact under concurrent requests; no Redis. |
| CSRF              | `SameSite=Lax` cookie + Go's `http.CrossOriginProtection`        | Standard library (Go 1.25+); no tokens to thread through the frontend.                           |
| Primary keys      | `bigint GENERATED ALWAYS AS IDENTITY`                           | Simplest mapping to Go `int64`. Guessable IDs are harmless because every query is scoped to the owner (FR-4). |
| Migrations        | goose, SQL files embedded in the API binary, applied at startup | One code path for dev, tests and production; sqlc reads the same files as its schema.           |
| Data access       | sqlc (`pgx/v5`) output **is** the repository layer               | A hand-written wrapper per query would be the premature abstraction spec principle 5 warns against. |
| HTTP routing      | Standard `net/http.ServeMux` method and wildcard patterns       | No third-party router needed.                                                                    |
| Frontend state    | Pinia for the current user only; task list local to its page    | Spec section 5: Pinia only for state that is truly shared.                                        |

Session cookies versus tokens, and the SPA rendering mode, are the first
candidates for ADRs.

------------------------------------------------------------------------

## 2. Architecture

``` text
Production

Browser ── HTTPS ──> Caddy ──┬── /api/*  ──> Go API :8080 ──> PostgreSQL 18
                             └── /*      ──> static SPA files (nuxt generate)

Development

Browser ──> nuxt dev :3000 ──┬── /api/*  ──> Go API :8080 ──> PostgreSQL 18
                             │   (nitro.devProxy)              (docker compose)
                             └── /*      ──> SPA with HMR
```

-   The browser only ever talks to one origin. In production Caddy
    routes by path; in development Nitro's `devProxy` forwards `/api/*`
    to the Go API.
-   The frontend never reaches PostgreSQL (spec section 3).
-   `GET /healthz` on the API is used by Docker health checks. Caddy
    only proxies `/api/*`, so it is not public.

**Departure** from the spec's `NUXT_PUBLIC_API_URL` example (section
14): the frontend calls the relative path `/api/v1`, so it needs no API
URL at runtime.

------------------------------------------------------------------------

## 3. Repository layout

The current Nuxt scaffold at the repository root moves into
`apps/web/`, following spec section 4.

``` text
tasks/
├── apps/
│   ├── web/                        Nuxt SPA
│   │   ├── app/
│   │   │   ├── assets/css/         Tailwind entry
│   │   │   ├── components/
│   │   │   │   ├── ui/             shadcn-vue components
│   │   │   │   └── tasks/          TaskForm, TaskItem, TaskList, TaskFilter
│   │   │   ├── composables/        useTasks, useTaskFilter
│   │   │   ├── layouts/            default, auth
│   │   │   ├── middleware/         auth.global.ts
│   │   │   ├── pages/              index, login, register
│   │   │   ├── plugins/            api.ts
│   │   │   ├── services/           auth.ts, tasks.ts
│   │   │   ├── stores/             auth.ts
│   │   │   ├── types/              api.ts
│   │   │   ├── utils/              validation.ts
│   │   │   └── app.vue
│   │   ├── nuxt.config.ts
│   │   ├── package.json
│   │   └── tsconfig.json
│   │
│   └── api/                        Go module
│       ├── cmd/server/             main.go
│       ├── internal/
│       │   ├── auth/               password hashing, session tokens
│       │   ├── config/             environment parsing
│       │   ├── database/           pgx pool, migration runner
│       │   ├── handlers/           HTTP handlers, requireAuth, JSON and errors
│       │   ├── middleware/         recover, logging, cross-origin check
│       │   ├── repositories/       sqlc output (generated, never edited)
│       │   └── services/           auth and task business rules
│       ├── migrations/             goose SQL files + embed.go
│       ├── queries/                users.sql, sessions.sql, tasks.sql
│       ├── sqlc.yaml
│       └── go.mod
│
├── tests/e2e/                      Playwright
├── docker/                         Dockerfiles, Caddyfile
├── docs/
├── docker-compose.yml
├── Makefile
├── .env.example
└── README.md
```

**Departure** from spec section 4: Go integration tests live inside the
Go module, next to the code they test, instead of in
`tests/integration/`. Go's `internal/` rule forbids importing
`apps/api/internal/...` from outside `apps/api/`. They sit behind the
`integration` build tag so `go test ./...` stays fast and needs no
Docker.

------------------------------------------------------------------------

## 4. Data model

One initial migration, `migrations/00001_init.sql`:

``` sql
-- +goose Up
CREATE TABLE users (
    id            bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         text        NOT NULL UNIQUE CHECK (email = lower(email)),
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash bytea       PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

CREATE TABLE tasks (
    id         bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title      text        NOT NULL CHECK (char_length(title) BETWEEN 1 AND 200),
    completed  boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX tasks_user_id_created_at_idx
    ON tasks (user_id, created_at DESC, id DESC);

CREATE TABLE login_attempts (
    email        text        PRIMARY KEY,
    attempts     integer     NOT NULL,
    window_start timestamptz NOT NULL
);

-- +goose Down
DROP TABLE login_attempts;
DROP TABLE tasks;
DROP TABLE sessions;
DROP TABLE users;
```

-   **Emails** are stored lowercased; the `CHECK` makes the unique
    constraint effectively case-insensitive without `citext`.
-   **Sessions** store the SHA-256 of the token, never the token, so a
    database leak yields no usable sessions.
-   **Title length** is enforced three times with one definition:
    Unicode code points. PostgreSQL's `char_length` counts code points,
    Go counts with `utf8.RuneCountInString`, and the frontend with
    `[...title].length`. JavaScript's `title.length` and HTML's
    `maxlength` count UTF-16 units and would disagree on emoji.
-   **Indexes** --- every foreign key has an index that starts with it.
    The task index also serves the list query, which filters by owner
    and sorts newest first.
-   `updated_at` is set by the `UPDATE` query itself; no triggers.
-   **Login attempts** are keyed by the normalized email and have no
    foreign key: unknown emails are counted too (see 5.5).

------------------------------------------------------------------------

## 5. Backend

### 5.1 Request pipeline

``` text
http.Server
  └─ log                method, path, status, duration (log/slog)
      └─ recover        panic → 500 INTERNAL_ERROR, logged with stack
          └─ CrossOriginProtection   non-safe cross-origin request → 403
              └─ ServeMux
                  ├─ public routes      register, login, healthz
                  └─ requireAuth        session cookie → user ID in context
                      └─ handler → service → repositories (sqlc) → PostgreSQL
```

-   Logging sits outside recover, so a request that panics is still
    logged, with the 500 that recover writes.
-   Request bodies are capped at 64 KiB (`http.MaxBytesReader`) and
    decoded with `DisallowUnknownFields`.
-   The server sets read-header, read, write and idle timeouts and shuts
    down gracefully on `SIGTERM`.
-   Logs are JSON in production and text in development. Passwords,
    session tokens and cookies are never logged.

### 5.2 Migrations

`migrations/embed.go` exposes the SQL files with `//go:embed *.sql`. At
startup the server opens a `database/sql` connection through
`github.com/jackc/pgx/v5/stdlib`, runs `goose.NewProvider(...).Up`, and
then opens the `pgxpool` the application uses. Integration tests call
the same function against a Testcontainers PostgreSQL.

### 5.3 Queries

sqlc reads `migrations/` as its schema (it skips goose's down
sections) and generates into `internal/repositories` with
`sql_package: "pgx/v5"` and `emit_interface: true`. Services depend on
the generated `Querier` interface, which unit tests replace with fakes.

| File           | Queries                                                                  |
|----------------|--------------------------------------------------------------------------|
| `users.sql`    | `CreateUser`, `GetUserByEmail`, `GetUserByID`                             |
| `sessions.sql` | `CreateSession`, `GetSessionUserID`, `DeleteSession`, `DeleteExpiredSessions` |
| `login_attempts.sql` | `RegisterLoginAttempt`, `ClearLoginAttempts`, `DeleteStaleLoginAttempts` |
| `tasks.sql`    | `ListTasks`, `CreateTask`, `UpdateTask`, `DeleteTask`                     |

**Ownership rule:** every task query takes `user_id` from the session and
has it in its `WHERE` clause. A task that belongs to someone else
matches no rows, and no rows is reported as `404 TASK_NOT_FOUND`. That
is FR-4's "fails as if the task did not exist", with no separate
permission check to forget.

``` sql
-- name: ListTasks :many
SELECT * FROM tasks
WHERE user_id = sqlc.arg('user_id')
  AND (sqlc.narg('completed')::boolean IS NULL
       OR completed = sqlc.narg('completed'))
ORDER BY created_at DESC, id DESC;

-- name: UpdateTask :one
UPDATE tasks
SET title      = coalesce(sqlc.narg('title'), title),
    completed  = coalesce(sqlc.narg('completed'), completed),
    updated_at = now()
WHERE id = sqlc.arg('id') AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteTask :execrows
DELETE FROM tasks
WHERE id = sqlc.arg('id') AND user_id = sqlc.arg('user_id');
```

### 5.4 Services

**AuthService**

-   `Register(email, password)` --- validates, hashes, then inserts the
    user and a new session in one transaction (sqlc's `WithTx`). A
    unique violation (SQLSTATE `23505`) becomes `EMAIL_TAKEN`; there is
    no check-then-insert race.
-   `Login(email, password)` --- first registers the attempt (5.5) and
    refuses it with `TOO_MANY_ATTEMPTS` when the limit is spent, without
    verifying the password. When the email is unknown, it still verifies
    the password against a fixed dummy hash, so response time does not
    reveal which emails exist. Both failures return the same
    `INVALID_CREDENTIALS`. A successful login clears the email's
    attempts and creates a new session, which also rules out session
    fixation.
-   `Logout(token)` --- deletes the session row.
-   `Authenticate(token)` --- hashes the token and looks up a session
    with `expires_at > now()`.

**TaskService** --- `List`, `Create`, `Update`, `Delete`. Applies the
title rules in 5.6, requires at least one field on update, and maps
"no rows" to `TASK_NOT_FOUND`.

### 5.5 Authentication details

**Passwords.** Argon2id from `golang.org/x/crypto/argon2` with
m=19456 KiB, t=2, p=1, a 16-byte random salt and a 32-byte key. Stored
as a PHC string (`$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>`), so the
parameters can be raised later without invalidating existing hashes.
Hashes are compared with `crypto/subtle.ConstantTimeCompare`.

**Session token.** 32 bytes from `crypto/rand`, base64url-encoded in the
cookie; the database keeps its SHA-256. Sessions expire 1 day (24
hours) after login, with no renewal. Once an hour the API deletes
expired sessions (`DeleteExpiredSessions`) and login-attempt rows whose
window has ended (`DeleteStaleLoginAttempts`).

**Cookie.**

| Attribute  | Value                                                  |
|------------|--------------------------------------------------------|
| Name       | `session`                                              |
| `HttpOnly` | yes                                                    |
| `Secure`   | when `APP_ENV=production`                               |
| `SameSite` | `Lax`                                                  |
| `Path`     | `/`                                                    |
| `Max-Age`  | 1 day; logout sends an expired cookie                  |

**CSRF.** `SameSite=Lax` keeps the cookie off cross-site `POST`,
`PATCH` and `DELETE` requests. `http.CrossOriginProtection` (checked
against Go 1.27.1) also rejects non-safe requests whose `Sec-Fetch-Site`
or `Origin` header marks them as cross-origin; its deny handler returns
the standard JSON error. No endpoint changes state on `GET`.

**Login attempt limit.** Each email gets 5 login attempts per 15-minute
window. The window starts at the first attempt; the 6th attempt inside
it is refused with `429 TOO_MANY_ATTEMPTS` and a `Retry-After` header
until the window ends. Refused attempts do not extend the window. A
successful login resets the count.

-   **Counted before the password is checked.** The attempt is
    registered first, so a burst of parallel requests cannot slip past
    the limit: the upsert locks the email's row, and exactly 5 requests
    per window see a count of 5 or less.
-   **Unknown emails are counted too.** Otherwise only real accounts
    would ever be blocked, and the `429` would reveal which emails
    exist.
-   **Keyed by the normalized email.** A malformed email is rejected
    with `400` before it touches the counter.

``` sql
-- name: RegisterLoginAttempt :one
INSERT INTO login_attempts (email, attempts, window_start)
VALUES (sqlc.arg('email'), 1, now())
ON CONFLICT (email) DO UPDATE
SET attempts     = CASE WHEN login_attempts.window_start <= now() - interval '15 minutes'
                        THEN 1 ELSE login_attempts.attempts + 1 END,
    window_start = CASE WHEN login_attempts.window_start <= now() - interval '15 minutes'
                        THEN now() ELSE login_attempts.window_start END
RETURNING attempts, window_start + interval '15 minutes' AS window_end;
```

The service refuses the attempt when `attempts > 5` and derives
`Retry-After` from `window_end`, so the window length lives only in this
query.

### 5.6 Validation

The backend is the definitive layer (spec section 10); the frontend
applies the same rules for fast feedback.

| Field      | Rule                                                                                   | PRD         |
|------------|----------------------------------------------------------------------------------------|-------------|
| `email`    | trimmed and lowercased; at most 254 characters; parsed by `net/mail.ParseAddress` and equal to the parsed address (rejects `Name <a@b.c>` forms) | FR-1        |
| `password` | 8 to 64 characters; not trimmed                                                         | FR-1        |
| `title`    | leading and trailing whitespace trimmed (`strings.TrimSpace`); 1 to 200 code points    | FR-5, FR-7  |
| `status`   | absent, `pending` or `completed`                                                        | FR-10       |

Lengths are counted in Unicode code points everywhere (see section 4).

------------------------------------------------------------------------

## 6. API

### 6.1 Conventions

-   Base path `/api/v1`, JSON only, `camelCase` fields.
-   Timestamps in RFC 3339, UTC.
-   Success: `{ "data": ... }`. `204` responses have no body.
-   Errors follow spec section 8. Validation errors add a `fields` map:

``` json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "The request has invalid fields",
    "fields": { "title": "Title must have at most 200 characters" }
  }
}
```

### 6.2 Endpoints

| Method | Path                    | Auth | Body                          | Success                 | Errors                                           |
|--------|-------------------------|------|-------------------------------|-------------------------|--------------------------------------------------|
| POST   | `/auth/register`        | no   | `{ email, password }`         | `201` user + cookie     | `400 VALIDATION_FAILED`, `409 EMAIL_TAKEN`        |
| POST   | `/auth/login`           | no   | `{ email, password }`         | `200` user + cookie     | `400 VALIDATION_FAILED`, `401 INVALID_CREDENTIALS`, `429 TOO_MANY_ATTEMPTS` |
| POST   | `/auth/logout`          | yes  | ---                           | `204` + expired cookie  | `401 UNAUTHENTICATED`                             |
| GET    | `/auth/me`              | yes  | ---                           | `200` user              | `401 UNAUTHENTICATED`                             |
| GET    | `/tasks?status=`        | yes  | ---                           | `200` task list         | `400 VALIDATION_FAILED`                           |
| POST   | `/tasks`                | yes  | `{ title }`                   | `201` task              | `400 VALIDATION_FAILED`                           |
| PATCH  | `/tasks/{id}`           | yes  | `{ title?, completed? }`      | `200` task              | `400 VALIDATION_FAILED`, `404 TASK_NOT_FOUND`     |
| DELETE | `/tasks/{id}`           | yes  | ---                           | `204`                   | `404 TASK_NOT_FOUND`                              |

Every authenticated route can also return `401 UNAUTHENTICATED`.

`PATCH` instead of the spec's `PUT` example, because each update changes
a single field. `status` omitted means all tasks. The list is not
paginated in the MVP (see section 13).

### 6.3 Resources

``` json
{ "id": 7, "email": "ana@example.com", "createdAt": "2026-09-25T14:03:11Z" }
```

``` json
{
  "id": 42,
  "title": "Buy milk",
  "completed": false,
  "createdAt": "2026-09-25T14:05:00Z",
  "updatedAt": "2026-09-25T14:05:00Z"
}
```

Responses are built from explicit response structs in `handlers`, never
from sqlc models, so `password_hash` cannot leak by accident.

### 6.4 Error codes

| Code                  | Status | When                                                    |
|-----------------------|--------|---------------------------------------------------------|
| `MALFORMED_REQUEST`   | 400    | body is not valid JSON, has unknown fields, or bad `{id}` |
| `VALIDATION_FAILED`   | 400    | a field breaks a rule in 5.6                             |
| `UNAUTHENTICATED`     | 401    | no session, or it expired                               |
| `INVALID_CREDENTIALS` | 401    | wrong email or password                                 |
| `FORBIDDEN_ORIGIN`    | 403    | rejected by `CrossOriginProtection`                      |
| `TASK_NOT_FOUND`      | 404    | no task with that ID for this user                      |
| `NOT_FOUND`           | 404    | unknown route                                           |
| `EMAIL_TAKEN`         | 409    | email already registered                                |
| `PAYLOAD_TOO_LARGE`   | 413    | body over 64 KiB                                        |
| `TOO_MANY_ATTEMPTS`   | 429    | login limit spent for this email; sent with `Retry-After` |
| `INTERNAL_ERROR`      | 500    | anything unexpected; details only in the logs           |

------------------------------------------------------------------------

## 7. Frontend

### 7.1 Configuration

-   `ssr: false`; `nuxt generate` builds static files for Caddy.
-   `nitro.devProxy` forwards `/api` to `API_URL` in development.
-   TypeScript strict (spec section 5).
-   `@pinia/nuxt`. Tailwind CSS 4 and shadcn-vue installed following
    shadcn-vue's Nuxt guide.

### 7.2 Routes

| Path        | Layout    | Access                               | PRD            |
|-------------|-----------|--------------------------------------|----------------|
| `/register` | `auth`    | guests only                          | FR-1           |
| `/login`    | `auth`    | guests only                          | FR-2           |
| `/`         | `default` | signed-in users; `?status=` filter   | FR-5 to FR-11  |

Guest pages declare `definePageMeta({ layout: 'auth', guest: true })`.
The `default` layout shows the user's email and a logout button.

### 7.3 Session handling

-   **`plugins/api.ts`** creates one `$fetch.create({ baseURL: '/api/v1' })`
    instance and provides `$api`, holding the `auth` and `tasks` service
    objects from `services/`. Services turn error responses into a typed
    `ApiError` (`status`, `code`, `message`, `fields`).
-   **`stores/auth.ts`** holds `user: User | null` and whether the
    session has been checked yet.
-   **`middleware/auth.global.ts`** calls `GET /auth/me` once per page
    load. Signed-out users are sent to `/login`; signed-in users who open
    a guest page are sent to `/`. This is only a UX convenience:
    protection lives in the API (spec section 9).
-   When any call other than `/auth/me` returns `401 UNAUTHENTICATED`,
    for example because the session expired, the plugin clears the store
    and navigates to `/login` (the Nuxt "custom `$fetch`" recipe).
    `401 INVALID_CREDENTIALS` on the login form does not redirect.
-   `429 TOO_MANY_ATTEMPTS` on the login form shows how long to wait,
    computed from `Retry-After`.

### 7.4 Task page

-   **State** lives in `useTasks()`, local to the page: the list, a
    loading flag, an error, and a pending flag per task.
-   **Filter** --- `useTaskFilter()` binds a tab control to
    `?status=`; the list reloads when the filter changes. An unknown
    value falls back to all (FR-10).
-   **Create** --- an input with a button; Enter submits. On success the
    task is added at the top and the input cleared. On failure the text
    stays, so the user can retry (FR-11).
-   **Complete** --- a checkbox per task. If the task no longer matches
    the active filter, it leaves the list.
-   **Edit** --- a button, or double-clicking the title, turns it into an
    input. Enter saves, Escape cancels.
-   **Delete** --- a button per task; permanent, no confirmation
    (FR-9).
-   **Updates are not optimistic.** The UI waits for the API, then
    changes local state. The affected task's controls are disabled
    meanwhile.
-   **States** (FR-11) --- skeleton rows while loading; an empty-state
    message; an inline alert for errors.

shadcn-vue components used: `button`, `input`, `label`, `checkbox`,
`card`, `tabs`, `alert`, `skeleton`. Icons from `lucide-vue-next`.

### 7.5 Accessibility and responsiveness

-   Every input has a visible label, or an accessible name when the
    design hides it.
-   Icon buttons carry names such as "Delete task: Buy milk".
-   Errors render in `role="alert"` regions so screen readers announce
    them.
-   After a delete, focus moves to the next task, or to the create
    input when the list is empty.
-   One column on every width; usable from 320 px up.

------------------------------------------------------------------------

## 8. Configuration

| Variable       | Used by               | Example                                            | Notes                                  |
|----------------|-----------------------|----------------------------------------------------|----------------------------------------|
| `DATABASE_URL` | API                   | `postgres://tasks:tasks@localhost:5432/tasks`       | required                               |
| `PORT`         | API                   | `8080`                                             | default `8080`                         |
| `APP_ENV`      | API                   | `development`                                      | `production` turns on `Secure` cookies and JSON logs |
| `API_URL`      | web, dev only         | `http://localhost:8080`                            | target of `nitro.devProxy`              |
| `POSTGRES_*`   | docker compose        | `POSTGRES_USER=tasks`                              | local database only                    |

`.env.example` lists all of them with development values and no real
credentials (spec section 14).

------------------------------------------------------------------------

## 9. Testing

The PRD's definition of done requires every acceptance criterion to be
covered by an automated test. The table assigns each requirement to the
cheapest test level that can prove it.

| PRD   | Backend unit (`testing`)                    | Backend HTTP (`httptest`)                         | Integration (Testcontainers)             | Frontend unit (Vitest)              | E2E (Playwright)                         |
|-------|---------------------------------------------|---------------------------------------------------|------------------------------------------|-------------------------------------|------------------------------------------|
| FR-1  | email normalization, password rules          | `201` + cookie; `409`; `400` with `fields`          | unique email regardless of case          | register form validation            | main flow                                |
| FR-2  | same error for unknown email and bad password; 6th attempt refused without verifying; success resets | identical `401` body for both cases; `429` with `Retry-After` | session lookup honours `expires_at`; concurrent attempts never exceed 5 per window; window resets once ended | wait message on `429` | reload keeps the user signed in          |
| FR-3  | ---                                         | `204` expires cookie; next call `401`               | session row deleted                      | middleware redirects                | main flow; `/` redirects after logout    |
| FR-4  | ---                                         | other user's task → `404`                          | update and delete scoped by owner touch 0 rows | ---                          | second user cannot see the first's tasks |
| FR-5  | trim; 1--200 code points                     | `201`; `400`                                       | `CHECK` rejects 0 and 201 code points    | title validation                    | main flow                                |
| FR-6  | ---                                         | response order                                    | newest first, ties by `id`               | ---                                 | main flow                                |
| FR-7  | same rules as FR-5                           | `200`; `404`                                       | `updated_at` changes                     | edit and cancel in `TaskItem`        | edit a title                             |
| FR-8  | ---                                         | `PATCH` `completed` both ways                      | ---                                      | completed styling                   | main flow                                |
| FR-9  | ---                                         | `204`; `404` on second delete                      | row removed                              | ---                                 | main flow                                |
| FR-10 | `status` parsing                             | invalid `status` → `400`                           | filter by `completed`                    | query binding, unknown value → all  | main flow; reload keeps the filter       |
| FR-11 | ---                                         | ---                                               | ---                                      | empty, loading, error; input kept   | ---                                      |

Also tested: migrations up and down (integration), `CrossOriginProtection`
rejecting a cross-origin `POST` (HTTP), body limit and unknown fields
(HTTP), and the error envelope on every error path (HTTP).

Tooling:

-   `go test ./...` runs unit and HTTP tests without Docker.
-   `go test -tags=integration ./...` starts `postgres:18` with
    testcontainers-go and applies the real migrations.
-   Components and composables rely on Nuxt auto-imports, so Vitest runs
    them through `@nuxt/test-utils` (not listed in the stack
    specification). Vitest 5 support starts at 4.3.1: earlier 4.x
    releases declare `vitest ^4.0.2` only. Checked on 2026-09-25 with
    `@nuxt/test-utils` 4.3.2, Vitest 5.0.1 and Nuxt 4.5.2: a smoke test
    using `mountSuspended`, `mockNuxtImport`, `useRoute` and `useState`
    passed. Require `@nuxt/test-utils` `^4.3.1`, plus `@vue/test-utils`
    and `happy-dom`, its default DOM.
-   `vitest.config.ts` defines two projects: `unit` (plain `node`
    environment, for functions like `utils/validation.ts`) and `nuxt`
    (`defineVitestProject`, for components, composables and stores).
-   Playwright runs from `tests/e2e/` against the full stack started by
    Docker Compose. Each test registers its own users, so tests do not
    share state.

------------------------------------------------------------------------

## 10. Local development and Docker

`docker-compose.yml` services:

| Service    | Image                                                  | Dev                         |
|------------|--------------------------------------------------------|-----------------------------|
| `postgres` | `postgres:18`, named volume, health check              | always                      |
| `api`      | built from `docker/api.Dockerfile` (multi-stage, static Go binary) | E2E and production-like runs |
| `web`      | built from `docker/web.Dockerfile`: Caddy + `nuxt generate` output | E2E and production-like runs |

Day-to-day development runs only `postgres` in Docker, with the API and
`nuxt dev` on the host for fast reloads.

`docker/Caddyfile`:

``` text
{$DOMAIN} {
    handle /api/* {
        reverse_proxy api:8080
    }
    handle {
        root * /srv
        try_files {path} /index.html
        file_server
    }
}
```

Makefile targets (spec section 15):

| Target                  | Does                                                           |
|-------------------------|----------------------------------------------------------------|
| `make dev`              | starts `postgres`, the API, and `nuxt dev`                     |
| `make test`             | Go unit and HTTP tests, Vitest                                 |
| `make test-integration` | Go tests with the `integration` tag                            |
| `make test-e2e`         | builds and starts the Compose stack, runs Playwright           |
| `make lint`             | `gofmt` check, `go vet`, ESLint, Prettier check, `nuxi typecheck` |
| `make generate`         | `sqlc generate`                                                 |
| `make migrate`          | applies migrations to `DATABASE_URL` without starting the server |
| `make build`            | builds both Docker images                                      |

------------------------------------------------------------------------

## 11. CI

GitHub Actions, one workflow on pull requests and on pushes to `main`,
in the order from spec section 16:

``` text
lint → typecheck → frontend tests → backend tests → integration → e2e → build
```

The workflow also runs `sqlc generate` and fails if the generated code
differs from what is committed. Deployment is out of scope for the MVP.

------------------------------------------------------------------------

## 12. Implementation plan

Each step ends with its tests passing.

1.  **Restructure** --- move the scaffold into `apps/web/`; create the
    Go module; add Compose with PostgreSQL, the Makefile and
    `.env.example`.
2.  **Database** --- the migration, sqlc setup and queries, and
    integration tests for constraints, ownership and ordering.
3.  **Auth API** --- hashing, sessions, middleware, the four auth
    endpoints, and their unit and HTTP tests.
4.  **Tasks API** --- the service, the four task endpoints, and their
    unit and HTTP tests.
5.  **Frontend foundation** --- Tailwind, shadcn-vue, Pinia, the API
    plugin, auth store, middleware, and the login and register pages.
6.  **Task page** --- composables and components, with unit tests.
7.  **Delivery** --- Dockerfiles, Caddy, Playwright tests, and the CI
    workflow.

------------------------------------------------------------------------

## 13. Risks and open questions

-   **Lockout abuse** --- anyone who knows a user's email can block
    their login for 15 minutes with 5 wrong passwords. Accepted for the
    MVP.
-   **Password spraying** --- the limit is per email, so one client can
    still try a common password against many emails. A per-IP limit is
    the follow-up; it needs no Redis while there is one API instance.
-   **Account enumeration** --- `EMAIL_TAKEN` on register reveals that an
    email has an account. This is inherent to FR-1 as written and
    accepted for the MVP.
-   **Unpaginated list** --- fine for personal lists; add cursor
    pagination on `(created_at, id)` if lists grow into the thousands.
-   **Migrations at startup** --- assumes one API instance. Running
    several would need goose's session locker or a separate migration
    step.
-   **PRD open questions** --- this spec assumes a 200-character title
    and deletion without confirmation. Changing either only touches the
    `CHECK` constraint and validators, or the task page.
