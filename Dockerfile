# Stage 1 — build the dashboard SPA (Vite). Build-only; nothing from node
# reaches the runtime image.
FROM node:22-alpine AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2 — build static Go binaries with the dashboard embedded.
FROM golang:1.26 AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Stage the built SPA where go:embed expects it (internal/web/dist).
COPY --from=web /web/dist/ ./internal/web/dist/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/dietdaemon ./cmd/dietdaemon

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/tune ./cmd/tune

# Bulk food-database importer. Datasets are user-provided (mounted volumes),
# so only the binary ships in the image.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/import-foods ./cmd/import-foods

# Minimal HTTP liveness probe for distroless HEALTHCHECK.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/healthcheck ./cmd/healthcheck

# Pre-create /data owned by nonroot so named volumes inherit the
# correct permissions. Without this, Docker mounts an empty root-owned
# directory and the nonroot user (65532) gets SQLITE_CANTOPEN.
RUN mkdir -p /data && chmod 777 /data

# Stage 3 — minimal runtime
FROM gcr.io/distroless/static:nonroot
WORKDIR /

COPY --from=builder /bin/dietdaemon   /bin/dietdaemon
COPY --from=builder /bin/tune          /bin/tune
COPY --from=builder /bin/import-foods  /bin/import-foods
COPY --from=builder /bin/healthcheck   /bin/healthcheck

# Seed /data with nonroot ownership. When a named volume is mounted
# here, Docker copies the owner (65532) from the image to the volume.
COPY --from=builder --chown=65532:65532 /data /data

EXPOSE 8080

# File-based liveness probe: main process touches /data/healthy every 5s.
# Works regardless of ENABLE_DASHBOARD (bot-only vs full-stack).
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/bin/healthcheck"]

ENTRYPOINT ["/bin/dietdaemon"]
