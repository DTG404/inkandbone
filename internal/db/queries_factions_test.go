package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactionCRUD(t *testing.T) {
	db := newTestDB(t)
	campaignID := seedCampaign(t, db)

	// Create
	id, err := db.CreateFaction(campaignID, "The Iron Legion", "A militaristic order", "faction", 8, `{"gold":500}`, "#c9a84c")
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	// Get
	f, err := db.GetFaction(id)
	require.NoError(t, err)
	require.NotNil(t, f)
	assert.Equal(t, "The Iron Legion", f.Name)
	assert.Equal(t, campaignID, f.CampaignID)
	assert.Equal(t, 8, f.Influence)
	assert.Equal(t, `{"gold":500}`, f.ResourcesJSON)

	// List
	factions, err := db.ListFactions(campaignID)
	require.NoError(t, err)
	require.Len(t, factions, 1)
	assert.Equal(t, "The Iron Legion", factions[0].Name)

	// Update
	err = db.UpdateFaction(id, "The Iron Legion (Reformed)", "A reformed militaristic order", "guild", 5, `{"gold":200}`, "#ff0000")
	require.NoError(t, err)

	f, _ = db.GetFaction(id)
	assert.Equal(t, "The Iron Legion (Reformed)", f.Name)
	assert.Equal(t, "guild", f.FactionType)
	assert.Equal(t, 5, f.Influence)
	assert.Equal(t, `{"gold":200}`, f.ResourcesJSON)
	assert.Equal(t, "#ff0000", f.Color)

	// Delete
	err = db.DeleteFaction(id)
	require.NoError(t, err)

	factions, _ = db.ListFactions(campaignID)
	assert.Len(t, factions, 0)
}

func TestFactionGet_NotFound(t *testing.T) {
	db := newTestDB(t)
	f, err := db.GetFaction(99999)
	require.Error(t, err)
	assert.Nil(t, f)
}
