# Contributing

## Setup

```bash
# Prerequisites: Go 1.26+, Node 22+, Docker (optional)
git clone https://github.com/gsaraiva2109/dietdaemon.git
cd dietdaemon

# Backend
cp .env.example .env
go mod download
go build ./cmd/dietdaemon

# Frontend (if working on web UI)
cd web
npm ci
npm run build    # produces dist/ that Go embeds

# Local checks, including Conventional Commit validation
go install honnef.co/go/tools/cmd/staticcheck@2026.1
pre-commit install
```

Use `docker compose up -d` for a full-stack dev environment with PostgreSQL.

## Branch and commit conventions

**Branches:** `feat/<slug>`, `fix/<slug>`, `refactor/<slug>`, `chore/<slug>`, `docs/<slug>`, `ci/<slug>`

**Commits:** [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)

```
feat(parser): add TACO food database support
fix(chat): lazy session creation stops empty rows
refactor(db): migrate to sqlx
chore: bump Go to 1.26.4
```

`pre-commit install` also installs a `commit-msg` hook that rejects subjects
outside this format. GitHub release notes group merged PRs by their labels, so
keep PR titles equally clear and add the relevant label:

| Release section | PR label                             |
|-----------------|--------------------------------------|
| 🔒 Security     | `security`                           |
| ✨ Enhancements  | `enhancement`                        |
| 🐛 Bug fixes    | `bug`                                |
| ⚡ Performance   | `performance`                        |
| 🛠 Maintenance  | `documentation`, `refactor`, `tests` |

GitHub generates the release body from those PRs, including **New Contributors**
only when a first-time contributor is part of that release.

## Architecture

DietDaemon has two parsing tiers and no LLM dependency by default:

| Tier | Parser                                    | Requirements             |
|------|-------------------------------------------|--------------------------|
| 0    | Deterministic tokenizer + unit dictionary | None                     |
| 1    | Embeddings via Ollama sidecar             | Ollama                   |
| 2    | Full LLM via Ollama sidecar               | Ollama + GPU recommended |

Core paths:

- `cmd/` — entry points (`dietdaemon` server, `tune` config tool)
- `core/` — business logic, meal parsing, macro resolution
- `internal/` — HTTP handlers, middleware, web embed
- `adapters/` — chat adapters (Telegram, Discord, Matrix)
- `web/` — TypeScript/React dashboard
- `migrations/` — PostgreSQL schema migrations
- `data/` — food database fixtures

Tier 0 must always work. AI features are behind feature flags and must never
block core meal-logging flow.

## Testing

```bash
go test ./... -count=1
```

Tests use a real PostgreSQL connection via `TEST_DATABASE_URL` (see `.env.example`).
Fixtures live in `fixtures/`.

Run `make staticcheck` and `make govulncheck` before pushing — CI treats both as required.

## PR workflow

1. Branch off `main`
2. Make changes, keep commits clean
3. Push — CI runs Go lint/test/build/staticcheck + govulncheck + frontend lint/tsc/build
4. Use the PR template, link the issue, add screenshots if UI changed
5. Merge when checks pass and review is done

## Chat channels

For questions, ideas, or help: open a [discussion](https://github.com/gsaraiva2109/dietdaemon/discussions).
