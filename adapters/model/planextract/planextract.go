// Package planextract holds the prompt and response contract shared by both
// diet-plan import paths (#193 paste-text, #194 photo/PDF), so the JSON
// schema the model is asked for lives in exactly one place. It mirrors
// labelextract's shape: a Prompt constant, private wire types, and
// ParseResponse, which is input-mode agnostic — the same contract serves a
// pasted-text completion call and a vision extraction call alike.
//
// Unlike labelextract, this package is not nested under adapters/model/internal:
// the #193 paste-text handler in internal/api builds the prompt and parses
// the response directly (there is no vision-adapter method to hide it
// behind, per decision #3 in the PR3 plan — #193 reuses the generic
// ports.ModelAdapter.Complete), and Go's internal-package visibility rule
// would otherwise block internal/api from importing it.
package planextract

import (
	"encoding/json"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/adapters/model/internal/jsonfence"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Prompt instructs the model to read pasted diet-plan text — in any
// language — and return the JSON contract ParseResponse expects. It is
// deliberately language-agnostic: the model reads English or Portuguese
// natively, so no locale branching is needed here.
const Prompt = `You are reading pasted text describing a diet/nutrition plan written by a nutritionist. The text may be in any language (e.g. Portuguese, English) — read it natively, no locale branching is needed.

A plan may prescribe different day types with different targets (for example a carb-cycling plan may prescribe a "training day" and a "rest day", each with its own macro targets). Each day type has:
- a name (e.g. "Dia de treino" / "Training day", "Dia de descanso" / "Rest day")
- macro targets: calories, protein, carbs, fat, fiber
- an optional daily water goal in milliliters
- one or more slots (meals), each with a label (e.g. "Café da manhã" / "Breakfast") and an optional time of day

Each slot may offer one or more interchangeable options ("Opção 1" / "Option 1", "Opção 2" / "Option 2") the person can pick between. Each option has one or more food items. Each item has:
- raw_name: the food name EXACTLY as written in the source text. Do NOT normalize, translate, correct spelling, or otherwise rewrite it. Preserve it verbatim, including its original language and any brand names — catalog matching against this exact string happens in a later step.
- an optional quantity and unit (e.g. quantity 100, unit "g"; or quantity 1, unit "unidade"/"unit")
- ad_libitum: true if the text says the item is free-quantity ("a vontade", "à vontade", "ad libitum", "as much as needed") instead of giving a quantity/unit. Use ad_libitum true for this case — never report it as quantity 0.

Apply this three-tier confidence rule to every field:
1. Confident: report the value.
2. Present but uncertain (unclear phrasing, ambiguous abbreviation, illegible fragment): still report your best-effort reading, but add that field's key to the low_confidence_fields array at its level (each day type has its own low_confidence_fields for its own fields such as "name", "water_goal_ml", "targets.calories", "targets.protein", "targets.carbs", "targets.fat", "targets.fiber"; each option has its own low_confidence_fields for its items' fields).
3. Not present in the text: the value must be JSON null. NEVER invent, guess, or estimate a value.

If the pasted text is not a diet plan, or is too garbled or incomplete to extract anything meaningful from, set unreadable to true and leave every other field null or empty.

Respond with ONLY this JSON object, no markdown fences, no commentary:
{
  "plan_name": string or null,
  "day_types": [
    {
      "name": string,
      "targets": {"Calories": number or null, "Protein": number or null, "Carbs": number or null, "Fat": number or null, "Fiber": number or null},
      "water_goal_ml": number or null,
      "slots": [
        {
          "label": string,
          "time_of_day": string or null,
          "options": [
            {
              "label": string,
              "items": [
                {"raw_name": string, "quantity": number or null, "unit": string or null, "ad_libitum": boolean}
              ],
              "low_confidence_fields": array of field name strings (may be empty)
            }
          ]
        }
      ],
      "low_confidence_fields": array of field name strings (may be empty)
    }
  ],
  "unreadable": boolean,
  "notes": string or null (any general free-text guidance from the plan that doesn't fit the structure above)
}`

// PhotoPrompt instructs the model to read a photographed or scanned
// diet-plan page and return the JSON contract ParseResponse expects. It
// mirrors Prompt's structure and field contract exactly; the only
// difference is the input framing (an image instead of pasted text) and an
// explicit call-out for multi-column carb-cycling tables (e.g. a table with
// "Training day" / "Rest day" as columns and meals as rows), a known hard
// case for photographed layouts — the model must not guess which column a
// value belongs to.
const PhotoPrompt = `You are reading a photographed or scanned page describing a diet/nutrition plan written by a nutritionist. The text on the page may be in any language (e.g. Portuguese, English) — read it natively, no locale branching is needed.

A plan may prescribe different day types with different targets (for example a carb-cycling plan may prescribe a "training day" and a "rest day", each with its own macro targets). Each day type has:
- a name (e.g. "Dia de treino" / "Training day", "Dia de descanso" / "Rest day")
- macro targets: calories, protein, carbs, fat, fiber
- an optional daily water goal in milliliters
- one or more slots (meals), each with a label (e.g. "Café da manhã" / "Breakfast") and an optional time of day

Each slot may offer one or more interchangeable options ("Opção 1" / "Option 1", "Opção 2" / "Option 2") the person can pick between. Each option has one or more food items. Each item has:
- raw_name: the food name EXACTLY as written on the page. Do NOT normalize, translate, correct spelling, or otherwise rewrite it. Preserve it verbatim, including its original language and any brand names — catalog matching against this exact string happens in a later step.
- an optional quantity and unit (e.g. quantity 100, unit "g"; or quantity 1, unit "unidade"/"unit")
- ad_libitum: true if the page says the item is free-quantity ("a vontade", "à vontade", "ad libitum", "as much as needed") instead of giving a quantity/unit. Use ad_libitum true for this case — never report it as quantity 0.

Known hard case: multi-column carb-cycling tables, where columns are day types (e.g. "Training day" / "Rest day") and rows are meals, with a food item or quantity written per column. Photographed tables like this are prone to column misalignment. If you cannot reliably tell which column a value belongs to — due to skew, glare, cropping, or ambiguous column boundaries — do NOT guess. Set unreadable to true instead of producing a plausible-looking but potentially wrong assignment.

Apply this three-tier confidence rule to every field:
1. Confident: report the value.
2. Present but uncertain (unclear phrasing, ambiguous abbreviation, illegible fragment, partial glare or blur): still report your best-effort reading, but add that field's key to the low_confidence_fields array at its level (each day type has its own low_confidence_fields for its own fields such as "name", "water_goal_ml", "targets.calories", "targets.protein", "targets.carbs", "targets.fat", "targets.fiber"; each option has its own low_confidence_fields for its items' fields).
3. Not present on the page: the value must be JSON null. NEVER invent, guess, or estimate a value.

If the photographed page is not a diet plan, or is too garbled, blurry, or incomplete to extract anything meaningful from — including the multi-column ambiguity case above — set unreadable to true and leave every other field null or empty.

Respond with ONLY this JSON object, no markdown fences, no commentary:
{
  "plan_name": string or null,
  "day_types": [
    {
      "name": string,
      "targets": {"Calories": number or null, "Protein": number or null, "Carbs": number or null, "Fat": number or null, "Fiber": number or null},
      "water_goal_ml": number or null,
      "slots": [
        {
          "label": string,
          "time_of_day": string or null,
          "options": [
            {
              "label": string,
              "items": [
                {"raw_name": string, "quantity": number or null, "unit": string or null, "ad_libitum": boolean}
              ],
              "low_confidence_fields": array of field name strings (may be empty)
            }
          ]
        }
      ],
      "low_confidence_fields": array of field name strings (may be empty)
    }
  ],
  "unreadable": boolean,
  "notes": string or null (any general free-text guidance from the plan that doesn't fit the structure above)
}`

type wireItem struct {
	RawName   string   `json:"raw_name"`
	Quantity  *float64 `json:"quantity"`
	Unit      *string  `json:"unit"`
	AdLibitum bool     `json:"ad_libitum"`
}

type wireOption struct {
	Label               string     `json:"label"`
	Items               []wireItem `json:"items"`
	LowConfidenceFields []string   `json:"low_confidence_fields"`
}

type wireSlot struct {
	Label     string       `json:"label"`
	TimeOfDay *string      `json:"time_of_day"`
	Options   []wireOption `json:"options"`
}

type wireDayType struct {
	Name                string       `json:"name"`
	Targets             types.Macros `json:"targets"`
	WaterGoalMl         *int         `json:"water_goal_ml"`
	Slots               []wireSlot   `json:"slots"`
	LowConfidenceFields []string     `json:"low_confidence_fields"`
}

type wireResponse struct {
	PlanName   *string       `json:"plan_name"`
	DayTypes   []wireDayType `json:"day_types"`
	Unreadable bool          `json:"unreadable"`
	Notes      *string       `json:"notes"`
}

// ParseResponse parses a model's raw text response (optionally
// markdown-fenced) into a PlanDraft.
func ParseResponse(raw string) (types.PlanDraft, error) {
	stripped := jsonfence.Strip(raw)

	var wr wireResponse
	if err := json.Unmarshal([]byte(stripped), &wr); err != nil {
		return types.PlanDraft{}, fmt.Errorf("planextract: decode response: %w", err)
	}

	return types.PlanDraft{
		PlanName:   wr.PlanName,
		DayTypes:   mapDayTypes(wr.DayTypes),
		Unreadable: wr.Unreadable,
		Notes:      wr.Notes,
	}, nil
}

func mapDayTypes(in []wireDayType) []types.PlanDraftDayType {
	out := make([]types.PlanDraftDayType, len(in))
	for i, d := range in {
		out[i] = types.PlanDraftDayType{
			Name:                d.Name,
			Targets:             d.Targets,
			WaterGoalMl:         d.WaterGoalMl,
			Slots:               mapSlots(d.Slots),
			LowConfidenceFields: d.LowConfidenceFields,
		}
	}
	return out
}

func mapSlots(in []wireSlot) []types.PlanDraftSlot {
	out := make([]types.PlanDraftSlot, len(in))
	for i, s := range in {
		out[i] = types.PlanDraftSlot{
			Label:     s.Label,
			TimeOfDay: s.TimeOfDay,
			Options:   mapOptions(s.Options),
		}
	}
	return out
}

func mapOptions(in []wireOption) []types.PlanDraftOption {
	out := make([]types.PlanDraftOption, len(in))
	for i, o := range in {
		out[i] = types.PlanDraftOption{
			Label:               o.Label,
			Items:               mapItems(o.Items),
			LowConfidenceFields: o.LowConfidenceFields,
		}
	}
	return out
}

func mapItems(in []wireItem) []types.PlanDraftItem {
	out := make([]types.PlanDraftItem, len(in))
	for i, it := range in {
		out[i] = types.PlanDraftItem{
			RawName:   it.RawName,
			Quantity:  it.Quantity,
			Unit:      it.Unit,
			AdLibitum: it.AdLibitum,
		}
	}
	return out
}
