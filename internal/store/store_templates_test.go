package store

import (
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

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
