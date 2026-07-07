// Package suggest implements DietDaemon's meal-suggestion engine: a
// rule-based macro-fit matcher (this file) plus an LLM-ranking orchestrator
// layered on top (engine.go, built separately).
package suggest

import (
	"math"
	"sort"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// CandidateItem is one food at a chosen serving size within a Candidate combo.
type CandidateItem struct {
	Food  types.FoodDetail
	Grams float64
}

// Candidate is one combination of 1-3 foods at chosen serving sizes, scored by
// how close its total macros land to the remaining target. Score is 0..1,
// higher is a closer fit.
type Candidate struct {
	Items  []CandidateItem
	Macros types.Macros
	Score  float64
}

// servingMultipliers are applied to a 100g base serving (0.5x = 50g ... 2x = 200g).
var servingMultipliers = []float64{0.5, 1.0, 1.5, 2.0}

// FindCombos searches pool for up to topN combinations of 1-3 items (at
// serving multipliers 0.5x/1x/1.5x/2x of a 100g base serving) whose combined
// macros best match remaining. Results are sorted best-first (highest Score).
//
// ponytail: bounded brute force over the caller-capped pool (expected ~15
// items) x combo size (1-3) x multiplier grid (4) — a few tens of thousands
// of evaluations, sub-millisecond. Swap for a real solver only if the pool
// cap needs to grow past ~20-30.
func FindCombos(pool []types.FoodDetail, remaining types.Macros, topN int) []Candidate {
	var results []Candidate

	// combo size 1
	for i := range pool {
		results = append(results, comboVariants(pool[i:i+1], remaining)...)
	}
	// combo size 2
	for i := range pool {
		for j := i + 1; j < len(pool); j++ {
			results = append(results, comboVariants([]types.FoodDetail{pool[i], pool[j]}, remaining)...)
		}
	}
	// combo size 3
	for i := range pool {
		for j := i + 1; j < len(pool); j++ {
			for k := j + 1; k < len(pool); k++ {
				results = append(results, comboVariants([]types.FoodDetail{pool[i], pool[j], pool[k]}, remaining)...)
			}
		}
	}

	sort.Slice(results, func(a, b int) bool { return results[a].Score > results[b].Score })

	if topN < len(results) {
		results = results[:topN]
	}
	return results
}

// comboVariants evaluates every serving-multiplier assignment for a fixed set
// of foods and returns one scored Candidate per variant.
func comboVariants(foods []types.FoodDetail, remaining types.Macros) []Candidate {
	// Build the multiplier index for each food position via odometer-style
	// counters, e.g. foods[0]'s multiplier cycles fastest.
	n := len(foods)
	idx := make([]int, n)
	var out []Candidate

	for {
		items := make([]CandidateItem, n)
		var total types.Macros
		for i, food := range foods {
			mult := servingMultipliers[idx[i]]
			items[i] = CandidateItem{Food: food, Grams: mult * 100}
			total = total.Add(food.Per100g.Scale(mult))
		}
		out = append(out, Candidate{
			Items:  items,
			Macros: total,
			Score:  score(total, remaining),
		})

		// advance odometer
		pos := 0
		for pos < n {
			idx[pos]++
			if idx[pos] < len(servingMultipliers) {
				break
			}
			idx[pos] = 0
			pos++
		}
		if pos == n {
			break
		}
	}
	return out
}

// score computes the fit of combo total macros against remaining. 1.0 is an
// exact match; it approaches 0 as deviation grows. Fiber is intentionally
// excluded (remaining fiber targets are often noisy or near-zero).
func score(total, remaining types.Macros) float64 {
	dev := math.Abs(total.Calories-remaining.Calories)/math.Max(remaining.Calories, 1) +
		math.Abs(total.Protein-remaining.Protein)/math.Max(remaining.Protein, 1) +
		math.Abs(total.Carbs-remaining.Carbs)/math.Max(remaining.Carbs, 1) +
		math.Abs(total.Fat-remaining.Fat)/math.Max(remaining.Fat, 1)
	return 1 / (1 + dev)
}
