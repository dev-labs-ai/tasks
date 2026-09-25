# Technical Specification --- Web System

## 1. Objective

Build a modern, modular, scalable, and easy-to-maintain web system,
using an architecture that clearly separates frontend, backend, and
database.

The defined architecture is:

**Nuxt 4 + Vue 3 + TypeScript + Bun + Tailwind CSS + shadcn-vue +
Pinia + Go + PostgreSQL**

The frontend must be responsible for the user interface and user
experience. The Go backend must handle the API, authentication,
authorization, business rules, and database access. The frontend must
not access PostgreSQL directly.

------------------------------------------------------------------------

## 2. Required stack

### Frontend

-   Nuxt 4
-   Vue 3
-   TypeScript
-   Node.js, required by the Nuxt CLI
-   Bun
-   Vite, through the Nuxt ecosystem
-   Pinia
-   Tailwind CSS
-   shadcn-vue
-   Vitest
-   Vue Test Utils

### Backend

-   Go
-   REST API
-   `pgx` for PostgreSQL access
-   `sqlc` for generating typed code from SQL
-   migrations with Goose or golang-migrate
-   `testing` for unit tests
-   `net/http/httptest` for HTTP tests

### Database

-   PostgreSQL

### Integration and E2E tests

-   Testcontainers
-   PostgreSQL in a container for integration tests
-   Playwright for end-to-end tests

### Infrastructure

-   Docker
-   Docker Compose
-   Caddy or Nginx as reverse proxy

### Versions

Pin the major version; minor and patch updates are allowed. Versions
verified on 2026-09-25.

| Component      | Version                  | Notes                                                                                                                  |
|----------------|--------------------------|------------------------------------------------------------------------------------------------------------------------|
| Nuxt           | 4.x                      | Nuxt 3 reached end of life on 2026-07-31. Nuxt 5 is expected in Q4 2026; Nuxt 4 stays supported for 6 months after it. |
| Vue            | 3.5.x                    | 3.6 is still a release candidate.                                                                                      |
| TypeScript     | 6.0.x                    | Do not use 7.0: `vue-tsc`, which `nuxi typecheck` relies on, does not support it yet. Re-evaluate when 7.1 ships.      |
| Node.js        | 24 LTS                   | The Nuxt CLI runs on Node.js even when Bun manages dependencies. Nuxt 4.5 requires `^22.19`, `^24.11` or `>=26`.       |
| Bun            | 1.x                      |                                                                                                                        |
| Vite           | provided by Nuxt         | Do not pin separately.                                                                                                 |
| Pinia          | 4.x (`@pinia/nuxt` 1.x)  | `@vue/devtools-api` must be installed alongside `pinia`. `@pinia/nuxt` 1.x requires Nuxt 4.                            |
| Tailwind CSS   | 4.x                      | Configured in CSS; there is no `tailwind.config.js`.                                                                   |
| shadcn-vue     | 2.x                      |                                                                                                                        |
| Vitest         | 5.x                      |                                                                                                                        |
| Vue Test Utils | 2.x                      |                                                                                                                        |
| Playwright     | 1.x                      |                                                                                                                        |
| Go             | 1.27                     | Go 1.25 has been out of support since 2026-08-19.                                                                      |
| pgx            | v5                       |                                                                                                                        |
| sqlc           | 1.x                      |                                                                                                                        |
| PostgreSQL     | 18                       | Supported until 2030-11. Do not adopt 19 while it is in beta.                                                          |

------------------------------------------------------------------------

## 3. Architecture

The main communication flow must be:

``` text
Browser
   |
   v
Nuxt 4 / Vue
   |
   | HTTP/HTTPS - REST API
   v
Go API
   |
   | SQL
   v
PostgreSQL
```

The frontend must never access PostgreSQL directly.

All relevant business rules must remain in the Go backend.

------------------------------------------------------------------------

## 4. Initial project structure

Use a monorepo structure:

``` text
project/
├── apps/
│   ├── web/
│   │   ├── app/
│   │   │   ├── assets/
│   │   │   ├── components/
│   │   │   │   └── ui/
│   │   │   ├── composables/
│   │   │   ├── layouts/
│   │   │   ├── middleware/
│   │   │   ├── pages/
│   │   │   ├── plugins/
│   │   │   ├── stores/
│   │   │   ├── services/
│   │   │   ├── types/
│   │   │   ├── utils/
│   │   │   └── app.vue
│   │   ├── nuxt.config.ts
│   │   ├── package.json
│   │   └── tsconfig.json
│   │
│   └── api/
│       ├── cmd/
│       │   └── server/
│       ├── internal/
│       │   ├── auth/
│       │   ├── handlers/
│       │   ├── middleware/
│       │   ├── services/
│       │   ├── repositories/
│       │   ├── database/
│       │   └── config/
│       ├── migrations/
│       ├── queries/
│       ├── sqlc.yaml
│       └── go.mod
│
├── tests/
│   ├── e2e/
│   └── integration/
│
├── docker/
├── docker-compose.yml
├── Makefile
└── README.md
```

