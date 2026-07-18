package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/digitalghost404/inkandbone/internal/ai"
	"github.com/digitalghost404/inkandbone/internal/db"
)

func (s *Server) handleListMapPins(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid map id", http.StatusBadRequest)
		return
	}
	pins, err := s.db.ListMapPins(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if pins == nil {
		pins = []db.MapPin{}
	}
	writeJSON(w, pins)
}

func (s *Server) handleListWorldNotes(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	q := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")
	tag := r.URL.Query().Get("tag")
	var revealed *bool
	if rv := r.URL.Query().Get("revealed"); rv == "true" {
		t := true
		revealed = &t
	} else if rv == "false" {
		f := false
		revealed = &f
	}
	notes, err := s.db.SearchWorldNotes(id, q, category, tag, revealed)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if notes == nil {
		notes = []db.WorldNote{}
	}
	writeJSON(w, notes)
}

func (s *Server) handlePatchWorldNote(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		respondError(w, "invalid world note id", http.StatusBadRequest)
		return
	}
	var body struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		TagsJSON string `json:"tags_json"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Title == "" || body.Content == "" {
		respondError(w, "title and content are required", http.StatusBadRequest)
		return
	}
	note, err := s.db.GetWorldNote(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if note == nil {
		http.Error(w, "world note not found", http.StatusNotFound)
		return
	}
	if err := s.db.UpdateWorldNote(id, body.Title, body.Content, body.TagsJSON); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventWorldNoteUpdated, Payload: &WorldNoteUpdatedPayload{CampaignID: RealtimeInt64(note.CampaignID), NoteID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handlePatchWorldNotePersonality(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body struct {
		PersonalityJSON string `json:"personality_json"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	note, err := s.db.GetWorldNote(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if note == nil {
		http.Error(w, "world note not found", http.StatusNotFound)
		return
	}
	if err := s.db.UpdateWorldNotePersonality(id, body.PersonalityJSON); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventWorldNoteUpdated, Payload: &WorldNoteUpdatedPayload{CampaignID: RealtimeInt64(note.CampaignID), NoteID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handlePatchWorldNoteRevealed(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid world note id", http.StatusBadRequest)
		return
	}
	var body struct {
		IsRevealed bool `json:"is_revealed"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	note, err := s.db.GetWorldNote(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if note == nil {
		http.Error(w, "world note not found", http.StatusNotFound)
		return
	}
	if err := s.db.PatchWorldNoteRevealed(id, body.IsRevealed); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventWorldNoteRevealed, Payload: &WorldNoteRevealedPayload{CampaignID: RealtimeInt64(note.CampaignID), ID: RealtimeInt64(id), IsRevealed: RealtimePtr(body.IsRevealed)}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListMaps(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	maps, err := s.db.ListMaps(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if maps == nil {
		maps = []db.Map{}
	}
	writeJSON(w, maps)
}

func (s *Server) handleGetMap(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid map id", http.StatusBadRequest)
		return
	}
	m, err := s.db.GetMap(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if m == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, m)
}

func (s *Server) handleDraftWorldNote(w http.ResponseWriter, r *http.Request) {
	if s.aiClient == nil {
		http.Error(w, "AI not configured — set ANTHROPIC_API_KEY", http.StatusServiceUnavailable)
		return
	}
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		Hint string `json:"hint"`
	}
	if err := decodeJSON(w, r, &body, shortJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Hint == "" {
		http.Error(w, "hint is required", http.StatusBadRequest)
		return
	}

	prompt := fmt.Sprintf(
		"Create a TTRPG world note for: %s\n\nRespond with exactly two lines:\nTitle: <short name>\nContent: <2-3 sentence description>",
		body.Hint,
	)
	generated, err := s.aiClient.Generate(r.Context(), prompt, 256)
	if err != nil {
		serverError(w, r, err)
		return
	}

	title, content := parseGeneratedNote(generated)
	if title == "" {
		title = body.Hint
	}
	if content == "" {
		content = generated
	}

	noteID, err := s.db.CreateWorldNote(id, title, content, "npc")
	if err != nil {
		serverError(w, r, err)
		return
	}

	created, err := s.db.GetWorldNote(noteID)
	if err != nil {
		serverError(w, r, err)
		return
	}

	s.bus.Publish(Event{Type: EventWorldNoteCreated, Payload: &WorldNoteCreatedPayload{CampaignID: RealtimeInt64(id), NoteID: RealtimeInt64(noteID), Title: RealtimePtr(title)}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created) //nolint:errcheck
}

func (s *Server) handleGenerateMap(w http.ResponseWriter, r *http.Request) {
	if s.aiClient == nil {
		http.Error(w, "AI not configured — set ANTHROPIC_API_KEY", http.StatusServiceUnavailable)
		return
	}
	completer, ok := s.aiClient.(ai.Completer)
	if !ok {
		http.Error(w, "AI client does not support generation", http.StatusServiceUnavailable)
		return
	}

	id, ok2 := parsePathID(r, "id")
	if !ok2 {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}

	var body struct {
		Name    string `json:"name"`
		Context string `json:"context"`
	}
	if err := decodeJSON(w, r, &body, shortJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Name == "" {
		body.Name = "Generated Map"
	}

	// Map generation requires precise SVG output — use the structured AI client,
	// not the narrative GM model.
	prompt := mapSystemPrompt + "\n\nGenerate a map for this TTRPG setting:\n\n" + body.Context
	svgRaw, err := completer.Generate(r.Context(), prompt, 4096)
	if err != nil {
		serverError(w, r, err)
		return
	}

	// Split at </svg> boundary: SVG content before, zones block after.
	svgEnd := strings.LastIndex(svgRaw, "</svg>")
	var svgPart, zonesPart string
	if svgEnd >= 0 {
		svgPart = svgRaw[:svgEnd+6]
		zonesPart = svgRaw[svgEnd+6:]
	} else {
		svgPart = svgRaw
	}
	svgContent, err := sanitizeGeneratedSVGResponse(svgPart)
	if err != nil {
		serverError(w, r, err)
		return
	}

	mapID, err := s.persistGeneratedSVG(id, body.Name, svgContent)
	if err != nil {
		serverError(w, r, err)
		return
	}

	// Parse optional [ZONES] block and create zones.
	if zonesPart != "" {
		if start := strings.Index(zonesPart, "[ZONES]"); start != -1 {
			inner := zonesPart[start+7:]
			if end := strings.Index(inner, "[/ZONES]"); end != -1 {
				inner = strings.TrimSpace(inner[:end])
				var zoneDefs []struct {
					Name   string  `json:"name"`
					X      float64 `json:"x"`
					Y      float64 `json:"y"`
					Width  float64 `json:"width"`
					Height float64 `json:"height"`
				}
				if jsonErr := json.Unmarshal([]byte(inner), &zoneDefs); jsonErr == nil {
					for _, z := range zoneDefs {
						if z.Name == "" || z.X < 0 || z.X > 1 || z.Y < 0 || z.Y > 1 ||
							z.Width <= 0 || z.Width > 1 || z.Height <= 0 || z.Height > 1 {
							continue
						}
						if _, err := s.db.CreateMapZone(mapID, z.Name, z.X, z.Y, z.Width, z.Height); err != nil {
							log.Printf("handleGenerateMap: create zone %q: %v", z.Name, err)
						}
					}
				}
			}
		}
	}

	m, err := s.db.GetMap(mapID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if m == nil {
		serverErrorText(w, r, "fetch created map")
		return
	}

	s.bus.Publish(Event{Type: EventMapCreated, Payload: &MapCreatedPayload{CampaignID: RealtimeInt64(id), MapID: RealtimeInt64(mapID)}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(m) //nolint:errcheck
}

func (s *Server) handleCreateMapPin(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid map id", http.StatusBadRequest)
		return
	}
	var body struct {
		X     float64 `json:"x"`
		Y     float64 `json:"y"`
		Label string  `json:"label"`
		Note  string  `json:"note"`
		Color string  `json:"color"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}

	pinID, err := s.db.AddMapPin(id, body.X, body.Y, body.Label, body.Note, body.Color)
	if err != nil {
		serverError(w, r, err)
		return
	}

	pins, err := s.db.ListMapPins(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	var created *db.MapPin
	for i := range pins {
		if pins[i].ID == pinID {
			created = &pins[i]
			break
		}
	}
	if created == nil {
		serverErrorText(w, r, "pin not found after create")
		return
	}

	s.bus.Publish(Event{Type: EventMapPinAdded, Payload: &MapPinAddedPayload{MapID: RealtimeInt64(id), PinID: RealtimeInt64(pinID)}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created) //nolint:errcheck
}

func (s *Server) handleListNPCs(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	npcs, err := s.db.ListSessionNPCs(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if npcs == nil {
		npcs = []db.SessionNPC{}
	}
	writeJSON(w, npcs)
}

func (s *Server) handleCreateNPC(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name string `json:"name"`
		Note string `json:"note"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	npc, err := s.db.CreateSessionNPC(id, body.Name, body.Note)
	if err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventNPCUpdated, Payload: &NPCUpdatedPayload{SessionID: RealtimeInt64(id), NPCID: RealtimeInt64(npc.ID)}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(npc) //nolint:errcheck
}

func (s *Server) handlePatchNPC(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid npc id", http.StatusBadRequest)
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	npc, err := s.db.GetSessionNPC(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if npc == nil {
		http.Error(w, "npc not found", http.StatusNotFound)
		return
	}
	if err := s.db.UpdateSessionNPC(id, body.Note); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventNPCUpdated, Payload: &NPCUpdatedPayload{SessionID: RealtimeInt64(npc.SessionID), NPCID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteNPC(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid npc id", http.StatusBadRequest)
		return
	}
	npc, err := s.db.GetSessionNPC(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if npc == nil {
		http.Error(w, "npc not found", http.StatusNotFound)
		return
	}
	if err := s.db.DeleteSessionNPC(id); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventNPCUpdated, Payload: &NPCUpdatedPayload{SessionID: RealtimeInt64(npc.SessionID), NPCID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListObjectives(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	objectives, err := s.db.ListObjectives(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if objectives == nil {
		objectives = []db.Objective{}
	}
	writeJSON(w, objectives)
}

func (s *Server) handleCreateObjective(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		ParentID    *int64 `json:"parent_id"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	if body.ParentID != nil {
		parent, err := s.db.GetObjective(*body.ParentID)
		if err != nil {
			serverError(w, r, err)
			return
		}
		if parent == nil {
			http.Error(w, "parent objective not found", http.StatusBadRequest)
			return
		}
		if parent.ParentID != nil {
			http.Error(w, "parent_id must reference a top-level objective", http.StatusBadRequest)
			return
		}
	}
	obj, err := s.db.CreateObjective(id, body.Title, body.Description, body.ParentID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventObjectiveUpdated, Payload: &ObjectiveUpdatedPayload{CampaignID: RealtimeInt64(id), ObjectiveID: RealtimeInt64(obj.ID)}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(obj) //nolint:errcheck
}

func (s *Server) handlePatchObjective(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid objective id", http.StatusBadRequest)
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}
	validStatuses := map[string]bool{"active": true, "completed": true, "failed": true}
	if !validStatuses[body.Status] {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}
	objective, err := s.db.GetObjective(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if objective == nil {
		http.Error(w, "objective not found", http.StatusNotFound)
		return
	}
	if err := s.db.UpdateObjectiveStatus(id, body.Status); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventObjectiveUpdated, Payload: &ObjectiveUpdatedPayload{CampaignID: RealtimeInt64(objective.CampaignID), ObjectiveID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteObjective(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid objective id", http.StatusBadRequest)
		return
	}
	objective, err := s.db.GetObjective(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if objective == nil {
		http.Error(w, "objective not found", http.StatusNotFound)
		return
	}
	if err := s.db.DeleteObjective(id); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventObjectiveUpdated, Payload: &ObjectiveUpdatedPayload{CampaignID: RealtimeInt64(objective.CampaignID), ObjectiveID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}

// handleDeduplicateObjectives removes duplicate objectives within a campaign,
// keeping the oldest copy of each title (case-insensitive).
// POST /api/campaigns/{id}/objectives/dedup
func (s *Server) handleDeduplicateObjectives(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid campaign id", http.StatusBadRequest)
		return
	}
	n, err := s.db.DeduplicateObjectives(campaignID)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if n > 0 {
		s.bus.Publish(Event{Type: EventObjectiveUpdated, Payload: &ObjectiveUpdatedPayload{CampaignID: RealtimeInt64(campaignID)}})
	}
	writeJSON(w, map[string]any{"deleted": n})
}

func (s *Server) handleListItems(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid character id", http.StatusBadRequest)
		return
	}
	items, err := s.db.ListItems(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if items == nil {
		items = []db.Item{}
	}
	writeJSON(w, items)
}

func (s *Server) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid character id", http.StatusBadRequest)
		return
	}
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Quantity    *int   `json:"quantity"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	qty := 1
	if body.Quantity != nil {
		qty = *body.Quantity
	}
	item, err := s.db.CreateItem(id, body.Name, body.Description, qty)
	if err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventItemUpdated, Payload: &ItemUpdatedPayload{CharacterID: RealtimeInt64(id), ItemID: RealtimeInt64(item.ID)}})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item) //nolint:errcheck
}

func (s *Server) handlePatchItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}
	existing, err := s.db.GetItem(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if existing == nil {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}
	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Quantity    *int    `json:"quantity"`
		Equipped    *bool   `json:"equipped"`
	}
	if err := decodeJSON(w, r, &body, ordinaryJSONLimit); err != nil {
		respondDecodeError(w, err)
		return
	}
	// Apply only provided fields.
	name := existing.Name
	description := existing.Description
	quantity := existing.Quantity
	equipped := existing.Equipped
	if body.Name != nil {
		name = *body.Name
	}
	if body.Description != nil {
		description = *body.Description
	}
	if body.Quantity != nil {
		quantity = *body.Quantity
	}
	if body.Equipped != nil {
		equipped = *body.Equipped
	}
	if err := s.db.UpdateItem(id, name, description, quantity, equipped); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventItemUpdated, Payload: &ItemUpdatedPayload{CharacterID: RealtimeInt64(existing.CharacterID), ItemID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.Error(w, "invalid item id", http.StatusBadRequest)
		return
	}
	existing, err := s.db.GetItem(id)
	if err != nil {
		serverError(w, r, err)
		return
	}
	if existing == nil {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}
	if err := s.db.DeleteItem(id); err != nil {
		serverError(w, r, err)
		return
	}
	s.bus.Publish(Event{Type: EventItemUpdated, Payload: &ItemUpdatedPayload{CharacterID: RealtimeInt64(existing.CharacterID), ItemID: RealtimeInt64(id)}})
	w.WriteHeader(http.StatusNoContent)
}
