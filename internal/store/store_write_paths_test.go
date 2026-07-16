package store

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestWritePathsRoundTrip(t *testing.T) {
	s, cleanup := tempDB(t)
	defer cleanup()
	now := time.Now().UTC()
	mustUser(t, s, types.User{ID: "write-user", Email: "write@example.com", CreatedAt: now})

	food := types.FoodMatch{FoodID: "write-food", Name: "Write Food", Source: "test", Per100g: types.Macros{Calories: 100}}
	if err := s.UpsertFood(ctx(), "write-user", food, nil); err != nil {
		t.Fatalf("UpsertFood: %v", err)
	}
	if err := s.SetSourcePrecedence(ctx(), "write-user", []string{"test"}); err != nil {
		t.Fatalf("SetSourcePrecedence: %v", err)
	}

	meal := types.Meal{ID: "write-meal", UserID: "write-user", RawText: "meal", At: now, CreatedAt: now}
	if err := s.SaveMeal(ctx(), meal); err != nil {
		t.Fatalf("SaveMeal: %v", err)
	}
	originalText := meal.RawText
	meal.RawText = "updated meal"
	if err := s.SaveMeal(ctx(), meal); err != nil {
		t.Fatalf("SaveMeal duplicate: %v", err)
	}
	if got, err := s.GetMeal(ctx(), meal.ID); err != nil || got.RawText != originalText {
		t.Fatalf("GetMeal = %+v, %v", got, err)
	}

	if err := s.SetNudgeRuleConfig(ctx(), "write-user", "protein", true, json.RawMessage(`{"min":0.5}`)); err != nil {
		t.Fatalf("SetNudgeRuleConfig: %v", err)
	}
	if err := s.DeleteNudgeRuleConfig(ctx(), "write-user", "protein"); err != nil {
		t.Fatalf("DeleteNudgeRuleConfig: %v", err)
	}

	template := types.MealTemplate{ID: "write-template", UserID: "write-user", Name: "Lunch", CreatedAt: now, LastUsed: now}
	if err := s.SaveTemplate(ctx(), template); err != nil {
		t.Fatalf("SaveTemplate: %v", err)
	}
	if got, err := s.GetTemplate(ctx(), template.ID); err != nil || got.Name != template.Name {
		t.Fatalf("GetTemplate = %+v, %v", got, err)
	}

	if err := s.UpsertProviderKey(ctx(), "write-user", "openai", "ciphertext"); err != nil {
		t.Fatalf("UpsertProviderKey: %v", err)
	}
	if got, found, err := s.GetProviderKey(ctx(), "write-user", "openai"); err != nil || !found || got != "ciphertext" {
		t.Fatalf("GetProviderKey = %q, %t, %v", got, found, err)
	}

	workout := types.Workout{ID: "write-workout", UserID: "write-user", Name: "Walk", DurationMin: 30, Intensity: "low", LoggedAt: now.Format(time.RFC3339)}
	if err := s.LogWorkout(ctx(), workout); err != nil {
		t.Fatalf("LogWorkout: %v", err)
	}
	if got, err := s.GetWorkout(ctx(), workout.ID); err != nil || got.Name != workout.Name {
		t.Fatalf("GetWorkout = %+v, %v", got, err)
	}
}
