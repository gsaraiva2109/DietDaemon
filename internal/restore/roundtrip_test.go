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

// ---------------------------------------------------------------------------
// must* helpers -- thin wrappers around the store/backup/restore calls that
// fail the test on error, keeping TestBackupRestoreRoundTrip's own body down
// to the sequence of operations under test rather than a wall of repeated
// error checks (each of which counts toward the cognitive-complexity budget
// the linter enforces on the test function itself).
// ---------------------------------------------------------------------------

func mustUpsertUser(t *testing.T, ctx context.Context, s *store.Store, id string) {
	t.Helper()
	if err := s.UpsertUser(ctx, types.User{ID: id, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("UpsertUser %s: %v", id, err)
	}
}

func mustSetBackupConfig(t *testing.T, ctx context.Context, s *store.Store, cfg types.BackupConfig) {
	t.Helper()
	if err := s.SetBackupConfig(ctx, cfg); err != nil {
		t.Fatalf("SetBackupConfig: %v", err)
	}
}

func mustCreatePlan(t *testing.T, ctx context.Context, s *store.Store, p types.DietPlan) types.DietPlan {
	t.Helper()
	created, err := s.CreatePlan(ctx, p)
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	return created
}

func mustSaveTemplate(t *testing.T, ctx context.Context, s *store.Store, tmpl types.MealTemplate) {
	t.Helper()
	if err := s.SaveTemplate(ctx, tmpl); err != nil {
		t.Fatalf("SaveTemplate %s: %v", tmpl.ID, err)
	}
}

func mustCreateDayType(t *testing.T, ctx context.Context, s *store.Store, dt types.DietPlanDayType) types.DietPlanDayType {
	t.Helper()
	created, err := s.CreateDayType(ctx, dt)
	if err != nil {
		t.Fatalf("CreateDayType %s: %v", dt.Name, err)
	}
	return created
}

func mustCreateSlot(t *testing.T, ctx context.Context, s *store.Store, sl types.DietPlanSlot) types.DietPlanSlot {
	t.Helper()
	created, err := s.CreateSlot(ctx, sl)
	if err != nil {
		t.Fatalf("CreateSlot: %v", err)
	}
	return created
}

func mustCreateSlotOption(t *testing.T, ctx context.Context, s *store.Store, opt types.DietPlanSlotOption) types.DietPlanSlotOption {
	t.Helper()
	created, err := s.CreateSlotOption(ctx, opt)
	if err != nil {
		t.Fatalf("CreateSlotOption: %v", err)
	}
	return created
}

func mustSetDayOverride(t *testing.T, ctx context.Context, s *store.Store, o types.DietPlanDayOverride) {
	t.Helper()
	if err := s.SetDayOverride(ctx, o); err != nil {
		t.Fatalf("SetDayOverride: %v", err)
	}
}

func mustGetPlanBundle(t *testing.T, ctx context.Context, s *store.Store, planID string) types.PlanBundle {
	t.Helper()
	bundle, err := s.GetPlanBundle(ctx, planID)
	if err != nil {
		t.Fatalf("GetPlanBundle: %v", err)
	}
	return bundle
}

func mustGetTemplate(t *testing.T, ctx context.Context, s *store.Store, id string) types.MealTemplate {
	t.Helper()
	tmpl, err := s.GetTemplate(ctx, id)
	if err != nil {
		t.Fatalf("GetTemplate %s: %v", id, err)
	}
	return tmpl
}

func mustLocalDisk(t *testing.T, dir string) *localdisk.Dest {
	t.Helper()
	dst, err := localdisk.New(dir)
	if err != nil {
		t.Fatalf("localdisk.New: %v", err)
	}
	return dst
}

func mustRunBackupOnce(t *testing.T, ctx context.Context, src *store.Store, dst *localdisk.Dest, userID string) {
	t.Helper()
	if err := backup.New(src, dst, nil, time.Hour).RunOnce(ctx, userID); err != nil {
		t.Fatalf("backup RunOnce: %v", err)
	}
}

func mustRunRestoreOnce(t *testing.T, ctx context.Context, target *store.Store, dst *localdisk.Dest, userID string, cfg types.BackupConfig) Summary {
	t.Helper()
	sum, err := New(target, dst).RunOnce(ctx, userID, cfg)
	if err != nil {
		t.Fatalf("restore RunOnce: %v", err)
	}
	return sum
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
	mustUpsertUser(t, ctx, src, "u1")
	mustSetBackupConfig(t, ctx, src, types.BackupConfig{
		UserID: "u1", Enabled: true, Destination: "local", LocalSubdir: "u1",
	})

	plan := mustCreatePlan(t, ctx, src, types.DietPlan{
		UserID: "u1", Name: "Cutting cycle", Notes: "from Dra. Ana",
		ValidFrom: "2026-01-05", CyclePattern: []string{"low", "high"}, CycleAnchorDate: "2026-01-05",
	})

	mustSaveTemplate(t, ctx, src, types.MealTemplate{
		ID: "tmpl-oats", UserID: "u1", Name: "Aveia",
		Items:     []types.ResolvedItem{{Parsed: types.ParsedItem{RawPhrase: "aveia", Quantity: 50, Unit: "g"}, Macros: types.Macros{Calories: 190, Carbs: 33}}},
		CreatedAt: time.Now().UTC(), LastUsed: time.Now().UTC(),
	})
	mustSaveTemplate(t, ctx, src, types.MealTemplate{
		ID: "tmpl-eggs", UserID: "u1", Name: "Ovos", CreatedAt: time.Now().UTC(), LastUsed: time.Now().UTC(),
	})

	dtLow := mustCreateDayType(t, ctx, src, types.DietPlanDayType{
		PlanID: plan.ID, Name: "Low-carb", Position: 0,
		Targets: types.Macros{Calories: 1800, Protein: 150, Carbs: 100, Fat: 60, Fiber: 25}, WaterGoalMl: 3000,
	})
	dtHigh := mustCreateDayType(t, ctx, src, types.DietPlanDayType{
		PlanID: plan.ID, Name: "High-carb", Position: 1,
		Targets: types.Macros{Calories: 2400, Protein: 150, Carbs: 300, Fat: 60, Fiber: 30}, WaterGoalMl: 3500,
	})

	slotBreakfast := mustCreateSlot(t, ctx, src, types.DietPlanSlot{DayTypeID: dtLow.ID, Position: 0, TimeOfDay: "07:00", Label: "Café da manhã"})
	mustCreateSlot(t, ctx, src, types.DietPlanSlot{DayTypeID: dtHigh.ID, Position: 0, TimeOfDay: "07:00", Label: "Café da manhã"})
	mustCreateSlotOption(t, ctx, src, types.DietPlanSlotOption{SlotID: slotBreakfast.ID, Position: 0, Label: "Opção 1", TemplateID: "tmpl-oats"})
	mustCreateSlotOption(t, ctx, src, types.DietPlanSlotOption{SlotID: slotBreakfast.ID, Position: 1, Label: "Opção 2", TemplateID: "tmpl-eggs"})

	mustSetDayOverride(t, ctx, src, types.DietPlanDayOverride{UserID: "u1", Date: "2026-01-10", DayTypeID: dtHigh.ID})

	srcBundle := mustGetPlanBundle(t, ctx, src, plan.ID)
	if len(srcBundle.DayTypes) != 2 || len(srcBundle.DayTypes[0].Slots) != 1 || len(srcBundle.DayTypes[0].Slots[0].Options) != 2 {
		t.Fatalf("source bundle shape unexpected: %+v", srcBundle)
	}

	// Back up the source database to local disk.
	dst := mustLocalDisk(t, dir)
	mustRunBackupOnce(t, ctx, src, dst, "u1")

	// Restore into a completely fresh database. The user account itself
	// isn't part of the backup (it's provisioned through signup/auth, not
	// disaster recovery), so the target needs it seeded first, same as a
	// real restore onto a freshly reprovisioned account.
	target := openTestStore(t, filepath.Join(dir, "target.db"))
	mustUpsertUser(t, ctx, target, "u1")
	cfg := types.BackupConfig{UserID: "u1", Destination: "local", LocalSubdir: "u1"}
	sum := mustRunRestoreOnce(t, ctx, target, dst, "u1", cfg)
	if sum.Plans != 1 || sum.DayTypes != 2 || sum.Slots != 2 || sum.SlotOptions != 2 || sum.DayOverrides != 1 || sum.Templates != 2 {
		t.Fatalf("unexpected restore summary: %+v", sum)
	}

	gotBundle := mustGetPlanBundle(t, ctx, target, plan.ID)
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
	gotTmpl := mustGetTemplate(t, ctx, target, "tmpl-oats")
	if gotTmpl.Name != "Aveia" || len(gotTmpl.Items) != 1 || gotTmpl.Items[0].Macros.Calories != 190 {
		t.Fatalf("restored template = %+v, want name Aveia with 1 item at 190 kcal", gotTmpl)
	}
}
