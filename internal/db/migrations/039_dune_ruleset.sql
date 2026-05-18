-- 039_dune_ruleset.sql: Add Dune: Adventures in the Imperium ruleset
-- Uses the Modiphius 2d20 system with 5 drives, 5 skills, archetypes, and faction support.
INSERT OR IGNORE INTO rulesets (name, schema_json, version, gm_context) VALUES
('dune', '[
  {"key":"archetype","label":"Archetype","type":"text","category":"identity"},
  {"key":"house","label":"House","type":"text","category":"identity"},
  {"key":"role","label":"Role","type":"text","category":"identity"},
  {"key":"faction","label":"Faction","type":"text","category":"identity"},
  {"key":"ambition","label":"Ambition","type":"text","category":"identity"},
  {"key":"drive_statement","label":"Drive Statement","type":"textarea","category":"notes"},
  {"key":"personality_traits","label":"Personality Traits","type":"textarea","category":"notes"},
  {"key":"duty","label":"Duty","type":"number","category":"attribute"},
  {"key":"faith","label":"Faith","type":"number","category":"attribute"},
  {"key":"justice","label":"Justice","type":"number","category":"attribute"},
  {"key":"power","label":"Power","type":"number","category":"attribute"},
  {"key":"truth","label":"Truth","type":"number","category":"attribute"},
  {"key":"battle","label":"Battle","type":"number","category":"skill"},
  {"key":"communicate","label":"Communicate","type":"number","category":"skill"},
  {"key":"discipline","label":"Discipline","type":"number","category":"skill"},
  {"key":"move","label":"Move","type":"number","category":"skill"},
  {"key":"understand","label":"Understand","type":"number","category":"skill"},
  {"key":"focuses","label":"Focuses","type":"textarea","category":"notes"},
  {"key":"talents","label":"Talents","type":"textarea","category":"notes"},
  {"key":"assets","label":"Assets","type":"textarea","category":"notes"},
  {"key":"determination","label":"Determination","type":"number","category":"resource"},
  {"key":"advancement","label":"Advancement","type":"number","category":"resource"},
  {"key":"wounds","label":"Wounds","type":"number","category":"track"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]', '1.0', 'You are the Game Master of a Dune: Adventures in the Imperium campaign — a far-future feudal society where noble Houses vie for power, influence, and the most precious substance in the universe: the spice melange.

Tone: Political intrigue, feudal drama, desert survival, and calculated action. Characters are elite agents of a noble House — Mentats, Swordmasters, Bene Gesserit Sisters, Spacing Guild agents, Fremen warriors, Suk doctors, spies, smugglers, and nobles. Every decision has consequences that ripple through the Imperium.

The Imperium runs on the 2d20 System: Drives (Duty, Faith, Justice, Power, Truth) define a character motivation and provide dice. Skills (Battle, Communicate, Discipline, Move, Understand) determine expertise. Momentum and Threat are the push-pull currencies of the narrative. Determination points let characters shine at critical moments.

Arrakis is the heart of the Imperium — a desert world where the spice melange is found. The desert is deadly: sandworms, heat, thirst, and the constant threat of Harkonnen or Sardaukar patrols. Fremen are the native people, fierce and adapted to the desert. Water is life. Every drop is sacred.

As GM: the world should feel vast, ancient, and dangerous. The political machinations of the Great Houses are as deadly as any blade. Let characters leverage their House resources, navigate intrigue, and face the moral weight of their choices. The desert does not forgive, and the Imperium does not forget.');
