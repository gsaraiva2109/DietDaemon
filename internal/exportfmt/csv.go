// Package exportfmt renders meals and daily rollups as CSV. It is shared by
// the on-demand REST export endpoint and the scheduled backup job so both
// produce byte-identical output from a single implementation.
package exportfmt

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// WriteMealsCSV writes meals as CSV to w: id,date,raw_text,kcal,protein,carbs,fat,fiber.
func WriteMealsCSV(w io.Writer, meals []types.Meal) error {
	if _, err := fmt.Fprintln(w, "id,date,raw_text,kcal,protein,carbs,fat,fiber"); err != nil {
		return err
	}
	for _, m := range meals {
		total := m.Total()
		escaped := strings.ReplaceAll(m.RawText, `"`, `""`)
		if _, err := fmt.Fprintf(w, "%s,%s,\"%s\",%.0f,%.1f,%.1f,%.1f,%.1f\n",
			m.ID, m.At.Format("2006-01-02"), escaped,
			total.Calories, total.Protein, total.Carbs, total.Fat, total.Fiber,
		); err != nil {
			return err
		}
	}
	return nil
}

// WriteRollupsCSV writes daily rollups as CSV to w.
func WriteRollupsCSV(w io.Writer, rollups []types.DailyRollup) error {
	const header = "date,consumed_kcal,consumed_protein,consumed_carbs,consumed_fat,consumed_fiber,target_kcal,target_protein,target_carbs,target_fat,target_fiber"
	if _, err := fmt.Fprintln(w, header); err != nil {
		return err
	}
	for _, r := range rollups {
		if _, err := fmt.Fprintf(w, "%s,%.0f,%.1f,%.1f,%.1f,%.1f,%.0f,%.1f,%.1f,%.1f,%.1f\n",
			r.Date,
			r.Consumed.Calories, r.Consumed.Protein, r.Consumed.Carbs, r.Consumed.Fat, r.Consumed.Fiber,
			r.Targets.Calories, r.Targets.Protein, r.Targets.Carbs, r.Targets.Fat, r.Targets.Fiber,
		); err != nil {
			return err
		}
	}
	return nil
}

// csvEscape quote-wraps a free-text field, doubling any embedded quotes, the
// same way WriteMealsCSV escapes raw_text.
func csvEscape(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// WriteWeightCSV writes weight entries as CSV to w: id,date,weight_kg,note.
// UserID and CreatedAt are not included; restore scopes rows to the -user
// CLI flag and re-stamps CreatedAt.
func WriteWeightCSV(w io.Writer, entries []types.WeightEntry) error {
	if _, err := fmt.Fprintln(w, "id,date,weight_kg,note"); err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := fmt.Fprintf(w, "%s,%s,%.2f,%s\n", e.ID, e.Date, e.WeightKg, csvEscape(e.Note)); err != nil {
			return err
		}
	}
	return nil
}

// WriteMeasurementsCSV writes body measurement entries as CSV to w.
func WriteMeasurementsCSV(w io.Writer, entries []types.MeasurementEntry) error {
	const header = "id,date,waist_cm,hips_cm,chest_cm,left_arm_cm,right_arm_cm,left_thigh_cm,right_thigh_cm,note"
	if _, err := fmt.Fprintln(w, header); err != nil {
		return err
	}
	for _, e := range entries {
		if _, err := fmt.Fprintf(w, "%s,%s,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%.2f,%s\n",
			e.ID, e.Date, e.WaistCm, e.HipsCm, e.ChestCm, e.LeftArmCm, e.RightArmCm, e.LeftThighCm, e.RightThighCm,
			csvEscape(e.Note),
		); err != nil {
			return err
		}
	}
	return nil
}

