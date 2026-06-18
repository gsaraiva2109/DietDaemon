# DietDaemon — Code Analysis Report

**Date:** 2026-06-18  
**Scope:** Full project (Go backend + React/TypeScript frontend + SQLite)  
**Size:** ~13,853 Go LOC | ~7,750 TS/TSX LOC | ~273 SQL LOC | 28 test files / 35 source files  
**Focus:** Quality · Security · Performance · Architecture

---

## Executive Summary

DietDaemon is a well-architected, single-binary nutrition tracker using hexagonal architecture. Strong separation of concerns, clean domain types, and progressive parser tiers. The codebase is mature — zero TODOs, every package documented, compile-time interface checks. Primary concerns are the fragile migration system, error message leakage in the API, an O(n²) rolling-average algorithm, and auth being optional-by-default.

**Overall Grade: B+/A-** — solid production foundation with a few targeted fixes needed.

---

## 1. Architecture

### 1.1 Hexagonal Ports & Adapters ✅

```
core/ports/ports.go   ← Interfaces (Store, Parser, Resolver, Notifier, …)
adapters/              ← Concrete implementations (telegram, discord, ollama, …)
internal/              ← Business logic (pipeline, resolver, store, scheduler, api)
```

Compile-time interface satisfaction guards every boundary:

```go
var (
    _ ports.Store          = (*Store)(nil)
    _ scheduler.Store      = (*Store)(nil)
    _ scheduler.NudgeStore = (*Store)(nil)
)
```

**Assessment:** Exemplary. Adapters can be swapped, tested, and extended independently.

### 1.2 Parser Tier Strategy ✅

| Tier | Name | Mechanism | Model Required |
|------|------|-----------|----------------|
| 0 | Deterministic | Grammar + fuzzy alias match | No |
| 1 | Embedding | Cosine nearest-neighbor on food vectors | Embedding only |
| 2 | LLM | Generative extraction + embedding match | LLM + Embedding |

Hot-swappable via `PARSER_TIER` env var. Tier 0 works offline — good fallback.

### 1.3 Pipeline Pattern ✅

```
Message → [STT] → Parse (Stage A) → Resolve (Stage B) → Persist → Reply
                                                              ↓
                                                       PendingMeal loop
                                                       (clarification Q&A)
```

The pending-meal clarification loop is elegant: items that can't resolve enter a conversational Q&A before being committed. No silent macro guessing.

### 1.4 Multi-User from Day One ✅

All tables keyed by `user_id` even though single-user is the default. When `MULTI_USER=true` is set, channel-to-user mapping and token authentication activate without schema changes.

### 1.5 Frontend Architecture ✅

- React 18 + React Router + TanStack Query + Recharts + Framer Motion
- Lazy-loaded routes (code splitting by page)
- Auth gate pattern with token probe
- Custom design system (CSS custom properties, dark/light themes)
- TypeScript mirrors Go domain types with explicit JSON key contracts

---

## 2. Quality

### 2.1 Migration System — MEDIUM

**Issue:** Migrations run on every boot by reading all `.sql` files from the `migrations/` directory and executing them unconditionally. No version tracking.

```go
// internal/store/store.go:84-95
for _, entry := range entries {
    content, err := migrations.FS.ReadFile(entry.Name())
    _, err = s.db.Exec(string(content))
}
```

**Risks:**
- No rollback capability
- If a migration fails mid-way, DB is in unknown state
- Relies entirely on `IF NOT EXISTS` / `CREATE TABLE IF NOT EXISTS` for idempotency
- Adding a column with `ALTER TABLE … ADD COLUMN` will fail on second boot (not all migrations use IF NOT EXISTS pattern)

**Recommendation:** Add a `schema_version` table. Track applied migrations by filename hash. Only run unapplied ones.

### 2.2 Error Message Leakage — HIGH

**Issue:** Raw Go errors exposed to API clients in multiple places:

```go
// internal/api/handler.go:185
json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})

// internal/api/handler.go:319
json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})

// internal/api/handler.go:1271 — writeErr fallback
json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
```

**Impact:** Internal errors (DB path, SQL errors, stack traces) leak to API consumers. This is an information disclosure vulnerability.

**Recommendation:** Wrap internal errors with user-safe messages. Only include `err.Error()` for validation errors where the detail is intentional (e.g., "weight_kg must be positive").

