package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSubTaskObjective(t *testing.T) {
	d := newTestDB(t)
	rsID, _ := d.CreateRuleset("testsys", `{}`, "1")
	campID, _ := d.CreateCampaign(rsID, "Camp", "")

	parent, err := d.CreateObjective(campID, "Find the artifact", "", nil)
	require.NoError(t, err)

	child, err := d.CreateObjective(campID, "Talk to the merchant", "", &parent.ID)
	require.NoError(t, err)
	assert.Equal(t, &parent.ID, child.ParentID)

	all, err := d.ListObjectives(campID)
	require.NoError(t, err)
	require.Len(t, all, 2)

	var parentRow, childRow *Objective
	for i := range all {
		if all[i].ID == parent.ID {
			parentRow = &all[i]
		} else {
			childRow = &all[i]
		}
	}
	require.NotNil(t, parentRow)
	require.NotNil(t, childRow)
	assert.Nil(t, parentRow.ParentID)
	assert.Equal(t, &parent.ID, childRow.ParentID)
}

func TestDeleteObjectiveCascadesSubTasks(t *testing.T) {
	d := newTestDB(t)
	rsID, _ := d.CreateRuleset("testsys", `{}`, "1")
	campID, _ := d.CreateCampaign(rsID, "Camp", "")

	parent, _ := d.CreateObjective(campID, "Main quest", "", nil)
	d.CreateObjective(campID, "Sub-task A", "", &parent.ID)
	d.CreateObjective(campID, "Sub-task B", "", &parent.ID)

	require.NoError(t, d.DeleteObjective(parent.ID))

	all, err := d.ListObjectives(campID)
	require.NoError(t, err)
	assert.Empty(t, all, "sub-tasks should be deleted with parent")
}

func TestDeleteObjectiveCascadesNestedDescendants(t *testing.T) {
	d := newTestDB(t)
	rulesetID, err := d.CreateRuleset("nested_objectives", `{}`, "1")
	require.NoError(t, err)
	campaignID, err := d.CreateCampaign(rulesetID, "Camp", "")
	require.NoError(t, err)

	root, err := d.CreateObjective(campaignID, "Root", "", nil)
	require.NoError(t, err)
	child, err := d.CreateObjective(campaignID, "Child", "", &root.ID)
	require.NoError(t, err)
	_, err = d.CreateObjective(campaignID, "Grandchild", "", &child.ID)
	require.NoError(t, err)

	require.NoError(t, d.DeleteObjective(root.ID))
	objectives, err := d.ListObjectives(campaignID)
	require.NoError(t, err)
	assert.Empty(t, objectives)
	require.NoError(t, foreignKeyCheck(d.db))
}
