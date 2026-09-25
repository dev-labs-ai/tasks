# Tasks --- Todo List MVP

**Status:** Draft
**Last updated:** 2026-09-25

Inputs: [prd.md](prd.md) and [techspec.md](techspec.md). The tasks follow
the seven steps of the tech spec's implementation plan (section 12).

-   Work in ID order; **Depends on** lists the tasks that must be done
    first.
-   Each task is one commit and lands with its tests passing.
-   **Refs** point to PRD requirements (FR-n), tech spec sections
    (§n) and stack specification sections (spec §n).
-   Assumes the PRD's open questions stay as written: 200-character
    titles and deletion without confirmation.

------------------------------------------------------------------------

## Progress

**1. Restructure**

-   [x] T01 --- Move the Nuxt scaffold into `apps/web/`
-   [x] T02 --- Create the Go API skeleton
-   [x] T03 --- Add Docker Compose, `.env.example` and the Makefile
-   [x] T04 --- Configure Nuxt as an SPA with the dev proxy
-   [x] T05 --- Set up linting and formatting

**2. Database**

-   [x] T06 --- Add the initial migration and the migration runner
-   [x] T07 --- Generate the repositories with sqlc

**3. Auth API**

-   [x] T08 --- Build the HTTP foundation
-   [x] T09 --- Implement password hashing and session tokens
-   [x] T10 --- Implement input validation
-   [x] T11 --- Implement the auth service
-   [x] T12 --- Expose the auth endpoints
-   [x] T13 --- Clean up expired sessions and login attempts

**4. Tasks API**

-   [x] T14 --- Implement the task service
-   [x] T15 --- Expose the task endpoints

**5. Frontend foundation**

-   [x] T16 --- Install Tailwind CSS, shadcn-vue and Pinia
-   [x] T17 --- Set up Vitest with `@nuxt/test-utils`
-   [x] T18 --- Add API types and client-side validation
-   [x] T19 --- Add the API client plugin
-   [x] T20 --- Add the auth store, route middleware and layouts
-   [x] T21 --- Build the register and login pages

**6. Task page**

-   [x] T22 --- Implement `useTasks`
-   [x] T23 --- Implement `useTaskFilter`
-   [x] T24 --- Build the task components
-   [x] T25 --- Build the task page

**7. Delivery**

-   [ ] T26 --- Build the production Docker images
-   [ ] T27 --- Write the E2E tests
-   [ ] T28 --- Add the CI workflow
-   [ ] T29 --- Document setup in the README

------------------------------------------------------------------------

## 1. Restructure

### T01 --- Move the Nuxt scaffold into `apps/web/`

**Depends on:** --- · **Refs:** §3

-   Move `app/`, `public/`, `nuxt.config.ts`, `package.json`,
    `bun.lock` and `tsconfig.json` into `apps/web/`.
-   Keep `README.md`, `LICENSE`, `docs/` and `.gitignore` at the root;
    make sure the ignore rules still match inside `apps/web/`.

**Done when** `bun install` and `bun run build` succeed in `apps/web/`,
and `bun dev` serves the welcome page.

### T02 --- Create the Go API skeleton

**Depends on:** --- · **Refs:** §3, §5.1, §8

-   `apps/api/go.mod` on Go 1.27; `cmd/server/main.go`.
-   `internal/config`: read `DATABASE_URL` (required), `PORT` (default
    `8080`) and `APP_ENV`.
-   `internal/database`: open the `pgxpool`.
-   `http.Server` with read-header, read, write and idle timeouts, and
    graceful shutdown on `SIGTERM`.
-   `log/slog`: JSON when `APP_ENV=production`, text otherwise.
-   `GET /healthz` pings the database.

**Done when** unit tests cover config parsing (missing `DATABASE_URL`,
defaults), and `/healthz` returns `200` with PostgreSQL up and an error
status with it down.

### T03 --- Add Docker Compose, `.env.example` and the Makefile

**Depends on:** T01, T02 · **Refs:** §8, §10

-   `docker-compose.yml` with the `postgres:18` service, a named volume
    and a health check.
-   `.env.example` with every variable from §8, development values only.
-   Makefile targets `dev` and `test`, filled in as later tasks add
    pieces.

**Done when** `make dev` starts PostgreSQL, the API and `nuxt dev`, and
`/healthz` answers `200`.

### T04 --- Configure Nuxt as an SPA with the dev proxy

**Depends on:** T01, T02 · **Refs:** §1, §2, §7.1

-   `ssr: false` in `nuxt.config.ts`.
-   `nitro.devProxy` forwards `/api` to `API_URL`, defaulting to
    `http://localhost:8080`; paths must reach the API unchanged.
-   TypeScript strict mode.

