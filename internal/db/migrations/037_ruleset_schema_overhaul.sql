-- 037_ruleset_schema_overhaul.sql
-- DIG-68: Convert all 5 legacy schemas to structured format
-- DIG-70: Add category field definitions across all 13 rulesets
-- DIG-76: Stabilize ruleset IDs (explicit IDs in seed)
--
-- The `category` field controls rendering:
--   attribute → pip dots (1-5)
--   track     → segmented bar (health-like)
--   skill     → number input in grid
--   identity  → text input
--   resource  → number input (XP, gold, etc.)
--   notes     → textarea

-- 1. dnd5e: legacy → structured with categories
UPDATE rulesets SET schema_json = '[
  {"key":"race","label":"Race","type":"text","category":"identity"},
  {"key":"class","label":"Class","type":"text","category":"identity"},
  {"key":"level","label":"Level","type":"number","category":"resource"},
  {"key":"hp","label":"HP","type":"number","category":"track"},
  {"key":"ac","label":"AC","type":"number","category":"resource"},
  {"key":"str","label":"STR","type":"number","category":"attribute"},
  {"key":"dex","label":"DEX","type":"number","category":"attribute"},
  {"key":"con","label":"CON","type":"number","category":"attribute"},
  {"key":"int","label":"INT","type":"number","category":"attribute"},
  {"key":"wis","label":"WIS","type":"number","category":"attribute"},
  {"key":"cha","label":"CHA","type":"number","category":"attribute"},
  {"key":"proficiency_bonus","label":"Proficiency Bonus","type":"number","category":"resource"},
  {"key":"skills","label":"Skills","type":"textarea","category":"notes"},
  {"key":"inventory","label":"Inventory","type":"textarea","category":"notes"},
  {"key":"spells","label":"Spells","type":"textarea","category":"notes"},
  {"key":"features","label":"Features","type":"textarea","category":"notes"}
]' WHERE name = 'dnd5e';

-- 2. ironsworn: legacy → structured with categories
UPDATE rulesets SET schema_json = '[
  {"key":"edge","label":"Edge","type":"number","category":"attribute"},
  {"key":"heart","label":"Heart","type":"number","category":"attribute"},
  {"key":"iron","label":"Iron","type":"number","category":"attribute"},
  {"key":"shadow","label":"Shadow","type":"number","category":"attribute"},
  {"key":"wits","label":"Wits","type":"number","category":"attribute"},
  {"key":"health","label":"Health","type":"number","category":"track"},
  {"key":"spirit","label":"Spirit","type":"number","category":"track"},
  {"key":"supply","label":"Supply","type":"number","category":"track"},
  {"key":"momentum","label":"Momentum","type":"number","category":"track"},
  {"key":"vows","label":"Vows","type":"textarea","category":"notes"},
  {"key":"bonds","label":"Bonds","type":"textarea","category":"notes"},
  {"key":"assets","label":"Assets","type":"textarea","category":"notes"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]' WHERE name = 'ironsworn';

