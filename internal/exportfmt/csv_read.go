package exportfmt

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// PhotoIndexEntry pairs a ProgressPhoto's metadata (Data left empty) with the
// filename its blob is stored under, as read from a photos.csv index.
type PhotoIndexEntry struct {
	Photo    types.ProgressPhoto
	Filename string
}

// readAll reads r as CSV and checks that the header row matches want exactly
// (column count and names), returning the data rows (header excluded).
func readAll(r io.Reader, want []string) ([][]string, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("exportfmt: read csv: %w", err)
	}
	if len(records) < 1 {
		return nil, fmt.Errorf("exportfmt: empty csv, expected header %v", want)
	}
	got := records[0]
	if len(got) != len(want) {
		return nil, fmt.Errorf("exportfmt: csv header has %d columns, want %d (%v)", len(got), len(want), want)
	}
	for i, name := range want {
		if got[i] != name {
			return nil, fmt.Errorf("exportfmt: csv header column %d is %q, want %q", i, got[i], name)
		}
	}
	return records[1:], nil
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

// ReadRollupsCSV parses the format written by WriteRollupsCSV. UserID is left
// zero-value; the restore caller sets it before calling the store.
func ReadRollupsCSV(r io.Reader) ([]types.DailyRollup, error) {
	want := []string{"date", "consumed_kcal", "consumed_protein", "consumed_carbs", "consumed_fat", "consumed_fiber",
		"target_kcal", "target_protein", "target_carbs", "target_fat", "target_fiber"}
	rows, err := readAll(r, want)
	if err != nil {
		return nil, err
	}
	out := make([]types.DailyRollup, 0, len(rows))
	for _, rec := range rows {
		if len(rec) != len(want) {
			return nil, fmt.Errorf("exportfmt: rollups row has %d columns, want %d", len(rec), len(want))
		}
		consumedKcal, _ := parseFloat(rec[1])
		consumedProtein, _ := parseFloat(rec[2])
		consumedCarbs, _ := parseFloat(rec[3])
		consumedFat, _ := parseFloat(rec[4])
		consumedFiber, _ := parseFloat(rec[5])
		targetKcal, _ := parseFloat(rec[6])
		targetProtein, _ := parseFloat(rec[7])
		targetCarbs, _ := parseFloat(rec[8])
		targetFat, _ := parseFloat(rec[9])
		targetFiber, _ := parseFloat(rec[10])
		out = append(out, types.DailyRollup{
			Date:     rec[0],
			Consumed: types.Macros{Calories: consumedKcal, Protein: consumedProtein, Carbs: consumedCarbs, Fat: consumedFat, Fiber: consumedFiber},
			Targets:  types.Macros{Calories: targetKcal, Protein: targetProtein, Carbs: targetCarbs, Fat: targetFat, Fiber: targetFiber},
		})
	}
	return out, nil
}

// ReadWeightCSV parses the format written by WriteWeightCSV. UserID is left
// zero-value; the restore caller sets it before calling the store.
func ReadWeightCSV(r io.Reader) ([]types.WeightEntry, error) {
	want := []string{"id", "date", "weight_kg", "note"}
	rows, err := readAll(r, want)
	if err != nil {
		return nil, err
	}
	out := make([]types.WeightEntry, 0, len(rows))
	for _, rec := range rows {
		if len(rec) != len(want) {
			return nil, fmt.Errorf("exportfmt: weight row has %d columns, want %d", len(rec), len(want))
		}
		weightKg, _ := parseFloat(rec[2])
		out = append(out, types.WeightEntry{ID: rec[0], Date: rec[1], WeightKg: weightKg, Note: rec[3]})
	}
	return out, nil
}

