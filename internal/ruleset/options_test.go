package ruleset

import "testing"

func TestVtMOptions_V5Clans(t *testing.T) {
	opts := CharacterOptions("vtm")
	clans, ok := opts["clan"]
	if !ok {
		t.Fatal("vtm options missing clan key")
	}
	want := []string{"Brujah", "Gangrel", "Malkavian", "Nosferatu", "Toreador", "Tremere", "Ventrue", "Caitiff", "Thin-Blooded"}
	if len(clans) < len(want) {
		t.Fatalf("expected at least %d clans, got %d: %v", len(want), len(clans), clans)
	}
	clanSet := map[string]bool{}
	for _, c := range clans {
		clanSet[c] = true
	}
	for _, w := range want {
		if !clanSet[w] {
			t.Errorf("missing clan %q", w)
		}
	}
}

func TestVtMOptions_PredatorType(t *testing.T) {
	opts := CharacterOptions("vtm")
	types, ok := opts["predator_type"]
	if !ok {
		t.Fatal("vtm options missing predator_type key")
	}
	if len(types) < 10 {
		t.Fatalf("expected at least 10 predator types, got %d", len(types))
	}
}

func TestVtMOptions_Sect(t *testing.T) {
	opts := CharacterOptions("vtm")
	if _, ok := opts["sect"]; !ok {
		t.Fatal("vtm options missing sect key")
	}
}

func TestVtMOptions_Generation(t *testing.T) {
	opts := CharacterOptions("vtm")
	gens, ok := opts["generation"]
	if !ok {
		t.Fatal("vtm options missing generation key")
	}
	if len(gens) < 6 {
		t.Fatalf("expected at least 6 generation values, got %d: %v", len(gens), gens)
	}
	found := false
	for _, g := range gens {
		if g == "15th (Thin-Blooded)" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected generation values to include %q, got %v", "15th (Thin-Blooded)", gens)
	}
}

func TestVtMPlayerGuideClans(t *testing.T) {
	opts := CharacterOptions("vtm")
	clans := opts["clan"]
	wantClans := []string{"Banu Haqim", "Hecata", "Lasombra", "Ministry", "Ravnos", "Salubri", "Tzimisce"}
	for _, c := range wantClans {
		found := false
		for _, clan := range clans {
			if clan == c { found = true; break }
		}
		if !found {
			t.Errorf("clan %q not found in options", c)
		}
	}
	charTypes := opts["character_type"]
	if len(charTypes) == 0 {
		t.Error("character_type options missing")
	}
	predTypes := opts["predator_type"]
	wantPreds := []string{"Farmer", "Montero", "Scene Queen", "Treasure Hunter", "Pursuer", "Witch Hunter"}
	for _, p := range wantPreds {
		found := false
		for _, pt := range predTypes {
			if pt == p { found = true; break }
		}
		if !found {
			t.Errorf("predator_type %q not found in options", p)
		}
	}
	gens := opts["generation"]
	foundSixteenth := false
	for _, g := range gens {
		if g == "16th (Thin-Blooded)" { foundSixteenth = true; break }
	}
	if !foundSixteenth {
		t.Error("16th generation missing from options")
	}
}
