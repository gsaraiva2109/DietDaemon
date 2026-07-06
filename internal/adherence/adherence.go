// Package adherence computes diet adherence metrics from daily rollup data.
package adherence

import (
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Streak returns the number of consecutive days ending at the last rollup
// (the most recent completed day). Walk backward through rollups (which are
// in ascending date order) and stop at the first day outside the band, with
// no target, or a date gap. The caller passes data ending yesterday (not
// today), so the streak reflects only completed days.
func Streak(rollups []types.DailyRollup, floorPct, ceilPct float64) int {
	if len(rollups) == 0 {
		return 0
	}

	var prevDate string
	count := 0

	for i := len(rollups) - 1; i >= 0; i-- {
		r := rollups[i]

		// Check date gap: walking backward, the current date must be exactly
		// one day before the previous entry. Skip the check for the first
		// (most recent) entry.
		if prevDate != "" && !isPrevDay(prevDate, r.Date) {
			break
		}

		// Must have a positive calorie target.
		if r.Targets.Calories <= 0 {
			break
		}

		// Must be within the band.
		floor := r.Targets.Calories * floorPct
		ceil := r.Targets.Calories * ceilPct
		if r.Consumed.Calories < floor || r.Consumed.Calories > ceil {
			break
		}

		count++
		prevDate = r.Date
	}

	return count
}

// isPrevDay returns true when curr is exactly one calendar day before prev.
func isPrevDay(prev, curr string) bool {
	p, err := time.Parse("2006-01-02", prev)
	if err != nil {
		return false
	}
	c, err := time.Parse("2006-01-02", curr)
	if err != nil {
		return false
	}
	next := c.AddDate(0, 0, 1)
	return p.Equal(next)
}
