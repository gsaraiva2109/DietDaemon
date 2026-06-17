// Package whisper implements ports.STTProvider by calling a whisper.cpp HTTP
// server. It sends the raw audio bytes as a multipart file upload to the
// /inference endpoint and returns the transcript with an optional language hint.
package whisper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// Compile-time interface check.
var _ ports.STTProvider = (*Provider)(nil)

// Provider satisfies ports.STTProvider by calling a whisper.cpp server.
type Provider struct {
	url    string // base URL, e.g. "http://localhost:8080"
	client *http.Client
}

// New returns a ready Provider. url is the whisper.cpp server base (no trailing
// slash).
func New(url string) *Provider {
	return &Provider{
		url:    strings.TrimRight(url, "/"),
		client: &http.Client{},
	}
}

// inferenceResponse is the JSON returned by whisper.cpp /inference.
type inferenceResponse struct {
	Text     string `json:"text"`
	Language string `json:"language"` // BCP-47, may be empty
}

// Transcribe sends audio bytes to the whisper server and returns the
// transcript text and an optional locale hint.
func (p *Provider) Transcribe(ctx context.Context, audio []byte) (string, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// Create a form file part for the audio. whisper.cpp servers typically
	// accept "file" as the field name.
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="audio.wav"`)
	h.Set("Content-Type", "audio/wav")
	fw, err := w.CreatePart(h)
	if err != nil {
		return "", "", fmt.Errorf("whisper: create form part: %w", err)
	}
	if _, err := fw.Write(audio); err != nil {
		return "", "", fmt.Errorf("whisper: write audio: %w", err)
	}
	w.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.url+"/inference", &buf)
	if err != nil {
		return "", "", fmt.Errorf("whisper: build request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("whisper: inference: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("whisper: status %d: %s", resp.StatusCode, string(body))
	}

	var ir inferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&ir); err != nil {
		return "", "", fmt.Errorf("whisper: decode: %w", err)
	}

	return strings.TrimSpace(ir.Text), ir.Language, nil
}