-- 3. vtm: legacy → structured with categories (V5)
-- Note: VtMCharacterSheet component in the frontend currently bypasses
-- the schema, but the structured format is needed for future data-driven rendering.
UPDATE rulesets SET schema_json = '[
  {"key":"character_type","label":"Character Type","type":"text","category":"identity"},
  {"key":"clan","label":"Clan","type":"text","category":"identity"},
  {"key":"predator_type","label":"Predator Type","type":"text","category":"identity"},
  {"key":"sect","label":"Sect","type":"text","category":"identity"},
  {"key":"generation","label":"Generation","type":"text","category":"identity"},
  {"key":"hunger","label":"Hunger","type":"number","category":"resource"},
  {"key":"blood_potency","label":"Blood Potency","type":"number","category":"resource"},
  {"key":"bane_severity","label":"Bane Severity","type":"number","category":"resource"},
  {"key":"humanity","label":"Humanity","type":"number","category":"resource"},
  {"key":"stains","label":"Stains","type":"number","category":"resource"},
  {"key":"strength","label":"Strength","type":"number","category":"attribute"},
  {"key":"dexterity","label":"Dexterity","type":"number","category":"attribute"},
  {"key":"stamina","label":"Stamina","type":"number","category":"attribute"},
  {"key":"charisma","label":"Charisma","type":"number","category":"attribute"},
  {"key":"manipulation","label":"Manipulation","type":"number","category":"attribute"},
  {"key":"composure","label":"Composure","type":"number","category":"attribute"},
  {"key":"intelligence","label":"Intelligence","type":"number","category":"attribute"},
  {"key":"wits","label":"Wits","type":"number","category":"attribute"},
  {"key":"resolve","label":"Resolve","type":"number","category":"attribute"},
  {"key":"athletics","label":"Athletics","type":"number","category":"skill"},
  {"key":"brawl","label":"Brawl","type":"number","category":"skill"},
  {"key":"craft","label":"Craft","type":"number","category":"skill"},
  {"key":"drive","label":"Drive","type":"number","category":"skill"},
  {"key":"firearms","label":"Firearms","type":"number","category":"skill"},
  {"key":"larceny","label":"Larceny","type":"number","category":"skill"},
  {"key":"melee","label":"Melee","type":"number","category":"skill"},
  {"key":"stealth","label":"Stealth","type":"number","category":"skill"},
  {"key":"survival","label":"Survival","type":"number","category":"skill"},
  {"key":"animal_ken","label":"Animal Ken","type":"number","category":"skill"},
  {"key":"etiquette","label":"Etiquette","type":"number","category":"skill"},
  {"key":"insight","label":"Insight","type":"number","category":"skill"},
  {"key":"intimidation","label":"Intimidation","type":"number","category":"skill"},
  {"key":"leadership","label":"Leadership","type":"number","category":"skill"},
  {"key":"performance","label":"Performance","type":"number","category":"skill"},
  {"key":"persuasion","label":"Persuasion","type":"number","category":"skill"},
  {"key":"streetwise","label":"Streetwise","type":"number","category":"skill"},
  {"key":"subterfuge","label":"Subterfuge","type":"number","category":"skill"},
  {"key":"academics","label":"Academics","type":"number","category":"skill"},
  {"key":"awareness","label":"Awareness","type":"number","category":"skill"},
  {"key":"finance","label":"Finance","type":"number","category":"skill"},
  {"key":"investigation","label":"Investigation","type":"number","category":"skill"},
  {"key":"medicine","label":"Medicine","type":"number","category":"skill"},
  {"key":"occult","label":"Occult","type":"number","category":"skill"},
  {"key":"politics","label":"Politics","type":"number","category":"skill"},
  {"key":"technology","label":"Technology","type":"number","category":"skill"},
  {"key":"animalism","label":"Animalism","type":"number","category":"skill"},
  {"key":"auspex","label":"Auspex","type":"number","category":"skill"},
  {"key":"blood_sorcery","label":"Blood Sorcery","type":"number","category":"skill"},
  {"key":"celerity","label":"Celerity","type":"number","category":"skill"},
  {"key":"dominate","label":"Dominate","type":"number","category":"skill"},
  {"key":"fortitude","label":"Fortitude","type":"number","category":"skill"},
  {"key":"obfuscate","label":"Obfuscate","type":"number","category":"skill"},
  {"key":"oblivion","label":"Oblivion","type":"number","category":"skill"},
  {"key":"potence","label":"Potence","type":"number","category":"skill"},
  {"key":"presence","label":"Presence","type":"number","category":"skill"},
  {"key":"protean","label":"Protean","type":"number","category":"skill"},
  {"key":"thin_blood_alchemy","label":"Thin-Blood Alchemy","type":"number","category":"skill"},
  {"key":"health_max","label":"Health Max","type":"number","category":"resource"},
  {"key":"health_superficial","label":"Health Superficial","type":"number","category":"resource"},
  {"key":"health_aggravated","label":"Health Aggravated","type":"number","category":"resource"},
  {"key":"willpower_max","label":"Willpower Max","type":"number","category":"resource"},
  {"key":"willpower_superficial","label":"Willpower Superficial","type":"number","category":"resource"},
  {"key":"willpower_aggravated","label":"Willpower Aggravated","type":"number","category":"resource"},
  {"key":"domitor","label":"Domitor","type":"text","category":"identity"},
  {"key":"bond_strength","label":"Bond Strength","type":"number","category":"resource"},
  {"key":"skill_specialties","label":"Skill Specialties","type":"textarea","category":"notes"},
  {"key":"merits_flaws","label":"Merits & Flaws","type":"textarea","category":"notes"},
  {"key":"convictions","label":"Convictions","type":"textarea","category":"notes"},
  {"key":"touchstones","label":"Touchstones","type":"textarea","category":"notes"},
  {"key":"ambition","label":"Ambition","type":"text","category":"identity"},
  {"key":"desire","label":"Desire","type":"text","category":"identity"},
  {"key":"rituals","label":"Rituals","type":"textarea","category":"notes"},
  {"key":"ceremonies","label":"Ceremonies","type":"textarea","category":"notes"},
  {"key":"known_formulae","label":"Known Formulae","type":"textarea","category":"notes"},
  {"key":"loresheets","label":"Loresheets","type":"textarea","category":"notes"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"},
  {"key":"xp","label":"XP","type":"number","category":"resource"}
]' WHERE name = 'vtm';