// ReadMeasurementsCSV parses the format written by WriteMeasurementsCSV.
// UserID is left zero-value; the restore caller sets it before calling the store.
func ReadMeasurementsCSV(r io.Reader) ([]types.MeasurementEntry, error) {
	want := []string{"id", "date", "waist_cm", "hips_cm", "chest_cm", "left_arm_cm", "right_arm_cm", "left_thigh_cm", "right_thigh_cm", "note"}
	rows, err := readAll(r, want)
	if err != nil {
		return nil, err
	}
	out := make([]types.MeasurementEntry, 0, len(rows))
	for _, rec := range rows {
		if len(rec) != len(want) {
			return nil, fmt.Errorf("exportfmt: measurements row has %d columns, want %d", len(rec), len(want))
		}
		waist, _ := parseFloat(rec[2])
		hips, _ := parseFloat(rec[3])
		chest, _ := parseFloat(rec[4])
		leftArm, _ := parseFloat(rec[5])
		rightArm, _ := parseFloat(rec[6])
		leftThigh, _ := parseFloat(rec[7])
		rightThigh, _ := parseFloat(rec[8])
		out = append(out, types.MeasurementEntry{
			ID: rec[0], Date: rec[1],
			WaistCm: waist, HipsCm: hips, ChestCm: chest,
			LeftArmCm: leftArm, RightArmCm: rightArm,
			LeftThighCm: leftThigh, RightThighCm: rightThigh,
			Note: rec[9],
		})
	}
	return out, nil
}

// ReadSleepCSV parses the format written by WriteSleepCSV. UserID is left
// zero-value; the restore caller sets it before calling the store.
func ReadSleepCSV(r io.Reader) ([]types.SleepLog, error) {
	want := []string{"id", "sleep_at", "wake_at", "quality", "note"}
	rows, err := readAll(r, want)
	if err != nil {
		return nil, err
	}
	out := make([]types.SleepLog, 0, len(rows))
	for _, rec := range rows {
		if len(rec) != len(want) {
			return nil, fmt.Errorf("exportfmt: sleep row has %d columns, want %d", len(rec), len(want))
		}
		var wakeAt *string
		if rec[2] != "" {
			wakeAt = new(rec[2])
		}
		out = append(out, types.SleepLog{ID: rec[0], SleepAt: rec[1], WakeAt: wakeAt, Quality: rec[3], Note: rec[4]})
	}
	return out, nil
}

// ReadWaterCSV parses the format written by WriteWaterCSV. UserID is left
// zero-value; the restore caller sets it before calling the store.
func ReadWaterCSV(r io.Reader) ([]types.WaterLog, error) {
	want := []string{"id", "amount_ml", "logged_at", "note"}
	rows, err := readAll(r, want)
	if err != nil {
		return nil, err
	}
	out := make([]types.WaterLog, 0, len(rows))
	for _, rec := range rows {
		if len(rec) != len(want) {
			return nil, fmt.Errorf("exportfmt: water row has %d columns, want %d", len(rec), len(want))
		}
		amountML, err := strconv.Atoi(rec[1])
		if err != nil {
			return nil, fmt.Errorf("exportfmt: water row %q: parse amount_ml: %w", rec[0], err)
		}
		out = append(out, types.WaterLog{ID: rec[0], AmountML: amountML, LoggedAt: rec[2], Note: rec[3]})
	}
	return out, nil
}

// ReadFastsCSV parses the format written by WriteFastsCSV. UserID is left
// zero-value; the restore caller sets it before calling the store.
func ReadFastsCSV(r io.Reader) ([]types.Fast, error) {
	want := []string{"id", "start_at", "end_at", "target_hours", "completed"}
	rows, err := readAll(r, want)
	if err != nil {
		return nil, err
	}
	out := make([]types.Fast, 0, len(rows))
	for _, rec := range rows {
		if len(rec) != len(want) {
			return nil, fmt.Errorf("exportfmt: fasts row has %d columns, want %d", len(rec), len(want))
		}
		startAt, err := time.Parse(time.RFC3339, rec[1])
		if err != nil {
			return nil, fmt.Errorf("exportfmt: fasts row %q: parse start_at: %w", rec[0], err)
		}
		var endAt *time.Time
		if rec[2] != "" {
			v, err := time.Parse(time.RFC3339, rec[2])
			if err != nil {
				return nil, fmt.Errorf("exportfmt: fasts row %q: parse end_at: %w", rec[0], err)
			}
			endAt = &v
		}
		targetHours, _ := parseFloat(rec[3])
		completed, _ := strconv.ParseBool(rec[4])
		out = append(out, types.Fast{ID: rec[0], StartAt: startAt, EndAt: endAt, TargetHours: targetHours, Completed: completed})
	}
	return out, nil
}

