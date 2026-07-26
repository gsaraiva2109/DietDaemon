package store

import (
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// TestListTemplatesForBackupIncludesPlanOwned confirms ListTemplatesForBackup
// -- unlike GetTemplates -- returns both user- and plan-owned templates, with
// owner_kind populated, so a restored plan-owned template can stay hidden
// from the user's own Templates list after restore.
func TestListTemplatesForBackupIncludesPlanOwned(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "backup-owner", CreatedAt: time.Now().UTC()})

	now := time.Now().UTC()
	userTmpl := types.MealTemplate{
		ID: "bk-user", UserID: "backup-owner", Name: "My breakfast", CreatedAt: now, LastUsed: now,
		Items: []types.ResolvedItem{{Macros: types.Macros{Calories: 200, Protein: 10, Carbs: 20, Fat: 5, Fiber: 2}}},
	}
	if err := s.SaveTemplate(ctx(), userTmpl); err != nil {
		t.Fatalf("SaveTemplate (user-owned): %v", err)
	}
	planTmpl := types.MealTemplate{
		ID: "bk-plan", UserID: "backup-owner", Name: "Opção 1", OwnerKind: types.TemplateOwnerPlan,
		CreatedAt: now, LastUsed: now,
	}
	if err := s.SaveTemplate(ctx(), planTmpl); err != nil {
		t.Fatalf("SaveTemplate (plan-owned): %v", err)
	}

	list, err := s.ListTemplatesForBackup(ctx(), "backup-owner")
	if err != nil {
		t.Fatalf("ListTemplatesForBackup: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListTemplatesForBackup = %+v, want 2 (both user- and plan-owned)", list)
	}
	byID := map[string]types.MealTemplate{}
	for _, tmpl := range list {
		byID[tmpl.ID] = tmpl
	}
	if got := byID["bk-user"]; got.OwnerKind != types.TemplateOwnerUser || len(got.Items) != 1 {
		t.Errorf("bk-user = %+v, want owner_kind=user and 1 item", got)
	}
	if got := byID["bk-plan"]; got.OwnerKind != types.TemplateOwnerPlan {
		t.Errorf("bk-plan = %+v, want owner_kind=plan", got)
	}
}

// TestRestoreTemplateIdempotent confirms RestoreTemplate preserves the
// original ID and owner_kind from the backup and is a safe no-op on a
// duplicate ID (the restore replay-safety convention shared by every other
// Restore* method in this store).
func TestRestoreTemplateIdempotent(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "restore-owner", CreatedAt: time.Now().UTC()})

	now := time.Now().UTC()
	tmpl := types.MealTemplate{
		ID: "restored-1", UserID: "restore-owner", Name: "Opção 1", OwnerKind: types.TemplateOwnerPlan,
		CreatedAt: now, LastUsed: now,
		Items: []types.ResolvedItem{{Macros: types.Macros{Calories: 400, Protein: 30, Carbs: 40, Fat: 12, Fiber: 6}}},
	}
	if err := s.RestoreTemplate(ctx(), tmpl); err != nil {
		t.Fatalf("RestoreTemplate: %v", err)
	}

	got, err := s.GetTemplate(ctx(), "restored-1")
	if err != nil {
		t.Fatalf("GetTemplate after restore: %v", err)
	}
	if got.Name != "Opção 1" || len(got.Items) != 1 || got.Items[0].Macros.Calories != 400 {
		t.Fatalf("restored template = %+v", got)
	}
	// A direct fetch by ID must still find it even though it's plan-owned
	// (GetTemplates would exclude it).
	list, err := s.GetTemplates(ctx(), "restore-owner")
	if err != nil {
		t.Fatalf("GetTemplates: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("GetTemplates = %+v, want plan-owned template excluded", list)
	}

	// Restoring the same ID again is a safe no-op, not an error.
	if err := s.RestoreTemplate(ctx(), tmpl); err != nil {
		t.Fatalf("RestoreTemplate (duplicate): %v", err)
	}
}