-- 4. coc: legacy → structured with categories
UPDATE rulesets SET schema_json = '[
  {"key":"occupation","label":"Occupation","type":"text","category":"identity"},
  {"key":"age","label":"Age","type":"number","category":"resource"},
  {"key":"hp","label":"HP","type":"number","category":"track"},
  {"key":"sanity","label":"Sanity","type":"number","category":"track"},
  {"key":"luck","label":"Luck","type":"number","category":"resource"},
  {"key":"mp","label":"MP","type":"number","category":"resource"},
  {"key":"str","label":"STR","type":"number","category":"attribute"},
  {"key":"con","label":"CON","type":"number","category":"attribute"},
  {"key":"siz","label":"SIZ","type":"number","category":"attribute"},
  {"key":"dex","label":"DEX","type":"number","category":"attribute"},
  {"key":"app","label":"APP","type":"number","category":"attribute"},
  {"key":"int","label":"INT","type":"number","category":"attribute"},
  {"key":"pow","label":"POW","type":"number","category":"attribute"},
  {"key":"edu","label":"EDU","type":"number","category":"attribute"},
  {"key":"skills","label":"Skills","type":"textarea","category":"notes"},
  {"key":"inventory","label":"Inventory","type":"textarea","category":"notes"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]' WHERE name = 'coc';

-- 5. cyberpunk: legacy → structured with categories
UPDATE rulesets SET schema_json = '[
  {"key":"role","label":"Role","type":"text","category":"identity"},
  {"key":"int","label":"INT","type":"number","category":"attribute"},
  {"key":"ref","label":"REF","type":"number","category":"attribute"},
  {"key":"cool","label":"COOL","type":"number","category":"attribute"},
  {"key":"tech","label":"TECH","type":"number","category":"attribute"},
  {"key":"lk","label":"LK","type":"number","category":"attribute"},
  {"key":"att","label":"ATT","type":"number","category":"attribute"},
  {"key":"ma","label":"MA","type":"number","category":"attribute"},
  {"key":"emp","label":"EMP","type":"number","category":"attribute"},
  {"key":"body","label":"BODY","type":"number","category":"attribute"},
  {"key":"humanity","label":"Humanity","type":"number","category":"resource"},
  {"key":"eurodollars","label":"Eurodollars","type":"number","category":"resource"},
  {"key":"skills","label":"Skills","type":"textarea","category":"notes"},
  {"key":"cyberware","label":"Cyberware","type":"textarea","category":"notes"},
  {"key":"gear","label":"Gear","type":"textarea","category":"notes"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]' WHERE name = 'cyberpunk';

-- 6. shadowrun: add categories
UPDATE rulesets SET schema_json = '[
  {"key":"metatype","label":"Metatype","type":"text","category":"identity"},
  {"key":"priority","label":"Priority","type":"text","category":"identity"},
  {"key":"body","label":"Body","type":"number","category":"attribute"},
  {"key":"agility","label":"Agility","type":"number","category":"attribute"},
  {"key":"reaction","label":"Reaction","type":"number","category":"attribute"},
  {"key":"strength","label":"Strength","type":"number","category":"attribute"},
  {"key":"willpower","label":"Willpower","type":"number","category":"attribute"},
  {"key":"logic","label":"Logic","type":"number","category":"attribute"},
  {"key":"intuition","label":"Intuition","type":"number","category":"attribute"},
  {"key":"charisma","label":"Charisma","type":"number","category":"attribute"},
  {"key":"edge","label":"Edge","type":"number","category":"resource"},
  {"key":"essence","label":"Essence","type":"number","category":"resource"},
  {"key":"physical_limit","label":"Phys Limit","type":"number","category":"resource"},
  {"key":"mental_limit","label":"Mental Limit","type":"number","category":"resource"},
  {"key":"social_limit","label":"Social Limit","type":"number","category":"resource"},
  {"key":"nuyen","label":"Nuyen","type":"number","category":"resource"},
  {"key":"karma","label":"Karma","type":"number","category":"resource"},
  {"key":"reputation","label":"Reputation","type":"number","category":"resource"},
  {"key":"notoriety","label":"Notoriety","type":"number","category":"resource"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]' WHERE name = 'shadowrun';

