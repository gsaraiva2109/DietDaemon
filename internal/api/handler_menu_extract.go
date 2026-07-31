package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// ---------------------------------------------------------------------------
// Photo menu dining mode (issue #201) — extracts dish candidates from a
// photographed restaurant menu for the user to pick from (editing if they
// want) before logging one as a meal. The uploaded image is never persisted
// or passed to h.store: it is read into memory, handed to the vision
// adapter, and discarded once this handler returns. Mirrors the diet-plan
// photo import path (#194, handler_plan_extract.go) file-for-file.
// ---------------------------------------------------------------------------

func (h *Handler) handleExtractMenuFromImage(w http.ResponseWriter, r *http.Request, userID string) {
	if h.visionAdapter == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "menu photo extraction is not configured on this server"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	// #nosec G120 — MaxBytesReader above bounds the body before ParseMultipartForm.
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file too large (max 5 MB)"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "file field required"})
		return
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(io.LimitReader(file, 5<<20))
	if err != nil {
		h.writeErr(w, err)
		return
	}

	mimeType := http.DetectContentType(data)
	if len(mimeType) < 6 || mimeType[:6] != "image/" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "uploaded file is not an image"})
		return
	}

	draft, err := h.visionAdapter.ExtractMenu(r.Context(), data, mimeType)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(draft)
}

// restaurantEstimateConfidence is forced onto every meal logged from a
// photographed menu dish, regardless of how confidently the parser resolved
// the food match: a photographed restaurant dish is inherently a rough
// estimate (no nutrition source prices whole plated dishes), so it must
// always be flagged low confidence (acceptance criterion 3).
const restaurantEstimateConfidence = 0.4

func (h *Handler) handleLogMenuDish(w http.ResponseWriter, r *http.Request, userID string) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": invalidJSONBodyPrefix + err.Error()})
		return
	}
	if body.Name == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "name field is required"})
		return
	}

	ctx := r.Context()
	locale := "en"
	if h.store != nil {
		u, err := h.store.GetUser(ctx, userID)
		if err == nil && u.Locale != "" {
			locale = u.Locale
		}
	}

	dishText := body.Name
	if body.Description != "" {
		dishText = body.Name + ". " + body.Description
	}

	items, _, err := h.logger.ParseAndResolve(ctx, userID, dishText, locale)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if len(items) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "couldn't recognize any food in that dish"})
		return
	}

	meal, err := h.logger.LogMealFromItems(ctx, userID, time.Now().UTC(), dishText, restaurantEstimateConfidence, items)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(meal)
}
