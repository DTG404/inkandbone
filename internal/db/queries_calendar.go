package db

import "database/sql"

type CalendarEvent struct {
	ID          int64  `json:"id"`
	CampaignID  int64  `json:"campaign_id"`
	InGameYear  int    `json:"in_game_year"`
	InGameMonth int    `json:"in_game_month"`
	InGameDay   int    `json:"in_game_day"`
	Title       string `json:"title"`
	Description string `json:"description"`
	EventType   string `json:"event_type"`
	SessionID   *int64 `json:"session_id"`
	CreatedAt   string `json:"created_at"`
}

// CampaignCalendarInfo holds the calendar state for a campaign.
type CampaignCalendarInfo struct {
	InGameYear    int    `json:"in_game_year"`
	InGameMonth   int    `json:"in_game_month"`
	InGameDay     int    `json:"in_game_day"`
	CalendarConfig string `json:"calendar_config"`
}

// GetCampaignDate returns the current in-game date and calendar config for a campaign.
func (db *DB) GetCampaignDate(campaignID int64) (*CampaignCalendarInfo, error) {
	row := db.db.QueryRow(
		`SELECT in_game_year, in_game_month, in_game_day, calendar_config
		 FROM campaigns WHERE id = ?`, campaignID,
	)
	var info CampaignCalendarInfo
	if err := row.Scan(&info.InGameYear, &info.InGameMonth, &info.InGameDay, &info.CalendarConfig); err != nil {
		return nil, err
	}
	return &info, nil
}

// UpdateCampaignDate sets the in-game date and optional calendar config for a campaign.
func (db *DB) UpdateCampaignDate(campaignID int64, year, month, day int, calendarConfig *string) error {
	if calendarConfig != nil {
		_, err := db.db.Exec(
			`UPDATE campaigns SET in_game_year = ?, in_game_month = ?, in_game_day = ?, calendar_config = ? WHERE id = ?`,
			year, month, day, *calendarConfig, campaignID,
		)
		return err
	}
	_, err := db.db.Exec(
		`UPDATE campaigns SET in_game_year = ?, in_game_month = ?, in_game_day = ? WHERE id = ?`,
		year, month, day, campaignID,
	)
	return err
}

// CreateCalendarEvent inserts a new calendar event.
func (db *DB) CreateCalendarEvent(campaignID int64, year, month, day int, title, description, eventType string, sessionID *int64) (int64, error) {
	res, err := db.db.Exec(
		`INSERT INTO calendar_events (campaign_id, in_game_year, in_game_month, in_game_day, title, description, event_type, session_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		campaignID, year, month, day, title, description, eventType, sessionID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetCalendarEvent returns a single calendar event by ID.
func (db *DB) GetCalendarEvent(id int64) (*CalendarEvent, error) {
	row := db.db.QueryRow(
		`SELECT id, campaign_id, in_game_year, in_game_month, in_game_day, title, description, event_type, session_id, created_at
		 FROM calendar_events WHERE id = ?`, id,
	)
	var e CalendarEvent
	var sessionID sql.NullInt64
	if err := row.Scan(&e.ID, &e.CampaignID, &e.InGameYear, &e.InGameMonth, &e.InGameDay, &e.Title, &e.Description, &e.EventType, &sessionID, &e.CreatedAt); err != nil {
		return nil, err
	}
	if sessionID.Valid {
		e.SessionID = &sessionID.Int64
	}
	return &e, nil
}

// ListCalendarEvents returns all calendar events for a campaign, ordered by date.
func (db *DB) ListCalendarEvents(campaignID int64) ([]CalendarEvent, error) {
	rows, err := db.db.Query(
		`SELECT id, campaign_id, in_game_year, in_game_month, in_game_day, title, description, event_type, session_id, created_at
		 FROM calendar_events WHERE campaign_id = ? ORDER BY in_game_year, in_game_month, in_game_day, created_at DESC`,
		campaignID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var e CalendarEvent
		var sessionID sql.NullInt64
		if err := rows.Scan(&e.ID, &e.CampaignID, &e.InGameYear, &e.InGameMonth, &e.InGameDay, &e.Title, &e.Description, &e.EventType, &sessionID, &e.CreatedAt); err != nil {
			return nil, err
		}
		if sessionID.Valid {
			e.SessionID = &sessionID.Int64
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// DeleteCalendarEvent removes a calendar event by ID.
func (db *DB) DeleteCalendarEvent(id int64) error {
	_, err := db.db.Exec(`DELETE FROM calendar_events WHERE id = ?`, id)
	return err
}
