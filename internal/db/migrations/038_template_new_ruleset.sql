-- 038_template_new_ruleset.sql
-- Template for adding a new ruleset.
-- Copy this file, rename to XXX_my_ruleset.sql, and fill in your values.
--
-- Fields in the schema_json array:
--   key:      Unique field identifier (lowercase_with_underscores)
--   label:    Display label in the UI (Human Readable)
--   type:     'text' | 'number' | 'textarea'
--   category: 'attribute' | 'track' | 'skill' | 'identity' | 'resource' | 'notes'
--     attribute → pip dots (1-5). e.g. STR, DEX, edge, brawn
--     track     → segmented bar. e.g. HP, health, wounds, endurance
--     skill     → number input. e.g. athletics, stealth, persuasion
--     identity  → text input. e.g. race, class, clan, background
--     resource  → number input. e.g. XP, gold, credits, level
--     notes     → textarea. e.g. notes, inventory, spells
--
-- After adding, also update RollStats() in internal/ruleset/random_stats.go
-- if you want auto-generated stats, and CharacterOptions() in
-- internal/ruleset/options.go if you have dropdown choices.

INSERT OR IGNORE INTO rulesets (name, schema_json, version, gm_context) VALUES
('my_ruleset', '[
  {"key":"name","label":"Name","type":"text","category":"identity"},
  {"key":"level","label":"Level","type":"number","category":"resource"},
  {"key":"str","label":"STR","type":"number","category":"attribute"},
  {"key":"dex","label":"DEX","type":"number","category":"attribute"},
  {"key":"hp","label":"HP","type":"number","category":"track"},
  {"key":"athletics","label":"Athletics","type":"number","category":"skill"},
  {"key":"inventory","label":"Inventory","type":"textarea","category":"notes"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]', '1.0', 'You are the GM of a my_ruleset campaign. Describe the world and adjudicate actions.');
