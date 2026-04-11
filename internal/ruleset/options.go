package ruleset

// CharacterOptions returns a map of field key → possible values for every
// chooseable (text/select) field in a given ruleset system. Fields whose
// values depend on another field (e.g. l5r family depends on clan) are
// omitted. Returns nil for unknown or fully-numeric systems.
func CharacterOptions(system string) map[string][]string {
	switch system {
	case "ironsworn":
		return nil // fully numeric, nothing to choose

	case "dnd5e":
		return map[string][]string{
			"race": {
				"Human", "Elf", "Dwarf", "Halfling", "Gnome",
				"Half-Elf", "Half-Orc", "Tiefling", "Dragonborn",
				"Aasimar", "Firbolg", "Goliath", "Kenku", "Lizardfolk",
				"Tabaxi", "Triton", "Yuan-ti Pureblood",
			},
			"class": {
				"Barbarian", "Bard", "Cleric", "Druid", "Fighter",
				"Monk", "Paladin", "Ranger", "Rogue", "Sorcerer",
				"Warlock", "Wizard", "Artificer", "Blood Hunter",
			},
			"background": {
				"Acolyte", "Charlatan", "Criminal", "Entertainer",
				"Folk Hero", "Guild Artisan", "Hermit", "Noble",
				"Outlander", "Sage", "Sailor", "Soldier", "Urchin",
			},
			"alignment": {
				"Lawful Good", "Neutral Good", "Chaotic Good",
				"Lawful Neutral", "True Neutral", "Chaotic Neutral",
				"Lawful Evil", "Neutral Evil", "Chaotic Evil",
			},
		}

	case "vtm":
		return map[string][]string{
			"character_type": {
				"Vampire", "Thin-Blooded", "Ghoul", "Mortal",
			},
			"clan": {
				// Core Rulebook
				"Brujah", "Gangrel", "Malkavian", "Nosferatu", "Toreador",
				"Tremere", "Ventrue", "Caitiff", "Thin-Blooded",
				// Player's Guide
				"Banu Haqim", "Hecata", "Lasombra", "Ministry", "Ravnos", "Salubri", "Tzimisce",
			},
			"predator_type": {
				// Core Rulebook
				"Alleycat", "Bagger", "Blood Leech", "Cleaner", "Consensualist",
				"Extortionist", "Graverobber", "Osiris", "Sandman", "Siren",
				// Player's Guide
				"Farmer", "Montero", "Scene Queen", "Treasure Hunter", "Pursuer", "Witch Hunter",
			},
			"sect": {
				"Camarilla", "Anarch", "Unaligned", "Sabbat (lapsed)",
			},
			"generation": {
				"10th", "11th", "12th", "13th", "14th", "15th (Thin-Blooded)", "16th (Thin-Blooded)",
			},
		}

	case "coc":
		return map[string][]string{
			"occupation": {
				"Antiquarian", "Artist", "Author", "Detective", "Doctor",
				"Engineer", "Journalist", "Military Officer", "Occultist",
				"Parapsychologist", "Police Inspector", "Professor", "Thief",
			},
		}

	case "cyberpunk", "cyberpunk_red":
		return map[string][]string{
			"role": {
				"Rockerboy", "Solo", "Netrunner", "Tech", "Medtech",
				"Media", "Cop", "Corporate", "Fixer", "Nomad",
			},
		}

	case "shadowrun":
		return map[string][]string{
			"metatype": {"Human", "Elf", "Dwarf", "Ork", "Troll"},
			"archetype": {
				"Street Samurai", "Adept", "Decker", "Technomancer",
				"Rigger", "Mage", "Shaman", "Face", "Infiltrator", "Fixer",
			},
		}

	case "wfrp":
		return map[string][]string{
			"species": {"Human", "Halfling", "Dwarf", "High Elf", "Wood Elf"},
			"career": {
				"Apothecary", "Engineer", "Lawyer", "Physician", "Scholar",
				"Wizard", "Agitator", "Artisan", "Beggar", "Investigator",
				"Merchant", "Rat Catcher", "Soldier", "Thief", "Entertainer",
				"Messenger", "Scout",
			},
		}

	case "starwars":
		return map[string][]string{
			"species": {
				"Human", "Twi'lek", "Rodian", "Wookiee", "Bothan",
				"Mon Calamari", "Trandoshan", "Duros", "Zabrak", "Togruta",
			},
			"career": {
				"Bounty Hunter", "Colonist", "Explorer", "Hired Gun",
				"Mystic", "Smuggler", "Technician",
			},
		}

	case "l5r":
		return map[string][]string{
			"clan": {
				"Crab", "Crane", "Dragon", "Lion",
				"Mantis", "Phoenix", "Scorpion", "Unicorn",
			},
		}

	case "theonering":
		return map[string][]string{
			"culture": {
				"Bardings", "Beornings", "Dwarves of Erebor",
				"Elves of Mirkwood", "Hobbits of the Shire",
				"Men of Bree", "Rangers of the North", "Woodmen of Wilderland",
			},
			"calling": {
				"Captain", "Champion", "Messenger",
				"Scholar", "Slayer", "Treasure Hunter", "Wanderer", "Warden",
			},
		}

	case "wrath_glory":
		return map[string][]string{
			// Official archetypes from the Cubicle 7 Wrath & Glory core rulebook,
			// organised by tier (Tier 1–4).
			"archetype": {
				// Tier 1
				"Sister Hospitaller", "Ministorum Priest", "Imperial Guardsman",
				"Inquisitorial Acolyte", "Inquisitorial Sage", "Ganger",
				"Corsair", "Boy", "Cultist",
				// Tier 2
				"Sister of Battle", "Sanctioned Psyker", "Skitarius",
				"Death Cult Assassin", "Tempestus Scion", "Rogue Trader",
				"Scavvy", "Space Marine Scout", "Ranger", "Kommando",
				"Rogue Psyker",
				// Tier 3
				"Tech-Priest", "Crusader", "Imperial Commissar", "Desperado",
				"Tactical Space Marine", "Warlock", "Nob",
				"Heretek", "Chaos Space Marine",
				// Tier 4
				"Inquisitor", "Primaris Intercessor",
			},
			// Official factions from Chapter 3 of the core rulebook.
			"faction": {
				"Adepta Sororitas", "Adeptus Astra Telepathica",
				"Adeptus Mechanicus", "Adeptus Ministorum",
				"Astra Militarum", "The Inquisition",
				"Rogue Trader Dynasties", "Scum",
				"Adeptus Astartes",
				"Aeldari", "Orks", "Chaos",
			},
		}

	case "blades":
		return map[string][]string{
			"playbook": {
				"Cutter", "Hound", "Leech", "Lurk", "Slide", "Spider", "Whisper",
			},
			"heritage": {
				"Akoros", "The Dagger Isles", "Iruvia",
				"Severos", "Skovlan", "Tycheros",
			},
			"background": {
				"Academic", "Labor", "Law", "Trade", "Military", "Underworld",
			},
			"vice": {
				"Faith", "Gambling", "Leisure", "Obligation",
				"Pleasure", "Stupor", "Weird",
			},
		}

	case "paranoia":
		return map[string][]string{
			"management_style": {
				"Authoritarian", "Bureaucratic", "Paranoid", "Obsequious",
			},
			"power_group": {
				"Armed Forces", "CPU", "HPD&MC", "IntSec",
				"PLC", "R&D", "Tech Services",
			},
			"secret_society": {
				"Anti-Mutant", "Communists", "FCCC-P", "Free Enterprise",
				"Humanists", "Mystics", "Pro Tech", "Romantics",
			},
		}

	default:
		return nil
	}
}