Nuxt 4 uses `app/` as its source directory: application code lives in
`apps/web/app/`, while configuration files stay at the root of
`apps/web/`.

The structure may evolve according to requirements, but it must avoid
premature complexity.

------------------------------------------------------------------------

## 5. Frontend

### Nuxt

Use Nuxt 4 as the main framework.

Use:

-   Nuxt pages and layouts;
-   middleware for route protection;
-   composables for reusable logic;
-   plugins only when truly necessary;
-   SSR when there is a benefit;
-   SPA/client-side rendering when it is more suitable for
    authenticated areas.

Do not create a second routing architecture outside of Nuxt.

### Vue

Use Vue 3 with the Composition API and `<script setup>`.

Avoid the Options API in new code.

### TypeScript

Use TypeScript in strict mode.

Avoid `any`.

Shared types must be defined explicitly and in an organized manner.

### Pinia

Use Pinia only for state that truly needs to be shared.

Do not put all application data in global stores.

Component-local state must remain local.

### Tailwind CSS

Use Tailwind CSS for styling.

Avoid unnecessary global CSS.

### shadcn-vue

Use shadcn-vue as the foundation for visual components.

Components must remain in the application code, allowing
customization.

Organize base components in:

``` text
app/components/ui/
```

Domain-specific components must live outside `app/components/ui`.

Example:

``` text
app/components/
├── ui/
│   ├── button/
│   ├── dialog/
│   ├── input/
│   └── table/
│
├── users/
├── dashboard/
└── projects/
```

------------------------------------------------------------------------

## 6. Go backend

The backend must be divided into clear responsibilities.

A possible organization:

``` text
internal/
├── auth/
├── handlers/
├── middleware/
├── services/
├── repositories/
├── database/
└── config/
```

### Handlers

Responsible for:

-   receiving HTTP requests;
-   validating basic input;
-   calling services;
-   setting HTTP status codes;
-   serializing responses.

Handlers must not contain complex business rules.

### Services

Hold the business rules.

Example:

``` text
Handler
   ↓
Service
   ↓
Repository
   ↓
PostgreSQL
```

### Repositories

Responsible for data access.

Use `sqlc` to generate code from the SQL queries.

Avoid SQL scattered across handlers and services.

------------------------------------------------------------------------

## 7. PostgreSQL

PostgreSQL is the application's official database.

Requirements:

-   versioned migrations;
-   foreign keys;
-   deliberately defined indexes;
-   constraints;
-   transactions when necessary;
-   clear SQL queries;
-   referential integrity.

Avoid creating unnecessary database abstractions.

### sqlc

Queries must be organized, for example:

``` text
queries/
├── users.sql
├── projects.sql
└── authentication.sql
```

The Go code generated by sqlc must not be edited manually.

------------------------------------------------------------------------

## 8. API

The API must use REST and JSON.

Example:

``` text
GET    /api/v1/users
GET    /api/v1/users/:id
POST   /api/v1/users
PUT    /api/v1/users/:id
DELETE /api/v1/users/:id
```

Use versioning:

``` text
/api/v1/...
```

Responses must have a consistent format.

Errors must have a standardized structure, for example:

``` json
{
  "error": {
    "code": "USER_NOT_FOUND",
    "message": "User not found"
  }
}
```

Do not return stack traces or internal details in production responses.

------------------------------------------------------------------------

## 9. Authentication and authorization

Authentication must be implemented in the Go backend.

The specific mechanism may be chosen during implementation, but it
must take into account:

-   secure session or access token + refresh token;
-   expiration;
-   logout;
-   CSRF protection when applicable;
-   `HttpOnly` cookies when the architecture uses cookie-based
    authentication;
-   authorization in the backend.

Never rely solely on frontend guards to protect resources.

All authorization must be validated in the backend.

------------------------------------------------------------------------

## 10. Validation

Validate data in two layers:

### Frontend

For fast user feedback.

### Backend

As the definitive validation layer.

The backend must never assume that data sent by the frontend is
trustworthy.

------------------------------------------------------------------------

## 11. Testing

The testing strategy has four levels.

