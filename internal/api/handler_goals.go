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
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
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

func (h *Handler) handleCalculateTDEE(w http.ResponseWriter, r *http.Request, userID string) {
	q := r.URL.Query()
	weightKg, _ := strconv.ParseFloat(q.Get("weight_kg"), 64)
	heightCm, _ := strconv.ParseFloat(q.Get("height_cm"), 64)
	age, _ := strconv.Atoi(q.Get("age"))
	gender := q.Get("gender")
	activity := q.Get("activity")

	if weightKg <= 0 || heightCm <= 0 || age <= 0 || gender == "" || activity == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "weight_kg, height_cm, age, gender, and activity query params are required",
		})
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
	endDate := time.Now().In(h.loc).Format("2006-01-02")
	startDate := time.Now().In(h.loc).AddDate(0, 0, -7).Format("2006-01-02")
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
	parsed, err := time.Parse("2006-01-02", birthDate)
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
