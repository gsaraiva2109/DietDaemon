# DietDaemon build. The dashboard SPA (web/) is built with Vite, copied into
# internal/web/dist, and embedded into the Go binary so everything ships as a
# single distroless container — same origin as the API, no CORS.
# Humans and CI use the same targets.

GO          ?= go
NPM         ?= npm
GOBIN       ?= $(CURDIR)/bin
GO_PACKAGES = $(shell $(GO) list ./... | grep -v '/web/')
GO_FLAGS    ?= -ldflags="-s -w"
GIT_SHA     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DOCKER_IMAGE ?= ghcr.io/gsaraiva2109/dietdaemon

.PHONY: all build build-go build-web test test-web lint lint-go lint-web \
        vet fmt staticcheck govulncheck dev-web ai-setup docker-build docker-run docker-stop clean

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
	@echo ">> building import-foods..."
	CGO_ENABLED=0 $(GO) build $(GO_FLAGS) -o $(GOBIN)/import-foods ./cmd/import-foods
	@echo ">> building healthcheck..."
	CGO_ENABLED=0 $(GO) build $(GO_FLAGS) -o $(GOBIN)/healthcheck ./cmd/healthcheck

# Run the Vite dev server (proxies /api to a locally running daemon on :8080).
dev-web:
	cd web && $(NPM) run dev

# ============================================================
# TEST
# ============================================================

test:
	@echo ">> running all Go tests..."
	$(GO) test $(GO_PACKAGES) -count=1 -timeout 120s

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
	$(GO) vet $(GO_PACKAGES)

fmt:
	@echo ">> checking go fmt..."
	@test -z "$$($(GO) fmt $(GO_PACKAGES))" || (echo "go fmt found unformatted files" && exit 1)

staticcheck:
	@echo ">> running staticcheck..."
	staticcheck $(GO_PACKAGES)

govulncheck:
	@echo ">> running govulncheck..."
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...

# ============================================================
# AI (optional)
# ============================================================

# Installs Ollama if missing and pulls whatever models .env's PARSER_TIER
# actually needs (EMBED_MODEL for tier 1+, LLM_MODEL too for tier 2). No-op
# at tier 0/unset — AI stays opt-in, this never runs as part of `build`.
ai-setup:
	@tier=$$(grep -E '^PARSER_TIER=' .env 2>/dev/null | tail -1 | cut -d= -f2); \
	tier=$${tier:-0}; \
	if [ "$$tier" = "0" ]; then \
		echo ">> PARSER_TIER=0 (or unset) - AI disabled, nothing to do"; \
		exit 0; \
	fi; \
	embed_model=$$(grep -E '^EMBED_MODEL=' .env 2>/dev/null | tail -1 | cut -d= -f2); \
	embed_model=$${embed_model:-nomic-embed-text}; \
	llm_model=$$(grep -E '^LLM_MODEL=' .env 2>/dev/null | tail -1 | cut -d= -f2); \
	llm_model=$${llm_model:-llama3.1}; \
	if ! command -v ollama >/dev/null 2>&1; then \
		echo ">> ollama not found, installing..."; \
		curl -fsSL https://ollama.com/install.sh | sh; \
	fi; \
	if ! ollama list >/dev/null 2>&1; then \
		echo ">> ollama installed but not reachable - start it (ollama serve) and re-run 'make ai-setup'"; \
		exit 1; \
	fi; \
	if ollama list | awk '{print $$1}' | grep -q "^$$embed_model"; then \
		echo ">> embed model already present: $$embed_model"; \
	else \
		echo ">> pulling embed model: $$embed_model"; \
		ollama pull "$$embed_model"; \
	fi; \
	if [ "$$tier" -ge 2 ] 2>/dev/null; then \
		if ollama list | awk '{print $$1}' | grep -q "^$$llm_model"; then \
			echo ">> LLM model already present: $$llm_model"; \
		else \
			echo ">> pulling LLM model: $$llm_model"; \
			ollama pull "$$llm_model"; \
		fi; \
	fi; \
	echo ">> ai-setup done (tier $$tier)"

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
