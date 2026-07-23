package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// BackupConfig has no json struct tags, so JSON keys are the Go field names
// verbatim (UserID, Destination, IntervalHrs, ...).

func TestBackupRoutesRequireAuth(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/settings/backup", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestGetBackupConfigDefaults(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/settings/backup", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	cfg := decodeJSON[types.BackupConfig](t, rec)
	if cfg.UserID != "test-user" || cfg.Destination != "local" || cfg.IntervalHrs != defaultBackupIntervalHrs {
		t.Errorf("unexpected default config: %+v", cfg)
	}
	if cfg.Enabled {
		t.Errorf("expected disabled by default")
	}
}

func TestGetBackupConfigExisting(t *testing.T) {
	store := newFakeMealStore()
	store.backupConfig = types.BackupConfig{UserID: "test-user", Enabled: true, Destination: "s3", S3Bucket: "my-bucket", IntervalHrs: 12}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/settings/backup", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	cfg := decodeJSON[types.BackupConfig](t, rec)
	if !cfg.Enabled || cfg.Destination != "s3" || cfg.S3Bucket != "my-bucket" || cfg.IntervalHrs != 12 {
		t.Errorf("unexpected stored config: %+v", cfg)
	}
}

func TestSetBackupConfig(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	body := map[string]any{
		"UserID":  "someone-else", // must be overridden by the authenticated user
		"Enabled": true, "Destination": "s3", "S3Bucket": "backups", "IntervalHrs": 6,
	}
	rec := doRequest(h, "PUT", "/api/v1/settings/backup", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.backupConfig.UserID != "test-user" {
		t.Errorf("expected authenticated user to override body UserID, got %q", store.backupConfig.UserID)
	}
	if store.backupConfig.Destination != "s3" || store.backupConfig.IntervalHrs != 6 || !store.backupConfig.Enabled {
		t.Errorf("unexpected persisted config: %+v", store.backupConfig)
	}
}

func TestSetBackupConfigDefaultsAppliedWhenEmpty(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "PUT", "/api/v1/settings/backup", map[string]any{}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if store.backupConfig.Destination != "local" || store.backupConfig.IntervalHrs != defaultBackupIntervalHrs {
		t.Errorf("expected defaults applied, got %+v", store.backupConfig)
	}
}

func TestSetBackupConfigInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	req := httptest.NewRequest("PUT", "/api/v1/settings/backup", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

// TestRunBackupNowDisabled covers the handleRunBackupNow branch reached when
// no backup.Runner is configured on the server (the default in tests).
func TestRunBackupNowDisabled(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/settings/backup/run", nil, nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
}
