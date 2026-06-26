CREATE TABLE decks (
  id                   INTEGER PRIMARY KEY AUTOINCREMENT,
  campaign_id          INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  name                 TEXT NOT NULL,
  cards_json           TEXT NOT NULL DEFAULT '[]',
  shuffled_order_json  TEXT NOT NULL DEFAULT '[]',
  draw_index           INTEGER NOT NULL DEFAULT 0,
  created_at           DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE deck_draws (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  deck_id    INTEGER NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
  card_json  TEXT NOT NULL,
  drawn_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
