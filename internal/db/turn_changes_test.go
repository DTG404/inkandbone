package db

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTurnChangesReplayAndScopedCorrections(t *testing.T) {
	d := newTestDB(t)
	session := setupSession(t, d)
	s, err := d.GetSession(session)
	require.NoError(t, err)
	c, err := d.CreateCharacter(s.CampaignID, "Sol")
	require.NoError(t, err)
	msg, err := d.CreateMessage(session, "assistant", "The door opens.", false, nil)
	require.NoError(t, err)
	turn := TurnSource{SessionID: session, MessageID: msg}
	require.NoError(t, d.UpdateCharacterData(c, `{"hp":10,"xp":0}`))
	_, err = d.ApplyTurnStats(turn, c, "stats", `{"hp":10,"xp":0}`, `{"hp":8,"xp":2}`)
	require.NoError(t, err)
	_, err = d.ApplyTurnCurrency(turn, c, 5)
	require.NoError(t, err)
	_, err = d.ApplyTurnCurrency(turn, c, 5)
	require.NoError(t, err)
	_, err = d.AddTurnEntity(turn, "npc", session, "Ada", "Pilot", 1)
	require.NoError(t, err)
	_, err = d.AddTurnEntity(turn, "item", c, "Key", "Brass", 1)
	require.NoError(t, err)
	objective, err := d.AddTurnEntity(turn, "objective", s.CampaignID, "Escape", "Leave town", 1)
	require.NoError(t, err)
	require.NoError(t, d.SetTurnObjectiveStatus(turn, s.CampaignID, objective, "Escape", "completed"))
	list, err := d.ListTurnChanges(session)
	require.NoError(t, err)
	require.Len(t, list, 7)
	byLabel := map[string]TurnChange{}
	for _, v := range list {
		byLabel[v.Label] = v
	}
	// Editing XP leaves HP safely reversible, even across a request retry.
	require.NoError(t, d.UpdateCharacterData(c, `{"hp":8,"xp":9}`))
	_, err = d.UndoTurnChange(session+1, byLabel["hp"].ID)
	require.Error(t, err)
	_, err = d.UndoTurnChange(session, byLabel["hp"].ID)
	require.NoError(t, err)
	_, err = d.UndoTurnChange(session, byLabel["hp"].ID)
	require.NoError(t, err)
	ch, err := d.GetCharacter(c)
	require.NoError(t, err)
	require.JSONEq(t, `{"hp":10,"xp":9}`, ch.DataJSON)
	require.Equal(t, int64(5), ch.CurrencyBalance)
	_, err = d.UndoTurnChange(session, byLabel["xp"].ID)
	require.ErrorIs(t, err, ErrChangeConflict)
	// Even an edit away and back is a later decision.
	require.NoError(t, d.UpdateCharacterCurrencyBalance(c, 4))
	require.NoError(t, d.UpdateCharacterCurrencyBalance(c, 5))
	_, err = d.UndoTurnChange(session, byLabel["Gold"].ID)
	require.ErrorIs(t, err, ErrChangeConflict)
	// Restoring a removal keeps original identity, equipment and metadata.
	items, err := d.ListItems(c)
	require.NoError(t, err)
	require.Len(t, items, 1)
	key := items[0]
	require.NoError(t, d.UpdateItem(key.ID, key.Name, key.Description, 3, true))
	require.NoError(t, d.RemoveTurnEntity(turn, "item", c, key.ID, key.Name))
	list, err = d.ListTurnChanges(session)
	require.NoError(t, err)
	loss := list[len(list)-1]
	require.Equal(t, "remove", loss.Operation)
	_, err = d.UndoTurnChange(session, loss.ID)
	require.NoError(t, err)
	restored, err := d.GetItem(key.ID)
	require.NoError(t, err)
	require.True(t, restored.Equipped)
	require.Equal(t, 3, restored.Quantity)
	_, err = d.UndoTurnChange(session, byLabel["Key"].ID)
	require.ErrorIs(t, err, ErrChangeConflict)
	_, err = d.UndoTurnChange(session, byLabel["Ada"].ID)
	require.NoError(t, err)
	_, err = d.AddTurnEntity(turn, "npc", session, "Ada", "Pilot", 1)
	require.NoError(t, err)
	npcs, err := d.ListSessionNPCs(session)
	require.NoError(t, err)
	require.Empty(t, npcs)
	_, err = d.UndoTurnChange(session, byLabel["Escape"].ID)
	require.NoError(t, err)
	obj, err := d.GetObjective(objective)
	require.NoError(t, err)
	require.Equal(t, "active", obj.Status)
	require.True(t, json.Valid(loss.Before))
}

func TestTurnChangesRejectForeignTargetsAndDependentObjectives(t *testing.T) {
	d := newTestDB(t)
	session := setupSession(t, d)
	s, _ := d.GetSession(session)
	msg, err := d.CreateMessage(session, "assistant", "test", false, nil)
	require.NoError(t, err)
	turn := TurnSource{SessionID: session, MessageID: msg}
	other := setupCampaign(t, d)
	_, err = d.AddTurnEntity(turn, "objective", other, "Foreign", "", 1)
	require.Error(t, err)
	id, err := d.AddTurnEntity(turn, "objective", s.CampaignID, "Parent", "", 1)
	require.NoError(t, err)
	_, err = d.CreateObjective(s.CampaignID, "Child", "", &id)
	require.NoError(t, err)
	changes, err := d.ListTurnChanges(session)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	_, err = d.UndoTurnChange(session, changes[0].ID)
	require.ErrorIs(t, err, ErrChangeConflict)
}
