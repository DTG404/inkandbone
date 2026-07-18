ALTER TABLE campaigns ADD COLUMN content_boundaries TEXT NOT NULL DEFAULT '';
ALTER TABLE campaigns ADD COLUMN narrative_locale TEXT NOT NULL DEFAULT 'en';
