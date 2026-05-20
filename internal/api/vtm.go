package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	mathrand "math/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/digitalghost404/inkandbone/internal/ai"
)

// vtmCrisisRE matches VtM-specific crisis keywords at word boundaries.
var vtmCrisisRE = regexp.MustCompile(
	`\b(frenzy|the beast|torpor|blood hunt|diablerie|masquerade breach|daybreak|sunrise)\b`,
)

// vtmMajorBreachRE matches major Masquerade breach keywords.
var vtmMajorBreachRE = regexp.MustCompile(
	`\b(caught on camera|viral|police|recorded|photographed|livestream|news crew)\b`,
)

// vtmModerateBreachRE matches moderate breach keywords.
var vtmModerateBreachRE = regexp.MustCompile(
	`\b(witnessed feeding|seen feeding|watched you feed|fangs exposed|transformation witnessed|seen your true form)\b`,
)

// vtmMinorBreachRE matches minor breach keywords.
var vtmMinorBreachRE = regexp.MustCompile(
	`\b(overheard|suspicious|noticed something|acting strange|too fast|too strong|inhuman)\b`,
)

// stainTriggerRE matches acts that cost Stains in VtM V5.
// Normal willing feeding does NOT cost Stains — only explicitly harmful or coerced acts do.
var stainTriggerRE = regexp.MustCompile(
	`\b(forced feeding|forcing|fed from|draining|drained dry|killed|slaughter|murder|diablerie|diablerized|breaking.*conviction|violated.*conviction|prey exclusion|broke the masquerade|unwilling)\b`,
)

// vtmNewNightRE matches phrases that signal a new night beginning in VtM.
var vtmNewNightRE = regexp.MustCompile(
	`\b(as dusk|at dusk|dusk falls|dusk arrives|dusk settles|dusk approaches|nightfall|as night falls|when night falls|the night falls|as the sun sets|the sun sets|sunset arrives|the evening begins|another night|the following night|next night|the next night|a new night|night has fallen|night has come|night has reclaimed|night reclaims|darkness falls|darkness descends|as darkness descends|fall of night|with the fall of night|the city awakens at night|as the darkness|as the night begins|that evening you|the next evening|night descends|night comes|the night arrives|night has settled|as night descends|night (begins|settles|covers|envelops|awakens))\b`,
)

// vtmStoryDayRE captures an in-story day name from the GM's opening scene.
var vtmStoryDayRE = regexp.MustCompile(
	`(?i)\b(monday|tuesday|wednesday|thursday|friday|saturday|sunday)\b`,
)

// vtmStoryDayIndex maps lowercase day name → JS Date.getDay() value (0=Sunday).
var vtmStoryDayIndex = map[string]int{
	"sunday": 0, "monday": 1, "tuesday": 2, "wednesday": 3,
	"thursday": 4, "friday": 5, "saturday": 6,
}

// vtmEmbraceRE matches language that may indicate a mortal character has been Embraced.
var vtmEmbraceRE = regexp.MustCompile(
	`(?i)\b(embrace[sd]?|the embrace|your embrace|sire[sd]|siring|turned into a vampire|made you a vampire|grants? you the gift|the gift of undeath|kindred now|one of us now|you are kindred|blood fills your veins|dark gift|welcomed into the embrace)\b`,
)

// rouseCheckRE matches the player's /rouse or "rouse check" command.
var rouseCheckRE = regexp.MustCompile(`(?i)(?:\b(rouse\s+check)\b|(?:^|\s)(/rouse)\b)`)

// bloodSurgeRE matches the /surge command.
var bloodSurgeRE = regexp.MustCompile(`(?i)(?:(?:^|\s)(/surge)\b|\b(blood\s+surge)\b)`)

// vtmEmbracePredatorTypes is the full list of VtM predator types for random assignment.
var vtmEmbracePredatorTypes = []string{
	"Alleycat", "Bagger", "Blood Leech", "Cleaner", "Consensualist",
	"Extortionist", "Graverobber", "Osiris", "Sandman", "Siren",
	"Farmer", "Montero", "Scene Queen", "Treasure Hunter", "Pursuer", "Witch Hunter",
}

