package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretCRUD(t *testing.T) {
	db := newTestDB(t)
	campaignID := seedCampaign(t, db)
	sessionID, err := db.CreateSession(campaignID, "S1", "2026-05-18")
	require.NoError(t, err)

	id, err := db.CreateSecret(campaignID, "The Lost Vault", "The secret vault is hidden beneath the old temple.", "secret")
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	s, err := db.GetSecret(id)
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, "The Lost Vault", s.Title)
	assert.Equal(t, campaignID, s.CampaignID)
	assert.Equal(t, "secret", s.Category)
	assert.False(t, s.Revealed)
	assert.Nil(t, s.RevealedAtSessionID)

	secrets, err := db.ListSecretsByCampaign(campaignID)
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "The Lost Vault", secrets[0].Title)

	err = db.RevealSecret(id, sessionID)
	require.NoError(t, err)
	s, _ = db.GetSecret(id)
	assert.True(t, s.Revealed)
	require.NotNil(t, s.RevealedAtSessionID)
	assert.Equal(t, sessionID, *s.RevealedAtSessionID)

	err = db.UpdateSecret(id, "The Hidden Vault", "Moved to a new location.", "clue")
	require.NoError(t, err)
	s, _ = db.GetSecret(id)
	assert.Equal(t, "The Hidden Vault", s.Title)
	assert.Equal(t, "clue", s.Category)

	err = db.DeleteSecret(id)
	require.NoError(t, err)
	secrets, _ = db.ListSecretsByCampaign(campaignID)
	assert.Len(t, secrets, 0)
}

func TestSecretGet_NotFound(t *testing.T) {
	db := newTestDB(t)
	s, err := db.GetSecret(99999)
	require.Error(t, err)
	assert.Nil(t, s)
}
