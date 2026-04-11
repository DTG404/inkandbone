package ruleset

import (
	"encoding/json"
	"strings"
)

// XPKey returns the character stats JSON field that holds the XP currency for the given system.
func XPKey(system string) string {
	switch system {
	case "shadowrun":
		return "karma"
	default:
		return "xp"
	}
}

// XPLabel returns the display label for the XP currency of the given system.
func XPLabel(system string) string {
	switch system {
	case "shadowrun":
		return "Karma"
	case "cyberpunk_red":
		return "IP"
	case "theonering":
		return "AP"
	default:
		return "XP"
	}
}

// vtmInClanDisciplines maps VtM clan → set of in-clan discipline field keys (lowercase, matching character sheet).
// Keys are stored lowercase for case-insensitive lookup.
// Caitiff have no in-clan disciplines — all cost out-of-clan rate (7× per dot).
var vtmInClanDisciplines = map[string][]string{
	"brujah":    {"celerity", "potence", "presence"},
	"gangrel":   {"animalism", "fortitude", "protean"},
	"malkavian": {"auspex", "dominate", "obfuscate"},
	"nosferatu": {"animalism", "obfuscate", "potence"},
	"toreador":  {"auspex", "celerity", "presence"},
	"tremere":   {"auspex", "blood_sorcery", "dominate"},
	"ventrue":   {"dominate", "fortitude", "presence"},
	"lasombra":  {"dominate", "oblivion", "potence"},
	"tzimisce":  {"animalism", "dominate", "protean"},
	"assamite":  {"blood_sorcery", "celerity", "obfuscate"},
	"giovanni":  {"dominate", "fortitude", "oblivion"},
	"ravnos":    {"animalism", "obfuscate", "presence"},
	"setite":    {"obfuscate", "presence", "protean"},
	// V5 supplement clan names.
	"banu_haqim": {"blood_sorcery", "celerity", "obfuscate"},
	"hecata":     {"auspex", "fortitude", "oblivion"},
	"ministry":   {"obfuscate", "presence", "protean"},
	"salubri":    {"auspex", "fortitude", "oblivion"},
	// Caitiff: no in-clan disciplines; explicit entry prevents nil lookup confusion.
	"caitiff": {},
}

// XPCostFor returns the XP cost for advancing field to newVal for the given system.
// statsJSON is required only for VtM discipline in/out-of-clan determination; pass "" otherwise.
// Returns 0 for systems where XP is threshold-based rather than spent (dnd5e level-up).
func XPCostFor(system, field string, newVal int, statsJSON string) int {
	switch system {
	case "wrath_glory":
		if strings.HasPrefix(field, "talent:") {
			name := strings.TrimPrefix(field, "talent:")
			return WGTalentCost(name)
		}
		return newVal * 4

	case "dnd5e":
		return 0 // XP not subtracted on level-up; threshold-based

	case "vtm":
		if field == "blood_potency" {
			return newVal * 10
		}
		// Disciplines: check in-clan for character's clan (field = raw key e.g. "animalism")
		vtmDisciplines := map[string]bool{
			"animalism": true, "auspex": true, "blood_sorcery": true, "celerity": true,
			"dominate": true, "fortitude": true, "obfuscate": true, "oblivion": true,
			"potence": true, "presence": true, "protean": true,
		}
		if vtmDisciplines[field] {
			multiplier := 7 // default out-of-clan
			if statsJSON != "" {
				var stats map[string]any
				if err := json.Unmarshal([]byte(statsJSON), &stats); err == nil {
					if clan, ok := stats["clan"].(string); ok {
						// Case-insensitive lookup: character sheet may store any capitalization.
						normalizedClan := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(clan), " ", "_"))
						for _, d := range vtmInClanDisciplines[normalizedClan] {
							if d == field {
								multiplier = 5
								break
							}
						}
					}
				}
			}
			return newVal * multiplier
		}
		// Attributes (full names matching character sheet)
		vtmAttribs := map[string]bool{
			"strength": true, "dexterity": true, "stamina": true,
			"charisma": true, "manipulation": true, "composure": true,
			"intelligence": true, "wits": true, "resolve": true,
		}
		if vtmAttribs[field] {
			return newVal * 4
		}
		// Skills: new_dots * 3
		return newVal * 3

	case "shadowrun":
		// Knowledge skills
		shadowrunKnowledgeSkills := map[string]bool{
			"street_knowledge":       true,
			"academic_knowledge":     true,
			"interest_knowledge":     true,
			"professional_knowledge": true,
		}
		if shadowrunKnowledgeSkills[field] {
			return newVal * 2
		}
		if field == "specialization" {
			return 5
		}
		// quality: flat 10
		if strings.HasPrefix(field, "quality:") {
			return 10
		}
		// Attributes or active skills: new_rating * 5
		return newVal * 5

	case "cyberpunk_red":
		if field == "role_ability" {
			return 30
		}
		return newVal * 10

	case "wfrp":
		return 10

	case "starwars":
		return newVal * 5

	case "l5r":
		l5rRings := map[string]bool{
			"air": true, "earth": true, "fire": true, "water": true, "void": true,
		}
		if l5rRings[field] {
			return newVal * 3
		}
		return newVal * 2

	case "theonering":
		return newVal

	case "blades":
		return 8 // full XP track (threshold)

	case "ironsworn":
		if strings.HasPrefix(field, "asset:") {
			if newVal >= 2 {
				return 1 // upgrade
			}
			return 2 // new asset
		}
		return 2

	default:
		return 0
	}
}

