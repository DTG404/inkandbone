package db

import (
	"testing"
)

func TestMacroCRUD(t *testing.T) {
	d := newTestDB(t)
	campID, _ := d.CreateCampaign(1, "c", "")
	charID, _ := d.CreateCharacter(campID, "Hero")

	// Create
	id1, err := d.CreateMacro(charID, "Attack", "I swing my sword.", "var(--gold)")
	if err != nil {
		t.Fatal(err)
	}
	id2, _ := d.CreateMacro(charID, "Dodge", "I dodge!", "#c0392b")

	// List — ordered by sort_order
	macros, err := d.ListMacros(charID)
	if err != nil {
		t.Fatal(err)
	}
	if len(macros) != 2 {
		t.Fatalf("want 2 macros, got %d", len(macros))
	}
	if macros[0].Label != "Attack" {
		t.Fatalf("want Attack first, got %s", macros[0].Label)
	}

	// Update
	if err := d.UpdateMacro(id1, "Slash", "I slash with my blade.", "#3498db"); err != nil {
		t.Fatal(err)
	}
	macros, _ = d.ListMacros(charID)
	if macros[0].Label != "Slash" {
		t.Fatalf("want Slash after update, got %s", macros[0].Label)
	}

	// Reorder
	if err := d.ReorderMacros(charID, []int64{id2, id1}); err != nil {
		t.Fatal(err)
	}
	macros, _ = d.ListMacros(charID)
	if macros[0].Label != "Dodge" {
		t.Fatalf("want Dodge first after reorder, got %s", macros[0].Label)
	}

	// Delete
	if err := d.DeleteMacro(id2); err != nil {
		t.Fatal(err)
	}
	macros, _ = d.ListMacros(charID)
	if len(macros) != 1 {
		t.Fatalf("want 1 macro after delete, got %d", len(macros))
	}
}

func TestMacroSortOrderAppends(t *testing.T) {
	d := newTestDB(t)
	campID, _ := d.CreateCampaign(1, "c", "")
	charID, _ := d.CreateCharacter(campID, "Hero")

	d.CreateMacro(charID, "A", "a", "")
	d.CreateMacro(charID, "B", "b", "")
	d.CreateMacro(charID, "C", "c", "")

	macros, _ := d.ListMacros(charID)
	for i, m := range macros {
		if m.SortOrder != i {
			t.Fatalf("macro %d: want sort_order %d, got %d", i, i, m.SortOrder)
		}
	}
}