// vtmEmbraceValidClans is the set of clans a sire can belong to (excludes Thin-Blooded, which is a character type).
var vtmEmbraceValidClans = map[string]bool{
	"Brujah": true, "Gangrel": true, "Malkavian": true, "Nosferatu": true,
	"Toreador": true, "Tremere": true, "Ventrue": true, "Caitiff": true,
	"Banu Haqim": true, "Hecata": true, "Lasombra": true, "Ministry": true,
	"Ravnos": true, "Salubri": true, "Tzimisce": true,
}

// vtmHungerDiceRoll executes a VtM V5 dice pool roll using the Hunger dice mechanic.
//
// In VtM V5 all dice are d10s. The pool is split: Hunger dice (red) replace normal dice
// up to the character's current Hunger level. Both types count 6+ as a success.
//   - Critical success = two or more 10s in the combined pool.
//   - Messy Critical  = critical success where at least one 10 is a Hunger die.
//   - Bestial Failure = 0 successes AND at least one Hunger die shows a 1.
//
// On a Messy Critical the clan Compulsion oracle is rolled and injected into the GM context.
func (s *Server) vtmHungerDiceRoll(ctx context.Context, sessionID int64, pool int, attribute string, dc int, reason, origExpr, charStatsJSON string) *rollCheckResult {
	// Parse current Hunger from character stats.
	hunger := 0
	if charStatsJSON != "" && charStatsJSON != "none" {
		var cs map[string]any
		if err := json.Unmarshal([]byte(charStatsJSON), &cs); err == nil {
			switch v := cs["hunger"].(type) {
			case float64:
				hunger = int(v)
			case int:
				hunger = v
			}
		}
	}
	if hunger < 0 {
		hunger = 0
	}
	if hunger > 5 {
		hunger = 5
	}

	hungerCount := hunger
	if hungerCount > pool {
		hungerCount = pool
	}
	normalCount := pool - hungerCount

	// Roll all dice.
	normal := make([]int, normalCount)
	hungerDice := make([]int, hungerCount)
	for i := range normal {
		normal[i] = mathrand.Intn(10) + 1
	}
	for i := range hungerDice {
		hungerDice[i] = mathrand.Intn(10) + 1
	}

	// Count successes (6+) and tens.
	successes := 0
	totalTens := 0
	hungerTens := 0
	hasHungerOne := false
	for _, r := range normal {
		if r >= 6 {
			successes++
		}
		if r == 10 {
			totalTens++
		}
	}
	for _, r := range hungerDice {
		if r >= 6 {
			successes++
		}
		if r == 10 {
			totalTens++
			hungerTens++
		}
		if r == 1 {
			hasHungerOne = true
		}
	}

	// Critical = 2+ tens; each pair of tens adds 1 extra success.
	critPairs := totalTens / 2
	successes += critPairs

	threshold := dc
	if threshold <= 0 {
		threshold = 1 // any success counts
	}
	success := successes >= threshold
	messyCritical := success && totalTens >= 2 && hungerTens >= 1
	bestialFail := !success && hasHungerOne

	// Build expression string for logging.
	expr := fmt.Sprintf("%dd10 (%dN+%dH)", pool, normalCount, hungerCount)

	// Log and broadcast.
	allRolls := append(normal, hungerDice...)
	breakdownBytes, _ := json.Marshal(allRolls)
	_, _ = s.db.LogDiceRoll(sessionID, expr, successes, string(breakdownBytes))
	s.bus.Publish(Event{Type: EventDiceRolled, Payload: map[string]any{
		"session_id":     sessionID,
		"expression":     expr,
		"result":         successes,
		"normal_dice":    normal,
		"hunger_dice":    hungerDice,
		"successes":      successes,
		"messy_critical": messyCritical,
		"bestial_fail":   bestialFail,
	}})

	// Messy Critical → roll clan Compulsion.
	var compulsion string
	if messyCritical {
		compulsion = s.vtmRollClanCompulsion(ctx, sessionID, charStatsJSON)
	}

	return &rollCheckResult{
		Expression:    expr,
		Total:         successes,
		Attribute:     attribute,
		DC:            threshold,
		Success:       success,
		Reason:        reason,
		MessyCritical: messyCritical,
		BestialFail:   bestialFail,
		Compulsion:    compulsion,
	}
}