// wgAttributes are the 7 W&G core attributes (full field names as stored in stats JSON).
var wgAttributes = []string{
	"strength", "agility", "toughness", "intellect", "willpower", "fellowship", "initiative",
}

// wgSkills are the 19 W&G skills.
var wgSkills = []string{
	"ws", "bs", "athletics", "awareness", "cunning", "deception", "fortitude",
	"insight", "intimidation", "investigation", "leadership", "medicae",
	"persuasion", "pilot", "psychic_mastery", "scholar", "stealth", "survival", "tech",
}

// ValidFields returns the list of advanceable field names for the given system.
// For talent advances (wrath_glory), the prefix "talent:" is used but talents are not
// enumerated here — use WGTalentExists to check a specific talent.
func ValidFields(system string) []string {
	switch system {
	case "wrath_glory":
		out := make([]string, 0, len(wgAttributes)+len(wgSkills))
		out = append(out, wgAttributes...)
		out = append(out, wgSkills...)
		return out

	case "dnd5e":
		return []string{"level"}

	case "vtm":
		return []string{
			// Attributes (full names matching character sheet)
			"strength", "dexterity", "stamina",
			"charisma", "manipulation", "composure",
			"intelligence", "wits", "resolve",
			// Skills
			"athletics", "brawl", "craft", "drive", "firearms", "larceny", "melee",
			"stealth", "survival", "animal_ken", "etiquette", "insight", "intimidation",
			"leadership", "performance", "persuasion", "streetwise", "subterfuge",
			"academics", "awareness", "finance", "investigation", "medicine",
			"occult", "politics", "technology",
			// Disciplines
			"animalism", "auspex", "blood_sorcery", "celerity", "dominate",
			"fortitude", "obfuscate", "oblivion", "potence", "presence", "protean",
			// Other
			"blood_potency",
		}

	case "shadowrun":
		return []string{
			"body", "agility", "reaction", "strength", "willpower", "logic",
			"intuition", "charisma", "edge", "magic", "resonance",
			"firearms", "close_combat", "piloting", "electronics", "cracking",
			"engineering", "biotech", "stealth", "athletics", "perception",
			"specialization",
		}

	case "cyberpunk_red":
		return []string{
			"athletics", "brawling", "concentration", "conversation", "education",
			"evasion", "first_aid", "handgun", "perception", "persuasion",
			"stealth", "streetwise", "tracking",
			"role_ability",
		}

	case "wfrp":
		return []string{
			"ws", "bs", "s", "t", "ag", "i", "dex", "int", "wp", "fel",
			"athletics", "bribery", "charm", "cool", "consume_alcohol",
			"dodge", "endurance", "evaluate", "gamble", "gossip", "haggle",
			"intimidate", "intuition", "leadership", "melee", "navigation",
			"outdoor_survival", "perception", "ride", "row", "stealth",
		}

	case "starwars":
		return []string{
			"astrogation", "athletics", "brawl", "charm", "coercion",
			"computers", "cool", "coordination", "core_worlds", "deception",
			"discipline", "education", "gunnery", "leadership", "lore",
			"mechanics", "medicine", "melee", "negotiation", "outer_rim",
			"perception", "piloting_planetary", "piloting_space", "ranged_heavy",
			"ranged_light", "resilience", "skullduggery", "stealth",
			"streetwise", "survival", "underworld", "vigilance", "xenology",
		}

	case "l5r":
		return []string{
			"air", "earth", "fire", "water", "void",
			"aesthetics", "arts", "courtesy", "culture", "design", "discourse",
			"fitness", "games", "government", "labor", "medicine", "meditation",
			"performance", "pressure_points", "psyche", "read_air",
			"smithing", "spiritual", "theology", "trade",
			"commerce", "command", "courtesy_l", "culture_l", "design_l",
			"government_l", "martial_arts_melee", "martial_arts_ranged",
			"martial_arts_unarmed", "ninjutsu",
		}

	case "theonering":
		return []string{
			"awe", "athletics", "awareness", "hunting", "song", "craft",
			"enhearten", "travel", "insight", "healing", "courtesy", "battle",
			"persuade", "stealth", "scan", "explore", "riddle", "lore",
		}

	case "blades":
		return []string{
			"action:Hunt", "action:Study", "action:Survey", "action:Tinker",
			"action:Finesse", "action:Prowl", "action:Skirmish", "action:Wreck",
			"action:Attune", "action:Command", "action:Consort", "action:Sway",
		}

	case "ironsworn":
		// Assets are dynamic; return a sentinel indicating asset advances are valid
		return []string{"asset:"}

	default:
		return nil
	}
}