**Done when** `curl -i http://localhost:3000/api/v1/anything` returns
Go's plain-text `404 page not found` rather than Nuxt's HTML. Recheck
that the path arrives unchanged once T08 adds request logging.

### T05 --- Set up linting and formatting

**Depends on:** T01, T02 · **Refs:** spec §12, §10

-   Go: `gofmt` check and `go vet`.
-   Web: ESLint, Prettier and `nuxi typecheck`.
-   `make lint` runs all of them.

**Done when** `make lint` passes on the current code and fails on a
deliberately misformatted file.

------------------------------------------------------------------------

## 2. Database

### T06 --- Add the initial migration and the migration runner

**Depends on:** T02, T03 · **Refs:** §4, §5.2, §9

-   `migrations/00001_init.sql` with `users`, `sessions`, `tasks` and
    `login_attempts`, exactly as in §4.
-   `migrations/embed.go` embeds the SQL files.
-   The migration runner uses goose's `NewProvider` over a
    `database/sql` connection from `pgx/v5/stdlib`; the server runs it at
    startup, and `make migrate` runs it alone.
-   Integration test harness: `integration` build tag, testcontainers-go
    with `postgres:18`, applies the real migrations.
    `make test-integration`.

**Done when** integration tests show:

-   migrations apply up and down cleanly;
-   `users.email` rejects uppercase and duplicates;
-   `tasks.title` rejects 0 and 201 code points and accepts 200 emoji;
-   deleting a user cascades to their sessions and tasks.

### T07 --- Generate the repositories with sqlc

**Depends on:** T06 · **Refs:** §5.3, §5.5

-   `sqlc.yaml`: schema from `migrations/`, queries from `queries/`,
    output to `internal/repositories`, `sql_package: "pgx/v5"`,
    `emit_interface: true`.
-   Queries in `users.sql`, `sessions.sql`, `login_attempts.sql` and
    `tasks.sql`, as listed in §5.3 and §5.5.
-   `make generate` runs `sqlc generate`; generated code is committed
    and never edited.

**Done when** integration tests show:

-   `ListTasks` returns newest first, ties broken by `id`, and filters
    by `completed`;
-   `UpdateTask` and `DeleteTask` on another user's task touch 0 rows;
-   `UpdateTask` changes `updated_at`;
-   `GetSessionUserID` ignores expired sessions;
-   `RegisterLoginAttempt` called concurrently never lets more than 5
    attempts through one window, and starts a new window once the old
    one has ended (set `window_start` in the past to test it);
-   the two cleanup queries delete only expired or stale rows.

------------------------------------------------------------------------

## 3. Auth API

### T08 --- Build the HTTP foundation

**Depends on:** T02 · **Refs:** §5.1, §6.1, §6.4

-   JSON helpers: decode with a 64 KiB limit and
    `DisallowUnknownFields`; write `{ "data": ... }` and the error
    envelope with optional `fields`.
-   Error codes from §6.4 as constants.
-   Middleware: recover, request logging, `http.CrossOriginProtection`
    with a deny handler that writes `FORBIDDEN_ORIGIN`.
-   Unknown routes return `NOT_FOUND` as JSON.

**Done when** HTTP tests cover: malformed JSON and unknown fields →
`400 MALFORMED_REQUEST`; body over 64 KiB → `413`; cross-origin `POST`
→ `403`; unknown route → `404`; a panicking handler → `500
INTERNAL_ERROR` with no details in the body.

### T09 --- Implement password hashing and session tokens

**Depends on:** T02 · **Refs:** §5.5

-   `internal/auth`: Argon2id hash and verify with m=19456 KiB, t=2,
    p=1, 16-byte salt and 32-byte key, stored as a PHC string; compared
    with `subtle.ConstantTimeCompare`; a fixed dummy hash for unknown
    emails.
-   Session tokens: 32 bytes from `crypto/rand`, base64url; SHA-256 for
    storage.

**Done when** unit tests show: hash and verify round-trip; a wrong
password fails; verification reads the parameters from the stored
string; malformed hashes are rejected; tokens are 32 bytes and differ
between calls.

### T10 --- Implement input validation

**Depends on:** T02 · **Refs:** §5.6, FR-1, FR-5, FR-10

-   Email: trim, lowercase, at most 254 characters, `net/mail` parse
    equal to the input.
-   Password: 8 to 64 code points, not trimmed.
-   Title: `strings.TrimSpace`, 1 to 200 code points.
-   Status: absent, `pending` or `completed`.

**Done when** unit tests cover the edges: `Name <a@b.c>` rejected,
mixed-case email normalized, 7/8/64/65-character passwords, a
whitespace-only title, 200 and 201 emoji, an unknown status.

### T11 --- Implement the auth service

**Depends on:** T07, T09, T10 · **Refs:** §5.4, §5.5, FR-1 to FR-3

