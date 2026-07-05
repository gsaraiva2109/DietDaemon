// Package exportfmt renders meals and daily rollups as CSV. It is shared by
// the on-demand REST export endpoint and the scheduled backup job so both
// produce byte-identical output from a single implementation.
package exportfmt

import (
	"fmt"
	"io"
	"strings"

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
