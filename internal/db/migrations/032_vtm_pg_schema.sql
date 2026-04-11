-- 032_vtm_pg_schema.sql: Extend VtM V5 schema with Player's Guide fields.
-- Adds character_type, thin-blood, ghoul, and extended text fields.
UPDATE rulesets SET
  schema_json = '{"system":"vtm","fields":[
    "character_type",
    "clan","predator_type","sect","generation",
    "hunger","blood_potency","bane_severity","humanity","stains",
    "strength","dexterity","stamina",
    "charisma","manipulation","composure",
    "intelligence","wits","resolve",
    "athletics","brawl","craft","drive","firearms","larceny","melee","stealth","survival",
    "animal_ken","etiquette","insight","intimidation","leadership","performance","persuasion","streetwise","subterfuge",
    "academics","awareness","finance","investigation","medicine","occult","politics","technology",
    "animalism","auspex","blood_sorcery","celerity","dominate","fortitude","obfuscate","oblivion","potence","presence","protean",
    "thin_blood_alchemy",
    "health_max","health_superficial","health_aggravated",
    "willpower_max","willpower_superficial","willpower_aggravated",
    "domitor","bond_strength",
    "skill_specialties","merits_flaws","convictions","touchstones","ambition","desire",
    "rituals","ceremonies","known_formulae","loresheets","notes","xp"
  ]}'
WHERE name = 'vtm';
