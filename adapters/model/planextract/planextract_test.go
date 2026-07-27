package planextract

import (
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestParseResponse(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{
			name: "unfenced json",
			raw:  `{"plan_name":"Plano de treino","day_types":[{"name":"Dia normal","targets":{"Calories":2200,"Protein":150,"Carbs":220,"Fat":70,"Fiber":25},"water_goal_ml":3000,"slots":[{"label":"Café da manhã","time_of_day":"08:00","options":[{"label":"Opção 1","items":[{"raw_name":"Ovos mexidos","quantity":3,"unit":"unidade","ad_libitum":false}],"low_confidence_fields":[]}]}],"low_confidence_fields":[]}],"unreadable":false,"notes":null}`,
		},
		{
			name: "fenced json",
			raw:  "```json\n" + `{"plan_name":"Plan","day_types":[],"unreadable":false,"notes":null}` + "\n```",
		},
		{
			name: "partial nulls",
			raw:  `{"plan_name":null,"day_types":[{"name":"Rest day","targets":{"Calories":1800,"Protein":null,"Carbs":null,"Fat":null,"Fiber":null},"water_goal_ml":null,"slots":[],"low_confidence_fields":["water_goal_ml"]}],"unreadable":false,"notes":null}`,
		},
		{
			name: "unreadable",
			raw:  `{"plan_name":null,"day_types":[],"unreadable":true,"notes":null}`,
		},
		{
			name:    "malformed",
			raw:     `not json`,
			wantErr: true,
		},
		{
			name: "portuguese food names preserved",
			raw:  `{"plan_name":"Plano cutting","day_types":[{"name":"Dia único","targets":{"Calories":2000,"Protein":160,"Carbs":180,"Fat":60,"Fiber":30},"water_goal_ml":2500,"slots":[{"label":"Almoço","time_of_day":"12:30","options":[{"label":"Opção única","items":[{"raw_name":"Arroz integral","quantity":100,"unit":"g","ad_libitum":false},{"raw_name":"Salada à vontade","quantity":null,"unit":null,"ad_libitum":true}],"low_confidence_fields":[]}]}],"low_confidence_fields":[]}],"unreadable":false,"notes":"Beber bastante água"}`,
		},
		{
			name: "multi day type carb cycling, multiple slots and options",
			raw: `{"plan_name":"Carb cycling","day_types":[` +
				`{"name":"Training day","targets":{"Calories":2600,"Protein":180,"Carbs":300,"Fat":70,"Fiber":30},"water_goal_ml":3500,` +
				`"slots":[` +
				`{"label":"Breakfast","time_of_day":"07:00","options":[{"label":"Option 1","items":[{"raw_name":"Oats","quantity":80,"unit":"g","ad_libitum":false}],"low_confidence_fields":[]},{"label":"Option 2","items":[{"raw_name":"Eggs","quantity":4,"unit":"unit","ad_libitum":false}],"low_confidence_fields":["items"]}]},` +
				`{"label":"Lunch","time_of_day":"12:00","options":[{"label":"Option 1","items":[{"raw_name":"Rice","quantity":150,"unit":"g","ad_libitum":false},{"raw_name":"Chicken breast","quantity":200,"unit":"g","ad_libitum":false}],"low_confidence_fields":[]}]}` +
				`],"low_confidence_fields":[]},` +
				`{"name":"Rest day","targets":{"Calories":1900,"Protein":170,"Carbs":120,"Fat":65,"Fiber":25},"water_goal_ml":3000,` +
				`"slots":[` +
				`{"label":"Breakfast","time_of_day":"07:00","options":[{"label":"Option 1","items":[{"raw_name":"Eggs","quantity":4,"unit":"unit","ad_libitum":false}],"low_confidence_fields":[]}]}` +
				`],"low_confidence_fields":["water_goal_ml"]}` +
				`],"unreadable":false,"notes":null}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseResponse(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseResponse(%q) want error, got nil", tc.raw)
				}
				if !strings.Contains(err.Error(), "planextract:") {
					t.Errorf("error %q does not carry planextract: prefix", err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseResponse(%q) unexpected error: %v", tc.raw, err)
			}
			_ = got
		})
	}
}

func TestParseResponseFieldValues(t *testing.T) {
	got, err := ParseResponse(`{"plan_name":"Plano de treino","day_types":[{"name":"Dia normal","targets":{"Calories":2200,"Protein":150,"Carbs":220,"Fat":70,"Fiber":25},"water_goal_ml":3000,"slots":[{"label":"Café da manhã","time_of_day":"08:00","options":[{"label":"Opção 1","items":[{"raw_name":"Ovos mexidos","quantity":3,"unit":"unidade","ad_libitum":false},{"raw_name":"Salada à vontade","quantity":null,"unit":null,"ad_libitum":true}],"low_confidence_fields":["items"]}]}],"low_confidence_fields":["name"]}],"unreadable":false,"notes":"Notas gerais"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertPlanFields(t, got)

	dt := got.DayTypes[0]
	assertDayTypeFields(t, dt)

	slot := dt.Slots[0]
	assertSlotFields(t, slot)

	assertOptionFields(t, slot.Options[0])
}

func assertPlanFields(t *testing.T, got types.PlanDraft) {
	t.Helper()
	if got.PlanName == nil || *got.PlanName != "Plano de treino" {
		t.Errorf("PlanName = %v, want Plano de treino", got.PlanName)
	}
	if got.Notes == nil || *got.Notes != "Notas gerais" {
		t.Errorf("Notes = %v, want Notas gerais", got.Notes)
	}
	if got.Unreadable {
		t.Errorf("Unreadable = true, want false")
	}
	if len(got.DayTypes) != 1 {
		t.Fatalf("DayTypes len = %d, want 1", len(got.DayTypes))
	}
}

func assertDayTypeFields(t *testing.T, dt types.PlanDraftDayType) {
	t.Helper()
	if dt.Name != "Dia normal" {
		t.Errorf("DayType.Name = %q, want Dia normal", dt.Name)
	}
	if dt.Targets.Calories != 2200 || dt.Targets.Protein != 150 {
		t.Errorf("DayType.Targets = %+v, want Calories=2200 Protein=150", dt.Targets)
	}
	if dt.WaterGoalMl == nil || *dt.WaterGoalMl != 3000 {
		t.Errorf("DayType.WaterGoalMl = %v, want 3000", dt.WaterGoalMl)
	}
	if len(dt.LowConfidenceFields) != 1 || dt.LowConfidenceFields[0] != "name" {
		t.Errorf("DayType.LowConfidenceFields = %v, want [name]", dt.LowConfidenceFields)
	}
	if len(dt.Slots) != 1 {
		t.Fatalf("Slots len = %d, want 1", len(dt.Slots))
	}
}

func assertSlotFields(t *testing.T, slot types.PlanDraftSlot) {
	t.Helper()
	if slot.Label != "Café da manhã" {
		t.Errorf("Slot.Label = %q, want Café da manhã", slot.Label)
	}
	if slot.TimeOfDay == nil || *slot.TimeOfDay != "08:00" {
		t.Errorf("Slot.TimeOfDay = %v, want 08:00", slot.TimeOfDay)
	}
	if len(slot.Options) != 1 {
		t.Fatalf("Options len = %d, want 1", len(slot.Options))
	}
}

func assertOptionFields(t *testing.T, opt types.PlanDraftOption) {
	t.Helper()
	if len(opt.Items) != 2 {
		t.Fatalf("Items len = %d, want 2", len(opt.Items))
	}
	if opt.Items[0].RawName != "Ovos mexidos" {
		t.Errorf("Items[0].RawName = %q, want Ovos mexidos (verbatim, not translated)", opt.Items[0].RawName)
	}
	if opt.Items[0].Quantity == nil || *opt.Items[0].Quantity != 3 {
		t.Errorf("Items[0].Quantity = %v, want 3", opt.Items[0].Quantity)
	}
	if opt.Items[0].AdLibitum {
		t.Errorf("Items[0].AdLibitum = true, want false")
	}
	if !opt.Items[1].AdLibitum {
		t.Errorf("Items[1].AdLibitum = false, want true (à vontade)")
	}
	if opt.Items[1].Quantity != nil {
		t.Errorf("Items[1].Quantity = %v, want nil for ad_libitum item", opt.Items[1].Quantity)
	}
}

func TestParseResponseUnreadable(t *testing.T) {
	got, err := ParseResponse(`{"plan_name":null,"day_types":[],"unreadable":true,"notes":null}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Unreadable {
		t.Errorf("Unreadable = false, want true")
	}
	if got.PlanName != nil {
		t.Errorf("PlanName = %v, want nil", got.PlanName)
	}
	if len(got.DayTypes) != 0 {
		t.Errorf("DayTypes = %v, want empty", got.DayTypes)
	}
}
