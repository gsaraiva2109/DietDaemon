// Package exportfmt renders meals and daily rollups as CSV. It is shared by
// the on-demand REST export endpoint and the scheduled backup job so both
// produce byte-identical output from a single implementation.
package exportfmt

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// WriteMealsCSV writes meals as CSV to w: id,date,raw_text,kcal,protein,carbs,fat,fiber.
func WriteMealsCSV(w io.Writer, meals []types.Meal) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	if err := cw.Write([]string{"id", "date", "raw_text", "kcal", "protein", "carbs", "fat", "fiber"}); err != nil {
		return err
	}
	for _, m := range meals {
		total := m.Total()
		if err := cw.Write([]string{
			m.ID, m.At.Format("2006-01-02"), m.RawText,
			fmt.Sprintf("%.0f", total.Calories), fmt.Sprintf("%.1f", total.Protein),
			fmt.Sprintf("%.1f", total.Carbs), fmt.Sprintf("%.1f", total.Fat), fmt.Sprintf("%.1f", total.Fiber),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteRollupsCSV writes daily rollups as CSV to w.
func WriteRollupsCSV(w io.Writer, rollups []types.DailyRollup) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	header := []string{
		"date", "consumed_kcal", "consumed_protein", "consumed_carbs", "consumed_fat", "consumed_fiber",
		"target_kcal", "target_protein", "target_carbs", "target_fat", "target_fiber",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, r := range rollups {
		if err := cw.Write([]string{
			r.Date,
			fmt.Sprintf("%.0f", r.Consumed.Calories), fmt.Sprintf("%.1f", r.Consumed.Protein),
			fmt.Sprintf("%.1f", r.Consumed.Carbs), fmt.Sprintf("%.1f", r.Consumed.Fat), fmt.Sprintf("%.1f", r.Consumed.Fiber),
			fmt.Sprintf("%.0f", r.Targets.Calories), fmt.Sprintf("%.1f", r.Targets.Protein),
			fmt.Sprintf("%.1f", r.Targets.Carbs), fmt.Sprintf("%.1f", r.Targets.Fat), fmt.Sprintf("%.1f", r.Targets.Fiber),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteWeightCSV writes weight entries as CSV to w: id,date,weight_kg,note.
// UserID and CreatedAt are not included; restore scopes rows to the -user
// CLI flag and re-stamps CreatedAt.
func WriteWeightCSV(w io.Writer, entries []types.WeightEntry) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	if err := cw.Write([]string{"id", "date", "weight_kg", "note"}); err != nil {
		return err
	}
	for _, e := range entries {
		if err := cw.Write([]string{e.ID, e.Date, fmt.Sprintf("%.2f", e.WeightKg), e.Note}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteMeasurementsCSV writes body measurement entries as CSV to w.
func WriteMeasurementsCSV(w io.Writer, entries []types.MeasurementEntry) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	header := []string{
		"id", "date", "waist_cm", "hips_cm", "chest_cm", "left_arm_cm", "right_arm_cm",
		"left_thigh_cm", "right_thigh_cm", "note",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, e := range entries {
		if err := cw.Write([]string{
			e.ID, e.Date,
			fmt.Sprintf("%.2f", e.WaistCm), fmt.Sprintf("%.2f", e.HipsCm), fmt.Sprintf("%.2f", e.ChestCm),
			fmt.Sprintf("%.2f", e.LeftArmCm), fmt.Sprintf("%.2f", e.RightArmCm),
			fmt.Sprintf("%.2f", e.LeftThighCm), fmt.Sprintf("%.2f", e.RightThighCm),
			e.Note,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteSleepCSV writes sleep logs as CSV to w: id,sleep_at,wake_at,quality,note.
// WakeAt writes as an empty field when nil (fast still in progress at backup time).
func WriteSleepCSV(w io.Writer, logs []types.SleepLog) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	if err := cw.Write([]string{"id", "sleep_at", "wake_at", "quality", "note"}); err != nil {
		return err
	}
	for _, s := range logs {
		wakeAt := ""
		if s.WakeAt != nil {
			wakeAt = *s.WakeAt
		}
		if err := cw.Write([]string{s.ID, s.SleepAt, wakeAt, s.Quality, s.Note}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteWorkoutsCSV writes workouts as CSV to w:
// id,name,duration_min,intensity,calories_burned,note,logged_at,external_id,exercises_json.
// Exercises marshal to a JSON array in exercises_json; CaloriesBurned and
// ExternalID write as empty fields when nil.
func WriteWorkoutsCSV(w io.Writer, workouts []types.Workout) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	header := []string{
		"id", "name", "duration_min", "intensity", "calories_burned", "note",
		"logged_at", "external_id", "exercises_json",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, wk := range workouts {
		caloriesBurned := ""
		if wk.CaloriesBurned != nil {
			caloriesBurned = fmt.Sprintf("%d", *wk.CaloriesBurned)
		}
		externalID := ""
		if wk.ExternalID != nil {
			externalID = *wk.ExternalID
		}
		exercisesJSON, err := json.Marshal(wk.Exercises)
		if err != nil {
			return fmt.Errorf("exportfmt: marshal exercises for workout %s: %w", wk.ID, err)
		}
		if err := cw.Write([]string{
			wk.ID, wk.Name, fmt.Sprintf("%d", wk.DurationMin), wk.Intensity, caloriesBurned,
			wk.Note, wk.LoggedAt, externalID, string(exercisesJSON),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteWaterCSV writes water logs as CSV to w: id,amount_ml,logged_at,note.
func WriteWaterCSV(w io.Writer, logs []types.WaterLog) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	if err := cw.Write([]string{"id", "amount_ml", "logged_at", "note"}); err != nil {
		return err
	}
	for _, l := range logs {
		if err := cw.Write([]string{l.ID, fmt.Sprintf("%d", l.AmountML), l.LoggedAt, l.Note}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteFastsCSV writes fasts as CSV to w: id,start_at,end_at,target_hours,completed.
// EndAt writes as an empty field when nil (fast still in progress at backup time).
func WriteFastsCSV(w io.Writer, fasts []types.Fast) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	if err := cw.Write([]string{"id", "start_at", "end_at", "target_hours", "completed"}); err != nil {
		return err
	}
	for _, f := range fasts {
		endAt := ""
		if f.EndAt != nil {
			endAt = f.EndAt.Format(time.RFC3339)
		}
		if err := cw.Write([]string{
			f.ID, f.StartAt.Format(time.RFC3339), endAt, fmt.Sprintf("%.2f", f.TargetHours), fmt.Sprintf("%v", f.Completed),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WritePhotosCSV writes a progress-photo metadata index as CSV to w:
// id,date,view,mime_type,filename. This is an index only — Data is not
// written here; each photo's blob is stored in a separate file named by
// PhotoFilename.
func WritePhotosCSV(w io.Writer, photos []types.ProgressPhoto) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	if err := cw.Write([]string{"id", "date", "view", "mime_type", "filename"}); err != nil {
		return err
	}
	for _, p := range photos {
		if err := cw.Write([]string{p.ID, p.Date, p.View, p.MimeType, PhotoFilename(p.ID)}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WritePlansCSV writes diet plans as CSV to w:
// id,name,notes,valid_from,valid_to,cycle_pattern_json,cycle_anchor_date,created_at,updated_at.
// UserID is not included; restore scopes rows to the -user CLI flag.
func WritePlansCSV(w io.Writer, plans []types.DietPlan) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	header := []string{
		"id", "name", "notes", "valid_from", "valid_to", "cycle_pattern_json",
		"cycle_anchor_date", "created_at", "updated_at",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, p := range plans {
		patternJSON, err := json.Marshal(p.CyclePattern)
		if err != nil {
			return fmt.Errorf("exportfmt: marshal cycle_pattern for plan %s: %w", p.ID, err)
		}
		if err := cw.Write([]string{
			p.ID, p.Name, p.Notes, p.ValidFrom, p.ValidTo,
			string(patternJSON), p.CycleAnchorDate,
			p.CreatedAt.UTC().Format(time.RFC3339), p.UpdatedAt.UTC().Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteDayTypesCSV writes diet plan day-types as CSV to w:
// id,plan_id,name,position,kcal,protein,carbs,fat,fiber,water_goal_ml.
func WriteDayTypesCSV(w io.Writer, dayTypes []types.DietPlanDayType) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	header := []string{"id", "plan_id", "name", "position", "kcal", "protein", "carbs", "fat", "fiber", "water_goal_ml"}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, dt := range dayTypes {
		if err := cw.Write([]string{
			dt.ID, dt.PlanID, dt.Name, fmt.Sprintf("%d", dt.Position),
			fmt.Sprintf("%.1f", dt.Targets.Calories), fmt.Sprintf("%.1f", dt.Targets.Protein),
			fmt.Sprintf("%.1f", dt.Targets.Carbs), fmt.Sprintf("%.1f", dt.Targets.Fat), fmt.Sprintf("%.1f", dt.Targets.Fiber),
			fmt.Sprintf("%d", dt.WaterGoalMl),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteSlotsCSV writes diet plan slots as CSV to w: id,day_type_id,position,time_of_day,label.
func WriteSlotsCSV(w io.Writer, slots []types.DietPlanSlot) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	if err := cw.Write([]string{"id", "day_type_id", "position", "time_of_day", "label"}); err != nil {
		return err
	}
	for _, sl := range slots {
		if err := cw.Write([]string{sl.ID, sl.DayTypeID, fmt.Sprintf("%d", sl.Position), sl.TimeOfDay, sl.Label}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteSlotOptionsCSV writes diet plan slot options as CSV to w: id,slot_id,position,label,template_id.
func WriteSlotOptionsCSV(w io.Writer, options []types.DietPlanSlotOption) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	if err := cw.Write([]string{"id", "slot_id", "position", "label", "template_id"}); err != nil {
		return err
	}
	for _, opt := range options {
		if err := cw.Write([]string{opt.ID, opt.SlotID, fmt.Sprintf("%d", opt.Position), opt.Label, opt.TemplateID}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteDayOverridesCSV writes diet plan day overrides as CSV to w: date,day_type_id.
// UserID is not included; restore scopes rows to the -user CLI flag.
func WriteDayOverridesCSV(w io.Writer, overrides []types.DietPlanDayOverride) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	if err := cw.Write([]string{"date", "day_type_id"}); err != nil {
		return err
	}
	for _, o := range overrides {
		if err := cw.Write([]string{o.Date, o.DayTypeID}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteTemplatesCSV writes meal templates (both user- and plan-owned) as CSV
// to w: id,name,owner_kind,created_at,last_used,items_json. UserID is not
// included; restore scopes rows to the -user CLI flag.
func WriteTemplatesCSV(w io.Writer, templates []types.MealTemplate) error {
	cw := csv.NewWriter(w)
	cw.UseCRLF = false
	header := []string{"id", "name", "owner_kind", "created_at", "last_used", "items_json"}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, t := range templates {
		itemsJSON, err := json.Marshal(t.Items)
		if err != nil {
			return fmt.Errorf("exportfmt: marshal items for template %s: %w", t.ID, err)
		}
		if err := cw.Write([]string{
			t.ID, t.Name, t.OwnerKind,
			t.CreatedAt.UTC().Format(time.RFC3339), t.LastUsed.UTC().Format(time.RFC3339),
			string(itemsJSON),
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// PhotoFilename returns the flat (no directory separators) filename used to
// store a progress photo's binary blob alongside the photos.csv index. Flat
// names matter: the localdisk backup destination strips any "/" via
// filepath.Base, so a nested path would silently collapse into the wrong
// file.
func PhotoFilename(id string) string {
	return "photo-" + id
}
