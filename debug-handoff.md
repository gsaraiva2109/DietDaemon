# TACO Data Corruption Debug — Handoff

**Date:** 2026-07-20
**Symptom:** Dashboard shows wrong macro values for TACO foods
**Example:** "Frango grelhado" shows `371/1550/10.3/2/10.6` (protein/carbs/fat/fiber/kcal), database has correct `31/0/3.6/0/165`

## Root Cause (confirmed)
Alpha.2 TACO parser had column-shift bug (#111): when pointed at the official TACO spreadsheet (XLSX with moisture%, kJ between columns), it parsed through the simple-CSV layout — moisture%→kcal, kcal→protein, kJ→carbs, protein→fat, fat→fiber. Alpha.2 wrote rows with bare numeric `food_id` ("41") and shuffled macros.

Alpha.3 parser is fixed and produces `food_id: "TACO041"` (prefixed). `BulkUpsertFoods` uses `ON CONFLICT(food_id) DO UPDATE` — different PKs ("41" vs "TACO041") → stale row never overwritten.

## What We Debugged

### Step 1 — Identified container version mismatch
- `docker inspect dietdaemon --format '{{.Config.Image}}'` → `v0.2.0-alpha.2` (Jul 18)
- Container was stale, alpha.2 had the buggy parser

### Step 2 — Redeployed via Dokploy
- User redeployed to alpha.3
- API still returned phantom `food_id: "41"` with wrong macros

### Step 3 — Attempted repair via import-foods
- Ran `import-foods -source taco -repair-macros` with docker run + dummy env vars
- Result: `rows_checked=597 rows_fixed=0`
- Repair matches by `(source, name)` — but the TACO food names in the DB may not exactly match the fresh import names, OR there were no stale rows to fix at that point (maybe they were already gone from prior reimport)

### Step 4 — Sqlite3 direct DB inspection
- Queried foods table: only found `TACO041` with CORRECT values (371/10.3/76.6/2.0/2.3)
- Searched for `food_id` exactly "41": returned nothing
- Searched for `food_id` with length < 5 (bare numeric IDs): returned nothing
- Confirmed DB file inode (7764106) — same file the daemon reads
- **Paradox:** DB had only correct data, API still returned phantom "41" row

### Step 5 — Checked for in-memory DB or copy
- `lsof` on the DB file: only one process, same inode
- No WAL/SHM files suggesting a different reader

### Step 6 — Verified TACO parser fix
- Examined `adapters/nutrition/taco/taco.go` — parser dispatch: `parseRows` → `rowsToFoods` (simple CSV) or `officialRowsToFoods` (official XLSX with column detection)
- PR #112 fixed the column mapping for official layout
- Test `TestOfficialSpreadsheetLayout` confirms correct parsing: 606/22.5/18.7/54/7.8 for Amendoim
- Embedded CSV `taco.csv` has 597 entries with `TACOxxx` IDs

### Step 7 — Verified container image update
- `docker inspect` confirmed the running container is alpha.3

### Step 8 — Nuked volume + reimport
- `docker volume rm dietdaemon_data`
- Re-imported TACO with alpha.3: `import-foods -source taco -db /data/dietdaemon.db`
- Container started successfully

### Step 9 — STILL BROKEN
- Dashboard still shows wrong macros after volume nuke + reimport + container restart

## Key Code Paths

### TACO parser
- `adapters/nutrition/taco/taco.go`: `New(path)` → detects CSV vs XLSX → `parseRows` → dispatches to `rowsToFoods` (simple) or `officialRowsToFoods` (official with moisture%/kJ columns)
- Embedded CSV: `adapters/nutrition/taco/taco.csv` (597 rows, `TACOxxx` IDs)
- Alpha.2 bug: `officialRowsToFoods` didn't exist yet — everything went through `rowsToFoods` which assumed simple 7-column CSV layout

### BulkUpsertFoods
- `internal/store/store_food_bulk.go:31-67`
- Uses `ON CONFLICT(food_id) DO UPDATE` — different PKs → new row inserted, stale row stays
- Has `plausibleMacros` guard — rejects rows where macros look corrupted (e.g., 606 protein for 2 kcal food)
- If alpha.3's `plausibleMacros` rejects the stale "41" row's values, the stale row would NEVER be touched

### RepairFoodMacros
- `internal/store/store_food_bulk.go:78-101`
- Matches by `(source, name)`, not `food_id`
- Designed exactly for this scenario — fixing rows unreachable by ON CONFLICT

### SearchCatalog
- `internal/store/store_food.go:467-516`
- `SELECT f.food_id, f.name, f.source, f.kcal_100g... FROM foods f LEFT JOIN user_food_stats...`
- Returns ALL foods, including any stale rows with bare numeric IDs

### import-foods CLI
- `cmd/import-foods/main.go`
- `config.Load()` requires full daemon env vars (dummy vars needed for docker run)
- Binary exists in distroless image at `/bin/import-foods`
- Must use `--entrypoint /bin/import-foods` to override daemon entrypoint

## Unresolved Questions

1. **Why does repair return `rows_fixed=0`?** The `RepairFoodMacros` query is `UPDATE foods SET ... WHERE source = ? AND name = ? AND owner_user_id IS NULL`. If the stale row has accented names ("Feijão") and the fresh import uses different accent normalization, they don't match. OR the stale row was already deleted by a prior operation.

2. **After volume nuke, where does the wrong data come from?** If volume was truly deleted and recreated, alpha.3 should write only correct data. Could the daemon be reading from a DIFFERENT DB path than the volume?

3. **Could there be a second DB copy?** The daemon might create a DB at a different path (not `/data/dietdaemon.db`). Check `config.Load()` DB path resolution.

4. **Could the frontend be caching?** Browser cache, service worker, or API response cache returning stale results from alpha.2 era.

5. **Was the volume actually deleted?** Dokploy might recreate volumes or have volume name mapping that differs from `dietdaemon_data`.

## Suggested Next Steps

1. **Verify DB path the daemon actually uses:**
   ```bash
   docker inspect dietdaemon --format '{{range .Mounts}}{{.Source}} -> {{.Destination}}{{"\n"}}{{end}}'
   ```

2. **Check ALL databases on the system:**
   ```bash
   find / -name "*.db" -newer /tmp 2>/dev/null
   ```

3. **Query the actual DB file the daemon reads:**
   ```bash
   # Get the actual mounted volume path first (step 1), then:
   sqlite3 <actual-path> "SELECT food_id, name, kcal_100g, protein_100g FROM foods WHERE source='taco' ORDER BY length(food_id) LIMIT 5;"
   ```

4. **Check for browser cache:**
   - Open DevTools → Network tab → check "Disable cache"
   - Hard reload (Ctrl+Shift+R)
   - Check the raw API response for `food_id: "41"`

5. **Check if import-foods actually wrote anything:**
   ```bash
   docker run --rm \
     -v dietdaemon_data:/data \
     -e MESSAGING_ADAPTER=telegram \
     -e TELEGRAM_BOT_TOKEN=dummy \
     -e NUTRITION_SOURCE=openfoodfacts \
     -e EMBED_ADAPTER=ollama \
     -e ENABLE_NOTIFICATIONS=false \
     --entrypoint /bin/import-foods \
     ghcr.io/gsaraiva2109/dietdaemon:v0.2.0-alpha.3 \
     -source taco -db /data/dietdaemon.db
   ```
   Expected output: `import-foods: source=taco dry_run=false rows=597`

6. **Dump ENTIRE foods table after import:**
   ```bash
   docker run --rm -v dietdaemon_data:/data alpine sh -c 'apk add --quiet sqlite && sqlite3 /data/dietdaemon.db "SELECT count(*), source FROM foods GROUP BY source;"'
   ```

## Files Involved
- `internal/store/store_food.go` — GetFoodForUser, SearchCatalog, foodMatchRow
- `internal/store/store_food_bulk.go` — BulkUpsertFoods, RepairFoodMacros, plausibleMacros
- `adapters/nutrition/taco/taco.go` — parser, parseRows, rowsToFoods, officialRowsToFoods
- `adapters/nutrition/taco/taco.csv` — embedded 597-row TACO catalog
- `cmd/import-foods/main.go` — import/repair/backfill CLI
- `internal/config/config.go` — Load(), DBPath resolution
- `Dockerfile` — 3-stage build, distroless runtime
- `docker-compose.yml` — `dietdaemon_data:/data` volume mount
