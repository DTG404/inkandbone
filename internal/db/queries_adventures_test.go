package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdventureCRUD(t *testing.T) {
	db := newTestDB(t)
	campaignID := seedCampaign(t, db)

	// Create
	id, err := db.CreateAdventure(campaignID, "The Lost Library", "Search for ancient knowledge", "active", 0)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	// Get
	a, err := db.GetAdventure(id)
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.Equal(t, "The Lost Library", a.Title)
	assert.Equal(t, campaignID, a.CampaignID)
	assert.Equal(t, "active", a.Status)
	assert.Equal(t, 0, a.SortOrder)

	// List
	adventures, err := db.ListAdventures(campaignID)
	require.NoError(t, err)
	require.Len(t, adventures, 1)
	assert.Equal(t, "The Lost Library", adventures[0].Title)

	// Update
	err = db.UpdateAdventure(id, "The Lost Library (Revised)", "Updated description", "completed", 1)
	require.NoError(t, err)

	a, _ = db.GetAdventure(id)
	assert.Equal(t, "The Lost Library (Revised)", a.Title)
	assert.Equal(t, "completed", a.Status)
	assert.Equal(t, 1, a.SortOrder)

	// Delete
	err = db.DeleteAdventure(id)
	require.NoError(t, err)

	adventures, _ = db.ListAdventures(campaignID)
	assert.Len(t, adventures, 0)
}

func TestAdventureGet_NotFound(t *testing.T) {
	db := newTestDB(t)
	a, err := db.GetAdventure(99999)
	require.Error(t, err)
	assert.Nil(t, a)
}

func TestAdventureSessions(t *testing.T) {
	db := newTestDB(t)
	campaignID := seedCampaign(t, db)

	// Create adventure
	advID, err := db.CreateAdventure(campaignID, "Adventure One", "", "active", 0)
	require.NoError(t, err)

	// Create sessions
	sess1ID, err := db.CreateSession(campaignID, "Session 1", "2026-01-01")
	require.NoError(t, err)
	sess2ID, err := db.CreateSession(campaignID, "Session 2", "2026-01-02")
	require.NoError(t, err)

	// Assign sessions to adventure
	err = db.SetSessionAdventure(sess1ID, &advID)
	require.NoError(t, err)
	err = db.SetSessionAdventure(sess2ID, &advID)
	require.NoError(t, err)

	// List sessions by adventure
	sessions, err := db.ListSessionsByAdventure(advID)
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	assert.Equal(t, "Session 1", sessions[0].Title)
	assert.Equal(t, "Session 2", sessions[1].Title)
	assert.Equal(t, advID, *sessions[0].AdventureID)

	// Remove session from adventure
	err = db.SetSessionAdventure(sess1ID, nil)
	require.NoError(t, err)

	sessions, _ = db.ListSessionsByAdventure(advID)
	require.Len(t, sessions, 1)
	assert.Equal(t, "Session 2", sessions[0].Title)
}

func TestAdventureCreate_DefaultsStatus(t *testing.T) {
	db := newTestDB(t)
	campaignID := seedCampaign(t, db)

	id, err := db.CreateAdventure(campaignID, "Default Status", "", "", 0)
	require.NoError(t, err)

	a, err := db.GetAdventure(id)
	require.NoError(t, err)
	assert.Equal(t, "upcoming", a.Status)
}