// ReadWorkoutsCSV parses the format written by WriteWorkoutsCSV, including
// exercises_json back into Exercises. UserID is left zero-value; the restore
// caller sets it before calling the store.
func ReadWorkoutsCSV(r io.Reader) ([]types.Workout, error) {
	want := []string{"id", "name", "duration_min", "intensity", "calories_burned", "note", "logged_at", "external_id", "exercises_json"}
	rows, err := readAll(r, want)
	if err != nil {
		return nil, err
	}
	out := make([]types.Workout, 0, len(rows))
	for _, rec := range rows {
		if len(rec) != len(want) {
			return nil, fmt.Errorf("exportfmt: workouts row has %d columns, want %d", len(rec), len(want))
		}
		durationMin, err := strconv.Atoi(rec[2])
		if err != nil {
			return nil, fmt.Errorf("exportfmt: workouts row %q: parse duration_min: %w", rec[0], err)
		}
		var caloriesBurned *int
		if rec[4] != "" {
			v, err := strconv.Atoi(rec[4])
			if err != nil {
				return nil, fmt.Errorf("exportfmt: workouts row %q: parse calories_burned: %w", rec[0], err)
			}
			caloriesBurned = &v
		}
		var externalID *string
		if rec[7] != "" {
			externalID = new(rec[7])
		}
		wk := types.Workout{
			ID: rec[0], Name: rec[1], DurationMin: durationMin, Intensity: rec[3],
			CaloriesBurned: caloriesBurned, Note: rec[5], LoggedAt: rec[6], ExternalID: externalID,
		}
		if rec[8] != "" {
			if err := json.Unmarshal([]byte(rec[8]), &wk.Exercises); err != nil {
				return nil, fmt.Errorf("exportfmt: workouts row %q: parse exercises_json: %w", rec[0], err)
			}
		}
		out = append(out, wk)
	}
	return out, nil
}

// ReadPhotosCSV parses the metadata index format written by WritePhotosCSV.
// Photo.Data is left empty; the blob is read separately from the file named
// by Filename.
func ReadPhotosCSV(r io.Reader) ([]PhotoIndexEntry, error) {
	want := []string{"id", "date", "view", "mime_type", "filename"}
	rows, err := readAll(r, want)
	if err != nil {
		return nil, err
	}
	out := make([]PhotoIndexEntry, 0, len(rows))
	for _, rec := range rows {
		if len(rec) != len(want) {
			return nil, fmt.Errorf("exportfmt: photos row has %d columns, want %d", len(rec), len(want))
		}
		out = append(out, PhotoIndexEntry{
			Photo:    types.ProgressPhoto{ID: rec[0], Date: rec[1], View: rec[2], MimeType: rec[3]},
			Filename: rec[4],
		})
	}
	return out, nil
}

// ReadMealsCSV parses the format written by WriteMealsCSV.
//
// This reconstruction is inherently LOSSY: meals.csv only carries meal-level
// macro totals, not the per-item breakdown, and only a date, not a
// time-of-day. Each row becomes a Meal with exactly one synthetic
// ResolvedItem whose Macros equal the row's totals, and At set to the row's
// date at midnight UTC. This is a property of the existing meals.csv export
// format (already used by production backups) and is not fixable here
// without breaking already-taken backups.
func ReadMealsCSV(r io.Reader) ([]types.Meal, error) {
	want := []string{"id", "date", "raw_text", "kcal", "protein", "carbs", "fat", "fiber"}
	rows, err := readAll(r, want)
	if err != nil {
		return nil, err
	}
	out := make([]types.Meal, 0, len(rows))
	for _, rec := range rows {
		if len(rec) != len(want) {
			return nil, fmt.Errorf("exportfmt: meals row has %d columns, want %d", len(rec), len(want))
		}
		at, err := time.Parse("2006-01-02", rec[1])
		if err != nil {
			return nil, fmt.Errorf("exportfmt: meals row %q: parse date: %w", rec[0], err)
		}
		kcal, _ := parseFloat(rec[3])
		protein, _ := parseFloat(rec[4])
		carbs, _ := parseFloat(rec[5])
		fat, _ := parseFloat(rec[6])
		fiber, _ := parseFloat(rec[7])
		macros := types.Macros{Calories: kcal, Protein: protein, Carbs: carbs, Fat: fat, Fiber: fiber}
		out = append(out, types.Meal{
			ID:      rec[0],
			At:      at,
			RawText: rec[2],
			Items: []types.ResolvedItem{
				{Macros: macros},
			},
			Confidence: 1,
			ParserTier: types.TierDeterministic,
			CreatedAt:  time.Now().UTC(),
		})
	}
	return out, nil
}
