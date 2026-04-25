-- 035_vtm_chronicle_dow.sql: Track the in-story day-of-week for Night 1 in VtM campaigns.
-- chronicle_night_start_dow is 0-6 (Sunday=0 … Saturday=6) matching JS Date.getDay().
-- -1 means undetected (waiting for the first GM scene to name a day).
ALTER TABLE campaigns ADD COLUMN chronicle_night_start_dow INTEGER NOT NULL DEFAULT -1;
