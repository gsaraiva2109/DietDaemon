// Package normalize provides shared unit-normalization logic used by every
// parser tier so that identical (qty, unit, food) tuples produce identical
// grams regardless of which parser extracted them.
//
// This is distinct from the phrase normalizer in internal/normalize, which
// handles food-alias folding for the store.
package normalize

import "strings"

// unitKind classifies how a unit converts to grams.
type unitKind int

const (
	kindMass   unitKind = iota // already a mass: grams = qty * gramsPerUnit
	kindVolume                 // volume/cooking: grams = qty * ml * density(1.0 g/ml)
	kindCount                  // countable ("2 eggs"): grams unknown
)

// unitDef describes one resolved unit. gramsPerUnit already folds in the
// density-1.0 assumption for volume/cooking units.
type unitDef struct {
	canonical    string
	gramsPerUnit float64
	kind         unitKind
}

// unitAliases maps normalized (lowercased, unaccented) PT/EN tokens to a unit.
var unitAliases = map[string]unitDef{}

func init() {
	mass := func(canon string, g float64, aliases ...string) {
		for _, a := range aliases {
			unitAliases[a] = unitDef{canon, g, kindMass}
		}
	}
	vol := func(canon string, gramsPerUnit float64, aliases ...string) {
		for _, a := range aliases {
			unitAliases[a] = unitDef{canon, gramsPerUnit, kindVolume}
		}
	}
	count := func(canon string, aliases ...string) {
		for _, a := range aliases {
			unitAliases[a] = unitDef{canon, 0, kindCount}
		}
	}

	// Mass.
	mass("g", 1, "g", "grama", "gramas", "gr", "grm")
	mass("kg", 1000, "kg", "quilo", "quilos", "kilo", "kilos")
	mass("mg", 0.001, "mg")
	mass("oz", 28.3495, "oz", "onca", "oncas", "ounce", "ounces")
	mass("lb", 453.592, "lb", "lbs", "libra", "libras", "pound", "pounds")

	// Volume (density 1.0 g/ml).
	vol("ml", 1, "ml", "mililitro", "mililitros", "milliliter", "milliliters")
	vol("l", 1000, "l", "lt", "litro", "litros", "liter", "liters", "litre", "litres")

	// Cooking measures (approximate, density 1.0).
	vol("tbsp", 15, "tbsp", "tablespoon", "tablespoons")
	vol("tsp", 5, "tsp", "teaspoon", "teaspoons")
	vol("cup", 240, "cup", "cups", "xicara", "xicaras")
	vol("glass", 200, "copo", "copos", "glass")
	// "colher"/"colheres" are refined to sopa/cha/cafe variants in refineColher;
	// the base entry makes them recognized as a unit token.
	vol("tbsp", 15, "colher", "colheres")

	// Explicit count words.
	count("unit", "unidade", "unidades", "un", "und", "unit", "units")
}

// accentRepl folds Portuguese accented characters to ASCII.
var accentRepl = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// NormalizeUnit maps a raw unit token to its canonical name and computes grams
// for the given quantity. locale is reserved for future unit-per-locale tables.
// When unit is empty or unrecognized it is treated as a count (grams=0).
func NormalizeUnit(qty float64, unit, _, _ string) (canonicalUnit string, grams float64) {
	unit = accentRepl.Replace(strings.ToLower(strings.TrimSpace(unit)))
	if unit == "" {
		return "unit", 0
	}

	def, ok := unitAliases[unit]
	if !ok {
		return "unit", 0
	}

	canonical := def.canonical
	g := qty * def.gramsPerUnit
	if def.kind == kindCount {
		g = 0
	}
	return canonical, g
}

// IsUnit reports whether the normalized token is a recognized unit.
func IsUnit(token string) bool {
	_, ok := unitAliases[accentRepl.Replace(strings.ToLower(strings.TrimSpace(token)))]
	return ok
}