### 11.1 Frontend --- Unit

Use:

-   Vitest
-   Vue Test Utils

Test:

-   functions;
-   composables;
-   stores;
-   components;
-   validations;
-   UI states.

### 11.2 Backend --- Unit

Use the standard package:

``` text
testing
```

Test mainly services and business rules.

### 11.3 Backend --- HTTP

Use:

``` text
net/http/httptest
```

Test handlers and endpoints.

### 11.4 Integration

Use Testcontainers with a real PostgreSQL.

Integration tests must validate:

``` text
Go
 ↓
Repository/sqlc
 ↓
PostgreSQL
```

Test:

-   queries;
-   constraints;
-   migrations;
-   transactions;
-   relationships.

Do not replace all database tests with mocks.

### 11.5 E2E

Use Playwright.

Tests must reproduce real flows:

``` text
Browser
 ↓
Nuxt
 ↓
Go API
 ↓
PostgreSQL
```

Examples:

-   login;
-   logout;
-   record creation;
-   editing;
-   deletion;
-   navigation;
-   permissions;
-   critical application flows.

------------------------------------------------------------------------

## 12. Code quality

Frontend:

-   TypeScript strict
-   ESLint
-   Prettier
-   Vitest

Backend:

-   `gofmt`
-   `go vet`
-   automated tests
-   Go linting when appropriate

Do not add tools without need.

------------------------------------------------------------------------

## 13. Docker

Development must have a reproducible environment.

Example:

``` text
docker compose up
```

It must allow starting at least:

``` text
PostgreSQL
```

and, when necessary:

``` text
backend
frontend
```

The production environment must have independent Docker images for the
required components.

Do not store secrets in the repository.

Use environment variables.

------------------------------------------------------------------------

## 14. Environment variables

Example:

``` text
DATABASE_URL=
API_URL=
NUXT_PUBLIC_API_URL=
APP_ENV=
PORT=
```

Secrets must never be committed to version control.

There must be an example file:

``` text
.env.example
```

without real credentials.

------------------------------------------------------------------------

## 15. Local development

The project must have simple commands.

Example:

``` bash
# Frontend
cd apps/web
bun install
bun dev

# Backend
cd apps/api
go run ./cmd/server
```

Or commands centralized in the Makefile:

``` bash
make dev
make test
make lint
make build
make migrate
```

------------------------------------------------------------------------

## 16. CI/CD

The pipeline must run at least:

``` text
Lint
  ↓
Typecheck
  ↓
Frontend tests
  ↓
Backend tests
  ↓
Integration tests
  ↓
E2E
  ↓
Build
```

Deployment must only occur after the required checks pass.

------------------------------------------------------------------------

## 17. Architectural principles

Follow these principles:

1.  Keep frontend and backend decoupled.
2.  Do not access PostgreSQL directly from the frontend.
3.  Concentrate business rules in the Go backend.
4.  Avoid complex business logic inside Vue components.
5.  Prefer simple code over premature abstractions.
6.  Use PostgreSQL as the source of truth for persistent data.
7.  Use explicit SQL and `sqlc` for typed database access.
8.  Unit test business rules.
9.  Use a real PostgreSQL in integration tests.
10. Use Playwright for critical flows.
11. Keep visual components reusable with shadcn-vue.
12. Keep TypeScript strict.
13. Do not introduce Redis, queues, microservices, or Kubernetes
    without a concrete need.
14. Design modules to allow future evolution without creating
    complexity up front.

------------------------------------------------------------------------

## 18. Additional dependencies

Do not automatically add:

-   Redis;
-   RabbitMQ;
-   Kafka;
-   Kubernetes;
-   microservices;
-   GraphQL;
-   Elasticsearch;
-   other databases.

These technologies must only be introduced when there is a concrete
technical requirement.

The initial architecture must remain:

``` text
Nuxt 4
   ↓
Go REST API
   ↓
PostgreSQL
```

with Docker for local and production infrastructure.

------------------------------------------------------------------------

## 19. Expected outcome

The system must result in a modern web application with:

-   a Nuxt/Vue frontend;
-   TypeScript;
-   dependency management with Bun;
-   UI based on Tailwind + shadcn-vue;
-   state with Pinia when necessary;
-   a Go backend;
-   a versioned REST API;
-   PostgreSQL;
-   migrations;
-   database access through pgx/sqlc;
-   unit tests;
-   integration tests;
-   E2E tests;
-   Docker;
-   a structure ready for CI/CD.

The priority must be **simplicity, clarity, security, testability,
and ease of maintenance**, avoiding architectural complexity before it
is needed.
