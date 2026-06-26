package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeckCRUD(t *testing.T) {
	d := newTestDB(t)
	campID := setupCampaign(t, d)
	sessID, err := d.CreateSession(campID, "S1", "2026-01-01")
	require.NoError(t, err)

	id, err := d.CreateDeck(campID, "Oracle", `[{"front":"Moon","back":"Change"},{"front":"Sun"}]`)
	require.NoError(t, err)
	assert.Positive(t, id)

	decks, err := d.ListDecks(campID)
	require.NoError(t, err)
	require.Len(t, decks, 1)
	assert.Equal(t, "Oracle", decks[0].Name)

	deck, err := d.GetDeck(id)
	require.NoError(t, err)
	require.NotNil(t, deck)

	require.NoError(t, d.ShuffleDeck(id, "[1,0]"))
	deck, _ = d.GetDeck(id)
	assert.Equal(t, "[1,0]", deck.ShuffledOrderJSON)
	assert.Equal(t, 0, deck.DrawIndex)

	require.NoError(t, d.DrawCard(id, 0, 1, sessID, `{"front":"Moon","back":"Change"}`))
	deck, _ = d.GetDeck(id)
	assert.Equal(t, 1, deck.DrawIndex)

	draws, err := d.ListDeckDraws(sessID)
	require.NoError(t, err)
	require.Len(t, draws, 1)

	require.NoError(t, d.DeleteDeck(id))
	decks, _ = d.ListDecks(campID)
	assert.Empty(t, decks)
}