// vtmRollClanCompulsion rolls on the clan-specific Compulsion oracle table for the
// active character's clan. Returns the compulsion description, or a generic Hunger
// Compulsion fallback if the clan table is not found.
func (s *Server) vtmRollClanCompulsion(ctx context.Context, sessionID int64, charStatsJSON string) string {
	clan := ""
	if charStatsJSON != "" && charStatsJSON != "none" {
		var cs map[string]any
		if json.Unmarshal([]byte(charStatsJSON), &cs) == nil {
			if c, ok := cs["clan"].(string); ok {
				clan = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(c), " ", "_"))
			}
		}
	}

	roll := mathrand.Intn(10) + 1

	// Look up ruleset ID for the session.
	var rulesetID *int64
	if sess, err := s.db.GetSession(sessionID); err == nil && sess != nil {
		if camp, err := s.db.GetCampaign(sess.CampaignID); err == nil && camp != nil {
			if rs, err := s.db.GetRuleset(camp.RulesetID); err == nil && rs != nil {
				rulesetID = &rs.ID
			}
		}
	}

	tableName := "compulsion_" + clan
	result, err := s.db.RollOracle(rulesetID, tableName, roll)
	if err != nil || result == "" {
		// Generic Hunger Compulsion fallback.
		result = "The Beast surges. The character must immediately seek to slake their Hunger through feeding — all other actions feel meaningless until Hunger drops below 3."
	}

	s.bus.Publish(Event{Type: "oracle_result", Payload: map[string]any{
		"session_id":   sessionID,
		"table":        tableName,
		"roll":         roll,
		"result":       result,
		"is_compulsion": true,
	}})

	return result
}

// handleVtMRouseCheck performs a Rouse Check for a VtM character.
// Rolls 1d10; 6+ = no Hunger change; 1-5 = Hunger +1.
// At Hunger 5, does not increase further but flags a Frenzy risk.
// Returns a string describing the result for injection into GM context.
func (s *Server) handleVtMRouseCheck(ctx context.Context, sessionID int64) string {
	charIDStr, err := s.db.GetSetting("active_character_id")
	if err != nil || charIDStr == "" {
		return ""
	}
	charID, err := strconv.ParseInt(charIDStr, 10, 64)
	if err != nil {
		return ""
	}
	char, err := s.db.GetCharacter(charID)
	if err != nil || char == nil || char.DataJSON == "" {
		return ""
	}
	var stats map[string]any
	if err := json.Unmarshal([]byte(char.DataJSON), &stats); err != nil {
		return ""
	}

	currentHunger := 0
	if v, ok := stats["hunger"]; ok {
		switch n := v.(type) {
		case int:
			currentHunger = n
		case float64:
			currentHunger = int(n)
		}
	}

	roll := mathrand.Intn(10) + 1
	_, _ = s.db.LogDiceRoll(sessionID, "1d10 (Rouse Check)", roll, "[]")
	s.bus.Publish(Event{Type: EventDiceRolled, Payload: map[string]any{
		"session_id": sessionID,
		"expression": "1d10 (Rouse Check)",
		"result":     roll,
	}})

	if roll >= 6 {
		return fmt.Sprintf("[ROUSE CHECK] Result: %d — Success. Hunger unchanged at %d.", roll, currentHunger)
	}

	// Hunger increases
	if currentHunger >= 5 {
		return fmt.Sprintf("[ROUSE CHECK] Result: %d — Failed. Hunger already at 5. FRENZY RISK: The character must resist a Hunger Frenzy (Composure + Resolve, difficulty 3).", roll)
	}

	newHunger := currentHunger + 1
	stats["hunger"] = newHunger
	dataJSON, err := json.Marshal(stats)
	if err != nil {
		return fmt.Sprintf("[ROUSE CHECK] Result: %d — Failed. Hunger should increase to %d but stat update failed.", roll, newHunger)
	}
	if err := s.db.UpdateCharacterData(charID, string(dataJSON)); err != nil {
		return fmt.Sprintf("[ROUSE CHECK] Result: %d — Failed. Hunger should increase to %d but stat update failed.", roll, newHunger)
	}
	s.bus.Publish(Event{Type: EventCharacterUpdated, Payload: map[string]any{"id": charID, "character_id": charID, "session_id": sessionID}})

	msg := fmt.Sprintf("[ROUSE CHECK] Result: %d — Failed. Hunger increases to %d.", roll, newHunger)
	if newHunger >= 4 {
		msg += " The Beast strains against the cage. Frenzy risk is elevated."
	}
	return msg
}

