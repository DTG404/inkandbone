package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSession(t *testing.T, d *DB) int64 {
	t.Helper()
	campID := setupCampaign(t, d)
	sessID, err := d.CreateSession(campID, "S1", "2026-04-03")
	require.NoError(t, err)
	return sessID
}

func TestCombatEncounters(t *testing.T) {
	d := newTestDB(t)
	sessID := setupSession(t, d)

	encID, err := d.CreateEncounter(sessID, "Goblin Ambush")
	require.NoError(t, err)

	active, err := d.GetActiveEncounter(sessID)
	require.NoError(t, err)
	require.NotNil(t, active)
	assert.Equal(t, "Goblin Ambush", active.Name)

	require.NoError(t, d.EndEncounter(encID))
	active, _ = d.GetActiveEncounter(sessID)
	assert.Nil(t, active)
}

func TestCombatants(t *testing.T) {
	d := newTestDB(t)
	sessID := setupSession(t, d)
	encID, err := d.CreateEncounter(sessID, "Fight")
	require.NoError(t, err)

	cID, err := d.AddCombatant(encID, "Orc Warrior", 15, 20, false, nil)
	require.NoError(t, err)

	list, err := d.ListCombatants(encID)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, "Orc Warrior", list[0].Name)
	assert.Equal(t, 20, list[0].HPMax)
	assert.Equal(t, 20, list[0].HPCurrent)

	require.NoError(t, d.UpdateCombatant(cID, 12, `["poisoned"]`))
	list, err = d.ListCombatants(encID)
	require.NoError(t, err)
	assert.Equal(t, 12, list[0].HPCurrent)
	assert.Equal(t, `["poisoned"]`, list[0].ConditionsJSON)
}

func TestAdvanceTurn(t *testing.T) {
	d := newTestDB(t)
	sessID := setupSession(t, d)
	encID, err := d.CreateEncounter(sessID, "Goblin Ambush")
	require.NoError(t, err)

	d.AddCombatant(encID, "Rogue", 18, 20, true, nil)
	d.AddCombatant(encID, "Goblin", 12, 10, false, nil)
	d.AddCombatant(encID, "Warrior", 15, 30, true, nil)

	// Initial index is 0
	enc, err := d.GetActiveEncounter(sessID)
	require.NoError(t, err)
	assert.Equal(t, 0, enc.ActiveTurnIndex)

	// Advance: 0 → 1
	next, err := d.AdvanceTurn(encID)
	require.NoError(t, err)
	assert.Equal(t, 1, next)

	// Advance: 1 → 2
	next, err = d.AdvanceTurn(encID)
	require.NoError(t, err)
	assert.Equal(t, 2, next)

	// Advance: 2 → 0 (wraps)
	next, err = d.AdvanceTurn(encID)
	require.NoError(t, err)
	assert.Equal(t, 0, next)

	// Verify persisted
	enc, err = d.GetActiveEncounter(sessID)
	require.NoError(t, err)
	assert.Equal(t, 0, enc.ActiveTurnIndex)
}

func TestAdvanceTurnNotFound(t *testing.T) {
	d := newTestDB(t)
	_, err := d.AdvanceTurn(99999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateCombatantVtMDamage_SuperficialHalving(t *testing.T) {
	d := newTestDB(t)
	sessID := setupSession(t, d)
	encID, err := d.CreateEncounter(sessID, "Test Fight")
	require.NoError(t, err)
	combID, err := d.AddCombatant(encID, "Vampire", 10, 6, true, nil)
	require.NoError(t, err)

	// Apply 4 superficial damage (vampires halve: 4→2 applied)
	err = d.UpdateCombatantVtMDamage(combID, 4, 0, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combatants, _ := d.ListCombatants(encID)
	if len(combatants) == 0 {
		t.Fatal("no combatants returned")
	}
	c := combatants[0]
	if c.DamageSuperficial != 2 {
		t.Errorf("expected 2 superficial after halving, got %d", c.DamageSuperficial)
	}
}

func TestUpdateCombatantVtMDamage_AggravatedDirect(t *testing.T) {
	d := newTestDB(t)
	sessID := setupSession(t, d)
	encID, err := d.CreateEncounter(sessID, "Test Fight")
	require.NoError(t, err)
	combID, err := d.AddCombatant(encID, "Vampire", 10, 6, true, nil)
	require.NoError(t, err)

	err = d.UpdateCombatantVtMDamage(combID, 0, 2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combatants, _ := d.ListCombatants(encID)
	c := combatants[0]
	if c.DamageAggravated != 2 {
		t.Errorf("expected 2 aggravated, got %d", c.DamageAggravated)
	}
}

func TestReorderCombatants(t *testing.T) {
	d := newTestDB(t)
	campID, _ := d.CreateCampaign(1, "camp", "")
	sessID, _ := d.CreateSession(campID, "sess", "")
	encID, _ := d.CreateEncounter(sessID, "enc")

	a, _ := d.AddCombatant(encID, "Alpha", 20, 10, true, nil)
	b, _ := d.AddCombatant(encID, "Beta", 15, 10, false, nil)
	c, _ := d.AddCombatant(encID, "Gamma", 10, 10, false, nil)

	// Default order: Alpha(0), Beta(1), Gamma(2)
	cs, _ := d.ListCombatants(encID)
	if cs[0].Name != "Alpha" {
		t.Fatalf("want Alpha first, got %s", cs[0].Name)
	}

	// Reorder: Gamma, Alpha, Beta
	if err := d.ReorderCombatants(encID, []int64{c, a, b}); err != nil {
		t.Fatal(err)
	}
	cs, _ = d.ListCombatants(encID)
	if cs[0].Name != "Gamma" {
		t.Fatalf("want Gamma first after reorder, got %s", cs[0].Name)
	}
	if cs[1].Name != "Alpha" {
		t.Fatalf("want Alpha second after reorder, got %s", cs[1].Name)
	}
}

func TestAddCombatantSortOrder(t *testing.T) {
	d := newTestDB(t)
	campID, _ := d.CreateCampaign(1, "camp", "")
	sessID, _ := d.CreateSession(campID, "sess", "")
	encID, _ := d.CreateEncounter(sessID, "enc")

	d.AddCombatant(encID, "First", 20, 10, true, nil)
	d.AddCombatant(encID, "Second", 5, 10, false, nil) // lower initiative → appends
	cs, _ := d.ListCombatants(encID)

	if cs[0].Name != "First" || cs[0].SortOrder != 0 {
		t.Fatalf("First should have sort_order 0, got %d", cs[0].SortOrder)
	}
	if cs[1].Name != "Second" || cs[1].SortOrder != 1 {
		t.Fatalf("Second should have sort_order 1, got %d", cs[1].SortOrder)
	}
}

func TestPatchCombatantInitiative(t *testing.T) {
	d := newTestDB(t)
	campID, _ := d.CreateCampaign(1, "camp", "")
	sessID, _ := d.CreateSession(campID, "sess", "")
	encID, _ := d.CreateEncounter(sessID, "enc")
	id, _ := d.AddCombatant(encID, "Hero", 10, 10, true, nil)

	if err := d.PatchCombatantInitiative(id, 18); err != nil {
		t.Fatal(err)
	}
	cs, _ := d.ListCombatants(encID)
	if cs[0].Initiative != 18 {
		t.Fatalf("want initiative 18, got %d", cs[0].Initiative)
	}
	// sort_order must not change
	if cs[0].SortOrder != 0 {
		t.Fatalf("sort_order must remain 0 after initiative patch, got %d", cs[0].SortOrder)
	}
}
