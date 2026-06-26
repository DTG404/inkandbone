package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkEmbedding(t *testing.T) {
	d := newTestDB(t)
	rsID, err := d.CreateRuleset("test", "{}", "test")
	require.NoError(t, err)

	chunks := []RulebookChunk{{Source: "Core", Heading: "Combat", Content: "Roll dice to hit."}}
	require.NoError(t, d.CreateRulebookChunks(rsID, chunks))

	pending, err := d.ListChunksForEmbedding(rsID)
	require.NoError(t, err)
	require.Len(t, pending, 1)

	emb := []float32{0.1, 0.2, 0.3}
	require.NoError(t, d.UpsertChunkEmbedding(pending[0].ID, emb))

	all, err := d.ListAllChunks(rsID)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.NotEmpty(t, all[0].Embedding)

	// no longer pending after embedding stored
	pending2, err := d.ListChunksForEmbedding(rsID)
	require.NoError(t, err)
	assert.Empty(t, pending2)
}