// autoVtMDisciplineRouseChecks uses the AI to detect how many Rouse Checks the
// character owes based on discipline use in the player action and GM narration,
// then fires that many Rouse Checks automatically. Only runs for VtM campaigns.
// Blood Surge (/surge) is excluded — it fires its own Rouse Check inline.
func (s *Server) autoVtMDisciplineRouseChecks(ctx context.Context, sessionID int64, playerAction, gmText string) {
	if !s.isAutomationEnabled(settingAutoCheckRoll) {
		return
	}
	completer, ok := s.aiClient.(ai.Completer)
	if !ok {
		return
	}
	if !s.canRunAutomation() {
		return
	}

	// Confirm VtM campaign.
	sess, err := s.db.GetSession(sessionID)
	if err != nil || sess == nil {
		return
	}
	camp, err := s.db.GetCampaign(sess.CampaignID)
	if err != nil || camp == nil {
		return
	}
	rs, err := s.db.GetRuleset(camp.RulesetID)
	if err != nil || rs == nil || rs.Name != "vtm" {
		return
	}

	// Mortals and ghouls do not use Rouse Checks for Disciplines.
	charIDStr, _ := s.db.GetSetting("active_character_id")
	if charIDStr != "" {
		if charID, err := strconv.ParseInt(charIDStr, 10, 64); err == nil {
			if char, err := s.db.GetCharacter(charID); err == nil && char != nil && char.DataJSON != "" {
				var charStats map[string]any
				if json.Unmarshal([]byte(char.DataJSON), &charStats) == nil {
					if ct, ok := charStats["character_type"].(string); ok && strings.ToLower(ct) == "mortal" {
						return
					}
				}
			}
		}
	}

	// Regex pre-filter: skip AI call if no discipline keyword appears in player action or GM text.
	// This avoids false positives from general vampire narrative that never mentions disciplines.
	disciplineKeywordRE := regexp.MustCompile(`(?i)\b(animalism|auspex|blood sorcery|celerity|dominate|fortitude|obfuscate|oblivion|potence|presence|protean|discipline|invoke|activate|channel|summon|command|compel|blur|vanish|invisible|shroud|meld|frenzy control|beast)\b`)
	if !disciplineKeywordRE.MatchString(playerAction) && !disciplineKeywordRE.MatchString(gmText) {
		return
	}

	prompt := fmt.Sprintf(`You are a Vampire: The Masquerade V5 rules engine.

Read the player action and GM narration below. Count how many Rouse Checks the character owes for EXPLICITLY ACTIVATED Discipline powers.

V5 ROUSE CHECK RULES — STRICT CRITERIA:
- Only count a Rouse Check when a Discipline power is EXPLICITLY activated. The player must declare "I use [Discipline]" OR the GM must narrate that the character activates/invokes/channels a named Discipline power.
- Disciplines: Animalism, Auspex, Blood Sorcery, Celerity, Dominate, Fortitude, Obfuscate, Oblivion, Potence, Presence, Protean.
- DO NOT count: vampires moving quickly (only count if Celerity is named), vampires taking a hit (only count if Fortitude is named), vampires being strong (only count if Potence is named), vampires sensing things (only count if Auspex is named).
- DO NOT count passive vampire traits (night vision, blood scent, hearing heartbeats, emotional intuition, natural predatory instinct).
- DO NOT count ambiguous descriptions where a Discipline is not explicitly named or invoked.
- Blood Surge is NOT counted here — it fires its own Rouse Check separately.
- Waking from day sleep is NOT counted here — handled separately.
- When in doubt, count 0. False negatives are far preferable to false positives.

Player action: %s
GM narration: %s

Reply with ONLY a single integer: the number of Rouse Checks owed (0 if none). No explanation.`, playerAction, gmText)

	var raw string
	err = retryWithBackoff(ctx, 2, func(ctx context.Context) error {
		var e error
		raw, e = completer.Generate(ctx, prompt, 10)
		return e
	})
	if err != nil {
		s.recordAutoFailure()
		return
	}
	s.recordAutoSuccess()
	raw = strings.TrimSpace(raw)
	count := 0
	if _, err := fmt.Sscanf(raw, "%d", &count); err != nil || count <= 0 {
		return
	}
	if count > 5 {
		count = 5 // sanity cap — no scene warrants more than 5 Rouse Checks
	}

	for range count {
		_ = s.handleVtMRouseCheck(ctx, sessionID)
	}
}

