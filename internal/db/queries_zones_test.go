package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapZoneCRUD(t *testing.T) {
	d := newTestDB(t)
	campID := setupCampaign(t, d)
	mapID, err := d.CreateMap(campID, "Dungeon", "maps/a.svg")
	require.NoError(t, err)

	zoneID, err := d.CreateMapZone(mapID, "throne room", 0.1, 0.2, 0.3, 0.25)
	require.NoError(t, err)
	assert.Positive(t, zoneID)

	zones, err := d.ListMapZones(mapID)
	require.NoError(t, err)
	require.Len(t, zones, 1)
	assert.False(t, zones[0].IsRevealed)

	unrevealed, err := d.ListUnrevealedZones(mapID)
	require.NoError(t, err)
	require.Len(t, unrevealed, 1)

	require.NoError(t, d.RevealZone(zoneID, true))
	zones, _ = d.ListMapZones(mapID)
	assert.True(t, zones[0].IsRevealed)

	unrevealed, _ = d.ListUnrevealedZones(mapID)
	assert.Empty(t, unrevealed)

	require.NoError(t, d.UpdateMapZone(zoneID, "great hall", 0.1, 0.2, 0.4, 0.3))
	zones, _ = d.ListMapZones(mapID)
	assert.Equal(t, "great hall", zones[0].Name)

	require.NoError(t, d.DeleteMapZone(zoneID))
	zones, _ = d.ListMapZones(mapID)
	assert.Empty(t, zones)
}
