package hevy

import (
	"encoding/json"
	"fmt"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ToWorkout converts a Hevy API workout into a DietDaemon domain Workout.
// Aggregation policy (locked): per exercise, sets = count of Hevy set entries,
// reps/weight_kg = max value across that exercise's sets (nil-safe), raw per-set
// data serialized as JSON in note. CaloriesBurned is nil (Hevy doesn't report it).
func ToWorkout(userID string, hw HevyWorkout) (types.Workout, error) {
	exercises := make([]types.WorkoutExercise, 0, len(hw.Exercises))
	for _, he := range hw.Exercises {
		setsCount := len(he.Sets)
		var maxReps *int
		var maxWeight *float64
		for _, s := range he.Sets {
			if s.Reps != nil {
				if maxReps == nil || *s.Reps > *maxReps {
					maxReps = new(*s.Reps)
				}
			}
			if s.WeightKg != nil {
				if maxWeight == nil || *s.WeightKg > *maxWeight {
					maxWeight = new(*s.WeightKg)
				}
			}
		}

		rawSets, err := json.Marshal(he.Sets)
		if err != nil {
			return types.Workout{}, fmt.Errorf("hevy: marshal sets for exercise %q: %w", he.Title, err)
		}

		exercises = append(exercises, types.WorkoutExercise{
			Name:     he.Title,
			Sets:     new(setsCount),
			Reps:     maxReps,
			WeightKg: maxWeight,
			Note:     string(rawSets),
		})
	}

	durationMin := int(hw.EndTime.Sub(hw.StartTime).Minutes())
	if durationMin < 0 {
		durationMin = 0
	}

	return types.Workout{
		UserID:      userID,
		Name:        hw.Title,
		DurationMin: durationMin,
		Intensity:   "moderate",
		LoggedAt:    hw.StartTime.UTC().Format("2006-01-02T15:04:05Z"),
		ExternalID:  new(hw.ID),
		Exercises:   exercises,
	}, nil
}
