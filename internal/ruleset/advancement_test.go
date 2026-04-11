package ruleset

import (
	"testing"
)

func TestXPKey(t *testing.T) {
	tests := []struct {
		system string
		want   string
	}{
		{"wrath_glory", "xp"},
		{"shadowrun", "karma"},
		{"dnd5e", "xp"},
		{"vtm", "xp"},
		{"cyberpunk_red", "xp"},
		{"blades", "xp"},
		{"wfrp", "xp"},
		{"starwars", "xp"},
		{"l5r", "xp"},
		{"theonering", "xp"},
		{"ironsworn", "xp"},
	}
	for _, tt := range tests {
		if got := XPKey(tt.system); got != tt.want {
			t.Errorf("XPKey(%q) = %q, want %q", tt.system, got, tt.want)
		}
	}
}

func TestXPLabel(t *testing.T) {
	tests := []struct {
		system string
		want   string
	}{
		{"wrath_glory", "XP"},
		{"shadowrun", "Karma"},
		{"cyberpunk_red", "IP"},
		{"theonering", "AP"},
		{"dnd5e", "XP"},
		{"vtm", "XP"},
	}
	for _, tt := range tests {
		if got := XPLabel(tt.system); got != tt.want {
			t.Errorf("XPLabel(%q) = %q, want %q", tt.system, got, tt.want)
		}
	}
}

func TestXPCostFor_wrathGlory(t *testing.T) {
	// Attribute advance: new_rating * 4
	if got := XPCostFor("wrath_glory", "toughness", 5, ""); got != 20 {
		t.Errorf("toughness to 5: got %d, want 20", got)
	}
	if got := XPCostFor("wrath_glory", "strength", 3, ""); got != 12 {
		t.Errorf("strength to 3: got %d, want 12", got)
	}
	// Skill advance: new_rating * 4
	if got := XPCostFor("wrath_glory", "athletics", 2, ""); got != 8 {
		t.Errorf("athletics to 2: got %d, want 8", got)
	}
	// Talent: uses WGTalentCost
	if got := XPCostFor("wrath_glory", "talent:Iron Will", 1, ""); got != 20 {
		t.Errorf("talent:Iron Will: got %d, want 20", got)
	}
}

func TestXPCostFor_dnd5e(t *testing.T) {
	// Level advance: no XP subtracted (threshold-based)
	if got := XPCostFor("dnd5e", "level", 2, ""); got != 0 {
		t.Errorf("dnd5e level up: got %d, want 0", got)
	}
}

func TestXPCostFor_vtm(t *testing.T) {
	// Attribute: new_dots * 4
	if got := XPCostFor("vtm", "strength", 3, ""); got != 12 {
		t.Errorf("vtm strength to 3: got %d, want 12", got)
	}
	// Skill: new_dots * 3
	if got := XPCostFor("vtm", "athletics", 2, ""); got != 6 {
		t.Errorf("vtm athletics to 2: got %d, want 6", got)
	}
	// Blood Potency: new_dots * 10
	if got := XPCostFor("vtm", "blood_potency", 2, ""); got != 20 {
		t.Errorf("vtm blood_potency to 2: got %d, want 20", got)
	}
	// In-clan discipline: new_dots * 5 (clan in statsJSON, capitalized)
	statsJSON := `{"clan":"Brujah","potence":1}`
	if got := XPCostFor("vtm", "potence", 2, statsJSON); got != 10 {
		t.Errorf("vtm in-clan discipline (potence) to 2: got %d, want 10", got)
	}
	// In-clan discipline: case-insensitive clan lookup
	statsJSONLower := `{"clan":"brujah","potence":1}`
	if got := XPCostFor("vtm", "potence", 2, statsJSONLower); got != 10 {
		t.Errorf("vtm in-clan discipline lowercase clan: got %d, want 10", got)
	}
	// Out-of-clan discipline: new_dots * 7
	statsJSON2 := `{"clan":"Brujah","oblivion":1}`
	if got := XPCostFor("vtm", "oblivion", 2, statsJSON2); got != 14 {
		t.Errorf("vtm out-of-clan discipline (oblivion) to 2: got %d, want 14", got)
	}
	// Caitiff: all disciplines out-of-clan (new_dots * 7)
	statsJSONCaitiff := `{"clan":"Caitiff","potence":1}`
	if got := XPCostFor("vtm", "potence", 2, statsJSONCaitiff); got != 14 {
		t.Errorf("vtm Caitiff discipline (potence) to 2: got %d, want 14", got)
	}
}