-- 7. wfrp: add categories
UPDATE rulesets SET schema_json = '[
  {"key":"species","label":"Species","type":"text","category":"identity"},
  {"key":"career","label":"Career","type":"text","category":"identity"},
  {"key":"career_level","label":"Career Level","type":"number","category":"resource"},
  {"key":"ws","label":"WS","type":"number","category":"attribute"},
  {"key":"bs","label":"BS","type":"number","category":"attribute"},
  {"key":"s","label":"S","type":"number","category":"attribute"},
  {"key":"t","label":"T","type":"number","category":"attribute"},
  {"key":"ag","label":"Ag","type":"number","category":"attribute"},
  {"key":"i","label":"I","type":"number","category":"attribute"},
  {"key":"dex","label":"Dex","type":"number","category":"attribute"},
  {"key":"int","label":"Int","type":"number","category":"attribute"},
  {"key":"wp","label":"WP","type":"number","category":"attribute"},
  {"key":"fel","label":"Fel","type":"number","category":"attribute"},
  {"key":"wounds","label":"Wounds","type":"number","category":"track"},
  {"key":"fate","label":"Fate","type":"number","category":"resource"},
  {"key":"fortune","label":"Fortune","type":"number","category":"resource"},
  {"key":"resilience","label":"Resilience","type":"number","category":"resource"},
  {"key":"resolve","label":"Resolve","type":"number","category":"resource"},
  {"key":"xp","label":"XP","type":"number","category":"resource"},
  {"key":"ambitions","label":"Ambitions","type":"textarea","category":"notes"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]' WHERE name = 'wfrp';

-- 8. starwars: add categories
UPDATE rulesets SET schema_json = '[
  {"key":"species","label":"Species","type":"text","category":"identity"},
  {"key":"career","label":"Career","type":"text","category":"identity"},
  {"key":"specialization","label":"Specialization","type":"text","category":"identity"},
  {"key":"brawn","label":"Brawn","type":"number","category":"attribute"},
  {"key":"agility","label":"Agility","type":"number","category":"attribute"},
  {"key":"intellect","label":"Intellect","type":"number","category":"attribute"},
  {"key":"cunning","label":"Cunning","type":"number","category":"attribute"},
  {"key":"willpower","label":"Willpower","type":"number","category":"attribute"},
  {"key":"presence","label":"Presence","type":"number","category":"attribute"},
  {"key":"wounds_threshold","label":"Wounds Max","type":"number","category":"track"},
  {"key":"wounds_current","label":"Wounds","type":"number","category":"track"},
  {"key":"strain_threshold","label":"Strain Max","type":"number","category":"track"},
  {"key":"strain_current","label":"Strain","type":"number","category":"track"},
  {"key":"soak","label":"Soak","type":"number","category":"resource"},
  {"key":"defense_melee","label":"Def Melee","type":"number","category":"resource"},
  {"key":"defense_ranged","label":"Def Ranged","type":"number","category":"resource"},
  {"key":"obligation","label":"Obligation","type":"number","category":"resource"},
  {"key":"credits","label":"Credits","type":"number","category":"resource"},
  {"key":"force_rating","label":"Force Rating","type":"number","category":"resource"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]' WHERE name = 'starwars';

-- 9. l5r: add categories
UPDATE rulesets SET schema_json = '[
  {"key":"clan","label":"Clan","type":"text","category":"identity"},
  {"key":"family","label":"Family","type":"text","category":"identity"},
  {"key":"school","label":"School","type":"text","category":"identity"},
  {"key":"school_rank","label":"School Rank","type":"number","category":"resource"},
  {"key":"air","label":"Air","type":"number","category":"attribute"},
  {"key":"earth","label":"Earth","type":"number","category":"attribute"},
  {"key":"fire","label":"Fire","type":"number","category":"attribute"},
  {"key":"water","label":"Water","type":"number","category":"attribute"},
  {"key":"void","label":"Void","type":"number","category":"attribute"},
  {"key":"endurance","label":"Endurance","type":"number","category":"track"},
  {"key":"composure","label":"Composure","type":"number","category":"track"},
  {"key":"focus","label":"Focus","type":"number","category":"resource"},
  {"key":"vigilance","label":"Vigilance","type":"number","category":"resource"},
  {"key":"glory","label":"Glory","type":"number","category":"resource"},
  {"key":"honor","label":"Honor","type":"number","category":"resource"},
  {"key":"status","label":"Status","type":"number","category":"resource"},
  {"key":"xp","label":"XP","type":"number","category":"resource"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]' WHERE name = 'l5r';

