package exportfmt

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestMealsCSVRoundTrip(t *testing.T) {
	meals := []types.Meal{
		{
			ID:      "m1",
			At:      time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
			RawText: `chicken, "extra" rice`,
			Items: []types.ResolvedItem{
				{Macros: types.Macros{Calories: 500, Protein: 40, Carbs: 50, Fat: 10, Fiber: 5}},
			},
		},
	}
	var buf bytes.Buffer
	if err := WriteMealsCSV(&buf, meals); err != nil {
		t.Fatalf("WriteMealsCSV: %v", err)
	}
	got, err := ReadMealsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadMealsCSV: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d meals, want 1", len(got))
	}
	if got[0].ID != "m1" || !got[0].At.Equal(meals[0].At) || got[0].RawText != meals[0].RawText {
		t.Errorf("got %+v, want id/at/rawtext matching %+v", got[0], meals[0])
	}
	if got[0].Total() != meals[0].Total() {
		t.Errorf("Total() = %+v, want %+v", got[0].Total(), meals[0].Total())
	}
}

func TestReadMealsCSV_LossyReconstruction(t *testing.T) {
	meals := []types.Meal{
		{
			ID:      "m1",
			At:      time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
			RawText: "chicken and rice",
			Items: []types.ResolvedItem{
				{Macros: types.Macros{Calories: 300, Protein: 30, Carbs: 20, Fat: 5, Fiber: 2}},
				{Macros: types.Macros{Calories: 200, Protein: 10, Carbs: 30, Fat: 5, Fiber: 3}},
			},
		},
		{
			ID:      "m2",
			At:      time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC),
			RawText: "eggs and toast",
			Items: []types.ResolvedItem{
				{Macros: types.Macros{Calories: 150, Protein: 12, Carbs: 5, Fat: 10, Fiber: 1}},
				{Macros: types.Macros{Calories: 120, Protein: 4, Carbs: 22, Fat: 2, Fiber: 1}},
				{Macros: types.Macros{Calories: 80, Protein: 2, Carbs: 15, Fat: 1, Fiber: 1}},
			},
		},
	}
	var buf bytes.Buffer
	if err := WriteMealsCSV(&buf, meals); err != nil {
		t.Fatalf("WriteMealsCSV: %v", err)
	}
	got, err := ReadMealsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadMealsCSV: %v", err)
	}
	if len(got) != len(meals) {
		t.Fatalf("got %d meals, want %d", len(got), len(meals))
	}
	for i, m := range got {
		if len(m.Items) != 1 {
			t.Errorf("meal %d: got %d items, want exactly 1", i, len(m.Items))
		}
		if m.Total() != meals[i].Total() {
			t.Errorf("meal %d: Total() = %+v, want %+v", i, m.Total(), meals[i].Total())
		}
	}
}

func TestRollupsCSVRoundTrip(t *testing.T) {
	rollups := []types.DailyRollup{
		{
			Date:     "2026-07-15",
			Consumed: types.Macros{Calories: 2200, Protein: 150.5, Carbs: 200.1, Fat: 60.2, Fiber: 25.3},
			Targets:  types.Macros{Calories: 2400, Protein: 160, Carbs: 220, Fat: 70, Fiber: 30},
		},
	}
	var buf bytes.Buffer
	if err := WriteRollupsCSV(&buf, rollups); err != nil {
		t.Fatalf("WriteRollupsCSV: %v", err)
	}
	got, err := ReadRollupsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadRollupsCSV: %v", err)
	}
	if len(got) != 1 || got[0].Date != rollups[0].Date {
		t.Fatalf("got %+v, want date %s", got, rollups[0].Date)
	}
	if got[0].Consumed != rollups[0].Consumed || got[0].Targets != rollups[0].Targets {
		t.Errorf("got %+v, want %+v", got[0], rollups[0])
	}
}

func TestWeightCSVRoundTrip(t *testing.T) {
	entries := []types.WeightEntry{
		{ID: "w1", Date: "2026-07-15", WeightKg: 82.35, Note: `feeling "great" today, up a bit`},
	}
	var buf bytes.Buffer
	if err := WriteWeightCSV(&buf, entries); err != nil {
		t.Fatalf("WriteWeightCSV: %v", err)
	}
	got, err := ReadWeightCSV(&buf)
	if err != nil {
		t.Fatalf("ReadWeightCSV: %v", err)
	}
	if len(got) != 1 || got[0] != entries[0] {
		t.Errorf("got %+v, want %+v", got, entries)
	}
}

