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

# Stage 3 — minimal runtime
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /bin/dietdaemon /bin/dietdaemon
COPY --from=builder /bin/tune        /bin/tune

EXPOSE 8080

ENTRYPOINT ["/bin/dietdaemon"]
