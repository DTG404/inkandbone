package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
)

const (
	ordinaryJSONLimit int64 = 64 << 10
	shortJSONLimit    int64 = 4 << 10
	imageUploadLimit  int64 = 10 << 20
	rulebookLimit     int64 = 50 << 20
)

var (
	errUnsupportedMediaType = errors.New("content type must be application/json")
	errTrailingJSON         = errors.New("request body must contain a single JSON value")
)

type requestIDContextKey struct{}

// withMaxBody wraps a handler with a MaxBytesReader limit on the request body.
func withMaxBody(n int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, n)
		next(w, r)
	}
}

// decodeJSON strictly decodes one size-limited JSON value from the request.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any, maxBytes int64) error {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return errUnsupportedMediaType
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return err
		}
		return errTrailingJSON
	}
	return nil
}

func respondDecodeError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	message := "invalid request body"
	var maxBytesError *http.MaxBytesError
	switch {
	case errors.Is(err, errUnsupportedMediaType):
		status = http.StatusUnsupportedMediaType
		message = errUnsupportedMediaType.Error()
	case errors.As(err, &maxBytesError):
		status = http.StatusRequestEntityTooLarge
		message = "request body too large"
	}
	respondError(w, message, status)
}

func parseMultipartForm(w http.ResponseWriter, r *http.Request, maxBytes int64) error {
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "multipart/form-data" {
		return errUnsupportedMediaType
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	return r.ParseMultipartForm(maxBytes)
}

func respondBodyError(w http.ResponseWriter, err error) {
	respondDecodeError(w, err)
}

func newRequestID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func requestID(r *http.Request) string {
	id, _ := r.Context().Value(requestIDContextKey{}).(string)
	return id
}

func withRequestID(r *http.Request, id string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, id))
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; font-src 'self' data:; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
}

func serverError(w http.ResponseWriter, r *http.Request, err error) {
	id := requestID(r)
	if id == "" {
		id = newRequestID()
		w.Header().Set("X-Request-ID", id)
	}
	log.Printf("request_id=%s internal_error=%q", id, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":      "internal server error",
		"request_id": id,
	})
}

func serverErrorText(w http.ResponseWriter, r *http.Request, message string) {
	serverError(w, r, errors.New(message))
}

// respondError writes a JSON error response with the given status code.
func respondError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

// respondJSON writes a JSON response.
func respondJSON(w http.ResponseWriter, v any) {
	writeJSON(w, v)
}
