package mfp

import "github.com/gsaraiva2109/dietdaemon/core/types"

// ToItem converts one MFP diary row into a resolved meal item. Unlike a
// live-logged item, the row's nutrition columns are already the absolute
// values for the portion MFP recorded (not per-100g), so Macros is taken
// directly from the row and Per100g is left zero — there's no serving size
// in grams to scale by, and none is needed since the absolute macros are
// already known.
func ToItem(row Row) types.ResolvedItem {
	macros := types.Macros{
		Calories: row.Calories,
		Protein:  row.ProteinG,
		Carbs:    row.CarbsG,
		Fat:      row.FatG,
		Fiber:    row.FiberG,
	}
	return types.ResolvedItem{
		Parsed: types.ParsedItem{
			RawPhrase: row.Food,
			Quantity:  1,
			Unit:      row.ServingSize,
		},
		Match: types.FoodMatch{
			Name:       row.Food,
			Source:     "mfp_import",
			MatchScore: 1,
		},
		Macros: macros,
	}
}
