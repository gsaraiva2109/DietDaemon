// Package mfp implements a one-shot import of a MyFitnessPal "Nutrition
// Diary" CSV export into DietDaemon. There is no ongoing sync: a user exports
// their diary once, runs cmd/import-mfp once, and is done — same shape as
// internal/importers/hevy, minus the live API client (MFP's export is a file,
// not an endpoint).
package mfp

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// mfpHeaders maps a canonical column key to the header names (matched
// case-insensitively, with any parenthetical unit suffix like "(g)" or
// "(mg)" ignored) that MyFitnessPal is known to use for it. Based on the
// standard MFP "Nutrition Diary" export format as of 2026; if a real export
// uses different column names, add the variant here.
var mfpHeaders = map[string][]string{
	"date":          {"date"},
	"meal":          {"meal"},
	"food":          {"food", "food name", "item"},
	"serving size":  {"serving size", "serving"},
	"calories":      {"calories", "energy"},
	"fat":           {"fat"},
	"carbohydrates": {"carbohydrates", "carbs"},
	"fiber":         {"fiber", "dietary fiber"},
	"protein":       {"protein"},
}

// Row is one line item from an MFP nutrition diary export: one food logged
// under one meal slot on one day.
type Row struct {
	Date        string // as exported, typically "YYYY-MM-DD"
	Meal        string // meal slot, e.g. "Breakfast", "Lunch", "Snacks"
	Food        string
	ServingSize string
	Calories    float64
	FatG        float64
	CarbsG      float64
	FiberG      float64
	ProteinG    float64
}

// ParseCSV reads an MFP nutrition diary export. Columns are located by
// header name (case-insensitive, unit-suffix-tolerant) rather than fixed
// position, since MFP's export column order is not guaranteed stable across
// export versions. Date, Meal, and Food are required columns; the nutrition
// columns default to 0 when absent.
func ParseCSV(r io.Reader) ([]Row, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("mfp: read csv: %w", err)
	}
	if len(records) < 1 {
		return nil, fmt.Errorf("mfp: empty csv")
	}

	col, err := indexHeaders(records[0])
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0, len(records)-1)
	for _, rec := range records[1:] {
		rows = append(rows, Row{
			Date:        field(rec, col, "date"),
			Meal:        field(rec, col, "meal"),
			Food:        field(rec, col, "food"),
			ServingSize: field(rec, col, "serving size"),
			Calories:    parseFloat(field(rec, col, "calories")),
			FatG:        parseFloat(field(rec, col, "fat")),
			CarbsG:      parseFloat(field(rec, col, "carbohydrates")),
			FiberG:      parseFloat(field(rec, col, "fiber")),
			ProteinG:    parseFloat(field(rec, col, "protein")),
		})
	}
	return rows, nil
}

// indexHeaders maps each canonical column key to its position in header.
// Returns an error if a required key (date, meal, food) has no match.
func indexHeaders(header []string) (map[string]int, error) {
	col := make(map[string]int, len(mfpHeaders))
	for i, h := range header {
		norm := normalizeHeader(h)
		for key, aliases := range mfpHeaders {
			if _, found := col[key]; found {
				continue
			}
			for _, alias := range aliases {
				if norm == alias {
					col[key] = i
					break
				}
			}
		}
	}

	for _, required := range []string{"date", "meal", "food"} {
		if _, ok := col[required]; !ok {
			return nil, fmt.Errorf("mfp: csv header missing required column %q", required)
		}
	}
	return col, nil
}

// normalizeHeader lowercases a header and strips a trailing parenthetical
// unit, e.g. "Fat (g)" -> "fat", "Sodium (mg)" -> "sodium".
func normalizeHeader(h string) string {
	h = strings.TrimSpace(h)
	if i := strings.Index(h, "("); i >= 0 {
		h = h[:i]
	}
	return strings.ToLower(strings.TrimSpace(h))
}

// field looks up key in col (two-value form, so an unmapped optional column
// correctly yields "" rather than defaulting to column 0) and returns the
// corresponding field of rec, or "" if the column is absent from this export
// or the row is short.
func field(rec []string, col map[string]int, key string) string {
	i, ok := col[key]
	if !ok || i < 0 || i >= len(rec) {
		return ""
	}
	return rec[i]
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}
