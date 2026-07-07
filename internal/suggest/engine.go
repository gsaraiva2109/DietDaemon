// Package suggest implements DietDaemon's meal-suggestion engine: a
// rule-based macro-fit matcher (matcher.go) plus an LLM-ranking orchestrator
// (this file) layered on top.
package suggest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// candidatePoolSize caps how many frequent foods feed the matcher — matcher.go's
// bounded brute force is sized for this.
const candidatePoolSize = 15

// topCandidates is how many rule-based combos are computed and offered to the LLM.
const topCandidates = 5

// Store is the subset of persistence the suggestion engine needs.
type Store interface {
	GetRollup(ctx context.Context, userID, localDate string) (types.DailyRollup, error)
	GetTargets(ctx context.Context, userID string) (types.DailyTargets, error)
	FrequentFoods(ctx context.Context, userID string, limit int) ([]types.FoodDetail, error)
}

// Engine orchestrates /suggest: compute remaining macros, find rule-based
// candidates, ask the completion adapter to rank/phrase them, and fall back to
// the top rule-based candidate if the model is unavailable or misbehaves.
type Engine struct {
	store Store
	model ports.ModelAdapter
	loc   *time.Location
}

// New returns a ready Engine.
func New(store Store, model ports.ModelAdapter, loc *time.Location) *Engine {
	return &Engine{store: store, model: model, loc: loc}
}

// Suggest computes what's left of the user's daily targets and returns candidate
// meals built from foods they already eat, ranked and phrased by the LLM when
// available.
func (e *Engine) Suggest(ctx context.Context, userID string) (types.MealSuggestion, error) {
	remaining, err := e.remainingMacros(ctx, userID)
	if err != nil {
		return types.MealSuggestion{}, err
	}

	pool, err := e.store.FrequentFoods(ctx, userID, candidatePoolSize)
	if err != nil {
		return types.MealSuggestion{}, fmt.Errorf("suggest: frequent foods: %w", err)
	}
	if len(pool) == 0 {
		return types.MealSuggestion{
			Remaining: remaining,
			Message:   "Log a few meals first so I know what you like to eat.",
			Source:    "rules",
		}, nil
	}

	candidates := FindCombos(pool, remaining, topCandidates)
	combos := toSuggestedCombos(candidates)
	rulesFallback := types.MealSuggestion{
		Remaining:  remaining,
		Candidates: combos,
		Message:    describeCombo(combos[0]),
		Source:     "rules",
	}

	model := e.model
	if override, ok := ports.ModelOverrideFromContext(ctx); ok {
		model = override
	}
	raw, err := model.Complete(ctx, rankPrompt(remaining, combos))
	if err != nil {
		// Model unavailable: fall back to the top rule-based candidate.
		return rulesFallback, nil
	}

	var resp rankResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil || resp.Message == "" {
		// Bad or empty JSON: fall back.
		return rulesFallback, nil
	}

	return types.MealSuggestion{
		Remaining:  remaining,
		Candidates: combos,
		Message:    resp.Message,
		Source:     "llm",
	}, nil
}

// remainingMacros computes targets minus what's been consumed today. A user
// with targets set but nothing logged today has no rollup row yet — that's the
// common "first check of the day" path, not an edge case, so it falls back to
// targets with zero consumed rather than erroring.
func (e *Engine) remainingMacros(ctx context.Context, userID string) (types.Macros, error) {
	localDate := time.Now().In(e.loc).Format("2006-01-02")

	rollup, err := e.store.GetRollup(ctx, userID, localDate)
	if err == nil {
		return rollup.Targets.Sub(rollup.Consumed), nil
	}
	if !errors.Is(err, types.ErrNotFound) {
		return types.Macros{}, fmt.Errorf("suggest: get rollup: %w", err)
	}

	targets, err := e.store.GetTargets(ctx, userID)
	if err != nil {
		return types.Macros{}, fmt.Errorf("suggest: get targets: %w", err)
	}
	return targets.Targets, nil
}

func toSuggestedCombos(candidates []Candidate) []types.SuggestedCombo {
	combos := make([]types.SuggestedCombo, len(candidates))
	for i, c := range candidates {
		items := make([]types.SuggestedItem, len(c.Items))
		for j, it := range c.Items {
			items[j] = types.SuggestedItem{
				FoodID: it.Food.FoodID,
				Name:   it.Food.Name,
				Grams:  it.Grams,
			}
		}
		combos[i] = types.SuggestedCombo{Items: items, Macros: c.Macros, Score: c.Score}
	}
	return combos
}

// describeCombo renders a candidate as plain text for the no-LLM fallback path.
func describeCombo(c types.SuggestedCombo) string {
	parts := make([]string, len(c.Items))
	for i, it := range c.Items {
		parts[i] = fmt.Sprintf("%s (%.0fg)", it.Name, it.Grams)
	}
	return fmt.Sprintf("Try: %s — about %.0f kcal, %.0fg protein.",
		strings.Join(parts, " + "), c.Macros.Calories, c.Macros.Protein)
}

// rankResponse is the expected JSON shape from the completion adapter.
type rankResponse struct {
	Message string `json:"message"`
}

// rankPrompt builds the completion-adapter prompt from remaining macros and
// the rule-based candidates it should rank/phrase.
func rankPrompt(remaining types.Macros, combos []types.SuggestedCombo) string {
	var b strings.Builder
	b.WriteString("You are a nutrition assistant. The user has these macros left today: ")
	fmt.Fprintf(&b, "%.0f kcal, %.0fg protein, %.0fg carbs, %.0fg fat.\n", remaining.Calories, remaining.Protein, remaining.Carbs, remaining.Fat)
	b.WriteString("Here are candidate meals built from foods they already eat, best macro fit first:\n")
	for i, c := range combos {
		parts := make([]string, len(c.Items))
		for j, it := range c.Items {
			parts[j] = fmt.Sprintf("%s (%.0fg)", it.Name, it.Grams)
		}
		fmt.Fprintf(&b, "%d. %s — %.0f kcal, %.0fg protein, %.0fg carbs, %.0fg fat\n",
			i+1, strings.Join(parts, " + "), c.Macros.Calories, c.Macros.Protein, c.Macros.Carbs, c.Macros.Fat)
	}
	b.WriteString(`Pick the best candidate (or a light combination of two) and phrase it as a short, friendly suggestion (1-2 sentences). Output ONLY JSON: {"message":"<your suggestion>"}`)
	return b.String()
}