func TestXPCostFor_shadowrun(t *testing.T) {
	// Attribute: new_rating * 5
	if got := XPCostFor("shadowrun", "body", 4, ""); got != 20 {
		t.Errorf("shadowrun body to 4: got %d, want 20", got)
	}
	// Specialization: 5
	if got := XPCostFor("shadowrun", "specialization", 1, ""); got != 5 {
		t.Errorf("shadowrun specialization: got %d, want 5", got)
	}
}

func TestXPCostFor_wfrp(t *testing.T) {
	// Flat 10 XP per advance
	if got := XPCostFor("wfrp", "ws", 1, ""); got != 10 {
		t.Errorf("wfrp ws: got %d, want 10", got)
	}
}

func TestXPCostFor_cyberpunkRed(t *testing.T) {
	// Skill: new_rating * 10
	if got := XPCostFor("cyberpunk_red", "athletics", 5, ""); got != 50 {
		t.Errorf("cyberpunk_red athletics to 5: got %d, want 50", got)
	}
	// Role ability: flat 30
	if got := XPCostFor("cyberpunk_red", "role_ability", 2, ""); got != 30 {
		t.Errorf("cyberpunk_red role_ability: got %d, want 30", got)
	}
}

func TestXPCostFor_starwars(t *testing.T) {
	// Skill: new_rating * 5
	if got := XPCostFor("starwars", "athletics", 3, ""); got != 15 {
		t.Errorf("starwars athletics to 3: got %d, want 15", got)
	}
}

func TestXPCostFor_l5r(t *testing.T) {
	// Ring: new_rank * 3
	if got := XPCostFor("l5r", "air", 3, ""); got != 9 {
		t.Errorf("l5r air to 3: got %d, want 9", got)
	}
	// Skill: new_rank * 2
	if got := XPCostFor("l5r", "athletics", 2, ""); got != 4 {
		t.Errorf("l5r athletics to 2: got %d, want 4", got)
	}
}

func TestXPCostFor_theonering(t *testing.T) {
	// Skill: new_rank * 1
	if got := XPCostFor("theonering", "athletics", 3, ""); got != 3 {
		t.Errorf("theonering athletics to 3: got %d, want 3", got)
	}
}

func TestXPCostFor_blades(t *testing.T) {
	// Blades advance triggers at xp >= 8; cost counted as 8 (the threshold)
	if got := XPCostFor("blades", "action:Hunt", 1, ""); got != 8 {
		t.Errorf("blades action: got %d, want 8", got)
	}
}

func TestXPCostFor_ironsworn(t *testing.T) {
	// Asset: 2 XP
	if got := XPCostFor("ironsworn", "asset:Shadow", 1, ""); got != 2 {
		t.Errorf("ironsworn new asset: got %d, want 2", got)
	}
	// Upgrade: 1 XP (new_value >= 2 means upgrade)
	if got := XPCostFor("ironsworn", "asset:Shadow", 2, ""); got != 1 {
		t.Errorf("ironsworn upgrade: got %d, want 1", got)
	}
}

func TestVtMInClanDisciplinesSpacedName(t *testing.T) {
	// "Banu Haqim" stored with a space must resolve to banu_haqim disciplines.
	discs, ok := VtMInClanDisciplinesFor("Banu Haqim")
	if !ok {
		t.Fatal("VtMInClanDisciplinesFor returned not-ok for 'Banu Haqim'")
	}
	want := []string{"blood_sorcery", "celerity", "obfuscate"}
	if len(discs) != len(want) {
		t.Fatalf("got %v, want %v", discs, want)
	}
	for i, d := range discs {
		if d != want[i] {
			t.Errorf("disc[%d] got %q want %q", i, d, want[i])
		}
	}
}

