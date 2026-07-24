package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// ---------------------------------------------------------------------------
// Goals & profile handlers -- profile CRUD, TDEE calculation, goal suggestions.
// ---------------------------------------------------------------------------

const dateLayout = "2006-01-02"

const errProfileMeasurementsOutOfRange = "profile measurements are out of range"

func (h *Handler) handleGetProfile(w http.ResponseWriter, r *http.Request, userID string) {
	profile, err := h.store.GetProfile(r.Context(), userID)
	if err != nil && !errors.Is(err, types.ErrNotFound) {
		h.writeErr(w, err)
		return
	}
	if errors.Is(err, types.ErrNotFound) {
		profile = types.UserProfile{UserID: userID, Onboarded: false}
	}
	_ = json.NewEncoder(w).Encode(profile)
}

func (h *Handler) handleUpsertProfile(w http.ResponseWriter, r *http.Request, userID string) {
	var body types.UserProfile
	if err := decodeRequestJSON(r, &body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	if msg, ok := h.validateProfileFields(body); !ok {
		writeValidationError(w, msg)
		return
	}
	now := time.Now().UTC()
	body.UserID = userID
	body.UpdatedAt = now
	if body.CreatedAt.IsZero() {
		body.CreatedAt = now
	}
	if err := h.store.UpsertProfile(r.Context(), body); err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

// validateProfileFields checks the fields set on a profile upsert request,
// returning the validation-error message and false on the first failure.
func (h *Handler) validateProfileFields(body types.UserProfile) (string, bool) {
	if msg, ok := validateProfileMeasurements(body); !ok {
		return msg, false
	}
	if body.BirthDate != "" && !validDate(body.BirthDate, h.loc) {
		return "birth_date must be a non-future YYYY-MM-DD date", false
	}
	return validateProfileEnums(body)
}

// validateProfileMeasurements checks the numeric measurement fields of a
// profile upsert request.
func validateProfileMeasurements(body types.UserProfile) (string, bool) {
	if body.HeightCm != 0 && (!isFinite(body.HeightCm) || body.HeightCm < 50 || body.HeightCm > 300) {
		return errProfileMeasurementsOutOfRange, false
	}
	if body.TargetWeightKg != 0 && (!isFinite(body.TargetWeightKg) || body.TargetWeightKg < 20 || body.TargetWeightKg > 500) {
		return errProfileMeasurementsOutOfRange, false
	}
	if body.WeeklyRate != 0 && (!isFinite(body.WeeklyRate) || body.WeeklyRate < 0) {
		return errProfileMeasurementsOutOfRange, false
	}
	return "", true
}

// validateProfileEnums checks the enum-like string fields of a profile
// upsert request.
func validateProfileEnums(body types.UserProfile) (string, bool) {
	if body.Gender != "" && !validGender(body.Gender) {
		return "gender is invalid", false
	}
	if body.ActivityLevel != "" && !validActivityLevel(body.ActivityLevel) {
		return "activity_level is invalid", false
	}
	if body.Goal != "" && !validGoal(body.Goal) {
		return "goal is invalid", false
	}
	return "", true
}

func (h *Handler) handleCalculateTDEE(w http.ResponseWriter, r *http.Request, userID string) {
	q := r.URL.Query()
	weightKg, weightErr := strconv.ParseFloat(q.Get("weight_kg"), 64)
	heightCm, heightErr := strconv.ParseFloat(q.Get("height_cm"), 64)
	age, ageErr := strconv.Atoi(q.Get("age"))
	gender := q.Get("gender")
	activity := q.Get("activity")

	if weightErr != nil || !isFinite(weightKg) || weightKg < 20 || weightKg > 500 {
		writeValidationError(w, "weight_kg must be between 20 and 500")
		return
	}
	if heightErr != nil || !isFinite(heightCm) || heightCm < 50 || heightCm > 300 {
		writeValidationError(w, "height_cm must be between 50 and 300")
		return
	}
	if ageErr != nil || age < 1 || age > 120 {
		writeValidationError(w, "age must be between 1 and 120")
		return
	}
	if !validGender(gender) || !validActivityLevel(activity) {
		writeValidationError(w, "gender and activity are invalid")
		return
	}

	params := types.TDEEParams{
		WeightKg:      weightKg,
		HeightCm:      heightCm,
		Age:           age,
		Gender:        gender,
		ActivityLevel: activity,
	}
	result := calculateTDEE(params)
	w.Header().Set("Cache-Control", "private, max-age=300")
	_ = json.NewEncoder(w).Encode(result)
}

func (h *Handler) handleGoalSuggestions(w http.ResponseWriter, r *http.Request, userID string) {
	profile, err := h.store.GetProfile(r.Context(), userID)
	if err != nil {
		// No profile yet.
		_ = json.NewEncoder(w).Encode(types.GoalSuggestion{
			Message: "Complete your profile to get personalized goal suggestions.",
		})
		return
	}

	// Get recent rollups for average intake.
	endDate := time.Now().In(h.loc).Format(dateLayout)
	startDate := time.Now().In(h.loc).AddDate(0, 0, -7).Format(dateLayout)
	rollups, _ := h.store.GetRollups(r.Context(), userID, startDate, endDate)

	var avgKcal float64
	for _, r := range rollups {
		avgKcal += r.Consumed.Calories
	}
	if len(rollups) > 0 {
		avgKcal /= float64(len(rollups))
	}

	// Get weight trend.
	trend, _ := h.store.WeightTrend(r.Context(), userID, 14)
	var currentLossKg float64
	if len(trend) >= 2 {
		currentLossKg = trend[0].RollingAvg - trend[len(trend)-1].RollingAvg
	}

	// Compute recommended kcal using TDEE.
	now := time.Now()
	birthDate := profile.BirthDate
	if birthDate == "" {
		_ = json.NewEncoder(w).Encode(types.GoalSuggestion{
			Message: "Add your birth date in Profile settings to get personalized goal suggestions.",
		})
		return
	}
	parsed, err := time.Parse(dateLayout, birthDate)
	if err != nil {
		_ = json.NewEncoder(w).Encode(types.GoalSuggestion{
			Message: "Birth date is invalid — update it in Profile settings.",
		})
		return
	}
	age := int(now.Sub(parsed).Hours() / 8766)

	if profile.HeightCm <= 0 {
		_ = json.NewEncoder(w).Encode(types.GoalSuggestion{
			Message: "Add your height in Profile settings to get personalized goal suggestions.",
		})
		return
	}

	// Get current weight for TDEE calc.
	weights, _ := h.store.ListWeight(r.Context(), userID, 30)
	if len(weights) == 0 {
		_ = json.NewEncoder(w).Encode(types.GoalSuggestion{
			Message: "Log your weight first to get personalized goal suggestions.",
		})
		return
	}
	currentWeight := weights[len(weights)-1].WeightKg

	params := types.TDEEParams{
		WeightKg:      currentWeight,
		HeightCm:      profile.HeightCm,
		Age:           age,
		Gender:        profile.Gender,
		ActivityLevel: profile.ActivityLevel,
	}
	tdee := calculateTDEE(params)

	recommendedKcal := tdee.MaintainCal
	switch profile.Goal {
	case "lose":
		recommendedKcal = tdee.CutCal
	case "gain":
		recommendedKcal = tdee.BulkCal
	}

	targetLossKg := currentWeight - profile.TargetWeightKg

	message := "Keep going! Track your meals consistently to reach your goals."
	switch profile.Goal {
	case "lose":
		if currentLossKg > 0 {
			message = fmt.Sprintf("You're losing ~%.1f kg/week. Keep it up!", currentLossKg)
		} else {
			message = "Weight is stable. Try reducing intake slightly to start losing."
		}
	case "gain":
		message = fmt.Sprintf("Aim for %.0f kcal/day to support muscle gain.", recommendedKcal)
	}

	_ = json.NewEncoder(w).Encode(types.GoalSuggestion{
		CurrentIntakeKcal: avgKcal,
		RecommendedKcal:   recommendedKcal,
		CurrentLossKg:     currentLossKg,
		TargetLossKg:      targetLossKg,
		Message:           message,
	})
}
