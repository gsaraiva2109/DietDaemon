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
		ex, err := toWorkoutExercise(he)
		if err != nil {
			return types.Workout{}, err
		}
		exercises = append(exercises, ex)
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

// toWorkoutExercise converts one Hevy exercise into a domain WorkoutExercise,
// per the aggregation policy documented on ToWorkout.
func toWorkoutExercise(he HevyExercise) (types.WorkoutExercise, error) {
	setsCount := len(he.Sets)
	maxReps, maxWeight := aggregateSets(he.Sets)

	rawSets, err := json.Marshal(he.Sets)
	if err != nil {
		return types.WorkoutExercise{}, fmt.Errorf("hevy: marshal sets for exercise %q: %w", he.Title, err)
	}

	return types.WorkoutExercise{
		Name:     he.Title,
		Sets:     new(setsCount),
		Reps:     maxReps,
		WeightKg: maxWeight,
		Note:     string(rawSets),
	}, nil
}

// aggregateSets computes the nil-safe max reps and max weight_kg across sets.
func aggregateSets(sets []HevySet) (maxReps *int, maxWeight *float64) {
	for _, s := range sets {
		if s.Reps != nil && (maxReps == nil || *s.Reps > *maxReps) {
			maxReps = new(*s.Reps)
		}
		if s.WeightKg != nil && (maxWeight == nil || *s.WeightKg > *maxWeight) {
			maxWeight = new(*s.WeightKg)
		}
	}
	return maxReps, maxWeight
}
