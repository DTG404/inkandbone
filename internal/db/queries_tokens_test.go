package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapTokenCRUD(t *testing.T) {
	d := newTestDB(t)
	campID := setupCampaign(t, d)

	mapID, err := d.CreateMap(campID, "Dungeon", "maps/a.svg")
	require.NoError(t, err)

	tokenID, err := d.PlaceToken(mapID, "character", 99, 0.5, 0.5)
	require.NoError(t, err)
	assert.Positive(t, tokenID)

	tokens, err := d.ListMapTokens(mapID)
	require.NoError(t, err)
	require.Len(t, tokens, 1)
	assert.Equal(t, "character", tokens[0].EntityType)

	// duplicate placement returns error
	_, err = d.PlaceToken(mapID, "character", 99, 0.3, 0.3)
	assert.Error(t, err)

	require.NoError(t, d.MoveToken(tokenID, 0.8, 0.2))
	tokens, _ = d.ListMapTokens(mapID)
	assert.InDelta(t, 0.8, tokens[0].X, 0.001)

	require.NoError(t, d.RemoveToken(tokenID))
	tokens, _ = d.ListMapTokens(mapID)
	assert.Empty(t, tokens)
}
