package db

import "database/sql"

type Deck struct {
	ID                int64  `json:"id"`
	CampaignID        int64  `json:"campaign_id"`
	Name              string `json:"name"`
	CardsJSON         string `json:"cards_json"`
	ShuffledOrderJSON string `json:"shuffled_order_json"`
	DrawIndex         int    `json:"draw_index"`
	CreatedAt         string `json:"created_at"`
}

type DeckDraw struct {
	ID        int64  `json:"id"`
	SessionID int64  `json:"session_id"`
	DeckID    int64  `json:"deck_id"`
	CardJSON  string `json:"card_json"`
	DrawnAt   string `json:"drawn_at"`
}

func (d *DB) ListDecks(campaignID int64) ([]Deck, error) {
	rows, err := d.db.Query(
		"SELECT id, campaign_id, name, cards_json, shuffled_order_json, draw_index, created_at FROM decks WHERE campaign_id = ? ORDER BY created_at",
		campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Deck
	for rows.Next() {
		var dk Deck
		if err := rows.Scan(&dk.ID, &dk.CampaignID, &dk.Name, &dk.CardsJSON, &dk.ShuffledOrderJSON, &dk.DrawIndex, &dk.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, dk)
	}
	return out, rows.Err()
}

func (d *DB) GetDeck(id int64) (*Deck, error) {
	var dk Deck
	err := d.db.QueryRow(
		"SELECT id, campaign_id, name, cards_json, shuffled_order_json, draw_index, created_at FROM decks WHERE id = ?",
		id,
	).Scan(&dk.ID, &dk.CampaignID, &dk.Name, &dk.CardsJSON, &dk.ShuffledOrderJSON, &dk.DrawIndex, &dk.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &dk, nil
}

func (d *DB) CreateDeck(campaignID int64, name, cardsJSON string) (int64, error) {
	res, err := d.db.Exec(
		"INSERT INTO decks (campaign_id, name, cards_json) VALUES (?, ?, ?)",
		campaignID, name, cardsJSON,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) DeleteDeck(id int64) error {
	_, err := d.db.Exec("DELETE FROM decks WHERE id = ?", id)
	return err
}

func (d *DB) ShuffleDeck(id int64, shuffledOrderJSON string) error {
	_, err := d.db.Exec(
		"UPDATE decks SET shuffled_order_json = ?, draw_index = 0 WHERE id = ?",
		shuffledOrderJSON, id,
	)
	return err
}

func (d *DB) DrawCard(deckID int64, newDrawIndex int, sessionID int64, cardJSON string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec("UPDATE decks SET draw_index = ? WHERE id = ?", newDrawIndex, deckID); err != nil {
		return err
	}
	if _, err := tx.Exec("INSERT INTO deck_draws (session_id, deck_id, card_json) VALUES (?, ?, ?)", sessionID, deckID, cardJSON); err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) ListDeckDraws(sessionID int64) ([]DeckDraw, error) {
	rows, err := d.db.Query(
		"SELECT id, session_id, deck_id, card_json, drawn_at FROM deck_draws WHERE session_id = ? ORDER BY drawn_at DESC",
		sessionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DeckDraw
	for rows.Next() {
		var dd DeckDraw
		if err := rows.Scan(&dd.ID, &dd.SessionID, &dd.DeckID, &dd.CardJSON, &dd.DrawnAt); err != nil {
			return nil, err
		}
		out = append(out, dd)
	}
	return out, rows.Err()
}