-   `Register`: validate, hash, then insert the user and a session in
    one transaction; SQLSTATE `23505` → `EMAIL_TAKEN`.
-   `Login`: register the attempt first; over 5 → `TOO_MANY_ATTEMPTS`
    with the window end, without verifying the password. Verify against
    the dummy hash when the email is unknown. Same
    `INVALID_CREDENTIALS` for both failures. On success clear the
    attempts and create a new session.
-   `Logout` and `Authenticate`.
-   Sessions expire 24 hours after creation.

**Done when** unit tests with a fake `Querier` show: duplicate email →
`EMAIL_TAKEN`; unknown email and wrong password give the same error;
the 6th attempt is refused without calling the password check; success
clears the attempts; `Authenticate` rejects unknown tokens.

### T12 --- Expose the auth endpoints

**Depends on:** T08, T11 · **Refs:** §5.5, §6.2, FR-1 to FR-3

-   `POST /auth/register`, `POST /auth/login`, `POST /auth/logout`,
    `GET /auth/me`.
-   `requireAuth` middleware: cookie → user ID in the request context.
-   Session cookie with the attributes in §5.5; `Retry-After` on `429`.
-   Response structs built by hand, never from sqlc models.

**Done when** HTTP tests cover: register → `201` with `HttpOnly`,
`SameSite=Lax`, `Max-Age=86400` and `Secure` only in production;
duplicate → `409`; invalid fields → `400` with `fields`; identical
`401` bodies for unknown email and wrong password; 6th login → `429`
with `Retry-After`; logout → `204` with an expired cookie, then `me` →
`401`; no response contains `password_hash`.

### T13 --- Clean up expired sessions and login attempts

**Depends on:** T02, T07 · **Refs:** §5.5

-   A ticker in the API process runs `DeleteExpiredSessions` and
    `DeleteStaleLoginAttempts` once an hour and stops on shutdown.

**Done when** a unit test with a short interval shows both queries run,
and that the job stops when its context is cancelled.

------------------------------------------------------------------------

## 4. Tasks API

### T14 --- Implement the task service

**Depends on:** T07, T10 · **Refs:** §5.3, §5.4, FR-4 to FR-10

-   `List`, `Create`, `Update`, `Delete`, always scoped by the user ID.
-   Title rules from T10; `Update` needs at least one field.
-   No rows → `TASK_NOT_FOUND`.

**Done when** unit tests with a fake `Querier` cover title validation,
an empty update, and the not-found mapping for update and delete.

### T15 --- Expose the task endpoints

**Depends on:** T08, T12, T14 · **Refs:** §6.2, §6.3, FR-4 to FR-10

-   `GET /tasks?status=`, `POST /tasks`, `PATCH /tasks/{id}`,
    `DELETE /tasks/{id}`, all behind `requireAuth`.
-   A non-numeric `{id}` → `MALFORMED_REQUEST`.

**Done when** HTTP tests cover: create → `201`; invalid title → `400`;
list order and `status` filter; invalid `status` → `400`; `PATCH`
`completed` both ways and the title; another user's task → `404` on
`PATCH` and `DELETE`; a second `DELETE` → `404`; no session → `401`.

------------------------------------------------------------------------

## 5. Frontend foundation

### T16 --- Install Tailwind CSS, shadcn-vue and Pinia

**Depends on:** T04 · **Refs:** §7.1, §7.4

-   Tailwind CSS 4 and shadcn-vue, following shadcn-vue's Nuxt guide.
-   shadcn-vue components: `button`, `input`, `label`, `checkbox`,
    `card`, `tabs`, `alert`, `skeleton`; icons from `@lucide/vue`.
-   `@pinia/nuxt` with `pinia` and `@vue/devtools-api`.

**Done when** `bun run build` and `make lint` pass, and a page renders a
shadcn-vue button styled by Tailwind.

### T17 --- Set up Vitest with `@nuxt/test-utils`

**Depends on:** T16 · **Refs:** §9

-   `vitest` 5, `@nuxt/test-utils` `^4.3.1`, `@vue/test-utils`,
    `happy-dom`.
-   `vitest.config.ts` with the `unit` and `nuxt` projects.
-   `make test` runs Vitest along with the Go tests.

**Done when** one test runs in each project from `make test`.

### T18 --- Add API types and client-side validation

**Depends on:** T17 · **Refs:** §5.6, §6.3, §7.3

-   `types/api.ts`: `User`, `Task`, `TaskStatus`, the error body.
-   `utils/validation.ts`: the same rules as T10, counting code points
    with `[...value].length`.

**Done when** unit tests mirror T10's edge cases, including 200 and 201
emoji.

### T19 --- Add the API client plugin

**Depends on:** T18 · **Refs:** §7.3

