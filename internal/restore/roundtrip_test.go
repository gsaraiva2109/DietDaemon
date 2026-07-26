package restore

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/backup"
	"github.com/gsaraiva2109/dietdaemon/internal/backup/localdisk"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

// openTestStore opens a fresh SQLite-backed *store.Store at path, running
// migrations. Mirrors internal/store's own (unexported) tempDB test helper,
// duplicated here since that package can't be imported for its test-only
// symbols.
func openTestStore(t *testing.T, path string) *store.Store {
	t.Helper()
	s, err := store.New("sqlite", path, store.SQLiteDialect(), nil)
	if err != nil {
		t.Fatalf("open store %s: %v", path, err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// TestBackupRestoreRoundTrip pins the #189 acceptance bar directly: build a
// plan with 2 day-types, slots, and options in a source database, back it
// up, restore into a completely fresh database, and confirm GetPlanBundle
// there returns an identical structure -- including the day override and
// the plan-owned meal_templates rows the options reference (a restore that
// skipped those would leave slot_options.template_id dangling, since foreign
// keys are enforced).
func TestBackupRestoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	src := openTestStore(t, filepath.Join(dir, "source.db"))
	if err := src.UpsertUser(ctx, types.User{ID: "u1", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("seed source user: %v", err)
	}
	if err := src.SetBackupConfig(ctx, types.BackupConfig{
		UserID: "u1", Enabled: true, Destination: "local", LocalSubdir: "u1",
	}); err != nil {
		t.Fatalf("set backup config: %v", err)
	}

	plan, err := src.CreatePlan(ctx, types.DietPlan{
		UserID: "u1", Name: "Cutting cycle", Notes: "from Dra. Ana",
		ValidFrom: "2026-01-05", CyclePattern: []string{"low", "high"}, CycleAnchorDate: "2026-01-05",
	})
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}

	if err := src.SaveTemplate(ctx, types.MealTemplate{
		ID: "tmpl-oats", UserID: "u1", Name: "Aveia",
		Items:     []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "aveia", Quantity: 50, Unit: "g"}, Macros: types.Macros{Calories: 190, Carbs: 33}}},
		CreatedAt: time.Now().UTC(), LastUsed: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed template oats: %v", err)
	}
	if err := src.SaveTemplate(ctx, types.MealTemplate{
		ID: "tmpl-eggs", UserID: "u1", Name: "Ovos", CreatedAt: time.Now().UTC(), LastUsed: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("seed template eggs: %v", err)
	}

	dtLow, err := src.CreateDayType(ctx, types.DietPlanDayType{
		PlanID: plan.ID, Name: "Low-carb", Position: 0,
		Targets: types.Macros{Calories: 1800, Protein: 150, Carbs: 100, Fat: 60, Fiber: 25}, WaterGoalMl: 3000,
	})
	if err != nil {
		t.Fatalf("CreateDayType low: %v", err)
	}
	dtHigh, err := src.CreateDayType(ctx, types.DietPlanDayType{
		PlanID: plan.ID, Name: "High-carb", Position: 1,
		Targets: types.Macros{Calories: 2400, Protein: 150, Carbs: 300, Fat: 60, Fiber: 30}, WaterGoalMl: 3500,
	})
	if err != nil {
		t.Fatalf("CreateDayType high: %v", err)
	}

	slotBreakfast, err := src.CreateSlot(ctx, types.DietPlanSlot{DayTypeID: dtLow.ID, Position: 0, TimeOfDay: "07:00", Label: "Café da manhã"})
	if err != nil {
		t.Fatalf("CreateSlot breakfast: %v", err)
	}
	if _, err := src.CreateSlot(ctx, types.DietPlanSlot{DayTypeID: dtHigh.ID, Position: 0, TimeOfDay: "07:00", Label: "Café da manhã"}); err != nil {
		t.Fatalf("CreateSlot high breakfast: %v", err)
	}
	if _, err := src.CreateSlotOption(ctx, types.DietPlanSlotOption{SlotID: slotBreakfast.ID, Position: 0, Label: "Opção 1", TemplateID: "tmpl-oats"}); err != nil {
		t.Fatalf("CreateSlotOption 1: %v", err)
	}
	if _, err := src.CreateSlotOption(ctx, types.DietPlanSlotOption{SlotID: slotBreakfast.ID, Position: 1, Label: "Opção 2", TemplateID: "tmpl-eggs"}); err != nil {
		t.Fatalf("CreateSlotOption 2: %v", err)
	}

	if err := src.SetDayOverride(ctx, types.DietPlanDayOverride{UserID: "u1", Date: "2026-01-10", DayTypeID: dtHigh.ID}); err != nil {
		t.Fatalf("SetDayOverride: %v", err)
	}

	srcBundle, err := src.GetPlanBundle(ctx, plan.ID)
	if err != nil {
		t.Fatalf("GetPlanBundle (source): %v", err)
	}
	if len(srcBundle.DayTypes) != 2 || len(srcBundle.DayTypes[0].Slots) != 1 || len(srcBundle.DayTypes[0].Slots[0].Options) != 2 {
		t.Fatalf("source bundle shape unexpected: %+v", srcBundle)
	}

	// Back up the source database to local disk.
	dst, err := localdisk.New(dir)
	if err != nil {
		t.Fatalf("localdisk.New: %v", err)
	}
	if err := backup.New(src, dst, nil, time.Hour).RunOnce(ctx, "u1"); err != nil {
		t.Fatalf("backup RunOnce: %v", err)
	}

	// Restore into a completely fresh database. The user account itself
	// isn't part of the backup (it's provisioned through signup/auth, not
	// disaster recovery), so the target needs it seeded first, same as a
	// real restore onto a freshly reprovisioned account.
	target := openTestStore(t, filepath.Join(dir, "target.db"))
	if err := target.UpsertUser(ctx, types.User{ID: "u1", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("seed target user: %v", err)
	}
	cfg := types.BackupConfig{UserID: "u1", Destination: "local", LocalSubdir: "u1"}
	sum, err := New(target, dst).RunOnce(ctx, "u1", cfg)
	if err != nil {
		t.Fatalf("restore RunOnce: %v", err)
	}
	if sum.Plans != 1 || sum.DayTypes != 2 || sum.Slots != 2 || sum.SlotOptions != 2 || sum.DayOverrides != 1 || sum.Templates != 2 {
		t.Fatalf("unexpected restore summary: %+v", sum)
	}

	gotBundle, err := target.GetPlanBundle(ctx, plan.ID)
	if err != nil {
		t.Fatalf("GetPlanBundle (restored): %v", err)
	}

	if gotBundle.Plan.ID != srcBundle.Plan.ID || gotBundle.Plan.Name != srcBundle.Plan.Name ||
		gotBundle.Plan.Notes != srcBundle.Plan.Notes || gotBundle.Plan.ValidFrom != srcBundle.Plan.ValidFrom ||
		gotBundle.Plan.CycleAnchorDate != srcBundle.Plan.CycleAnchorDate ||
		!reflect.DeepEqual(gotBundle.Plan.CyclePattern, srcBundle.Plan.CyclePattern) {
		t.Fatalf("restored plan = %+v, want %+v", gotBundle.Plan, srcBundle.Plan)
	}
	if !gotBundle.Plan.CreatedAt.Equal(srcBundle.Plan.CreatedAt) || !gotBundle.Plan.UpdatedAt.Equal(srcBundle.Plan.UpdatedAt) {
		t.Fatalf("restored plan timestamps = %+v/%+v, want %+v/%+v",
			gotBundle.Plan.CreatedAt, gotBundle.Plan.UpdatedAt, srcBundle.Plan.CreatedAt, srcBundle.Plan.UpdatedAt)
	}
	if !reflect.DeepEqual(gotBundle.DayTypes, srcBundle.DayTypes) {
		t.Fatalf("restored day-type/slot/option tree = %+v, want %+v", gotBundle.DayTypes, srcBundle.DayTypes)
	}

	gotOverride, err := target.GetDayOverride(ctx, "u1", "2026-01-10")
	if err != nil || gotOverride.DayTypeID != dtHigh.ID {
		t.Fatalf("GetDayOverride (restored) = %+v, err = %v, want day type %s", gotOverride, err, dtHigh.ID)
	}

	// The plan-owned templates the options reference must exist in the
	// target too, or the round trip merely "succeeded" while quietly losing
	// the prescribed foods.
	gotTmpl, err := target.GetTemplate(ctx, "tmpl-oats")
	if err != nil {
		t.Fatalf("GetTemplate (restored): %v", err)
	}
	if gotTmpl.Name != "Aveia" || len(gotTmpl.Items) != 1 || gotTmpl.Items[0].Macros.Calories != 190 {
		t.Fatalf("restored template = %+v, want name Aveia with 1 item at 190 kcal", gotTmpl)
	}
}