// TestRestoreTemplateDefaultsOwnerKind confirms a backup taken before the
// owner_kind column existed (empty OwnerKind) restores as TemplateOwnerUser.
func TestRestoreTemplateDefaultsOwnerKind(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "restore-owner-2", CreatedAt: time.Now().UTC()})

	now := time.Now().UTC()
	tmpl := types.MealTemplate{ID: "restored-2", UserID: "restore-owner-2", Name: "Legacy", CreatedAt: now, LastUsed: now}
	if err := s.RestoreTemplate(ctx(), tmpl); err != nil {
		t.Fatalf("RestoreTemplate: %v", err)
	}

	list, err := s.GetTemplates(ctx(), "restore-owner-2")
	if err != nil {
		t.Fatalf("GetTemplates: %v", err)
	}
	if len(list) != 1 || list[0].ID != "restored-2" {
		t.Fatalf("GetTemplates = %+v, want the restored template visible (owner_kind defaulted to user)", list)
	}
}

// TestGetTemplatesExcludesPlanOwned confirms plan-owned templates (backing a
// DietPlanSlotOption, owner_kind = "plan") don't clutter the user's own
// Templates list, while a direct GetTemplate by ID still finds them --
// slot-option editing needs to load the backing template even though the
// list view hides it.
func TestGetTemplatesExcludesPlanOwned(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "tmpl-owner", CreatedAt: time.Now().UTC()})

	now := time.Now().UTC()
	userTmpl := types.MealTemplate{ID: "t-user", UserID: "tmpl-owner", Name: "My breakfast", CreatedAt: now, LastUsed: now}
	if err := s.SaveTemplate(ctx(), userTmpl); err != nil {
		t.Fatalf("SaveTemplate (user-owned): %v", err)
	}
	planTmpl := types.MealTemplate{
		ID: "t-plan", UserID: "tmpl-owner", Name: "Opção 1", OwnerKind: types.TemplateOwnerPlan,
		CreatedAt: now, LastUsed: now,
	}
	if err := s.SaveTemplate(ctx(), planTmpl); err != nil {
		t.Fatalf("SaveTemplate (plan-owned): %v", err)
	}

	list, err := s.GetTemplates(ctx(), "tmpl-owner")
	if err != nil {
		t.Fatalf("GetTemplates: %v", err)
	}
	if len(list) != 1 || list[0].ID != "t-user" {
		t.Fatalf("GetTemplates = %+v, want only the user-owned template", list)
	}

	// A direct fetch by ID still finds the plan-owned template -- slot-option
	// editing needs this even though the list view hides it.
	got, err := s.GetTemplate(ctx(), "t-plan")
	if err != nil {
		t.Fatalf("GetTemplate(t-plan): %v", err)
	}
	if got.ID != "t-plan" {
		t.Fatalf("GetTemplate(t-plan) = %+v", got)
	}
}

// TestSaveTemplateOwnerKindImmutableAfterCreate confirms a re-save (e.g. the
// last_used bump on template log) can't flip owner_kind after creation.
func TestSaveTemplateOwnerKindImmutableAfterCreate(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	mustUser(t, s, types.User{ID: "tmpl-owner-2", CreatedAt: time.Now().UTC()})

	now := time.Now().UTC()
	tmpl := types.MealTemplate{
		ID: "t-plan-2", UserID: "tmpl-owner-2", Name: "Opção 1", OwnerKind: types.TemplateOwnerPlan,
		CreatedAt: now, LastUsed: now,
	}
	if err := s.SaveTemplate(ctx(), tmpl); err != nil {
		t.Fatalf("SaveTemplate: %v", err)
	}

	// Re-save without OwnerKind set (mirrors handleLogTemplate's last_used
	// bump, which never touches OwnerKind) must not demote it to "user".
	tmpl.Name = "Opção 1 revisada"
	tmpl.OwnerKind = ""
	if err := s.SaveTemplate(ctx(), tmpl); err != nil {
		t.Fatalf("SaveTemplate (re-save): %v", err)
	}

	list, err := s.GetTemplates(ctx(), "tmpl-owner-2")
	if err != nil {
		t.Fatalf("GetTemplates: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("GetTemplates = %+v, want the plan-owned template still excluded after re-save", list)
	}
}
