-- Canonical template for contributors adding a ruleset.
-- Copy this file to a new incrementally numbered migration. Do not copy or
-- modify historical migration 038, which remains immutable migration history.
--
-- Schema field categories:
--   attribute: pip dots; track: segmented bar; skill/resource: number input;
--   identity: text/select input; notes: textarea.
-- Add default and options values whenever schema-driven character creation can
-- use them. Add Go ruleset logic only for genuinely complex stat generation.

INSERT OR IGNORE INTO rulesets (name, schema_json, version, gm_context) VALUES
('replace_with_ruleset_name', '[
  {"key":"name","label":"Name","type":"text","category":"identity","default":""},
  {"key":"level","label":"Level","type":"number","category":"resource","min":1,"default":"1"},
  {"key":"strength","label":"Strength","type":"number","category":"attribute","min":1,"max":5,"default":"1"},
  {"key":"health","label":"Health","type":"number","category":"track","min":0,"max":10,"default":"10"},
  {"key":"profession","label":"Profession","type":"text","category":"identity","options":["Warrior","Scholar"]},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes","default":""}
]', '1.0', 'Describe this ruleset and its GM guidance.');