// CanAffordAny returns true if the character can afford at least one valid advancement
// for the system, given their current XP and stats. Pass statsJSON for VtM discipline checks.
// Returns false for coc and paranoia (no XP advancement).
func CanAffordAny(system string, currentXP int, statsJSON string) bool {
	switch system {
	case "coc", "paranoia":
		return false
	}

	// For D&D 5e, check level threshold
	if system == "dnd5e" {
		var stats map[string]any
		if err := json.Unmarshal([]byte(statsJSON), &stats); err != nil {
			return false
		}
		level := 1
		if lv, ok := stats["level"].(float64); ok {
			level = int(lv)
		}
		if level >= 20 {
			return false
		}
		thresholds := []int{0, 300, 900, 2700, 6500, 14000, 23000, 34000, 48000,
			64000, 85000, 100000, 120000, 140000, 165000, 195000, 225000, 265000, 305000, 355000}
		return currentXP >= thresholds[level]
	}

	// For Blades, check XP >= 8
	if system == "blades" {
		return currentXP >= 8
	}

	// For all other systems, find minimum cost across valid fields
	var stats map[string]any
	if statsJSON != "" {
		json.Unmarshal([]byte(statsJSON), &stats) //nolint:errcheck
	}

	for _, field := range ValidFields(system) {
		if field == "asset:" {
			// Ironsworn: cheapest is 1 XP (upgrade)
			if currentXP >= 1 {
				return true
			}
			continue
		}
		// Get current value of field
		currentVal := 0
		if stats != nil {
			if v, ok := stats[field].(float64); ok {
				currentVal = int(v)
			}
		}
		cost := XPCostFor(system, field, currentVal+1, statsJSON)
		if cost > 0 && currentXP >= cost {
			return true
		}
	}

	// W&G: cheapest purchasable talent costs 10 XP (see wgTalentTable minimum).
	if system == "wrath_glory" && currentXP >= 10 {
		return true
	}

	// VtM: cheapest discipline advance is in-clan tier 1 at new_dots*5 = 1*5 = 5 XP.
	if system == "vtm" && currentXP >= 5 {
		return true
	}

	return false
}

// VtMInClanDisciplinesFor returns the in-clan discipline field keys for a VtM clan.
// Lookup is case-insensitive. The second return value is false if the clan is not recognized.
// Caitiff returns an empty slice (all disciplines are out-of-clan for them).
func VtMInClanDisciplinesFor(clan string) ([]string, bool) {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(clan), " ", "_"))
	discs, ok := vtmInClanDisciplines[normalized]
	return discs, ok
}

// CostRulesDescription returns a brief human-readable summary of XP costs for a system.
func CostRulesDescription(system string) string {
	switch system {
	case "wrath_glory":
		return "Attribute advance: new_rating × 4 XP. Skill advance: new_rating × 4 XP. Talent: fixed XP cost per talent (10–60)."
	case "dnd5e":
		return "Level up when XP ≥ threshold. XP thresholds: 0, 300, 900, 2700, 6500, 14000, 23000, 34000, 48000, 64000, 85000, 100000, 120000, 140000, 165000, 195000, 225000, 265000, 305000, 355000."
	case "vtm":
		return "Attribute: new_dots × 4. Skill: new_dots × 3. In-clan Discipline: new_dots × 5. Out-of-clan Discipline: new_dots × 7. Blood Potency: new_dots × 10."
	case "cyberpunk_red":
		return "Skill advance: new_rating × 10 IP. Role ability rank: 30 IP flat."
	case "shadowrun":
		return "Attribute: new_rating × 5 Karma. Active skill: new_rating × 5 Karma. Knowledge skill: new_rating × 2 Karma. Specialization: 5 Karma."
	case "wfrp":
		return "Any characteristic, skill, or talent advance costs 10 XP each."
	case "starwars":
		return "Skill advance: new_rating × 5 XP. Treat all skills as career for simplicity."
	case "l5r":
		return "Ring advance: new_rank × 3 XP. Skill advance: new_rank × 2 XP."
	case "theonering":
		return "Skill advance: new_rank × 1 AP."
	case "blades":
		return "XP track fills to 8. On advance, XP resets to 0 and character gains 1 action dot or special ability."
	case "ironsworn":
		return "Asset: 2 XP. Asset upgrade: 1 XP."
	default:
		return "XP advance cost varies by system."
	}
}

