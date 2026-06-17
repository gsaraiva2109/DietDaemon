# Stage 1 — build static binary
FROM golang:1.26 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/dietdaemon ./cmd/dietdaemon

# Stage 2 — minimal runtime
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /bin/dietdaemon /bin/dietdaemon

EXPOSE 8080

ENTRYPOINT ["/bin/dietdaemon"]
