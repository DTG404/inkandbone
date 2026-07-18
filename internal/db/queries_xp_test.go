package db_test

import (
	"testing"

	"github.com/digitalghost404/inkandbone/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestDB(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { d.Close() })
	return d
}

func TestXPLog(t *testing.T) {
	d := newTestDB(t)
	rsID, _ := d.CreateRuleset("testsys", `{}`, "1")
	campID, _ := d.CreateCampaign(rsID, "Camp", "")
	sessID, _ := d.CreateSession(campID, "S1", "2026-04-04")

	// Create with amount
	amt := 150
	entry, err := d.CreateXP(sessID, "Defeated the bandit lord", &amt)
	require.NoError(t, err)
	assert.Equal(t, "Defeated the bandit lord", entry.Note)
	require.NotNil(t, entry.Amount)
	assert.Equal(t, 150, *entry.Amount)

	// Create without amount
	entry2, err := d.CreateXP(sessID, "Roleplay milestone", nil)
	require.NoError(t, err)
	assert.Nil(t, entry2.Amount)

	// List
	all, err := d.ListXP(sessID)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	// Delete
	require.NoError(t, d.DeleteXP(entry.ID))
	all, err = d.ListXP(sessID)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}
