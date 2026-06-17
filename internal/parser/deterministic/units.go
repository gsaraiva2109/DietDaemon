package deterministic

import "strings"

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
