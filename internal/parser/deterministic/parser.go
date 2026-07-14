// Package deterministic implements the Tier-0 parser: no model, just a tokenizer
// and a bilingual (PT/EN) unit dictionary. It turns disciplined shorthand such
// as "200g frango, 2 ovos" or "200g chicken, 2 eggs" into ParsedItems. This is
// the default parser, so DietDaemon is fully usable with zero LLM and zero GPU.
//
// Volume and cooking measures are converted to grams assuming a density of
// 1.0 g/ml; count-based items ("2 eggs") carry no grams and are left for the
// resolver to map to a food-specific portion.
package deterministic

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	unitnorm "github.com/gsaraiva2109/dietdaemon/internal/parser/normalize"
)

// Parser is the Tier-0 deterministic parser.
type Parser struct{}

// New returns a ready Tier-0 parser. It holds no state and is safe for
// concurrent use.
func New() *Parser { return &Parser{} }

// Tier reports that this is the deterministic strategy.
func (p *Parser) Tier() types.ParserTier { return types.TierDeterministic }

var (
	// decimalComma converts a decimal comma between digits ("1,5") to a dot so
	// the comma can safely double as an item separator elsewhere.
	decimalComma = regexp.MustCompile(`([0-9]),([0-9])`)
	// itemSep splits a message into food items on punctuation or the PT/EN
	// conjunctions "e"/"and".
	itemSep = regexp.MustCompile(`(?i)(?:\s*[,;+&\n]\s*|\s+(?:e|and)\s+)`)
	// qtyRe captures a leading quantity and the remainder of a segment.
	qtyRe = regexp.MustCompile(`^([0-9]+(?:[.,][0-9]+)?)\s*(.*)$`)
)

// leadingFillers lists PT/EN verb/filler phrases that commonly open a casual
// food mention ("Comi arroz...", "I ate rice...") but carry no nutritional
// meaning of their own. Longer phrases are listed first so "vou comer" and
// "acabei de comer" strip as a whole rather than leaving a shorter entry to
// match a prefix of them first. Each entry keeps its trailing space so
// stripLeadingFiller only matches a leading whole word, never mid-phrase
// (mirrors stripConnector's "de "/"do "/"da "/"of " style below).
var leadingFillers = []string{
	"acabei de comer ", "vou comer ", "comendo ", "comer ", "comi ",
	"i ate ", "i had ", "eating ", "ate ", "had ",
}

// stripLeadingFiller removes one leading filler/verb phrase from s, if
// present, so the embedding matcher and food lookups see just the food
// phrase (e.g. "comi arroz" -> "arroz"). Only the start of s is checked, so
// unusual orderings such as "arroz comi" are left untouched.
func stripLeadingFiller(s string) string {
	for _, f := range leadingFillers {
		if strings.HasPrefix(s, f) {
			return strings.TrimSpace(s[len(f):])
		}
	}
	return s
}

// Extract implements ports.Parser. confidence is the mean per-item confidence
// (0..1): clean "quantity + mass-unit + food" scores highest; count-based and
// quantity-less items score lower.
func (p *Parser) Extract(_ context.Context, text, locale string) ([]types.ParsedItem, float64, error) {
	text = decimalComma.ReplaceAllString(strings.TrimSpace(text), "$1.$2")
	segments := itemSep.Split(text, -1)

	var items []types.ParsedItem
	var confSum float64
	for _, seg := range segments {
		if strings.TrimSpace(seg) == "" {
			continue
		}
		item, conf, ok := parseSegment(seg, locale)
		if !ok {
			continue
		}
		items = append(items, item)
		confSum += conf
	}
	if len(items) == 0 {
		return nil, 0, nil
	}
	return items, confSum / float64(len(items)), nil
}

// parseSegment parses one item segment, e.g. "200g frango" or "2 ovos".
func parseSegment(seg, locale string) (types.ParsedItem, float64, bool) {
	norm := stripLeadingFiller(normalize(seg))
	item := types.ParsedItem{Locale: locale, Quantity: 1}
	conf := 1.0

	rest := norm
	if m := qtyRe.FindStringSubmatch(norm); m != nil {
		item.Quantity = parseNumber(m[1])
		rest = strings.TrimSpace(m[2])
	} else {
		// No explicit quantity: assume one portion, but lower confidence.
		conf *= 0.5
	}

	unit, grams, food, hadUnit := consumeUnit(item.Quantity, rest)
	food = strings.TrimSpace(food)
	if food == "" {
		return types.ParsedItem{}, 0, false // nothing identifies the food
	}

	item.Unit = unit
	item.NormalizedGrams = grams
	item.RawPhrase = food
	if !hadUnit {
		// Count-based ("2 ovos"): grams unknown, resolver maps a portion.
		conf *= 0.85
	}
	return item, conf, true
}

// consumeUnit pulls a leading unit token (single- or multi-word) off rest,
// returning the canonical unit, grams for the quantity, the remaining food
// phrase, and whether a unit was actually recognized.
func consumeUnit(qty float64, rest string) (unit string, grams float64, food string, hadUnit bool) {
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "unit", 0, "", false
	}

	// Check if the first token is a recognized unit using the shared table.
	if !unitnorm.IsUnit(fields[0]) {
		// No unit: the whole remainder is the food, treated as a count.
		return "unit", 0, rest, false
	}

	remaining := strings.Join(fields[1:], " ")

	// colher variants need special handling — the spoon type is in the next word.
	if fields[0] == "colher" || fields[0] == "colheres" {
		canonical, g, rem := refineColher(qty, remaining)
		return canonical, g, stripConnector(rem), true
	}

	canonical, g := unitnorm.NormalizeUnit(qty, fields[0], remaining, "")
	return canonical, g, stripConnector(remaining), true
}

// refineColher resolves the Portuguese spoon variants ("colher de sopa/chá/...")
// to their approximate volumes; a bare "colher" defaults to a tablespoon.
func refineColher(qty float64, remaining string) (canonical string, grams float64, food string) {
	variants := []struct {
		phrase string
		ml     float64
		canon  string
	}{
		{"de sopa", 15, "tbsp"},
		{"de sobremesa", 10, "dessert-spoon"},
		{"de cha", 5, "tsp"},
		{"de cafe", 2, "coffee-spoon"},
	}
	for _, v := range variants {
		if strings.HasPrefix(remaining, v.phrase) {
			return v.canon, qty * v.ml, strings.TrimSpace(remaining[len(v.phrase):])
		}
	}
	return "tbsp", qty * 15, remaining
}

// stripConnector removes a leading PT/EN connector ("de arroz", "of rice") left
// after a unit so only the food phrase remains.
func stripConnector(food string) string {
	for _, c := range []string{"de ", "do ", "da ", "of "} {
		if strings.HasPrefix(food, c) {
			return strings.TrimSpace(food[len(c):])
		}
	}
	return food
}

func parseNumber(s string) float64 {
	f, _ := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	return f
}
