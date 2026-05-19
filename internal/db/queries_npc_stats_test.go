package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNpcStatCRUD(t *testing.T) {
	d := newTestDB(t)
	campaignID := seedCampaign(t, d)

	ac := 15
	initMod := 2

	// Create
	id, err := d.CreateNpcStat(campaignID, "Goblin Scout", "scout", `{"str":8,"dex":14}`, 7, &ac, initMod, `["Stealth +4"]`, `["Nimble Escape"]`, `["Shortbow","Dagger"]`, "A nimble goblin look out")
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	// Create another with nil AC
	id2, err := d.CreateNpcStat(campaignID, "Gelatinous Cube", "ooze", `{"str":16}`, 84, nil, -5, `[]`, `["Engulf"]`, `[]`, "")
	require.NoError(t, err)
	assert.Greater(t, id2, int64(0))

	// Get
	n, err := d.GetNpcStat(id)
	require.NoError(t, err)
	require.NotNil(t, n)
	assert.Equal(t, "Goblin Scout", n.Name)
	assert.Equal(t, campaignID, n.CampaignID)
	assert.Equal(t, "scout", n.Role)
	assert.Equal(t, 7, n.HPMax)
	require.NotNil(t, n.ArmorClass)
	assert.Equal(t, 15, *n.ArmorClass)
	assert.Equal(t, 2, n.InitiativeMod)
	assert.Equal(t, `["Stealth +4"]`, n.Skills)
	assert.Equal(t, `["Nimble Escape"]`, n.Abilities)

	// List
	stats, err := d.ListNpcStats(campaignID)
	require.NoError(t, err)
	require.Len(t, stats, 2)

	// Update
	newAC := 17
	err = d.UpdateNpcStats(id, "Goblin Scout (Elite)", "scout", `{"str":10,"dex":16}`, 14, &newAC, 3, `["Stealth +6","Perception +3"]`, `["Nimble Escape","Sneak Attack"]`, `["Shortbow+1","Dagger","Potion of Healing"]`, "An elite scout")
	require.NoError(t, err)

	n, _ = d.GetNpcStat(id)
	assert.Equal(t, "Goblin Scout (Elite)", n.Name)
	assert.Equal(t, 14, n.HPMax)
	require.NotNil(t, n.ArmorClass)
	assert.Equal(t, 17, *n.ArmorClass)
	assert.Equal(t, 3, n.InitiativeMod)
	assert.Contains(t, n.Skills, "Perception")

	// Delete
	err = d.DeleteNpcStat(id)
	require.NoError(t, err)

	stats, _ = d.ListNpcStats(campaignID)
	assert.Len(t, stats, 1)
}

func TestNpcStatGet_NotFound(t *testing.T) {
	d := newTestDB(t)
	n, err := d.GetNpcStat(99999)
	require.Error(t, err)
	assert.Nil(t, n)
}