-- 10. theonering: add categories
UPDATE rulesets SET schema_json = '[
  {"key":"culture","label":"Culture","type":"text","category":"identity"},
  {"key":"calling","label":"Calling","type":"text","category":"identity"},
  {"key":"body","label":"Body","type":"number","category":"attribute"},
  {"key":"heart","label":"Heart","type":"number","category":"attribute"},
  {"key":"wits","label":"Wits","type":"number","category":"attribute"},
  {"key":"endurance","label":"Endurance","type":"number","category":"track"},
  {"key":"endurance_max","label":"Endurance Max","type":"number","category":"track"},
  {"key":"hope","label":"Hope","type":"number","category":"track"},
  {"key":"hope_max","label":"Hope Max","type":"number","category":"track"},
  {"key":"shadow_points","label":"Shadow Pts","type":"number","category":"resource"},
  {"key":"shadow_scars","label":"Shadow Scars","type":"number","category":"resource"},
  {"key":"valour","label":"Valour","type":"number","category":"resource"},
  {"key":"wisdom","label":"Wisdom","type":"number","category":"resource"},
  {"key":"standing","label":"Standing","type":"number","category":"resource"},
  {"key":"fellowship_score","label":"Fellowship","type":"number","category":"resource"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]' WHERE name = 'theonering';

-- 11. wrath_glory: add categories
UPDATE rulesets SET schema_json = '[
  {"key":"archetype","label":"Archetype","type":"text","category":"identity"},
  {"key":"faction","label":"Faction","type":"text","category":"identity"},
  {"key":"rank","label":"Rank","type":"number","category":"resource"},
  {"key":"keywords","label":"Keywords","type":"text","category":"identity"},
  {"key":"strength","label":"Strength","type":"number","category":"attribute"},
  {"key":"agility","label":"Agility","type":"number","category":"attribute"},
  {"key":"toughness","label":"Toughness","type":"number","category":"attribute"},
  {"key":"intellect","label":"Intellect","type":"number","category":"attribute"},
  {"key":"willpower","label":"Willpower","type":"number","category":"attribute"},
  {"key":"fellowship","label":"Fellowship","type":"number","category":"attribute"},
  {"key":"initiative","label":"Initiative","type":"number","category":"attribute"},
  {"key":"ws","label":"Weapon Skill","type":"number","category":"skill"},
  {"key":"bs","label":"Ballistic Skill","type":"number","category":"skill"},
  {"key":"athletics","label":"Athletics","type":"number","category":"skill"},
  {"key":"awareness","label":"Awareness","type":"number","category":"skill"},
  {"key":"cunning","label":"Cunning","type":"number","category":"skill"},
  {"key":"deception","label":"Deception","type":"number","category":"skill"},
  {"key":"fortitude","label":"Fortitude","type":"number","category":"skill"},
  {"key":"insight","label":"Insight","type":"number","category":"skill"},
  {"key":"intimidation","label":"Intimidation","type":"number","category":"skill"},
  {"key":"investigation","label":"Investigation","type":"number","category":"skill"},
  {"key":"leadership","label":"Leadership","type":"number","category":"skill"},
  {"key":"medicae","label":"Medicae","type":"number","category":"skill"},
  {"key":"persuasion","label":"Persuasion","type":"number","category":"skill"},
  {"key":"pilot","label":"Pilot","type":"number","category":"skill"},
  {"key":"psychic_mastery","label":"Psychic Mastery","type":"number","category":"skill"},
  {"key":"scholar","label":"Scholar","type":"number","category":"skill"},
  {"key":"stealth","label":"Stealth","type":"number","category":"skill"},
  {"key":"survival","label":"Survival","type":"number","category":"skill"},
  {"key":"tech","label":"Tech","type":"number","category":"skill"},
  {"key":"speed","label":"Speed","type":"number","category":"resource"},
  {"key":"defence","label":"Defence","type":"number","category":"resource"},
  {"key":"resilience","label":"Resilience","type":"number","category":"resource"},
  {"key":"determination","label":"Determination","type":"number","category":"resource"},
  {"key":"resolve","label":"Resolve","type":"number","category":"resource"},
  {"key":"conviction","label":"Conviction","type":"number","category":"resource"},
  {"key":"influence","label":"Influence","type":"number","category":"resource"},
  {"key":"wounds","label":"Wounds","type":"number","category":"track"},
  {"key":"shock","label":"Shock","type":"number","category":"track"},
  {"key":"corruption","label":"Corruption","type":"number","category":"resource"},
  {"key":"wrath","label":"Wrath","type":"number","category":"resource"},
  {"key":"glory","label":"Glory","type":"number","category":"resource"},
  {"key":"ruin","label":"Ruin","type":"number","category":"resource"},
  {"key":"wealth","label":"Wealth Tier","type":"number","category":"resource"},
  {"key":"xp","label":"XP","type":"number","category":"resource"},
  {"key":"talents","label":"Talents","type":"textarea","category":"notes"},
  {"key":"powers","label":"Psychic Powers","type":"textarea","category":"notes"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]' WHERE name = 'wrath_glory';

