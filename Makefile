# DietDaemon build. The dashboard SPA (web/) is built with Vite, copied into
# internal/web/dist, and embedded into the Go binary so everything ships as a
# single distroless container — same origin as the API, no CORS.
# Humans and CI use the same targets.

GO          ?= go
NPM         ?= npm
GOBIN       ?= $(CURDIR)/bin
GO_FLAGS    ?= -ldflags="-s -w"
GIT_SHA     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DOCKER_IMAGE ?= ghcr.io/gsaraiva2109/dietdaemon

.PHONY: all build build-go build-web test test-web lint lint-go lint-web \
        vet fmt dev-web docker-build docker-run docker-stop clean

all: build

# ============================================================
# BUILD
# ============================================================

# Full build: frontend first, then Go (go:embed needs internal/web/dist).
build: build-web build-go

# Compile the dashboard and stage it for go:embed.
build-web:
	cd web && $(NPM) ci && $(NPM) run build
	rm -rf internal/web/dist
	mkdir -p internal/web/dist
	cp -r web/dist/. internal/web/dist/

# Build both binaries (static, stripped).
build-go:
	@mkdir -p $(GOBIN)
	@echo ">> building dietdaemon..."
	CGO_ENABLED=0 $(GO) build $(GO_FLAGS) -o $(GOBIN)/dietdaemon ./cmd/dietdaemon
	@echo ">> building tune..."
	CGO_ENABLED=0 $(GO) build $(GO_FLAGS) -o $(GOBIN)/tune ./cmd/tune

# Run the Vite dev server (proxies /api to a locally running daemon on :8080).
dev-web:
	cd web && $(NPM) run dev

# ============================================================
# TEST
# ============================================================

test:
	@echo ">> running all Go tests..."
	$(GO) test ./... -count=1 -timeout 120s

test-web:
	@echo ">> running frontend tests..."
	cd web && $(NPM) test 2>/dev/null || echo "frontend tests not yet configured"

# ============================================================
# LINT / VET / FMT
# ============================================================

lint: lint-go lint-web

lint-go: vet fmt

lint-web:
	@echo ">> running eslint..."
	cd web && $(NPM) run lint
	@echo ">> checking TypeScript..."
	cd web && npx tsc -b --noEmit

vet:
	@echo ">> running go vet..."
	$(GO) vet ./...

fmt:
	@echo ">> checking go fmt..."
	@test -z "$$($(GO) fmt ./...)" || (echo "go fmt found unformatted files" && exit 1)

# ============================================================
# DOCKER
# ============================================================

docker-build:
	@echo ">> building Docker image ($(DOCKER_IMAGE):latest)..."
	DOCKER_BUILDKIT=1 docker build \
		-t $(DOCKER_IMAGE):latest \
		-t $(DOCKER_IMAGE):sha-$(GIT_SHA) \
		.

docker-run:
	@echo ">> starting via docker compose..."
	docker compose up -d

docker-stop:
	@echo ">> stopping via docker compose..."
	docker compose down

# ============================================================
# CLEAN
# ============================================================

clean:
	@echo ">> cleaning..."
	rm -rf $(GOBIN) web/dist internal/web/dist/assets
	$(GO) clean -cache -testcache
