package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorldNoteReveal(t *testing.T) {
	d := newTestDB(t)
	campID := setupCampaign(t, d)

	id, err := d.CreateWorldNote(campID, "The Vault", "Hidden place", "location")
	require.NoError(t, err)

	// default is not revealed
	note, err := d.GetWorldNote(id)
	require.NoError(t, err)
	assert.False(t, note.IsRevealed)

	// reveal it
	require.NoError(t, d.PatchWorldNoteRevealed(id, true))
	note, err = d.GetWorldNote(id)
	require.NoError(t, err)
	assert.True(t, note.IsRevealed)

	// filter by revealed=true
	revealed := true
	results, err := d.SearchWorldNotes(campID, "", "", "", &revealed)
	require.NoError(t, err)
	assert.Len(t, results, 1)

	// filter by revealed=false returns none
	notRevealed := false
	results, err = d.SearchWorldNotes(campID, "", "", "", &notRevealed)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestGetLatestMap(t *testing.T) {
	d := newTestDB(t)
	campID := setupCampaign(t, d)

	m, err := d.GetLatestMap(campID)
	require.NoError(t, err)
	assert.Nil(t, m)

	id1, err := d.CreateMap(campID, "Town Square", "maps/a.svg")
	require.NoError(t, err)
	_, err = d.CreateMap(campID, "Dungeon", "maps/b.svg")
	require.NoError(t, err)
	_ = id1

	latest, err := d.GetLatestMap(campID)
	require.NoError(t, err)
	require.NotNil(t, latest)
	assert.Equal(t, "Dungeon", latest.Name)
}
