# Build context: the repository root.
# The Nuxt CLI runs on Node.js; Bun only installs the dependencies.
FROM node:24-slim AS build
COPY --from=oven/bun:1 /usr/local/bin/bun /usr/local/bin/bun
WORKDIR /app
COPY apps/web/package.json apps/web/bun.lock ./
# Scripts wait: postinstall runs `nuxt prepare`, which needs the sources.
RUN bun install --frozen-lockfile --ignore-scripts
COPY apps/web/ ./
RUN bun run generate

FROM caddy:2
COPY docker/Caddyfile /etc/caddy/Caddyfile
COPY --from=build /app/.output/public /srv
