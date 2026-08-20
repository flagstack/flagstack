# syntax=docker/dockerfile:1.7

FROM node:24-bookworm-slim AS frontend
WORKDIR /src/frontend
RUN corepack enable && corepack prepare pnpm@11.22.0 --activate
COPY frontend/package.json ./
RUN pnpm install --no-frozen-lockfile
COPY frontend/ ./
RUN pnpm build

FROM golang:1.26.6-bookworm AS backend
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN go generate ./ent \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/switchonyourcode ./cmd/switchonyourcode \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/switchonyourcode-migrate ./cmd/switchonyourcode-migrate

FROM debian:bookworm-slim AS runtime
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=backend /out/switchonyourcode /app/switchonyourcode
COPY --from=backend /out/switchonyourcode-migrate /app/switchonyourcode-migrate
COPY --from=frontend /src/frontend/dist /app/frontend
RUN chown -R 65532:65532 /app
USER 65532:65532
ENV SWITCHONYOURCODE_HTTP_ADDR=:8080 \
    SWITCHONYOURCODE_STATIC_DIR=/app/frontend
EXPOSE 8080
CMD ["/app/switchonyourcode"]