-   `plugins/api.ts`: one `$fetch.create({ baseURL: '/api/v1' })`,
    provided as `$api` with `auth` and `tasks` services from
    `services/`.
-   Error responses become a typed `ApiError`.
-   `401 UNAUTHENTICATED` from any call other than `/auth/me` clears the
    auth store and navigates to `/login`.

**Done when** tests show the `ApiError` mapping, the redirect on `401
UNAUTHENTICATED`, and no redirect on `401 INVALID_CREDENTIALS` or on
`/auth/me`.

### T20 --- Add the auth store, route middleware and layouts

**Depends on:** T19 · **Refs:** §7.2, §7.3, FR-2, FR-3

-   `stores/auth.ts`: `user` and whether the session was checked.
-   `middleware/auth.global.ts`: one `GET /auth/me` per page load;
    signed-out users → `/login`; signed-in users on guest pages → `/`.
-   `auth` layout for guest pages; `default` layout with the user's
    email and a logout button.

**Done when** tests cover both redirects and that logout clears the
store and lands on `/login`.

### T21 --- Build the register and login pages

**Depends on:** T20 · **Refs:** §7.2, §7.5, FR-1, FR-2

-   Forms with labels, client-side validation and field errors from
    the API's `fields`.
-   `429` on login shows how long to wait, from `Retry-After`.
-   Success lands on `/`.

**Done when** tests cover client-side errors, API field errors, the
generic login error and the wait message.

------------------------------------------------------------------------

## 6. Task page

### T22 --- Implement `useTasks`

**Depends on:** T19 · **Refs:** §7.4, FR-5 to FR-9, FR-11

-   State: list, loading flag, error, pending flag per task.
-   `create`, `update`, `remove`; no optimistic updates.
-   A task that stops matching the active filter after an update leaves
    the list.

**Done when** tests with a mocked `$api` cover each action, the pending
flags, errors, and the filter rule.

### T23 --- Implement `useTaskFilter`

**Depends on:** T17 · **Refs:** §7.4, FR-10

-   Binds the filter to `?status=`; an unknown value means all.

**Done when** tests cover reading the query, writing it, and the
unknown-value fallback.

### T24 --- Build the task components

**Depends on:** T16, T18 · **Refs:** §7.4, §7.5, FR-5 to FR-10

-   `TaskForm`: Enter submits; the text stays on failure.
-   `TaskItem`: checkbox; edit via button or double-click, Enter saves,
    Escape cancels; delete button named "Delete task: <title>";
    controls disabled while pending; completed styling.
-   `TaskList` and `TaskFilter` (tabs).

**Done when** component tests cover submit, keep-on-error, edit and
cancel, the accessible names, and the disabled state.

### T25 --- Build the task page

**Depends on:** T20, T22, T23, T24 · **Refs:** §7.4, §7.5, FR-6, FR-11

-   `pages/index.vue` wires the composables and components.
-   States: skeleton while loading, empty message, `role="alert"` for
    errors.
-   After a delete, focus moves to the next task, or to the create input
    when the list is empty.
-   One column, usable from 320 px.

**Done when** tests cover the loading, empty and error states and the
focus move after delete.

------------------------------------------------------------------------

## 7. Delivery

### T26 --- Build the production Docker images

**Depends on:** T15, T25 · **Refs:** §2, §10

-   `docker/api.Dockerfile`: multi-stage, static Go binary.
-   `docker/web.Dockerfile`: `nuxt generate`, served by Caddy with
    `docker/Caddyfile` from §10.
-   Compose gains the `api` and `web` services; `make build`.

**Done when** `docker compose up` serves the app on one origin, `/api/*`
reaches Go, `/healthz` through Caddy does not reach the API, and
reloading `/login` serves the SPA.

### T27 --- Write the E2E tests

**Depends on:** T26 · **Refs:** §9, PRD §6

-   Playwright in `tests/e2e/`, against the Compose stack;
    `make test-e2e`.
-   Each test registers its own users.

**Done when** these pass:

-   the main flow: register → create → complete → filter by completed →
    delete → log out;
-   a reload keeps the user signed in and keeps the filter;
-   editing a title;
-   a second user cannot see the first user's tasks;
-   `/` redirects to `/login` after logout.

### T28 --- Add the CI workflow

**Depends on:** T05, T27 · **Refs:** §11

-   GitHub Actions on pull requests and pushes to `main`: lint →
    typecheck → frontend tests → backend tests → integration → E2E →
    build.
-   Fails if `sqlc generate` changes committed files.

**Done when** the workflow passes on a pull request.

### T29 --- Document setup in the README

**Depends on:** T28 · **Refs:** §8, §10

-   Replace the Nuxt starter README: prerequisites, `.env` setup, the
    Makefile targets, and links to the PRD and tech spec.

**Done when** a fresh clone reaches a running app by following only
the README.
