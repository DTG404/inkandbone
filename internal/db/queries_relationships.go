package db

import (
	"database/sql"
	"time"
)

// Relationship represents a named relationship between two characters/NPCs in a campaign.
type Relationship struct {
	ID               int64     `json:"id"`
	CampaignID       int64     `json:"campaign_id"`
	FromName         string    `json:"from_name"`
	ToName           string    `json:"to_name"`
	RelationshipType string    `json:"relationship_type"`
	Description      string    `json:"description"`
	CreatedAt        time.Time `json:"created_at"`
}

func (db *DB) GetRelationship(id int64) (*Relationship, error) {
	var relationship Relationship
	err := db.db.QueryRow(
		`SELECT id, campaign_id, from_name, to_name, relationship_type, description, created_at
		 FROM relationships WHERE id = ?`, id,
	).Scan(&relationship.ID, &relationship.CampaignID, &relationship.FromName, &relationship.ToName, &relationship.RelationshipType, &relationship.Description, &relationship.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &relationship, nil
}

// CreateRelationship inserts a new relationship and returns its ID.
func (db *DB) CreateRelationship(campaignID int64, fromName, toName, relType, description string) (int64, error) {
	res, err := db.db.Exec(
		`INSERT INTO relationships (campaign_id, from_name, to_name, relationship_type, description)
		 VALUES (?, ?, ?, ?, ?)`,
		campaignID, fromName, toName, relType, description,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListRelationships returns all relationships for a campaign.
func (db *DB) ListRelationships(campaignID int64) ([]Relationship, error) {
	rows, err := db.db.Query(
		`SELECT id, campaign_id, from_name, to_name, relationship_type, description, created_at
		 FROM relationships WHERE campaign_id = ? ORDER BY created_at ASC`,
		campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []Relationship
	for rows.Next() {
		var r Relationship
		if err := rows.Scan(&r.ID, &r.CampaignID, &r.FromName, &r.ToName, &r.RelationshipType, &r.Description, &r.CreatedAt); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

// UpdateRelationship changes the type and description of an existing relationship.
func (db *DB) UpdateRelationship(id int64, relType, description string) error {
	_, err := db.db.Exec(
		`UPDATE relationships SET relationship_type = ?, description = ? WHERE id = ?`,
		relType, description, id,
	)
	return err
}

// DeleteRelationship removes a relationship by ID.
func (db *DB) DeleteRelationship(id int64) error {
	_, err := db.db.Exec(`DELETE FROM relationships WHERE id = ?`, id)
	return err
}
