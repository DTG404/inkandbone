package db

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaignNarrativePreferencesMigrationAndRoundTrip(t *testing.T) {
	database, err := Open(filepath.Join(t.TempDir(), "campaign-preferences.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	rulesetID, err := database.CreateRuleset("preferences-test", "[]", "1")
	require.NoError(t, err)
	campaignID, err := database.CreateCampaign(rulesetID, "Campaign", "")
	require.NoError(t, err)

	campaign, err := database.GetCampaign(campaignID)
	require.NoError(t, err)
	assert.Equal(t, "", campaign.ContentBoundaries)
	assert.Equal(t, "en", campaign.NarrativeLocale)

	guidance, boundaries, locale := "Measured pacing", "No body horror", "de-DE"
	require.NoError(t, database.UpdateCampaignConfig(campaignID, nil, nil, &guidance, &boundaries, &locale))
	campaign, err = database.GetCampaign(campaignID)
	require.NoError(t, err)
	assert.Equal(t, guidance, campaign.SystemPromptOverride)
	assert.Equal(t, boundaries, campaign.ContentBoundaries)
	assert.Equal(t, locale, campaign.NarrativeLocale)
}
