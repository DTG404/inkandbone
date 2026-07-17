package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeJSONTransportErrors(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
		maxBytes    int64
		wantStatus  int
	}{
		{
			name:        "oversized body",
			contentType: "application/json",
			body:        `{"name":"too large"}`,
			maxBytes:    8,
			wantStatus:  http.StatusRequestEntityTooLarge,
		},
		{
			name:        "oversized trailing whitespace",
			contentType: "application/json",
			body:        `{"name":"ok"}` + strings.Repeat(" ", 100),
			maxBytes:    16,
			wantStatus:  http.StatusRequestEntityTooLarge,
		},
		{
			name:        "wrong content type",
			contentType: "text/plain",
			body:        `{"name":"valid json"}`,
			maxBytes:    ordinaryJSONLimit,
			wantStatus:  http.StatusUnsupportedMediaType,
		},
		{
			name:        "unknown field",
			contentType: "application/json; charset=utf-8",
			body:        `{"name":"valid","extra":true}`,
			maxBytes:    ordinaryJSONLimit,
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "trailing json",
			contentType: "application/json",
			body:        `{"name":"first"} {"name":"second"}`,
			maxBytes:    ordinaryJSONLimit,
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			var body struct {
				Name string `json:"name"`
			}

			err := decodeJSON(w, req, &body, tt.maxBytes)
			require.Error(t, err)
			respondDecodeError(w, err)

			assert.Equal(t, tt.wantStatus, w.Code)
			var response map[string]string
			require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
			assert.Contains(t, response, "error")
		})
	}
}

func TestMiddlewareMultipartLimitIncludesOverhead(t *testing.T) {
	var encoded bytes.Buffer
	writer := multipart.NewWriter(&encoded)
	part, err := writer.CreateFormFile("image", "test.png")
	require.NoError(t, err)
	_, err = part.Write(bytes.Repeat([]byte("x"), 32))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(encoded.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	err = parseMultipartForm(w, req, int64(encoded.Len()-1))
	require.Error(t, err)
	respondBodyError(w, err)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestMiddlewareOpaqueErrorAndRequestID(t *testing.T) {
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /api/fail", func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, requestID(r))
		serverError(w, r, errors.New("database password=do-not-expose"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/fail", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	requestIDHeader := w.Header().Get("X-Request-ID")
	require.NotEmpty(t, requestIDHeader)
	var response map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
	assert.Equal(t, map[string]string{
		"error":      "internal server error",
		"request_id": requestIDHeader,
	}, response)
	assert.NotContains(t, w.Body.String(), "password")
}

func TestMiddlewareSecurityHeaders(t *testing.T) {
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /api/ok", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/ok", nil))

	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "camera=(), microphone=(), geolocation=()", w.Header().Get("Permissions-Policy"))
	assert.Equal(t,
		"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' data: https://fonts.gstatic.com; img-src 'self' data: blob:; connect-src 'self' https://fonts.googleapis.com https://fonts.gstatic.com; frame-src https://www.youtube.com; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'",
		w.Header().Get("Content-Security-Policy"),
	)
}
