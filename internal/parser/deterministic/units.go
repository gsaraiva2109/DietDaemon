package deterministic

import "strings"

// unitKind classifies how a unit converts to grams.
type unitKind int

const (
	kindMass   unitKind = iota // already a mass: grams = qty * gramsPerUnit
	kindVolume                 // volume/cooking: grams = qty * ml * density(1.0 g/ml)
	kindCount                  // countable ("2 eggs"): grams unknown, left to the resolver
)

// unitDef describes one resolved unit. gramsPerUnit already folds in the
// density-1.0 assumption for volume/cooking units.
type unitDef struct {
	canonical    string
	gramsPerUnit float64
	kind         unitKind
}

// unitAliases maps normalized (lowercased, unaccented) PT/EN tokens to a unit.
// The dictionary is intentionally bilingual so the same parser handles
// "200g chicken" and "200g frango" without any language detection.
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

	// Explicit count words (bare counts are also the default when no unit matches).
	count("unit", "unidade", "unidades", "un", "und", "unit", "units")
}

// accentRepl folds the Portuguese accented characters we care about down to
// ASCII, avoiding an x/text dependency for such a small set.
var accentRepl = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// normalize lowercases, trims, and strips accents so unit lookups and food
// phrases are matched consistently (the store normalizes the same way).
func normalize(s string) string {
	return accentRepl.Replace(strings.ToLower(strings.TrimSpace(s)))
}
