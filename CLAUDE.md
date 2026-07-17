## graphify

This project has a graphify knowledge graph at graphify-out/.

Rules:
- Before answering architecture or codebase questions, read graphify-out/GRAPH_REPORT.md for god nodes and community structure
- If graphify-out/wiki/index.md exists, navigate it instead of reading raw files
- For cross-module "how does X relate to Y" questions, prefer `graphify query "<question>"`, `graphify path "<A>" "<B>"`, or `graphify explain "<concept>"` over grep — these traverse the graph's EXTRACTED + INFERRED edges instead of scanning files
- After modifying code files in this session, run `graphify update .` to keep the graph current (AST-only, no API cost)

## Backend validation

Run the Make targets instead of raw `go ... ./...` commands; they exclude `web/`, which is frontend-only and contains vendored Go fixtures.

- `make test` — Go tests
- `make vet` — Go vet
- `make fmt` — formatting check
- `make staticcheck` — Go static analysis (required in PR CI; pre-commit runs it locally on Go changes)
- `make govulncheck` — Go vulnerability scan (full module, incl. `web/`, for accurate reachability)