// bloodPotencyBonusDice returns the bonus dice granted by Blood Surge for a given Blood Potency.
func bloodPotencyBonusDice(bp int) int {
	switch {
	case bp >= 10:
		return 4
	case bp >= 7:
		return 3
	case bp >= 4:
		return 2
	default:
		return 1
	}
}

// handleVtMBloodSurge performs a Rouse Check and returns bonus dice count.
// Returns a string for injection into GM context.
func (s *Server) handleVtMBloodSurge(ctx context.Context, sessionID int64) string {
	rouseResult := s.handleVtMRouseCheck(ctx, sessionID)

	charIDStr, _ := s.db.GetSetting("active_character_id")
	charID, _ := strconv.ParseInt(charIDStr, 10, 64)
	char, err := s.db.GetCharacter(charID)
	if err != nil || char == nil {
		return rouseResult
	}
	var stats map[string]any
	_ = json.Unmarshal([]byte(char.DataJSON), &stats)
	bp := 1
	if v, ok := stats["blood_potency"]; ok {
		switch n := v.(type) {
		case int:
			bp = n
		case float64:
			bp = int(n)
		}
	}
	bonus := bloodPotencyBonusDice(bp)
	return rouseResult + fmt.Sprintf(" [BLOOD SURGE] Add %d bonus dice to the next roll this turn (Blood Potency %d).", bonus, bp)
}

// autoDetectVtMNightDOW sets the in-story day-of-week for Night 1 on a VtM campaign
// the first time a GM scene names a weekday. After that it becomes a no-op.
func (s *Server) autoDetectVtMNightDOW(ctx context.Context, sessionID int64, gmText string) {
	if !vtmStoryDayRE.MatchString(gmText) {
		return
	}
	sess, err := s.db.GetSession(sessionID)
	if err != nil || sess == nil {
		return
	}
	camp, err := s.db.GetCampaign(sess.CampaignID)
	if err != nil || camp == nil {
		return
	}

	ruleset, err := s.db.GetRuleset(camp.RulesetID)
	if err != nil || ruleset == nil || ruleset.Name != "vtm" {
		return
	}

	// Already detected — nothing to do.
	if camp.ChronicleNightStartDOW >= 0 {
		return
	}

	match := vtmStoryDayRE.FindString(gmText)
	if match == "" {
		return
	}
	dow, ok := vtmStoryDayIndex[strings.ToLower(match)]
	if !ok {
		return
	}

	// Back-calculate: current night is chronicle_night; Night 1 was (chronicle_night-1) days before today's story day.
	night1DOW := ((dow - (camp.ChronicleNight - 1)) % 7 + 7) % 7
	if err := s.db.SetCampaignChronicleNightStartDOW(camp.ID, night1DOW); err != nil {
		log.Printf("autoDetectVtMNightDOW: failed to store DOW: %v", err)
		return
	}
	s.bus.Publish(Event{
		Type: "campaign_updated",
		Payload: map[string]any{
			"campaign_id":               camp.ID,
			"chronicle_night_start_dow": night1DOW,
		},
	})
	log.Printf("autoDetectVtMNightDOW: campaign %d Night 1 anchored to DOW %d (detected %q on night %d)", camp.ID, night1DOW, match, camp.ChronicleNight)
}

