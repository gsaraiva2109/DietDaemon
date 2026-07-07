package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/importers/hevy"
)

// hevyClient is the subset of hevy.Client the import handler needs. Defining it
// here lets tests inject a fake without touching the real HTTP client.
type hevyClient interface {
	ListWorkouts(ctx context.Context, page, pageSize int) ([]hevy.HevyWorkout, int, error)
}

// ---------------------------------------------------------------------------
// Hevy key management (mirrors BYOK AI-key endpoints in handler_settings.go)
// ---------------------------------------------------------------------------

// hevyKeyStatus is the response body for GET /api/v1/settings/hevy-key.
type hevyKeyStatus struct {
	HasKey bool `json:"has_key"`
}

// handleGetHevyKey returns whether the user has a stored Hevy API key.
func (h *Handler) handleGetHevyKey(w http.ResponseWriter, r *http.Request, userID string) {
	_, found, err := h.store.GetUserHevyKey(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(hevyKeyStatus{HasKey: found})
}

// handleSetHevyKey encrypts and stores a Hevy API key.
func (h *Handler) handleSetHevyKey(w http.ResponseWriter, r *http.Request, userID string) {
	if h.cfg == nil || len(h.cfg.AIKeyEncKey) == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "AI_KEY_ENC_KEY is not configured"})
		return
	}

	var body struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if body.Key == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "key is required"})
		return
	}

	ct, err := auth.Encrypt([]byte(body.Key), h.cfg.AIKeyEncKey)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	encKey := base64.RawStdEncoding.EncodeToString(ct)

	if err := h.store.SetUserHevyKey(r.Context(), userID, encKey); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleDeleteHevyKey removes the user's stored Hevy API key.
func (h *Handler) handleDeleteHevyKey(w http.ResponseWriter, r *http.Request, userID string) {
	if err := h.store.DeleteUserHevyKey(r.Context(), userID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Hevy import endpoint
// ---------------------------------------------------------------------------

// importResult is the summary returned after a Hevy import completes.
type importResult struct {
	Imported          int `json:"imported"`
	SkippedDuplicates int `json:"skipped_duplicates"`
	Total             int `json:"total"`
}

// handleImportHevy runs a one-time Hevy workout import for the authenticated user.
// POST /api/v1/import/hevy
func (h *Handler) handleImportHevy(w http.ResponseWriter, r *http.Request, userID string) {
	if h.cfg == nil || len(h.cfg.AIKeyEncKey) == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "AI_KEY_ENC_KEY is not configured"})
		return
	}

	encKey, found, err := h.store.GetUserHevyKey(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if !found {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "no Hevy API key configured — set a key first via POST /api/v1/settings/hevy-key"})
		return
	}

	ct, err := base64.RawStdEncoding.DecodeString(encKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to decode stored key"})
		return
	}
	plaintext, err := auth.Decrypt(ct, h.cfg.AIKeyEncKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to decrypt stored key"})
		return
	}

	client := hevy.NewClient(string(plaintext))
	result, err := runHevyImport(r.Context(), client, h.store, userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}

// runHevyImport pages through all Hevy workouts, transforms each one, and
// imports them via store.ImportWorkout. Accepts a hevyClient interface so
// handler tests can inject a fake.
func runHevyImport(ctx context.Context, client hevyClient, store MealStore, userID string) (importResult, error) {
	var result importResult
	page := 1
	for {
		workouts, pageCount, err := client.ListWorkouts(ctx, page, 10)
		if err != nil {
			return result, err
		}
		result.Total += len(workouts)

		for _, hw := range workouts {
			w, err := hevy.ToWorkout(userID, hw)
			if err != nil {
				return result, err
			}
			w.ID = newHandlerID()
			if err := store.ImportWorkout(ctx, w); err != nil {
				return result, err
			}
			result.Imported++
		}

		if page >= pageCount {
			break
		}
		page++
	}

	result.SkippedDuplicates = result.Total - result.Imported
	return result, nil
}
