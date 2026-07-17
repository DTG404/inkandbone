package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) handleGetRuleset(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid ruleset id", http.StatusBadRequest)
		return
	}
	rs, err := s.db.GetRuleset(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rs == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, rs)
}

func (s *Server) handlePatchCharacter(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		respondError(w, "invalid character id", http.StatusBadRequest)
		return
	}
	var body struct {
		DataJSON        *string `json:"data_json"`
		CurrencyBalance *int64  `json:"currency_balance"`
		CurrencyLabel   *string `json:"currency_label"`
	}
	if err := decodeJSON(r, &body); err != nil {
		respondError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.DataJSON != nil {
		if err := s.db.UpdateCharacterData(id, *body.DataJSON); err != nil {
			respondError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if body.CurrencyBalance != nil {
		balance := *body.CurrencyBalance
		if balance < 0 {
			balance = 0
		}
		if err := s.db.UpdateCharacterCurrencyBalance(id, balance); err != nil {
			respondError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if body.CurrencyLabel != nil {
		if err := s.db.UpdateCharacterCurrencyLabel(id, *body.CurrencyLabel); err != nil {
			respondError(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	s.bus.Publish(Event{Type: EventCharacterUpdated, Payload: map[string]any{"id": id, "character_id": id}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUploadPortrait(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid character id", http.StatusBadRequest)
		return
	}
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("portrait")
	if err != nil {
		http.Error(w, "portrait is required: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	filename := fmt.Sprintf("%d_%s", id, filepath.Base(header.Filename))
	ext := strings.ToLower(filepath.Ext(filename))
	if _, ok := portraitAssetTypes[ext]; !ok {
		http.Error(w, "unsupported image format", http.StatusBadRequest)
		return
	}
	destDir := filepath.Join(s.dataDir, "portraits")
	if err := os.MkdirAll(destDir, 0750); err != nil {
		http.Error(w, "mkdir: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := writeValidatedUpload(destDir, filename, file, portraitAssetTypes); err != nil {
		if err == errInvalidAsset {
			http.Error(w, "image content does not match its format", http.StatusBadRequest)
			return
		}
		http.Error(w, "write file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	relativePath := "portraits/" + filename
	if err := s.db.UpdateCharacterPortrait(id, relativePath); err != nil {
		removeStoredAsset(destDir, filename)
		http.Error(w, "db: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.bus.Publish(Event{Type: EventCharacterUpdated, Payload: map[string]any{
		"id":            id,
		"character_id":  id,
		"portrait_path": relativePath,
	}})
	writeJSON(w, map[string]string{"portrait_path": relativePath})
}
