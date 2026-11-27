package db

// PerceivedNarration returns only GM passages addressed to the selected character.
// Older replies lack a speaker ID; associate those with the preceding public
// player action. Unattributed turns are usable only in a single-character game.
func (d *DB) PerceivedNarration(campaignID, sessionID, characterID int64) ([]string, error) {
	rows, err := d.db.Query(`
		SELECT substr(m.content, -2400) FROM messages m JOIN sessions s ON s.id = m.session_id
		WHERE s.campaign_id = ? AND s.id <= ? AND m.role = 'assistant' AND m.whisper = 0
		AND coalesce(m.character_id,
		  (SELECT u.character_id FROM messages u WHERE u.session_id = m.session_id
		   AND u.role = 'user' AND u.whisper = 0 AND u.id < m.id ORDER BY u.id DESC LIMIT 1),
		  (SELECT min(id) FROM characters WHERE campaign_id = s.campaign_id HAVING count(*) = 1)
		) = ?
		ORDER BY s.id DESC, m.id DESC LIMIT 12`, campaignID, sessionID, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var passages []string
	for rows.Next() {
		var passage string
		if err := rows.Scan(&passage); err != nil {
			return nil, err
		}
		passages = append(passages, passage)
	}
	return passages, rows.Err()
}