// FieldHints returns a string listing valid field keys for the system, suitable for
// injection into an AI prompt. This prevents the AI from guessing human-readable
// names (e.g. "ballistic_skill") instead of the actual JSON stat keys (e.g. "bs").
func FieldHints(system string) string {
	switch system {
	case "wrath_glory":
		return `Valid field keys (use EXACTLY as shown):
Attributes: strength, agility, toughness, intellect, willpower, fellowship, initiative
Skills: ws (Weapon Skill), bs (Ballistic Skill), athletics, awareness, cunning, deception,
  fortitude, insight, intimidation, investigation, leadership, medicae, persuasion,
  pilot, psychic_mastery, scholar, stealth, survival, tech
Talents: talent:<Name> (e.g. "talent:Iron Will") — use exact talent name`

	case "wfrp":
		return `Valid field keys (use EXACTLY as shown):
Characteristics: ws, bs, s, t, ag, i, dex, int, wp, fel
Skills: athletics, bribery, charm, cool, consume_alcohol, dodge, endurance, evaluate,
  gamble, gossip, haggle, intimidate, intuition, leadership, melee, navigation,
  outdoor_survival, perception, ride, row, stealth`

	case "shadowrun":
		return `Valid field keys (use EXACTLY as shown):
Attributes: body, agility, reaction, strength, willpower, logic, intuition, charisma, edge, magic, resonance
Active skills: firearms, close_combat, piloting, electronics, cracking, engineering, biotech, stealth, athletics, perception
Other: specialization`

	case "vtm":
		return `Valid field keys (use EXACTLY as shown):
Attributes: strength, dexterity, stamina, charisma, manipulation, composure, intelligence, wits, resolve
Skills: athletics, brawl, craft, drive, firearms, larceny, melee, stealth, survival,
  animal_ken, etiquette, insight, intimidation, leadership, performance, persuasion,
  streetwise, subterfuge, academics, awareness, finance, investigation, medicine,
  occult, politics, technology
Disciplines: animalism, auspex, blood_sorcery, celerity, dominate, fortitude,
  obfuscate, oblivion, potence, presence, protean
Other: blood_potency`

	case "blades":
		return `Valid field keys (use EXACTLY as shown):
Actions: action:Hunt, action:Study, action:Survey, action:Tinker,
  action:Finesse, action:Prowl, action:Skirmish, action:Wreck,
  action:Attune, action:Command, action:Consort, action:Sway`

	default:
		fields := ValidFields(system)
		if len(fields) == 0 {
			return ""
		}
		return "Valid field keys: " + strings.Join(fields, ", ")
	}
}

// WGRecalcDerived recalculates W&G derived stats that depend on the given field.
// stats is the full character stats map (modified in place).
// field is the attribute that was just advanced (e.g., "toughness").
// Requires "archetype" key in stats to look up tier.
func WGRecalcDerived(stats map[string]any, field string) {
	tier := 1
	if archName, ok := stats["archetype"].(string); ok {
		if def, exists := wgArchetypes[archName]; exists {
			tier = def.tier
		}
	}

	getInt := func(key string) int {
		if v, ok := stats[key].(float64); ok {
			return int(v)
		}
		if v, ok := stats[key].(int); ok {
			return v
		}
		return 0
	}

	switch field {
	case "toughness":
		tgh := getInt("toughness")
		stats["wounds"] = (tier * 2) + tgh
		stats["resilience"] = tgh + 1
		stats["determination"] = tgh
	case "willpower":
		wil := getInt("willpower")
		resolve := wil - 1
		if resolve < 1 {
			resolve = 1
		}
		stats["shock"] = wil + tier
		stats["resolve"] = resolve
		stats["conviction"] = wil
	case "fellowship":
		fel := getInt("fellowship")
		influence := fel - 1
		if influence < 0 {
			influence = 0
		}
		stats["influence"] = influence
	case "initiative":
		ini := getInt("initiative")
		defence := ini - 1
		if defence < 0 {
			defence = 0
		}
		stats["defence"] = defence
	}
}