func TestMeasurementsCSVRoundTrip(t *testing.T) {
	entries := []types.MeasurementEntry{
		{
			ID: "meas1", Date: "2026-07-15",
			WaistCm: 80.5, HipsCm: 95.25, ChestCm: 100.1,
			LeftArmCm: 30.2, RightArmCm: 30.4,
			LeftThighCm: 55.6, RightThighCm: 55.8,
			Note: "post-workout",
		},
	}
	var buf bytes.Buffer
	if err := WriteMeasurementsCSV(&buf, entries); err != nil {
		t.Fatalf("WriteMeasurementsCSV: %v", err)
	}
	got, err := ReadMeasurementsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadMeasurementsCSV: %v", err)
	}
	if len(got) != 1 || got[0] != entries[0] {
		t.Errorf("got %+v, want %+v", got, entries)
	}
}

func TestSleepCSVRoundTrip(t *testing.T) {
	wake := "2026-07-15 07:00:00"
	entries := []types.SleepLog{
		{ID: "s1", SleepAt: "2026-07-14 23:00:00", WakeAt: &wake, Quality: "good", Note: "slept well"},
		{ID: "s2", SleepAt: "2026-07-15 23:00:00", WakeAt: nil, Quality: "", Note: ""},
	}
	var buf bytes.Buffer
	if err := WriteSleepCSV(&buf, entries); err != nil {
		t.Fatalf("WriteSleepCSV: %v", err)
	}
	got, err := ReadSleepCSV(&buf)
	if err != nil {
		t.Fatalf("ReadSleepCSV: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d entries, want 2", len(got))
	}
	if got[0].ID != entries[0].ID || got[0].WakeAt == nil || *got[0].WakeAt != wake || got[0].Quality != "good" || got[0].Note != "slept well" {
		t.Errorf("row 0: got %+v", got[0])
	}
	if got[1].WakeAt != nil {
		t.Errorf("row 1: WakeAt = %v, want nil", *got[1].WakeAt)
	}
}

func TestWorkoutsCSVRoundTrip(t *testing.T) {
	sets, reps := 3, 10
	weight := 60.5
	calories := 350
	extID := "hevy-123"
	workouts := []types.Workout{
		{
			ID: "wo1", Name: "Leg Day, heavy", DurationMin: 45, Intensity: "high",
			CaloriesBurned: &calories, Note: `felt "strong"`, LoggedAt: "2026-07-15 18:00:00",
			ExternalID: &extID,
			Exercises: []types.WorkoutExercise{
				{ID: "ex1", WorkoutID: "wo1", Name: "Squat", Sets: &sets, Reps: &reps, WeightKg: &weight, Note: "PR"},
			},
		},
		{
			ID: "wo2", Name: "Rest day walk", DurationMin: 20, Intensity: "low",
			LoggedAt: "2026-07-16 08:00:00",
		},
	}
	var buf bytes.Buffer
	if err := WriteWorkoutsCSV(&buf, workouts); err != nil {
		t.Fatalf("WriteWorkoutsCSV: %v", err)
	}
	got, err := ReadWorkoutsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadWorkoutsCSV: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d workouts, want 2", len(got))
	}
	if got[0].Name != workouts[0].Name || got[0].Note != workouts[0].Note ||
		got[0].CaloriesBurned == nil || *got[0].CaloriesBurned != calories ||
		got[0].ExternalID == nil || *got[0].ExternalID != extID {
		t.Errorf("workout 0: got %+v", got[0])
	}
	if len(got[0].Exercises) != 1 || got[0].Exercises[0].Name != "Squat" ||
		*got[0].Exercises[0].Sets != sets || *got[0].Exercises[0].Reps != reps || *got[0].Exercises[0].WeightKg != weight {
		t.Errorf("workout 0 exercises: got %+v", got[0].Exercises)
	}
	if got[1].CaloriesBurned != nil || got[1].ExternalID != nil || len(got[1].Exercises) != 0 {
		t.Errorf("workout 1: got %+v, want nil-able fields nil/empty", got[1])
	}
}