// WriteSleepCSV writes sleep logs as CSV to w: id,sleep_at,wake_at,quality,note.
// WakeAt writes as an empty field when nil (fast still in progress at backup time).
func WriteSleepCSV(w io.Writer, logs []types.SleepLog) error {
	if _, err := fmt.Fprintln(w, "id,sleep_at,wake_at,quality,note"); err != nil {
		return err
	}
	for _, s := range logs {
		wakeAt := ""
		if s.WakeAt != nil {
			wakeAt = *s.WakeAt
		}
		if _, err := fmt.Fprintf(w, "%s,%s,%s,%s,%s\n", s.ID, s.SleepAt, wakeAt, s.Quality, csvEscape(s.Note)); err != nil {
			return err
		}
	}
	return nil
}

// WriteWorkoutsCSV writes workouts as CSV to w:
// id,name,duration_min,intensity,calories_burned,note,logged_at,external_id,exercises_json.
// Exercises marshal to a JSON array in exercises_json; CaloriesBurned and
// ExternalID write as empty fields when nil.
func WriteWorkoutsCSV(w io.Writer, workouts []types.Workout) error {
	const header = "id,name,duration_min,intensity,calories_burned,note,logged_at,external_id,exercises_json"
	if _, err := fmt.Fprintln(w, header); err != nil {
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
		if _, err := fmt.Fprintf(w, "%s,%s,%d,%s,%s,%s,%s,%s,%s\n",
			wk.ID, csvEscape(wk.Name), wk.DurationMin, wk.Intensity, caloriesBurned,
			csvEscape(wk.Note), wk.LoggedAt, externalID, csvEscape(string(exercisesJSON)),
		); err != nil {
			return err
		}
	}
	return nil
}

// WriteWaterCSV writes water logs as CSV to w: id,amount_ml,logged_at,note.
func WriteWaterCSV(w io.Writer, logs []types.WaterLog) error {
	if _, err := fmt.Fprintln(w, "id,amount_ml,logged_at,note"); err != nil {
		return err
	}
	for _, l := range logs {
		if _, err := fmt.Fprintf(w, "%s,%d,%s,%s\n", l.ID, l.AmountML, l.LoggedAt, csvEscape(l.Note)); err != nil {
			return err
		}
	}
	return nil
}

// WriteFastsCSV writes fasts as CSV to w: id,start_at,end_at,target_hours,completed.
// EndAt writes as an empty field when nil (fast still in progress at backup time).
func WriteFastsCSV(w io.Writer, fasts []types.Fast) error {
	if _, err := fmt.Fprintln(w, "id,start_at,end_at,target_hours,completed"); err != nil {
		return err
	}
	for _, f := range fasts {
		endAt := ""
		if f.EndAt != nil {
			endAt = f.EndAt.Format(time.RFC3339)
		}
		if _, err := fmt.Fprintf(w, "%s,%s,%s,%.2f,%v\n",
			f.ID, f.StartAt.Format(time.RFC3339), endAt, f.TargetHours, f.Completed,
		); err != nil {
			return err
		}
	}
	return nil
}

// WritePhotosCSV writes a progress-photo metadata index as CSV to w:
// id,date,view,mime_type,filename. This is an index only — Data is not
// written here; each photo's blob is stored in a separate file named by
// PhotoFilename.
func WritePhotosCSV(w io.Writer, photos []types.ProgressPhoto) error {
	if _, err := fmt.Fprintln(w, "id,date,view,mime_type,filename"); err != nil {
		return err
	}
	for _, p := range photos {
		if _, err := fmt.Fprintf(w, "%s,%s,%s,%s,%s\n", p.ID, p.Date, p.View, p.MimeType, PhotoFilename(p.ID)); err != nil {
			return err
		}
	}
	return nil
}

// PhotoFilename returns the flat (no directory separators) filename used to
// store a progress photo's binary blob alongside the photos.csv index. Flat
// names matter: the localdisk backup destination strips any "/" via
// filepath.Base, so a nested path would silently collapse into the wrong
// file.
func PhotoFilename(id string) string {
	return "photo-" + id
}
