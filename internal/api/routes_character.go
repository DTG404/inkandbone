package api

import (
	"fmt"
	"log"
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
		serverError(w, r, err)
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
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.DataJSON != nil {
		if err := s.db.UpdateCharacterData(id, *body.DataJSON); err != nil {
			serverError(w, r, err)
			return
		}
	}
	if body.CurrencyBalance != nil {
		balance := *body.CurrencyBalance
		if balance < 0 {
			balance = 0
		}
		if err := s.db.UpdateCharacterCurrencyBalance(id, balance); err != nil {
			serverError(w, r, err)
			return
		}
	}
	if body.CurrencyLabel != nil {
		if err := s.db.UpdateCharacterCurrencyLabel(id, *body.CurrencyLabel); err != nil {
			serverError(w, r, err)
			return
		}
	}
	s.bus.Publish(Event{Type: EventCharacterUpdated, Payload: &CharacterUpdatedPayload{ID: RealtimeInt64(id), CharacterID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUploadPortrait(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid character id", http.StatusBadRequest)
		return
	}
	if err := parseMultipartForm(w, r, 5<<20); err != nil {
		respondBodyError(w, err)
		return
	}
	file, header, err := r.FormFile("portrait")
	if err != nil {
		http.Error(w, "portrait is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(filepath.Base(header.Filename)))
	if _, ok := portraitAssetTypes[ext]; !ok {
		http.Error(w, "unsupported image format", http.StatusBadRequest)
		return
	}
	filename := fmt.Sprintf("%d_%s%s", id, randomHex(16), ext)
	destDir := filepath.Join(s.dataDir, "portraits")
	if err := os.MkdirAll(destDir, 0750); err != nil {
		serverError(w, r, err)
		return
	}
	if err := writeValidatedUpload(destDir, filename, file, portraitAssetTypes); err != nil {
		if err == errInvalidAsset {
			http.Error(w, "image content does not match its format", http.StatusBadRequest)
			return
		}
		serverError(w, r, err)
		return
	}

	relativePath := "portraits/" + filename
	if err := s.db.UpdateCharacterPortrait(id, relativePath); err != nil {
		if cleanupErr := removeStoredAsset(destDir, filename); cleanupErr != nil {
			log.Printf("portrait upload cleanup failed: %v", cleanupErr)
		}
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventCharacterUpdated, Payload: &CharacterUpdatedPayload{ID: RealtimeInt64(id), CharacterID: RealtimeInt64(id), PortraitPath: RealtimePtr(relativePath)}})
	writeJSON(w, map[string]string{"portrait_path": relativePath})
}