func TestReadWorkoutsCSV_Errors(t *testing.T) {
	const header = "id,name,duration_min,intensity,calories_burned,note,logged_at,external_id,exercises_json\n"
	tests := []struct {
		name    string
		row     string
		wantErr string
	}{
		{
			name:    "invalid duration_min",
			row:     "wo1,Leg Day,not-a-number,high,,,2026-07-15 18:00:00,,\n",
			wantErr: "parse duration_min",
		},
		{
			name:    "invalid calories_burned",
			row:     "wo1,Leg Day,45,high,not-a-number,,2026-07-15 18:00:00,,\n",
			wantErr: "parse calories_burned",
		},
		{
			name:    "invalid exercises_json",
			row:     "wo1,Leg Day,45,high,,,2026-07-15 18:00:00,,{not valid json\n",
			wantErr: "parse exercises_json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadWorkoutsCSV(strings.NewReader(header + tt.row))
			if err == nil {
				t.Fatalf("ReadWorkoutsCSV: want error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ReadWorkoutsCSV error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestWaterCSVRoundTrip(t *testing.T) {
	logs := []types.WaterLog{
		{ID: "wa1", AmountML: 250, LoggedAt: "2026-07-15 10:00:00", Note: "post-run"},
	}
	var buf bytes.Buffer
	if err := WriteWaterCSV(&buf, logs); err != nil {
		t.Fatalf("WriteWaterCSV: %v", err)
	}
	got, err := ReadWaterCSV(&buf)
	if err != nil {
		t.Fatalf("ReadWaterCSV: %v", err)
	}
	if len(got) != 1 || got[0] != logs[0] {
		t.Errorf("got %+v, want %+v", got, logs)
	}
}

func TestFastsCSVRoundTrip(t *testing.T) {
	start := time.Date(2026, 7, 14, 20, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	fasts := []types.Fast{
		{ID: "f1", StartAt: start, EndAt: &end, TargetHours: 16, Completed: true},
		{ID: "f2", StartAt: start, EndAt: nil, TargetHours: 18.5, Completed: false},
	}
	var buf bytes.Buffer
	if err := WriteFastsCSV(&buf, fasts); err != nil {
		t.Fatalf("WriteFastsCSV: %v", err)
	}
	got, err := ReadFastsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadFastsCSV: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d fasts, want 2", len(got))
	}
	if !got[0].StartAt.Equal(start) || got[0].EndAt == nil || !got[0].EndAt.Equal(end) || got[0].TargetHours != 16 || !got[0].Completed {
		t.Errorf("fast 0: got %+v", got[0])
	}
	if !got[1].StartAt.Equal(start) || got[1].EndAt != nil || got[1].TargetHours != 18.5 || got[1].Completed {
		t.Errorf("fast 1: got %+v", got[1])
	}
}

func TestPlansCSVRoundTrip(t *testing.T) {
	plans := []types.DietPlan{
		{
			ID: "p1", Name: `Cutting, "cycle"`, Notes: "from Dra. Ana",
			ValidFrom: "2026-01-05", ValidTo: "2026-06-30",
			CyclePattern:    []string{"dt-low", "dt-high"},
			CycleAnchorDate: "2026-01-05",
			CreatedAt:       time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC),
			UpdatedAt:       time.Date(2026, 1, 6, 11, 0, 0, 0, time.UTC),
		},
		{
			ID: "p2", Name: "Open-ended", ValidFrom: "2026-07-01", ValidTo: "",
			CyclePattern:    nil,
			CycleAnchorDate: "2026-07-01",
			CreatedAt:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	var buf bytes.Buffer
	if err := WritePlansCSV(&buf, plans); err != nil {
		t.Fatalf("WritePlansCSV: %v", err)
	}
	got, err := ReadPlansCSV(&buf)
	if err != nil {
		t.Fatalf("ReadPlansCSV: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d plans, want 2", len(got))
	}
	if got[0].Name != plans[0].Name || len(got[0].CyclePattern) != 2 || got[0].CyclePattern[0] != "dt-low" ||
		got[0].ValidTo != plans[0].ValidTo || !got[0].CreatedAt.Equal(plans[0].CreatedAt) || !got[0].UpdatedAt.Equal(plans[0].UpdatedAt) {
		t.Errorf("plan 0: got %+v", got[0])
	}
	if len(got[1].CyclePattern) != 0 || got[1].ValidTo != "" {
		t.Errorf("plan 1: got %+v, want empty cycle_pattern and valid_to", got[1])
	}
}

func TestReadPlansCSV_Errors(t *testing.T) {
	const header = "id,name,notes,valid_from,valid_to,cycle_pattern_json,cycle_anchor_date,created_at,updated_at\n"
	tests := []struct {
		name    string
		row     string
		wantErr string
	}{
		{
			name:    "invalid cycle_pattern_json",
			row:     "p1,X,,2026-01-01,,{not valid,2026-01-01,2026-01-01T00:00:00Z,2026-01-01T00:00:00Z\n",
			wantErr: "parse cycle_pattern_json",
		},
		{
			name:    "invalid created_at",
			row:     "p1,X,,2026-01-01,,,2026-01-01,not-a-time,2026-01-01T00:00:00Z\n",
			wantErr: "parse created_at",
		},
		{
			name:    "invalid updated_at",
			row:     "p1,X,,2026-01-01,,,2026-01-01,2026-01-01T00:00:00Z,not-a-time\n",
			wantErr: "parse updated_at",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadPlansCSV(strings.NewReader(header + tt.row))
			if err == nil {
				t.Fatalf("ReadPlansCSV: want error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ReadPlansCSV error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestDayTypesCSVRoundTrip(t *testing.T) {
	dayTypes := []types.DietPlanDayType{
		{
			ID: "dt1", PlanID: "p1", Name: `Low-carb, "cheat day"`, Position: 0,
			Targets:     types.Macros{Calories: 1800, Protein: 150.5, Carbs: 100.2, Fat: 60.1, Fiber: 25.3},
			WaterGoalMl: 3000,
		},
	}
	var buf bytes.Buffer
	if err := WriteDayTypesCSV(&buf, dayTypes); err != nil {
		t.Fatalf("WriteDayTypesCSV: %v", err)
	}
	got, err := ReadDayTypesCSV(&buf)
	if err != nil {
		t.Fatalf("ReadDayTypesCSV: %v", err)
	}
	if len(got) != 1 || got[0].ID != "dt1" || got[0].Name != dayTypes[0].Name ||
		got[0].Targets != dayTypes[0].Targets || got[0].WaterGoalMl != 3000 {
		t.Errorf("got %+v, want %+v", got, dayTypes)
	}
}

func TestReadDayTypesCSV_Errors(t *testing.T) {
	const header = "id,plan_id,name,position,kcal,protein,carbs,fat,fiber,water_goal_ml\n"
	tests := []struct {
		name    string
		row     string
		wantErr string
	}{
		{name: "invalid position", row: "dt1,p1,X,not-a-number,1800,150,100,60,25,3000\n", wantErr: "parse position"},
		{name: "invalid water_goal_ml", row: "dt1,p1,X,0,1800,150,100,60,25,not-a-number\n", wantErr: "parse water_goal_ml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadDayTypesCSV(strings.NewReader(header + tt.row))
			if err == nil {
				t.Fatalf("ReadDayTypesCSV: want error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ReadDayTypesCSV error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestSlotsCSVRoundTrip(t *testing.T) {
	slots := []types.DietPlanSlot{
		{ID: "sl1", DayTypeID: "dt1", Position: 0, TimeOfDay: "07:00", Label: `Café da manhã, "cedo"`},
	}
	var buf bytes.Buffer
	if err := WriteSlotsCSV(&buf, slots); err != nil {
		t.Fatalf("WriteSlotsCSV: %v", err)
	}
	got, err := ReadSlotsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadSlotsCSV: %v", err)
	}
	if len(got) != 1 || got[0] != slots[0] {
		t.Errorf("got %+v, want %+v", got, slots)
	}
}

func TestReadSlotsCSV_Errors(t *testing.T) {
	const header = "id,day_type_id,position,time_of_day,label\n"
	_, err := ReadSlotsCSV(strings.NewReader(header + "sl1,dt1,not-a-number,07:00,X\n"))
	if err == nil || !strings.Contains(err.Error(), "parse position") {
		t.Errorf("ReadSlotsCSV error = %v, want substring %q", err, "parse position")
	}
}

func TestSlotOptionsCSVRoundTrip(t *testing.T) {
	options := []types.DietPlanSlotOption{
		{ID: "opt1", SlotID: "sl1", Position: 0, Label: `Opção 1, "principal"`, TemplateID: "tmpl1"},
	}
	var buf bytes.Buffer
	if err := WriteSlotOptionsCSV(&buf, options); err != nil {
		t.Fatalf("WriteSlotOptionsCSV: %v", err)
	}
	got, err := ReadSlotOptionsCSV(&buf)
	if err != nil {
		t.Fatalf("ReadSlotOptionsCSV: %v", err)
	}
	if len(got) != 1 || got[0] != options[0] {
		t.Errorf("got %+v, want %+v", got, options)
	}
}

func TestReadSlotOptionsCSV_Errors(t *testing.T) {
	const header = "id,slot_id,position,label,template_id\n"
	_, err := ReadSlotOptionsCSV(strings.NewReader(header + "opt1,sl1,not-a-number,X,tmpl1\n"))
	if err == nil || !strings.Contains(err.Error(), "parse position") {
		t.Errorf("ReadSlotOptionsCSV error = %v, want substring %q", err, "parse position")
	}
}

func TestDayOverridesCSVRoundTrip(t *testing.T) {
	overrides := []types.DietPlanDayOverride{
		{Date: "2026-07-15", DayTypeID: "dt-low"},
		{Date: "2026-07-16", DayTypeID: "dt-high"},
	}
	var buf bytes.Buffer
	if err := WriteDayOverridesCSV(&buf, overrides); err != nil {
		t.Fatalf("WriteDayOverridesCSV: %v", err)
	}
	got, err := ReadDayOverridesCSV(&buf)
	if err != nil {
		t.Fatalf("ReadDayOverridesCSV: %v", err)
	}
	if len(got) != 2 || got[0].Date != overrides[0].Date || got[0].DayTypeID != overrides[0].DayTypeID ||
		got[1].Date != overrides[1].Date || got[1].DayTypeID != overrides[1].DayTypeID {
		t.Errorf("got %+v, want %+v (UserID intentionally left zero-value)", got, overrides)
	}
}

func TestTemplatesCSVRoundTrip(t *testing.T) {
	templates := []types.MealTemplate{
		{
			ID: "t1", Name: `Café da manhã, "padrão"`, OwnerKind: types.TemplateOwnerPlan,
			CreatedAt: time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC),
			LastUsed:  time.Date(2026, 1, 2, 8, 0, 0, 0, time.UTC),
			Items: []types.ResolvedItem{
				{Macros: types.Macros{Calories: 300, Protein: 20, Carbs: 30, Fat: 10, Fiber: 5}},
			},
		},
		{
			ID: "t2", Name: "No items", OwnerKind: types.TemplateOwnerUser,
			CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			LastUsed:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	var buf bytes.Buffer
	if err := WriteTemplatesCSV(&buf, templates); err != nil {
		t.Fatalf("WriteTemplatesCSV: %v", err)
	}
	got, err := ReadTemplatesCSV(&buf)
	if err != nil {
		t.Fatalf("ReadTemplatesCSV: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d templates, want 2", len(got))
	}
	if got[0].ID != "t1" || got[0].Name != templates[0].Name || got[0].OwnerKind != types.TemplateOwnerPlan ||
		!got[0].CreatedAt.Equal(templates[0].CreatedAt) || !got[0].LastUsed.Equal(templates[0].LastUsed) ||
		len(got[0].Items) != 1 || got[0].Items[0].Macros != templates[0].Items[0].Macros {
		t.Errorf("template 0: got %+v", got[0])
	}
	if got[1].OwnerKind != types.TemplateOwnerUser || len(got[1].Items) != 0 {
		t.Errorf("template 1: got %+v", got[1])
	}
}

func TestReadTemplatesCSV_Errors(t *testing.T) {
	const header = "id,name,owner_kind,created_at,last_used,items_json\n"
	tests := []struct {
		name    string
		row     string
		wantErr string
	}{
		{name: "invalid created_at", row: "t1,X,user,not-a-time,2026-01-01T00:00:00Z,\n", wantErr: "parse created_at"},
		{name: "invalid last_used", row: "t1,X,user,2026-01-01T00:00:00Z,not-a-time,\n", wantErr: "parse last_used"},
		{name: "invalid items_json", row: "t1,X,user,2026-01-01T00:00:00Z,2026-01-01T00:00:00Z,{not valid json\n", wantErr: "parse items_json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadTemplatesCSV(strings.NewReader(header + tt.row))
			if err == nil {
				t.Fatalf("ReadTemplatesCSV: want error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ReadTemplatesCSV error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestPhotosCSVRoundTrip(t *testing.T) {
	photos := []types.ProgressPhoto{
		{ID: "p1", Date: "2026-07-15", View: "front", MimeType: "image/jpeg", Data: []byte("blob-bytes-not-written")},
	}
	var buf bytes.Buffer
	if err := WritePhotosCSV(&buf, photos); err != nil {
		t.Fatalf("WritePhotosCSV: %v", err)
	}
	got, err := ReadPhotosCSV(&buf)
	if err != nil {
		t.Fatalf("ReadPhotosCSV: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d photos, want 1", len(got))
	}
	p := got[0].Photo
	if p.ID != "p1" || p.Date != "2026-07-15" || p.View != "front" || p.MimeType != "image/jpeg" ||
		len(p.Data) != 0 || got[0].Filename != PhotoFilename("p1") {
		t.Errorf("got %+v, want id=p1 date=2026-07-15 view=front mime=image/jpeg data=empty filename=%s", got[0], PhotoFilename("p1"))
	}
}
