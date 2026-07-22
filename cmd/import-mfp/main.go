// Command import-mfp is a one-shot import of a MyFitnessPal "Nutrition
// Diary" CSV export into DietDaemon. Run it once per user, ever — there is
// no ongoing sync. MFP rows are date-only (no timestamp), so -tz plus a
// fixed meal-slot-to-time-of-day mapping fills in a logged_at.
//
// Usage:
//
//	go run ./cmd/import-mfp -user <user-id> -csv diary.csv -db ./data/dietdaemon.db
//	go run ./cmd/import-mfp -user <user-id> -csv diary.csv -db ./data/dietdaemon.db -dry-run
//	go run ./cmd/import-mfp -user <user-id> -csv diary.csv -db ./data/dietdaemon.db -tz America/New_York
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/importers/mfp"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

func main() {
	userID := flag.String("user", "", "user ID to import meals for (required)")
	csvPath := flag.String("csv", "", "path to MyFitnessPal nutrition diary CSV export (required)")
	dbPath := flag.String("db", "", "SQLite database path (required)")
	dryRun := flag.Bool("dry-run", false, "parse and count meals without writing to the store")
	tz := flag.String("tz", "UTC", "IANA timezone name used to convert MFP's date-only rows to a logged_at time")
	flag.Parse()

	if *userID == "" || *csvPath == "" || *dbPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	// A large diary export can take a moment to write; let ctrl-c stop
	// cleanly rather than killing the process mid-import, matching
	// cmd/import-foods.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *userID, *csvPath, *dbPath, *tz, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "import-mfp: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, userID, csvPath, dbPath, tz string, dryRun bool) error {
	// #nosec G304 -- path provided by operator at CLI, intentional file read
	f, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("open csv: %w", err)
	}
	defer func() { _ = f.Close() }()

	rows, err := mfp.ParseCSV(f)
	if err != nil {
		return fmt.Errorf("parse csv: %w", err)
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return fmt.Errorf("load timezone %q: %w", tz, err)
	}

	meals, err := groupIntoMeals(userID, rows, loc)
	if err != nil {
		return fmt.Errorf("build meals: %w", err)
	}

	itemCount := 0
	for _, m := range meals {
		itemCount += len(m.Items)
	}

	if dryRun {
		fmt.Printf("import-mfp: dry_run=true meals=%d items=%d\n", len(meals), itemCount)
		return nil
	}

	st, err := store.New("sqlite", dbPath, store.SQLiteDialect(), nil)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() {
		if cerr := st.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "import-mfp: close store: %v\n", cerr)
		}
	}()

	imported, err := importMeals(ctx, st, meals)
	if err != nil {
		return err
	}

	// imported counts calls that returned no error, which includes rows
	// SaveMeal silently no-opped as duplicates (same limitation as
	// ImportWorkout) — it's a success count, not a distinct-insert count.
	fmt.Printf("import-mfp: dry_run=false meals=%d items=%d imported=%d\n", len(meals), itemCount, imported)
	return nil
}

// mealSaver is the subset of *store.Store that importMeals needs, so tests
// can swap in a real temp store without depending on the rest of ports.Store.
type mealSaver interface {
	SaveMeal(ctx context.Context, m types.Meal) error
}

// importMeals writes each meal via SaveMeal, which no-ops on a duplicate
// external_id (see internal/store/store_meals.go), making a re-run of the
// same export safe. SaveMeal doesn't report whether a given call inserted or
// no-opped, so "imported" here just means "called without error" — the
// skipped-duplicates count in run's log line is inferred from the total, not
// measured per-row.
func importMeals(ctx context.Context, st mealSaver, meals []types.Meal) (int, error) {
	imported := 0
	for _, m := range meals {
		if ctx.Err() != nil {
			return imported, ctx.Err()
		}
		if err := st.SaveMeal(ctx, m); err != nil {
			return imported, fmt.Errorf("save meal %s: %w", *m.ExternalID, err)
		}
		imported++
	}
	return imported, nil
}

// groupIntoMeals groups CSV rows by (Date, Meal slot) — MFP logs each food
// as its own row, but DietDaemon logs one meal with multiple items — and
// converts each group into one types.Meal. Group order follows each group's
// first appearance in rows, so output order is deterministic.
func groupIntoMeals(userID string, rows []mfp.Row, loc *time.Location) ([]types.Meal, error) {
	type key struct{ date, slot string }
	var order []key
	grouped := make(map[key][]mfp.Row)

	for _, row := range rows {
		k := key{date: row.Date, slot: row.Meal}
		if _, seen := grouped[k]; !seen {
			order = append(order, k)
		}
		grouped[k] = append(grouped[k], row)
	}

	now := time.Now().UTC()
	meals := make([]types.Meal, 0, len(order))
	for _, k := range order {
		at, err := mealTime(k.date, k.slot, loc)
		if err != nil {
			return nil, err
		}

		group := grouped[k]
		items := make([]types.ResolvedItem, 0, len(group))
		for _, row := range group {
			items = append(items, mfp.ToItem(row))
		}

		meals = append(meals, types.Meal{
			ID:         newImportID(),
			UserID:     userID,
			At:         at,
			RawText:    fmt.Sprintf("MyFitnessPal import: %s, %s", k.slot, k.date),
			Items:      items,
			Confidence: 1,
			ParserTier: types.TierDeterministic,
			CreatedAt:  now,
			ExternalID: new("mfp:" + k.date + ":" + strings.ToLower(strings.TrimSpace(k.slot))),
		})
	}
	return meals, nil
}

// mealSlotTimes maps a normalized MFP meal slot name to a representative
// time of day. MFP diary rows carry no timestamp, only a date and a slot
// name, so this is the best available substitute for a real logged_at.
var mealSlotTimes = map[string][2]int{
	"breakfast": {8, 0},
	"lunch":     {12, 0},
	"dinner":    {18, 0},
	"snack":     {15, 0},
	"snacks":    {15, 0},
}

// mfpDateLayouts are the date formats MFP is known to export, tried in
// order. MFP's export locale affects which one applies.
var mfpDateLayouts = []string{"2006-01-02", "01/02/2006", "1/2/2006"}

// mealTime combines an MFP row's date and meal slot into a UTC timestamp,
// using mealSlotTimes for the time-of-day and slot 12:00 as the fallback for
// an unrecognized slot name (e.g. a custom MFP meal name).
func mealTime(date, slot string, loc *time.Location) (time.Time, error) {
	var d time.Time
	var err error
	for _, layout := range mfpDateLayouts {
		d, err = time.ParseInLocation(layout, date, loc)
		if err == nil {
			break
		}
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("unrecognized date %q (tried %v)", date, mfpDateLayouts)
	}

	hm, ok := mealSlotTimes[strings.ToLower(strings.TrimSpace(slot))]
	if !ok {
		hm = [2]int{12, 0}
	}
	return time.Date(d.Year(), d.Month(), d.Day(), hm[0], hm[1], 0, 0, loc).UTC(), nil
}

// importIDCounter guarantees distinct IDs even for meals generated within
// the same nanosecond in a tight loop.
var importIDCounter int64

func newImportID() string {
	n := atomic.AddInt64(&importIDCounter, 1)
	return strconv.FormatInt(time.Now().UnixNano(), 36) + strconv.FormatInt(n, 36)
}
