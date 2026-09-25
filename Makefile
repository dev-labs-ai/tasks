-include .env

SQLC_VERSION := v1.31.1

.PHONY: dev dev-api dev-web db migrate test test-integration lint lint-api lint-web fmt generate

# dev: start PostgreSQL, the API and nuxt dev; Ctrl+C stops the last two
dev: db
	@$(MAKE) --no-print-directory -j2 dev-api dev-web

dev-api: .env
	cd apps/api && DATABASE_URL="$(DATABASE_URL)" PORT="$(PORT)" APP_ENV="$(APP_ENV)" go run ./cmd/server

# PORT is removed because nuxt dev would otherwise listen on the API's port.
dev-web: .env apps/web/node_modules
	cd apps/web && env -u PORT API_URL="$(API_URL)" bun dev

# db: start PostgreSQL and wait until it accepts connections
db: .env
	docker compose up -d --wait postgres

# migrate: apply pending migrations without starting the server
migrate: db
	cd apps/api && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate

# test: run the unit tests
test:
	cd apps/api && go test ./...

# test-integration: run the Go tests that need PostgreSQL; needs Docker
test-integration:
	cd apps/api && go test -tags=integration ./...

# lint: check formatting, static analysis and types without changing files
lint: lint-api lint-web

lint-api:
	@unformatted=$$(gofmt -l apps/api); \
	if [ -n "$$unformatted" ]; then echo "gofmt needed:"; echo "$$unformatted"; exit 1; fi
	cd apps/api && go vet ./... && go vet -tags=integration ./...

lint-web: apps/web/node_modules
	cd apps/web && bun run lint && bun run format:check && bun run typecheck

# generate: regenerate internal/repositories from queries/ with sqlc
generate:
	cd apps/api && go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate

# fmt: rewrite Go and web sources in place
fmt: apps/web/node_modules
	gofmt -w apps/api
	cd apps/web && bun run format

.env:
	cp .env.example .env
	@echo "Created .env from .env.example"

apps/web/node_modules: apps/web/bun.lock
	cd apps/web && bun install --frozen-lockfile
	@touch $@
