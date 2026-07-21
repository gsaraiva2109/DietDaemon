package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

// ErrorCode is the stable, machine-readable API error contract.
type ErrorCode string

const (
	ErrorNotFound         ErrorCode = "not_found"
	ErrorConflict         ErrorCode = "conflict"
	ErrorValidation       ErrorCode = "validation_error"
	ErrorUnauthorized     ErrorCode = "unauthorized"
	ErrorRateLimited      ErrorCode = "rate_limited"
	ErrorInternal         ErrorCode = "internal_error"
	ErrorForbidden        ErrorCode = "forbidden"
	ErrorMethodNotAllowed ErrorCode = "method_not_allowed"
	ErrorNotImplemented   ErrorCode = "not_implemented"
	ErrorUnavailable      ErrorCode = "service_unavailable"
)

type errorEnvelope struct {
	Error struct {
		Code    ErrorCode `json:"code"`
		Message string    `json:"message"`
	} `json:"error"`
}

func writeAPIError(w http.ResponseWriter, status int, code ErrorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	var body errorEnvelope
	body.Error.Code = code
	body.Error.Message = message
	_ = json.NewEncoder(w).Encode(body)
}

func errorForStatus(status int) (ErrorCode, string) {
	switch status {
	case http.StatusBadRequest, http.StatusRequestEntityTooLarge, http.StatusUnprocessableEntity:
		return ErrorValidation, "Invalid request."
	case http.StatusUnauthorized:
		return ErrorUnauthorized, "Unauthorized."
	case http.StatusForbidden:
		return ErrorForbidden, "Forbidden."
	case http.StatusNotFound:
		return ErrorNotFound, "Not found."
	case http.StatusConflict:
		return ErrorConflict, "Conflict."
	case http.StatusMethodNotAllowed:
		return ErrorMethodNotAllowed, "Method not allowed."
	case http.StatusTooManyRequests:
		return ErrorRateLimited, "Too many requests."
	case http.StatusNotImplemented:
		return ErrorNotImplemented, "Not implemented."
	case http.StatusServiceUnavailable:
		return ErrorUnavailable, "Service unavailable."
	default:
		return ErrorInternal, "Internal server error."
	}
}

// errorEnvelopeWriter delays ordinary API responses so legacy handlers cannot
// leak an implementation error. Streaming and downloads pass through as soon
// as their protocol headers are committed.
type errorEnvelopeWriter struct {
	http.ResponseWriter
	status      int
	buf         bytes.Buffer
	passthrough bool
}

func (w *errorEnvelopeWriter) WriteHeader(status int) {
	if w.status != 0 || w.passthrough {
		return
	}
	w.status = status
	if w.streamingOrDownload() {
		w.passthrough = true
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *errorEnvelopeWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if w.passthrough {
		return w.ResponseWriter.Write(p)
	}
	return w.buf.Write(p)
}

func (w *errorEnvelopeWriter) Flush() {
	if !w.passthrough {
		w.passthrough = true
		if w.status == 0 {
			w.status = http.StatusOK
		}
		w.ResponseWriter.WriteHeader(w.status)
		_, _ = w.ResponseWriter.Write(w.buf.Bytes())
		w.buf.Reset()
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *errorEnvelopeWriter) streamingOrDownload() bool {
	contentType := w.Header().Get("Content-Type")
	return strings.HasPrefix(contentType, "text/event-stream") || w.Header().Get("Content-Disposition") != ""
}

func (w *errorEnvelopeWriter) finish() {
	if w.passthrough {
		return
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.status >= http.StatusBadRequest {
		code, message := errorForStatus(w.status)
		if w.status < http.StatusInternalServerError {
			message = publicErrorMessage(w.buf.Bytes(), message)
		}
		writeAPIError(w.ResponseWriter, w.status, code, message)
		return
	}
	w.ResponseWriter.WriteHeader(w.status)
	_, _ = w.ResponseWriter.Write(w.buf.Bytes())
}

func publicErrorMessage(body []byte, fallback string) string {
	var payload struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(body, &payload) != nil || payload.Error == "" {
		return fallback
	}
	message := strings.TrimSpace(payload.Error)
	if i := strings.IndexByte(message, ':'); i >= 0 {
		message = message[:i]
	}
	if message == "" {
		return fallback
	}
	return message
}

func withAPIErrorEnvelope(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ww := &errorEnvelopeWriter{ResponseWriter: w}
		next.ServeHTTP(ww, r)
		ww.finish()
	}
}
