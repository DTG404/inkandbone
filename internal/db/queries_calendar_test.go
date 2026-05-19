package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaignDate_Defaults(t *testing.T) {
	db := newTestDB(t)
	campaignID := seedCampaign(t, db)

	info, err := db.GetCampaignDate(campaignID)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, 1, info.InGameYear)
	assert.Equal(t, 1, info.InGameMonth)
	assert.Equal(t, 1, info.InGameDay)
	assert.Equal(t, "{}", info.CalendarConfig)
}

func TestUpdateCampaignDate(t *testing.T) {
	db := newTestDB(t)
	campaignID := seedCampaign(t, db)

	cfg := `{"months":12,"month_names":["January","February"]}`
	err := db.UpdateCampaignDate(campaignID, 1247, 3, 15, &cfg)
	require.NoError(t, err)

	info, err := db.GetCampaignDate(campaignID)
	require.NoError(t, err)
	assert.Equal(t, 1247, info.InGameYear)
	assert.Equal(t, 3, info.InGameMonth)
	assert.Equal(t, 15, info.InGameDay)
	assert.Equal(t, cfg, info.CalendarConfig)
}

func TestUpdateCampaignDate_NilConfig(t *testing.T) {
	db := newTestDB(t)
	campaignID := seedCampaign(t, db)

	err := db.UpdateCampaignDate(campaignID, 999, 6, 20, nil)
	require.NoError(t, err)

	info, err := db.GetCampaignDate(campaignID)
	require.NoError(t, err)
	assert.Equal(t, 999, info.InGameYear)
	assert.Equal(t, 6, info.InGameMonth)
	assert.Equal(t, 20, info.InGameDay)
	assert.Equal(t, "{}", info.CalendarConfig)
}

func TestCalendarEventCRUD(t *testing.T) {
	db := newTestDB(t)
	campaignID := seedCampaign(t, db)
	sessionID := seedSession(t, db, campaignID)

	// Create
	eid, err := db.CreateCalendarEvent(campaignID, 1247, 3, 15, "The Battle of Iron Pass", "A decisive battle", "battle", &sessionID)
	require.NoError(t, err)
	assert.Greater(t, eid, int64(0))

	// Get
	e, err := db.GetCalendarEvent(eid)
	require.NoError(t, err)
	require.NotNil(t, e)
	assert.Equal(t, "The Battle of Iron Pass", e.Title)
	assert.Equal(t, 1247, e.InGameYear)
	assert.Equal(t, 3, e.InGameMonth)
	assert.Equal(t, 15, e.InGameDay)
	assert.Equal(t, "battle", e.EventType)
	require.NotNil(t, e.SessionID)
	assert.Equal(t, sessionID, *e.SessionID)

	// List
	events, err := db.ListCalendarEvents(campaignID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "The Battle of Iron Pass", events[0].Title)

	// Create another event without session
	eid2, err := db.CreateCalendarEvent(campaignID, 1247, 3, 16, "Peace Treaty Signed", "End of war", "note", nil)
	require.NoError(t, err)
	assert.Greater(t, eid2, int64(0))

	e2, err := db.GetCalendarEvent(eid2)
	require.NoError(t, err)
	assert.Nil(t, e2.SessionID)

	events, err = db.ListCalendarEvents(campaignID)
	require.NoError(t, err)
	assert.Len(t, events, 2)

	// Delete
	err = db.DeleteCalendarEvent(eid)
	require.NoError(t, err)

	events, _ = db.ListCalendarEvents(campaignID)
	require.Len(t, events, 1)

	_, err = db.GetCalendarEvent(eid)
	require.Error(t, err)
}

func TestCalendarEvent_NotFound(t *testing.T) {
	db := newTestDB(t)
	_, err := db.GetCalendarEvent(99999)
	require.Error(t, err)
}

func TestListCalendarEvents_Empty(t *testing.T) {
	db := newTestDB(t)
	campaignID := seedCampaign(t, db)

	events, err := db.ListCalendarEvents(campaignID)
	require.NoError(t, err)
	assert.Len(t, events, 0)
}

func TestUpdateCampaignDate_ClearConfig(t *testing.T) {
	db := newTestDB(t)
	campaignID := seedCampaign(t, db)

	cfg := `{"custom":true}`
	err := db.UpdateCampaignDate(campaignID, 5, 10, 20, &cfg)
	require.NoError(t, err)

	// Update date with empty config
	emptyCfg := `{}`
	err = db.UpdateCampaignDate(campaignID, 5, 11, 1, &emptyCfg)
	require.NoError(t, err)

	info, err := db.GetCampaignDate(campaignID)
	require.NoError(t, err)
	assert.Equal(t, 5, info.InGameYear)
	assert.Equal(t, 11, info.InGameMonth)
	assert.Equal(t, 1, info.InGameDay)
	assert.Equal(t, "{}", info.CalendarConfig)
}
