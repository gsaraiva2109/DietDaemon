package api

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func writeValidationError(w http.ResponseWriter, message string) {
	writeAPIError(w, http.StatusBadRequest, ErrorValidation, message)
}

func decodeRequestJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

// decodeOptionalRequestJSON accepts an empty body for endpoints with documented defaults,
// while still rejecting malformed JSON.
func decodeOptionalRequestJSON(r *http.Request, dst any) error {
	err := decodeRequestJSON(r, dst)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func validDate(date string, loc *time.Location) bool {
	parsed, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil || parsed.Format("2006-01-02") != date {
		return false
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return !parsed.After(today)
}

func validGender(value string) bool {
	return value == "male" || value == "female" || value == "other"
}

func validActivityLevel(value string) bool {
	switch value {
	case "sedentary", "light", "moderate", "active", "very_active":
		return true
	default:
		return false
	}
}

func validGoal(value string) bool {
	return value == "cut" || value == "maintain" || value == "bulk"
}

func validSleepQuality(value string) bool {
	switch value {
	case "ok", "poor", "fair", "good", "great":
		return true
	default:
		return false
	}
}

func validWorkoutIntensity(value string) bool {
	switch value {
	case "light", "moderate", "heavy":
		return true
	default:
		return false
	}
}

func validNutritionSource(value string) bool {
	switch value {
	case "openfoodfacts", "taco", "usda":
		return true
	default:
		return false
	}
}

func validMacros(m types.Macros) bool {
	return isFinite(m.Calories) && m.Calories >= 0 &&
		isFinite(m.Protein) && m.Protein >= 0 &&
		isFinite(m.Carbs) && m.Carbs >= 0 &&
		isFinite(m.Fat) && m.Fat >= 0 &&
		isFinite(m.Fiber) && m.Fiber >= 0
}

func boundedQueryInt(w http.ResponseWriter, r *http.Request, field string, fallback, min, max int) (int, bool) {
	value := r.URL.Query().Get(field)
	if value == "" {
		return fallback, true
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < min || parsed > max {
		writeValidationError(w, field+" is out of range")
		return 0, false
	}
	return parsed, true
}