-- 12. blades: add categories
UPDATE rulesets SET schema_json = '[
  {"key":"playbook","label":"Playbook","type":"text","category":"identity"},
  {"key":"heritage","label":"Heritage","type":"text","category":"identity"},
  {"key":"background","label":"Background","type":"text","category":"identity"},
  {"key":"vice","label":"Vice","type":"text","category":"identity"},
  {"key":"hunt","label":"Hunt","type":"number","category":"skill"},
  {"key":"study","label":"Study","type":"number","category":"skill"},
  {"key":"survey","label":"Survey","type":"number","category":"skill"},
  {"key":"tinker","label":"Tinker","type":"number","category":"skill"},
  {"key":"finesse","label":"Finesse","type":"number","category":"skill"},
  {"key":"prowl","label":"Prowl","type":"number","category":"skill"},
  {"key":"skirmish","label":"Skirmish","type":"number","category":"skill"},
  {"key":"wreck","label":"Wreck","type":"number","category":"skill"},
  {"key":"attune","label":"Attune","type":"number","category":"skill"},
  {"key":"command","label":"Command","type":"number","category":"skill"},
  {"key":"consort","label":"Consort","type":"number","category":"skill"},
  {"key":"sway","label":"Sway","type":"number","category":"skill"},
  {"key":"stress","label":"Stress","type":"number","category":"track"},
  {"key":"trauma","label":"Trauma","type":"number","category":"track"},
  {"key":"coin","label":"Coin","type":"number","category":"resource"},
  {"key":"stash","label":"Stash","type":"number","category":"resource"},
  {"key":"load","label":"Load","type":"number","category":"resource"},
  {"key":"xp_insight","label":"XP Insight","type":"number","category":"resource"},
  {"key":"xp_prowess","label":"XP Prowess","type":"number","category":"resource"},
  {"key":"xp_resolve","label":"XP Resolve","type":"number","category":"resource"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]' WHERE name = 'blades';

-- 13. paranoia: add categories
UPDATE rulesets SET schema_json = '[
  {"key":"full_name","label":"Full Name","type":"text","category":"identity"},
  {"key":"sector","label":"Sector","type":"text","category":"identity"},
  {"key":"security_clearance","label":"Clearance","type":"text","category":"identity"},
  {"key":"management_style","label":"Mgmt Style","type":"text","category":"identity"},
  {"key":"power_group","label":"Power Group","type":"text","category":"identity"},
  {"key":"secret_society","label":"Secret Society","type":"text","category":"identity"},
  {"key":"violence","label":"Violence","type":"number","category":"attribute"},
  {"key":"treachery","label":"Treachery","type":"number","category":"attribute"},
  {"key":"happiness","label":"Happiness","type":"number","category":"attribute"},
  {"key":"straight","label":"Straight","type":"number","category":"attribute"},
  {"key":"moxie","label":"Moxie","type":"number","category":"attribute"},
  {"key":"credits","label":"Credits","type":"number","category":"resource"},
  {"key":"clone_number","label":"Clone #","type":"number","category":"resource"},
  {"key":"treason_points","label":"Treason Pts","type":"number","category":"resource"},
  {"key":"notes","label":"Notes","type":"textarea","category":"notes"}
]' WHERE name = 'paranoia';
