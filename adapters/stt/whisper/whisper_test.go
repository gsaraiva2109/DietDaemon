package whisper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTranscribe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/inference" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(inferenceResponse{
			Text:     "duzentos gramas de frango",
			Language: "pt",
		})
	}))
	defer srv.Close()

	p := New(srv.URL)
	text, lang, err := p.Transcribe(t.Context(), []byte("fake-audio-data"))
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if text != "duzentos gramas de frango" {
		t.Errorf("text = %q", text)
	}
	if lang != "pt" {
		t.Errorf("lang = %q, want %q", lang, "pt")
	}
}

func TestTranscribeEmptyAudio(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(inferenceResponse{Text: ""})
	}))
	defer srv.Close()

	p := New(srv.URL)
	text, _, err := p.Transcribe(t.Context(), []byte{})
	if err != nil {
		t.Fatalf("Transcribe: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
}

func TestTranscribeHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := New(srv.URL)
	if _, _, err := p.Transcribe(t.Context(), []byte("audio")); err == nil {
		t.Error("expected error on 500, got nil")
	}
}
