# Tasks

A private to-do list: register, add tasks, complete, filter, edit and
delete them. Each user sees only their own tasks.

- **Frontend** — Nuxt 4 (client-rendered SPA), Vue 3, TypeScript, Tailwind
  CSS 4, shadcn-vue, Pinia. Lives in [`apps/web`](apps/web).
- **Backend** — Go REST API with server-side sessions, pgx and sqlc. Lives
  in [`apps/api`](apps/api).
- **Database** — PostgreSQL 18, migrated with goose at API startup.

What it does and why: [docs/prd.md](docs/prd.md). How it is built:
[docs/techspec.md](docs/techspec.md). Implementation checklist:
[docs/tasks.md](docs/tasks.md).

## Prerequisites

| Tool                      | Version                                   |
| ------------------------- | ----------------------------------------- |
| Go                        | 1.27                                      |
| Node.js                   | 24 LTS (the Nuxt CLI runs on it)          |
| Bun                       | 1.x (installs the web dependencies)       |
| Docker with Compose       | any recent version                        |
| GNU Make                  | any                                       |

## Run it

```bash
git clone git@github.com:dev-labs-ai/tasks.git
cd tasks
make dev
```

Open <http://localhost:3000> and create an account.

`make dev` does everything on the first run: it copies `.env.example` to
`.env`, installs the web dependencies, starts PostgreSQL in Docker, and runs
the API (port 8080) and `nuxt dev` (port 3000). The API applies the
database migrations when it starts. Press Ctrl+C to stop the API and the web
app; PostgreSQL keeps running until `docker compose down`.

In development the browser only talks to `localhost:3000`: Nuxt forwards
`/api/*` to the Go API, so the session cookie works without CORS.

## Configuration

Settings come from environment variables; `.env.example` lists them all with
development values, and `make` reads them from `.env`. The main ones:

| Variable       | Used by | Purpose                                               |
| -------------- | ------- | ----------------------------------------------------- |
| `DATABASE_URL` | API     | PostgreSQL connection string (required)               |
| `PORT`         | API     | HTTP port, default `8080`                              |
| `APP_ENV`      | API     | `development` or `production` (Secure cookies, JSON logs) |
| `API_URL`      | web dev | where `nuxt dev` forwards `/api`                       |
| `WEB_PORT`     | Compose | host port of the production-like stack, default `8088` |
| `DOMAIN`       | Compose | host name for automatic HTTPS in Caddy                 |

Never commit `.env` or real credentials.

## Make targets

| Command                 | What it does                                                     |
| ----------------------- | ---------------------------------------------------------------- |
| `make dev`              | PostgreSQL, the API and `nuxt dev`                               |
| `make db`               | only PostgreSQL                                                  |
| `make migrate`          | apply migrations without starting the server                     |
| `make test`             | Go unit and HTTP tests, and the Vitest suites                    |
| `make test-integration` | Go tests against a real PostgreSQL (Testcontainers; needs Docker) |
| `make test-e2e`         | build and start the production stack, run the Playwright tests   |
| `make lint`             | gofmt, go vet, ESLint, Prettier and type checks                  |
| `make fmt`              | format Go and web sources in place                               |
| `make generate`         | regenerate the sqlc repositories after editing `apps/api/queries` |
| `make build`            | build the production images                                      |

## Production-like stack

```bash
docker compose up -d --build
```

Open <http://localhost:8088>. Caddy serves the generated SPA and forwards
`/api/*` to the API container; everything is on one origin. Set `DOMAIN` to
a real host name to get automatic HTTPS.

## Project layout

```text
apps/api      Go API: cmd/ (server, migrate), internal/, migrations/, queries/
apps/web      Nuxt app: app/ (pages, components, composables, stores), test/
tests/e2e     Playwright end-to-end tests
docker        Dockerfiles and the Caddyfile
docs          PRD, tech spec, task list and stack specification
```

## License

[MIT](LICENSE)
