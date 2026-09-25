-include .env

.PHONY: dev dev-api dev-web db test

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

# test: run the unit tests
test:
	cd apps/api && go test ./...

.env:
	cp .env.example .env
	@echo "Created .env from .env.example"

apps/web/node_modules: apps/web/bun.lock
	cd apps/web && bun install --frozen-lockfile
	@touch $@
