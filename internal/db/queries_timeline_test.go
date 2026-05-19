package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSessionTimeline_emptySession(t *testing.T) {
	d := newTestDB(t)
	rsID, err := d.CreateRuleset(t.Name(), "{}", "test")
	require.NoError(t, err)
	campID, err := d.CreateCampaign(rsID, "Camp", "")
	require.NoError(t, err)
	sessID, err := d.CreateSession(campID, "S1", "2026-04-03")
	require.NoError(t, err)

	entries, err := d.GetSessionTimeline(sessID)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestGetSessionTimeline_mergesAndSorts(t *testing.T) {
	d := newTestDB(t)
	rsID, err := d.CreateRuleset(t.Name(), "{}", "test")
	require.NoError(t, err)
	campID, err := d.CreateCampaign(rsID, "Camp", "")
	require.NoError(t, err)
	sessID, err := d.CreateSession(campID, "S1", "2026-04-03")
	require.NoError(t, err)

	_, err = d.CreateMessage(sessID, "user", "Hello", false, nil)
	require.NoError(t, err)
	_, err = d.LogDiceRoll(sessID, "1d20+5", 18, "[13]")
	require.NoError(t, err)

	entries, err := d.GetSessionTimeline(sessID)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	types := map[string]bool{}
	for _, e := range entries {
		types[e.Type] = true
		assert.NotEmpty(t, e.Timestamp)
		assert.NotEmpty(t, e.Data)
	}
	assert.True(t, types["message"], "expected a message entry")
	assert.True(t, types["dice_roll"], "expected a dice_roll entry")
}

func TestGetSessionTimeline_sortedByTimestamp(t *testing.T) {
	d := newTestDB(t)
	rsID, err := d.CreateRuleset(t.Name(), "{}", "test")
	require.NoError(t, err)
	campID, err := d.CreateCampaign(rsID, "Camp", "")
	require.NoError(t, err)
	sessID, err := d.CreateSession(campID, "S1", "2026-04-03")
	require.NoError(t, err)

	_, err = d.LogDiceRoll(sessID, "1d6", 4, "[4]")
	require.NoError(t, err)
	_, err = d.CreateMessage(sessID, "assistant", "The die lands on 4.", false, nil)
	require.NoError(t, err)

	entries, err := d.GetSessionTimeline(sessID)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	// Timestamps must be non-decreasing.
	assert.LessOrEqual(t, entries[0].Timestamp, entries[1].Timestamp)
}
