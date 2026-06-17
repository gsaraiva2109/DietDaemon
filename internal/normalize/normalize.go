// Package normalize provides shared phrase normalization (lowercase, trim,
// unaccent) used across the store and nutrition adapters so that "Frângó" and
// "frango" match the same index.
package normalize

import "strings"

// Normalize lowercases, trims whitespace, and strips common Latin diacritics
// so that accented Portuguese/Spanish/French/English search phrases match
// their ASCII-normalised database representations.
func Normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return unaccent(s)
}

// unaccent maps precomposed accented Latin characters to their ASCII base.
// It covers PT, EN, ES, FR diacritics and is intentionally simple — fuzzy
// matching is layered on later.
func unaccent(s string) string {
	replacer := strings.NewReplacer(
		// Uppercase.
		"À", "A", "Á", "A", "Â", "A", "Ã", "A", "Ä", "A", "Å", "A",
		"Æ", "AE", "Ç", "C",
		"È", "E", "É", "E", "Ê", "E", "Ë", "E",
		"Ì", "I", "Í", "I", "Î", "I", "Ï", "I",
		"Ð", "D", "Ñ", "N",
		"Ò", "O", "Ó", "O", "Ô", "O", "Õ", "O", "Ö", "O", "Ø", "O",
		"Ù", "U", "Ú", "U", "Û", "U", "Ü", "U",
		"Ý", "Y",
		// Lowercase.
		"à", "a", "á", "a", "â", "a", "ã", "a", "ä", "a", "å", "a",
		"æ", "ae", "ç", "c",
		"è", "e", "é", "e", "ê", "e", "ë", "e",
		"ì", "i", "í", "i", "î", "i", "ï", "i",
		"ð", "d", "ñ", "n",
		"ò", "o", "ó", "o", "ô", "o", "õ", "o", "ö", "o", "ø", "o",
		"ù", "u", "ú", "u", "û", "u", "ü", "u",
		"ý", "y", "ÿ", "y",
	)
	return replacer.Replace(s)
}