### 2.3 Silent Time Parsing Failure — MEDIUM

```go
// internal/store/store.go:986-995
func parseUTC(s string) time.Time {
    if s == "" {
        return time.Time{}
    }
    t, err := time.Parse(time.RFC3339, s)
    if err != nil {
        return time.Time{}  // ← silent failure
    }
    return t.UTC()
}
```

**Impact:** Corrupted timestamps produce zero-value times, which propagate through queries and comparisons silently. Could produce wrong date-based rollup lookups.

**Recommendation:** Either log the parse error at minimum, or return an error. At the call sites, treat zero time as a sentinel.

### 2.4 Magic Defaults in Goal Suggestions — LOW

```go
// internal/api/handler.go:1083
var currentWeight float64 = 70  // hardcoded fallback

// internal/api/handler.go:1075
age := 30  // hardcoded fallback
```

**Impact:** Users without weight data or birth date get inaccurate TDEE calculations. The suggestions use these values silently.

**Recommendation:** Return a clear "insufficient data" response instead of computing with defaults.

### 2.5 TDEE Carbs Formula — HIGH

```go
// internal/api/handler.go:1259-1261
Fat:   tdee * 0.25 / 9,
Carbs: (tdee - (p.WeightKg*2.2*4 + tdee*0.25)) / 4,
```

**Issue:** The carbs formula subtracts `tdee*0.25` (raw calories from fat ≈500-750) but should subtract `Fat * 9` (the actual fat calories computed on line 1259). These are different values — `tdee*0.25` is ~500-750 calories while `Fat*9` = `(tdee*0.25/9)*9` = `tdee*0.25`, so they ARE equal. Actually let me re-check…

`Fat = tdee * 0.25 / 9` → fat grams.
`Fat * 9 = tdee * 0.25` → fat calories.

`Carbs = (tdee - (protein_cal + fat_cal)) / 4`
`protein_cal = weightKg * 2.2 * 4`
`fat_cal = tdee * 0.25`

So `Carbs = (tdee - weightKg*2.2*4 - tdee*0.25) / 4` — this IS correct mathematically. False alarm.

Wait, let me verify: `(tdee - (p.WeightKg*2.2*4 + tdee*0.25)) / 4` = `(tdee - protein_cal - fat_cal) / 4`. Yes, this is correct. Not a bug.

### 2.6 Test Coverage — LOW

28 test files for 35 source files = good file-level coverage. But I can't assess line coverage without running tests. The presence of contract tests in `tests/contract/` suggests good architectural testing discipline.

**Recommendation:** Add CI coverage report to `.github/workflows/`.

---

## 3. Security

### 3.1 Auth Optional by Default — MEDIUM

```go
// internal/api/handler.go:200-205
if h.authToken != "" {
    return h.authenticateStaticToken(r)
}
return "default", nil  // ← no auth required
```

**Impact:** When `API_AUTH_TOKEN` is unset and `MULTI_USER=false`, the entire API is open to anyone who can reach the port. In Docker deployments behind a reverse proxy, this may be intentional. But the default is permissive.

**Recommendation:** Log a warning at startup when no auth is configured. Consider requiring auth for non-localhost requests even in single-user mode.

### 3.2 Token in localStorage — LOW

```typescript
// web/src/lib/api.ts:23
const TOKEN_KEY = 'dd.token'
export function getToken(): string | null {
    return localStorage.getItem(TOKEN_KEY)
}
```

**Impact:** XSS vulnerability could leak the API token. This is the standard SPA tradeoff — acceptable for a self-hosted nutrition tracker.

**Recommendation:** Document the risk. Consider `httpOnly` cookie option for production deployments behind a reverse proxy.

### 3.3 No Rate Limiting — LOW

The API has no per-IP or per-user rate limiting. For a self-hosted tool this is low risk, but in multi-user deployments, one user could overwhelm the SQLite DB.

**Recommendation:** Add a simple token-bucket rate limiter middleware for multi-user mode.

### 3.4 File Upload Bounds — ✅

```go
// internal/api/handler.go:869-871
r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
if err := r.ParseMultipartForm(5 << 20); err != nil {
```

Photo uploads are correctly bounded at 5MB with `MaxBytesReader` before parsing. Good.

