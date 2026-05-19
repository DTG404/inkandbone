package db

import (
	"database/sql"
	"time"
)

type Secret struct {
	ID                  int64     `json:"id"`
	CampaignID          int64     `json:"campaign_id"`
	Title               string    `json:"title"`
	Content             string    `json:"content"`
	Category            string    `json:"category"`
	Revealed            bool      `json:"revealed"`
	RevealedAtSessionID *int64    `json:"revealed_at_session_id,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
}

func (db *DB) CreateSecret(campaignID int64, title, content, category string) (int64, error) {
	res, err := db.db.Exec(
		`INSERT INTO secrets (campaign_id, title, content, category)
		 VALUES (?, ?, ?, ?)`,
		campaignID, title, content, category,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (db *DB) GetSecret(id int64) (*Secret, error) {
	row := db.db.QueryRow(
		`SELECT id, campaign_id, title, content, category, revealed, revealed_at_session_id, created_at
		 FROM secrets WHERE id = ?`, id,
	)
	var s Secret
	var revealedInt int
	var revealedAtSession sql.NullInt64
	if err := row.Scan(&s.ID, &s.CampaignID, &s.Title, &s.Content, &s.Category, &revealedInt, &revealedAtSession, &s.CreatedAt); err != nil {
		return nil, err
	}
	s.Revealed = revealedInt != 0
	if revealedAtSession.Valid {
		s.RevealedAtSessionID = &revealedAtSession.Int64
	}
	return &s, nil
}

func (db *DB) ListSecretsByCampaign(campaignID int64) ([]Secret, error) {
	rows, err := db.db.Query(
		`SELECT id, campaign_id, title, content, category, revealed, revealed_at_session_id, created_at
		 FROM secrets WHERE campaign_id = ? ORDER BY created_at DESC`,
		campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var secrets []Secret
	for rows.Next() {
		var s Secret
		var revealedInt int
		var revealedAtSession sql.NullInt64
		if err := rows.Scan(&s.ID, &s.CampaignID, &s.Title, &s.Content, &s.Category, &revealedInt, &revealedAtSession, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.Revealed = revealedInt != 0
		if revealedAtSession.Valid {
			s.RevealedAtSessionID = &revealedAtSession.Int64
		}
		secrets = append(secrets, s)
	}
	return secrets, rows.Err()
}

func (db *DB) RevealSecret(id int64, sessionID int64) error {
	_, err := db.db.Exec(
		`UPDATE secrets SET revealed = 1, revealed_at_session_id = ? WHERE id = ?`,
		sessionID, id,
	)
	return err
}

func (db *DB) UpdateSecret(id int64, title, content, category string) error {
	_, err := db.db.Exec(
		`UPDATE secrets SET title = ?, content = ?, category = ? WHERE id = ?`,
		title, content, category, id,
	)
	return err
}

func (db *DB) DeleteSecret(id int64) error {
	_, err := db.db.Exec(`DELETE FROM secrets WHERE id = ?`, id)
	return err
}