func TestCanAffordAny(t *testing.T) {
	// W&G with 8 XP: can afford skill at rating 2 (cost 8)
	statsJSON := `{"xp":8,"strength":2,"agility":2,"toughness":2,"intellect":2,"willpower":2,"fellowship":2,"initiative":2,"ws":0,"bs":0,"athletics":0,"awareness":0,"cunning":0,"deception":0,"fortitude":0,"insight":0,"intimidation":0,"investigation":0,"leadership":0,"medicae":0,"persuasion":0,"pilot":0,"psychic_mastery":0,"scholar":0,"stealth":0,"survival":0,"tech":0}`
	if !CanAffordAny("wrath_glory", 8, statsJSON) {
		t.Error("expected CanAffordAny=true for W&G with 8 XP and zero skills")
	}
	// W&G with 0 XP: cannot afford anything
	if CanAffordAny("wrath_glory", 0, statsJSON) {
		t.Error("expected CanAffordAny=false for W&G with 0 XP")
	}
	// CoC: always false (no XP advancement)
	if CanAffordAny("coc", 999, "{}") {
		t.Error("expected CanAffordAny=false for coc")
	}
	// Paranoia: always false
	if CanAffordAny("paranoia", 999, "{}") {
		t.Error("expected CanAffordAny=false for paranoia")
	}
}

func TestValidFields_wrathGlory(t *testing.T) {
	fields := ValidFields("wrath_glory")
	// Must include attribute fields
	want := map[string]bool{
		"strength": true, "agility": true, "toughness": true,
		"intellect": true, "willpower": true, "fellowship": true, "initiative": true,
		"athletics": true, "ws": true, "bs": true,
	}
	got := map[string]bool{}
	for _, f := range fields {
		got[f] = true
	}
	for k := range want {
		if !got[k] {
			t.Errorf("ValidFields(wrath_glory) missing %q", k)
		}
	}
	// Must not include derived stat fields
	derived := []string{"wounds", "defence", "resilience", "determination", "resolve", "conviction", "influence", "shock", "speed"}
	for _, d := range derived {
		if got[d] {
			t.Errorf("ValidFields(wrath_glory) should not include derived %q", d)
		}
	}
}

func TestWGRecalcDerived(t *testing.T) {
	base := func() map[string]any {
		return map[string]any{
			"toughness":  float64(5),
			"willpower":  float64(4),
			"fellowship": float64(3),
			"initiative": float64(4),
			"archetype":  "Imperial Guardsman", // tier 1
		}
	}

	t.Run("toughness", func(t *testing.T) {
		stats := base()
		WGRecalcDerived(stats, "toughness")
		if stats["wounds"] != 7 { // (1*2)+5
			t.Errorf("wounds: got %v, want 7", stats["wounds"])
		}
		if stats["resilience"] != 6 { // 5+1
			t.Errorf("resilience: got %v, want 6", stats["resilience"])
		}
		if stats["determination"] != 5 {
			t.Errorf("determination: got %v, want 5", stats["determination"])
		}
	})

	t.Run("willpower", func(t *testing.T) {
		stats := base()
		WGRecalcDerived(stats, "willpower")
		if stats["shock"] != 5 { // 4+1 (tier=1)
			t.Errorf("shock: got %v, want 5", stats["shock"])
		}
		if stats["resolve"] != 3 { // max(1, 4-1)
			t.Errorf("resolve: got %v, want 3", stats["resolve"])
		}
		if stats["conviction"] != 4 {
			t.Errorf("conviction: got %v, want 4", stats["conviction"])
		}
	})

	t.Run("willpower_floor", func(t *testing.T) {
		// willpower=1 → resolve should be clamped to 1
		stats := base()
		stats["willpower"] = float64(1)
		WGRecalcDerived(stats, "willpower")
		if stats["resolve"] != 1 {
			t.Errorf("resolve floor: got %v, want 1", stats["resolve"])
		}
	})

	t.Run("fellowship", func(t *testing.T) {
		stats := base()
		WGRecalcDerived(stats, "fellowship")
		if stats["influence"] != 2 { // 3-1
			t.Errorf("influence: got %v, want 2", stats["influence"])
		}
	})

	t.Run("fellowship_floor", func(t *testing.T) {
		stats := base()
		stats["fellowship"] = float64(1)
		WGRecalcDerived(stats, "fellowship")
		if stats["influence"] != 0 { // 1-1=0, not negative
			t.Errorf("influence floor: got %v, want 0", stats["influence"])
		}
	})

	t.Run("initiative", func(t *testing.T) {
		stats := base()
		WGRecalcDerived(stats, "initiative")
		if stats["defence"] != 3 { // 4-1
			t.Errorf("defence: got %v, want 3", stats["defence"])
		}
	})
}