### 3.5 SQL Injection — ✅

All queries use parameterized placeholders (`?`). The only dynamic SQL is the `IN (?, ?, ?)` placeholder expansion in `loadItems`, which builds the parameter list programmatically — no user input concatenation.

---

## 4. Performance

### 4.1 WeightTrend O(n²) Rolling Average — MEDIUM

```go
// internal/store/store.go:1360-1381
for i, e := range entries {
    start := i - 6
    if start < 0 { start = 0 }
    sum := 0.0
    for j := start; j <= i; j++ {
        sum += entries[j].WeightKg
    }
    wt.RollingAvg = sum / float64(count)
}
```

**Impact:** For 365 entries, this does ~2,555 iterations instead of ~365 with a sliding window. Not catastrophic for this data size, but unnecessary.

**Recommendation:** Maintain a running sum in a 7-element ring buffer. O(n) single pass.

### 4.2 SQLite EXCLUSIVE Locking — NOTE

```go
// internal/store/store.go:52
db.Exec("PRAGMA locking_mode = EXCLUSIVE")
```

**Impact:** Single-writer by design. Correct for the personal-use case. Will bottleneck at ~10-20 concurrent API users. Intentional tradeoff documented in code comments.

### 4.3 Embedding Nearest-Neighbor via Application Code — LOW

The `SQLIndex.Nearest` method loads all vectors for a user and computes cosine similarity in Go. Fine for <1,000 foods per user, will degrade beyond ~10,000.

**Recommendation:** If food library grows large, consider SQLite's `vec0` extension or approximate nearest-neighbor.

### 4.4 Frontend Bundle Splitting — ✅

Routes are lazy-loaded via `React.lazy()`. Recharts (~300KB) only ships when Trends/Summary is visited. Good discipline.

### 4.5 React Query Stale Time — ✅

```typescript
queries: { staleTime: 15_000, retry: 1, refetchOnWindowFocus: false }
```

Sensible defaults for a dashboard. 15s stale time balances freshness with server load.

---

## 5. Code Quality Metrics

| Metric | Value | Grade |
|--------|-------|-------|
| Package documentation | 100% of packages have doc comments | A+ |
| Interface satisfaction checks | Compile-time `var _ Iface = (*Impl)(nil)` | A+ |
| Test file ratio | 28/35 (80%) | B+ |
| Error handling | Contextual wrapping with `%w` throughout | A |
| Dead code / TODOs | Zero found | A+ |
| Consistent naming | Idiomatic Go throughout | A |
| Configuration validation | Fail-fast, collects all problems | A |
| Godoc-quality comments | Every exported symbol documented | A |

---

## 6. Priority Recommendations

### High Priority

1. **Fix error message leakage in API** — Replace `err.Error()` in `writeErr` fallback with generic "internal server error". Keep detail only for validation errors.

2. **Add migration versioning** — Create `schema_versions` table. Track SHA256 of applied migrations. Skip already-applied files.

### Medium Priority

3. **Log warning on no-auth startup** — When `API_AUTH_TOKEN=""` and dashboard is enabled, log: `"dashboard running without authentication"`.

4. **Fix `parseUTC` silent failure** — Log parse errors. Return sentinel or error.

5. **Optimize `WeightTrend` to O(n)** — Sliding window instead of nested loop.

6. **Remove magic defaults from goal suggestions** — Return "insufficient data" when profile/weight data is missing.

### Low Priority

7. **Add CI test coverage report** — Integrate `go test -cover` into GitHub Actions.

8. **Add rate limiting middleware** — For multi-user mode.

9. **Document localStorage token risk** — Note in README or security docs.

---

## 7. What's Working Well

- **Hexagonal architecture** is clean, testable, and swappable
- **Parser tier system** is a novel progressive-enhancement approach
- **Pending meal clarification loop** prevents silent macro guessing — strong UX design
- **Compile-time interface checks** prevent regressions
- **Multi-user-ready schema** from day one — no future rewrite needed
- **Single-binary deployment** with embedded SPA — zero CORS, simple ops
- **Fail-fast config validation** — all problems reported at once
- **Good frontend code splitting** — lazy routes, tree-shakeable charts
- **Comprehensive domain types** — 6 phases of features all modeled cleanly
- **Docker multi-stage build** — distroless static, no CGO, small image