// autoDetectVtMEmbrace runs after every GM response for VtM mortal characters.
// It uses a regex pre-filter and then an AI confirmation step to detect if the
// mortal has been Embraced. On confirmation it transforms the character into a
// full Vampire: clan from sire, random predator type, and reset vampire stats.
func (s *Server) autoDetectVtMEmbrace(ctx context.Context, sessionID int64, gmText string) {
	if s.aiClient == nil {
		return
	}
	// Regex pre-filter — skip AI call if no embrace language present.
	if !vtmEmbraceRE.MatchString(gmText) {
		return
	}

	completer, ok := s.aiClient.(ai.Completer)
	if !ok {
		return
	}
	if !s.canRunAutomation() {
		return
	}

	// Resolve session → campaign → ruleset.
	sess, err := s.db.GetSession(sessionID)
	if err != nil || sess == nil {
		return
	}
	camp, err := s.db.GetCampaign(sess.CampaignID)
	if err != nil || camp == nil {
		return
	}
	ruleset, err := s.db.GetRuleset(camp.RulesetID)
	if err != nil || ruleset == nil || ruleset.Name != "vtm" {
		return
	}

	// Resolve active character and confirm it is a Mortal.
	charIDStr, err := s.db.GetSetting("active_character_id")
	if err != nil || charIDStr == "" {
		return
	}
	charID, err := strconv.ParseInt(charIDStr, 10, 64)
	if err != nil {
		return
	}
	char, err := s.db.GetCharacter(charID)
	if err != nil || char == nil || char.DataJSON == "" {
		return
	}
	var stats map[string]any
	if err := json.Unmarshal([]byte(char.DataJSON), &stats); err != nil {
		return
	}
	charType, _ := stats["character_type"].(string)
	if strings.ToLower(charType) != "mortal" {
		return
	}

	// Ask the AI to confirm the Embrace and identify the sire's clan.
	prompt := fmt.Sprintf(`You are analyzing a Vampire: The Masquerade V5 story segment.

GM NARRATION:
%s

TASK: Determine whether the PLAYER CHARACTER (a mortal) was definitively Embraced (turned into a Kindred vampire) in this narration.

Respond with a JSON object and nothing else:
- "embraced": true if the player character was Embraced, false otherwise
- "clan": the clan name of the sire who performed the Embrace (one of: Brujah, Gangrel, Malkavian, Nosferatu, Toreador, Tremere, Ventrue, Caitiff, Banu Haqim, Hecata, Lasombra, Ministry, Ravnos, Salubri, Tzimisce). Use "Caitiff" if the sire's clan is unknown or ambiguous.

Only set "embraced" to true if the narration clearly describes the player character undergoing the Embrace — do not trigger on NPC embraces or hypothetical references.

Example: {"embraced": true, "clan": "Nosferatu"}`, gmText)

	var raw string
	err = retryWithBackoff(ctx, 2, func(ctx context.Context) error {
		var e error
		raw, e = completer.Generate(ctx, prompt, 80)
		return e
	})
	if err != nil {
		log.Printf("autoDetectVtMEmbrace: AI call failed: %v", err)
		s.recordAutoFailure()
		return
	}
	s.recordAutoSuccess()

	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return
	}

	var result struct {
		Embraced bool   `json:"embraced"`
		Clan     string `json:"clan"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &result); err != nil || !result.Embraced {
		return
	}

	// Validate clan; fall back to Caitiff if unrecognized.
	sireClan := strings.TrimSpace(result.Clan)
	if !vtmEmbraceValidClans[sireClan] {
		sireClan = "Caitiff"
	}

	// Pick a random predator type.
	predatorType := vtmEmbracePredatorTypes[mathrand.Intn(len(vtmEmbracePredatorTypes))]

	// Apply the Embrace: transform mortal → vampire.
	log.Printf("autoDetectVtMEmbrace: EMBRACING char=%d old_type=%q new_clan=%q sireClan=%q", charID, charType, sireClan, sireClan)
	stats["character_type"] = "Vampire"
	stats["clan"] = sireClan
	stats["predator_type"] = predatorType
	stats["hunger"] = float64(1)
	stats["blood_potency"] = float64(1)
	stats["bane_severity"] = float64(1)
	stats["humanity"] = float64(7)
	stats["stains"] = float64(0)
	// Set sect to Unaligned if not already set.
	if sect, _ := stats["sect"].(string); sect == "" {
		stats["sect"] = "Unaligned"
	}
	// Initialise discipline fields to 0 if absent.
	for _, disc := range []string{
		"animalism", "auspex", "blood_sorcery", "celerity", "dominate",
		"fortitude", "obfuscate", "oblivion", "potence", "presence", "protean",
	} {
		if _, ok := stats[disc]; !ok {
			stats[disc] = float64(0)
		}
	}

	updated, err := json.Marshal(stats)
	if err != nil {
		log.Printf("autoDetectVtMEmbrace: marshal failed: %v", err)
		return
	}
	if err := s.db.UpdateCharacterData(charID, string(updated)); err != nil {
		log.Printf("autoDetectVtMEmbrace: DB update failed: %v", err)
		return
	}

	s.bus.Publish(Event{
		Type: EventCharacterUpdated,
		Payload: map[string]any{
			"character_id":  charID,
			"session_id":    sessionID,
			"embrace":       true,
			"clan":          sireClan,
			"predator_type": predatorType,
		},
	})
	log.Printf("autoDetectVtMEmbrace: character %d embraced into clan %s (predator type: %s)", charID, sireClan, predatorType)
}

// autoUpdateMasquerade checks GM text for Masquerade breach keywords and decrements
// masquerade_integrity for VtM sessions. No-op for non-VtM sessions.
func (s *Server) autoUpdateMasquerade(ctx context.Context, sessionID int64, gmText string) {
	if !s.isAutomationEnabled(settingAutoUpdateMasq) {
		return
	}
	sess, err := s.db.GetSession(sessionID)
	if err != nil || sess == nil {
		return
	}
	camp, err := s.db.GetCampaign(sess.CampaignID)
	if err != nil || camp == nil {
		return
	}
	rs, err := s.db.GetRuleset(camp.RulesetID)
	if err != nil || rs == nil || rs.Name != "vtm" {
		return
	}

	lower := strings.ToLower(gmText)
	delta := 0
	if vtmMajorBreachRE.MatchString(lower) {
		delta = -3
	} else if vtmModerateBreachRE.MatchString(lower) {
		delta = -2
	} else if vtmMinorBreachRE.MatchString(lower) {
		delta = -1
	}
	if delta == 0 {
		return
	}

	current, err := s.db.GetMasqueradeIntegrity(sessionID)
	if err != nil {
		return
	}
	newLevel := current + delta
	if newLevel < 0 {
		newLevel = 0
	}
	_ = s.db.UpdateMasqueradeIntegrity(sessionID, newLevel)
	s.bus.Publish(Event{Type: EventSessionUpdated, Payload: map[string]any{
		"session_id":           sessionID,
		"masquerade_integrity": newLevel,
	}})
}

// detectAndApplyVtMStains scans text for Humanity-violating acts and adds Stains.
// After adding a Stain, checks whether a Remorse roll is required (stains >= 11 - humanity)
// and auto-applies it: roll Humanity dice (d10s, 6+ = success), pass → stains reset,
// fail → humanity -1 and stains reset.
func (s *Server) detectAndApplyVtMStains(ctx context.Context, sessionID int64, text string) {
	if !stainTriggerRE.MatchString(strings.ToLower(text)) {
		return
	}
	charIDStr, err := s.db.GetSetting("active_character_id")
	if err != nil || charIDStr == "" {
		return
	}
	charID, err := strconv.ParseInt(charIDStr, 10, 64)
	if err != nil {
		return
	}
	char, err := s.db.GetCharacter(charID)
	if err != nil || char == nil || char.DataJSON == "" {
		return
	}
	var stats map[string]any
	if err := json.Unmarshal([]byte(char.DataJSON), &stats); err != nil {
		return
	}

	getIntStat := func(key string) int {
		switch n := stats[key].(type) {
		case int:
			return n
		case float64:
			return int(n)
		}
		return 0
	}

	stains := getIntStat("stains")
	if stains >= 10 {
		return
	}
	stains++
	stats["stains"] = float64(stains)

	humanity := getIntStat("humanity")
	if humanity <= 0 {
		humanity = 7 // sensible default if not set
	}

	// Remorse threshold: stains >= (11 - humanity)
	remorseThreshold := 11 - humanity
	if remorseThreshold < 1 {
		remorseThreshold = 1
	}

	if stains >= remorseThreshold {
		// Roll Remorse Check: dice pool = humanity (min 1), looking for 6+ on each d10
		pool := humanity
		if pool < 1 {
			pool = 1
		}
		successes := 0
		expr := fmt.Sprintf("%dd10 (Remorse Check)", pool)
		rolls := make([]int, pool)
		for i := range rolls {
			r := mathrand.Intn(10) + 1
			rolls[i] = r
			if r >= 6 {
				successes++
			}
		}
		rollsJSON, _ := json.Marshal(rolls)
		totalRoll := 0
		for _, r := range rolls {
			if r > totalRoll {
				totalRoll = r // highest die for logging
			}
		}
		_, _ = s.db.LogDiceRoll(sessionID, expr, totalRoll, string(rollsJSON))
		s.bus.Publish(Event{Type: EventDiceRolled, Payload: map[string]any{
			"session_id": sessionID,
			"expression": expr,
			"result":     totalRoll,
			"rolls":      rolls,
			"successes":  successes,
		}})

		// Apply result
		stats["stains"] = float64(0)
		if successes == 0 {
			// Failed Remorse: lose 1 Humanity
			newHumanity := humanity - 1
			if newHumanity < 0 {
				newHumanity = 0
			}
			stats["humanity"] = float64(newHumanity)
		}
		// On success stains just reset to 0 (already set above)
	}

	updated, err := json.Marshal(stats)
	if err != nil {
		return
	}
	if err := s.db.UpdateCharacterData(charID, string(updated)); err != nil {
		return
	}
	s.bus.Publish(Event{Type: EventCharacterUpdated, Payload: map[string]any{
		"id":           charID,
		"character_id": charID,
		"session_id":   sessionID,
	}})
}

// autoUpdateChronicleNight detects when a new night begins in a VtM session
// and increments the campaign's chronicle_night counter. Zero AI cost — keyword only.
// Also fires a Rouse Check to rise (Hunger may increase) and restores Willpower
// by min(Composure, Resolve), both of which happen mechanically every night in V5.
func (s *Server) autoUpdateChronicleNight(ctx context.Context, sessionID int64, gmText string) {
	if !s.isAutomationEnabled(settingAutoUpdateNight) {
		return
	}
	if !vtmNewNightRE.MatchString(strings.ToLower(gmText)) {
		return
	}
	sess, err := s.db.GetSession(sessionID)
	if err != nil || sess == nil {
		return
	}
	camp, err := s.db.GetCampaign(sess.CampaignID)
	if err != nil || camp == nil {
		return
	}
	rs, err := s.db.GetRuleset(camp.RulesetID)
	if err != nil || rs == nil || rs.Name != "vtm" {
		return
	}
	newNight := camp.ChronicleNight + 1
	if err := s.db.UpdateCampaignChronicleNight(camp.ID, newNight); err != nil {
		return
	}
	s.bus.Publish(Event{Type: "campaign_updated", Payload: map[string]any{
		"campaign_id":     camp.ID,
		"chronicle_night": newNight,
	}})

	// Rouse Check to rise: every vampire makes a Rouse Check when waking for the night.
	// handleVtMRouseCheck handles the d10 roll, updates Hunger if failed, and broadcasts.
	_ = s.handleVtMRouseCheck(ctx, sessionID)

	// Willpower restoration: sleeping the day restores min(Composure, Resolve) willpower.
	// This is deterministic — no AI call needed.
	s.vtmRestoreWillpowerOnWake(sessionID)
}

// vtmRestoreWillpowerOnWake restores Willpower by min(Composure, Resolve) when the
// vampire wakes for a new night, capped at willpower_max. Pure calculation, no AI.
func (s *Server) vtmRestoreWillpowerOnWake(sessionID int64) {
	charIDStr, err := s.db.GetSetting("active_character_id")
	if err != nil || charIDStr == "" {
		return
	}
	charID, err := strconv.ParseInt(charIDStr, 10, 64)
	if err != nil {
		return
	}
	char, err := s.db.GetCharacter(charID)
	if err != nil || char == nil || char.DataJSON == "" {
		return
	}
	var stats map[string]any
	if err := json.Unmarshal([]byte(char.DataJSON), &stats); err != nil {
		return
	}

	getInt := func(key string) int {
		switch v := stats[key].(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			n, _ := strconv.Atoi(v)
			return n
		}
		return 0
	}

	composure := getInt("composure")
	resolve := getInt("resolve")
	wMax := getInt("willpower_max")
	wCurrent := getInt("willpower_superficial")

	restore := composure
	if resolve < restore {
		restore = resolve
	}
	if restore <= 0 {
		return
	}

	newW := wCurrent + restore
	if newW > wMax {
		newW = wMax
	}
	if newW == wCurrent {
		return
	}

	stats["willpower_superficial"] = float64(newW)
	updated, err := json.Marshal(stats)
	if err != nil {
		return
	}
	if err := s.db.UpdateCharacterData(charID, string(updated)); err != nil {
		return
	}
	s.bus.Publish(Event{Type: EventCharacterUpdated, Payload: map[string]any{
		"id":           charID,
		"character_id": charID,
		"session_id":   sessionID,
		"data_json":    string(updated),
	}})
}
