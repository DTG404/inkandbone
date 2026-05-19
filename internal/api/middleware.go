package api

import (
	"encoding/json"
	"net/http"
)

// withMaxBody wraps a handler with a MaxBytesReader limit on the request body.
func withMaxBody(n int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, n)
		next(w, r)
	}
}

// decodeJSON decodes the JSON request body into v.
func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
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
