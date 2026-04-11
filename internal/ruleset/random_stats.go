package ruleset

import (
	"math/rand"
	"sort"
	"strings"
)

// wgArchetypeDef defines the properties of a W&G archetype used during character creation.
type wgArchetypeDef struct {
	faction   string
	species   string
	tier      int
	attrMin   [7]int // strength, agility, toughness, intellect, willpower, fellowship, initiative
	skillMin  map[string]int
	abilities []string // starting archetype abilities (free, cannot be repurchased)
}

// wgArchetypes maps W&G archetype names to their definitions.
var wgArchetypes = map[string]wgArchetypeDef{
	"Sister Hospitaller": {
		faction: "Adepta Sororitas", species: "Human", tier: 1,
		attrMin:  [7]int{3, 3, 3, 3, 3, 4, 3},
		skillMin: map[string]int{"medicae": 2, "persuasion": 1},
		abilities: []string{"Loyal Compassion"},
	},
	"Ministorum Priest": {
		faction: "Adeptus Ministorum", species: "Human", tier: 1,
		attrMin:  [7]int{3, 3, 3, 3, 4, 4, 3},
		skillMin: map[string]int{"leadership": 1, "persuasion": 2},
		abilities: []string{"Fiery Invective"},
	},
	"Imperial Guardsman": {
		faction: "Astra Militarum", species: "Human", tier: 1,
		attrMin:  [7]int{3, 3, 3, 3, 3, 3, 3},
		skillMin: map[string]int{"ws": 1, "bs": 1},
		abilities: []string{"Look Out, Sir!"},
	},
	"Inquisitorial Acolyte": {
		faction: "Inquisition", species: "Human", tier: 1,
		attrMin:  [7]int{3, 3, 3, 3, 3, 3, 3},
		skillMin: map[string]int{"investigation": 1, "insight": 1},
		abilities: []string{"Inquisitorial Decree"},
	},
	"Inquisitorial Sage": {
		faction: "Inquisition", species: "Human", tier: 1,
		attrMin:  [7]int{3, 3, 3, 4, 3, 3, 3},
		skillMin: map[string]int{"scholar": 2, "investigation": 1},
		abilities: []string{"Administratum Records"},
	},
	"Ganger": {
		faction: "Underhive", species: "Human", tier: 1,
		attrMin:  [7]int{3, 4, 3, 3, 3, 3, 3},
		skillMin: map[string]int{"cunning": 1, "stealth": 1},
		abilities: []string{"Scrounger"},
	},
	"Corsair": {
		faction: "Rogue Traders", species: "Human", tier: 1,
		attrMin:  [7]int{3, 4, 3, 3, 3, 3, 4},
		skillMin: map[string]int{"pilot": 1, "bs": 1},
		abilities: []string{"Dancing the Blade's Edge"},
	},
	"Boy": {
		faction: "Orks", species: "Ork", tier: 1,
		attrMin:  [7]int{4, 3, 4, 2, 3, 3, 3},
		skillMin: map[string]int{"ws": 1, "intimidation": 1},
		abilities: []string{"Get Stuck In"},
	},
	"Cultist": {
		faction: "Chaos Undivided", species: "Human", tier: 1,
		attrMin:  [7]int{3, 3, 3, 3, 3, 3, 3},
		skillMin: map[string]int{"deception": 1, "stealth": 1},
		abilities: []string{"Enemy Within", "Corruption"},
	},
	"Sister of Battle": {
		faction: "Adepta Sororitas", species: "Human", tier: 2,
		attrMin:  [7]int{3, 3, 4, 3, 4, 3, 3},
		skillMin: map[string]int{"ws": 2, "fortitude": 2},
		abilities: []string{"Purity of Faith"},
	},
	"Sanctioned Psyker": {
		faction: "Astra Militarum", species: "Human", tier: 2,
		attrMin:  [7]int{3, 3, 3, 4, 4, 3, 3},
		skillMin: map[string]int{"psychic_mastery": 2, "scholar": 1},
		abilities: []string{"Psyker", "Unlock Disciplines"},
	},
	"Skitarius": {
		faction: "Adeptus Mechanicus", species: "Human", tier: 2,
		attrMin:  [7]int{3, 3, 4, 3, 3, 3, 3},
		skillMin: map[string]int{"tech": 1, "bs": 2},
		abilities: []string{"Heavily Augmented"},
	},
	"Death Cult Assassin": {
		faction: "Officio Assassinorum", species: "Human", tier: 2,
		attrMin:  [7]int{4, 5, 3, 3, 3, 3, 5},
		skillMin: map[string]int{"ws": 3, "stealth": 2},
		abilities: []string{"Glancing Blow"},
	},
	"Tempestus Scion": {
		faction: "Astra Militarum", species: "Human", tier: 2,
		attrMin:  [7]int{3, 4, 4, 3, 4, 3, 4},
		skillMin: map[string]int{"bs": 3, "athletics": 2},
		abilities: []string{"Elite Soldier"},
	},
	"Rogue Trader": {
		faction: "Rogue Traders", species: "Human", tier: 2,
		attrMin:  [7]int{3, 3, 3, 3, 4, 5, 3},
		skillMin: map[string]int{"leadership": 2, "persuasion": 2},
		abilities: []string{"Warrant of Trade"},
	},
	"Scavvy": {
		faction: "Underhive", species: "Human", tier: 2,
		attrMin:  [7]int{3, 3, 3, 3, 3, 3, 3},
		skillMin: map[string]int{"survival": 1, "cunning": 1},
		abilities: []string{"Mutant"},
	},
	"Space Marine Scout": {
		faction: "Adeptus Astartes", species: "Space Marine", tier: 2,
		attrMin:  [7]int{4, 4, 5, 3, 4, 3, 4},
		skillMin: map[string]int{"stealth": 2, "awareness": 2},
		abilities: []string{"Use the Terrain"},
	},
	"Ranger": {
		faction: "Aeldari", species: "Aeldari", tier: 2,
		attrMin:  [7]int{3, 4, 3, 4, 3, 3, 5},
		skillMin: map[string]int{"stealth": 3, "survival": 2},
		abilities: []string{"From the Shadows"},
	},
	"Kommando": {
		faction: "Orks", species: "Ork", tier: 2,
		attrMin:  [7]int{4, 4, 4, 3, 3, 3, 4},
		skillMin: map[string]int{"stealth": 2, "cunning": 1},
		abilities: []string{"Kunnin' Plan"},
	},
	"Rogue Psyker": {
		faction: "Chaos Undivided", species: "Human", tier: 2,
		attrMin:  [7]int{3, 3, 3, 4, 5, 3, 3},
		skillMin: map[string]int{"psychic_mastery": 3, "deception": 1},
		abilities: []string{"Psyker", "Unlock Disciplines", "Corruption"},
	},
	"Tech-Priest": {
		faction: "Adeptus Mechanicus", species: "Human", tier: 3,
		attrMin:  [7]int{3, 3, 4, 5, 3, 3, 3},
		skillMin: map[string]int{"tech": 3, "scholar": 2},
		abilities: []string{"Rite of Repair"},
	},
	"Crusader": {
		faction: "Adeptus Ministorum", species: "Human", tier: 3,
		attrMin:  [7]int{4, 3, 4, 3, 4, 3, 3},
		skillMin: map[string]int{"ws": 3, "fortitude": 2},
		abilities: []string{"Armour of Faith"},
	},
	"Imperial Commissar": {
		faction: "Astra Militarum", species: "Human", tier: 3,
		attrMin:  [7]int{3, 3, 3, 3, 5, 4, 3},
		skillMin: map[string]int{"leadership": 3, "intimidation": 2},
		abilities: []string{"Fearsome Respect"},
	},
	"Desperado": {
		faction: "Rogue Traders", species: "Human", tier: 3,
		attrMin:  [7]int{3, 5, 3, 3, 3, 3, 5},
		skillMin: map[string]int{"bs": 3, "cunning": 2},
		abilities: []string{"Valuable Prey"},
	},
	"Tactical Space Marine": {
		faction: "Adeptus Astartes", species: "Space Marine", tier: 3,
		attrMin:  [7]int{5, 4, 6, 4, 5, 4, 4},
		skillMin: map[string]int{"ws": 4, "bs": 4, "athletics": 3},
		abilities: []string{"Tactical Versatility"},
	},
	"Warlock": {
		faction: "Aeldari", species: "Aeldari", tier: 3,
		attrMin:  [7]int{4, 4, 4, 5, 5, 4, 5},
		skillMin: map[string]int{"psychic_mastery": 4, "ws": 3},
		abilities: []string{"Runes of Battle", "Unlock Disciplines"},
	},
	"Nob": {
		faction: "Orks", species: "Ork", tier: 3,
		attrMin:  [7]int{6, 4, 6, 3, 4, 4, 4},
		skillMin: map[string]int{"ws": 4, "intimidation": 3},
		abilities: []string{"The Green Tide"},
	},
	"Heretek": {
		faction: "Chaos Undivided", species: "Human", tier: 3,
		attrMin:  [7]int{3, 3, 5, 6, 3, 3, 3},
		skillMin: map[string]int{"tech": 4, "scholar": 3},
		abilities: []string{"Rite of Repair", "Corruption"},
	},
	"Chaos Space Marine": {
		faction: "Chaos Undivided", species: "Space Marine", tier: 3,
		attrMin:  [7]int{6, 4, 6, 4, 5, 4, 4},
		skillMin: map[string]int{"ws": 4, "bs": 4, "intimidation": 3},
		abilities: []string{"Tactical Versatility", "Corruption"},
	},
	"Inquisitor": {
		faction: "Inquisition", species: "Human", tier: 4,
		attrMin:  [7]int{4, 4, 4, 5, 5, 5, 4},
		skillMin: map[string]int{"investigation": 4, "leadership": 3, "scholar": 3},
		abilities: []string{"Unchecked Authority"},
	},
	"Primaris Intercessor": {
		faction: "Adeptus Astartes", species: "Space Marine", tier: 4,
		attrMin:  [7]int{6, 5, 6, 4, 5, 4, 5},
		skillMin: map[string]int{"bs": 5, "athletics": 4},
		abilities: []string{"Intercessor Focus"},
	},
}

// Abilities returns the archetype's starting abilities slice.
func (d wgArchetypeDef) Abilities() []string { return d.abilities }

// WGArchetypeView is an exported view of a wgArchetypeDef for use outside this package.
type WGArchetypeView struct {
	def wgArchetypeDef
}

// Abilities returns the archetype's starting abilities (free, cannot be repurchased).
func (v WGArchetypeView) Abilities() []string { return v.def.abilities }

// WGArchetypeDefFor looks up the W&G archetype definition for the given name.
// Returns an exported WGArchetypeView and true if found, or zero-value and false if not found.
func WGArchetypeDefFor(name string) (WGArchetypeView, bool) {
	def, ok := wgArchetypes[name]
	return WGArchetypeView{def: def}, ok
}

// wgSpeciesSpeed maps species to their base Speed value (core rulebook p.193).
var wgSpeciesSpeed = map[string]int{
	"Human":          6,
	"Space Marine":   7,
	"Primaris":       7,
	"Aeldari":        8,
	"Ork":            6,
}

// randPick returns a random element from the given slice.
func randPick(options []string) string {
	if len(options) == 0 {
		return ""
	}
	return options[rand.Intn(len(options))]
}

// RollStats returns a map of randomly generated starting field values for the
// given ruleset system. Every schema field is populated — text fields are
// randomly chosen from canonical options. An unrecognised system returns an
// empty map.
func RollStats(system, archetype string) map[string]any {
	switch system {
	case "dnd5e":
		return map[string]any{
			"race":              randPick([]string{"Human", "Elf", "Dwarf", "Halfling", "Gnome", "Half-Elf", "Half-Orc", "Tiefling", "Dragonborn"}),
			"class":             randPick([]string{"Barbarian", "Bard", "Cleric", "Druid", "Fighter", "Monk", "Paladin", "Ranger", "Rogue", "Sorcerer", "Warlock", "Wizard"}),
			"background":        randPick([]string{"Acolyte", "Charlatan", "Criminal", "Entertainer", "Folk Hero", "Guild Artisan", "Hermit", "Noble", "Outlander", "Sage", "Sailor", "Soldier", "Urchin"}),
			"alignment":         "True Neutral",
			"str":               roll4d6DropLowest(),
			"dex":               roll4d6DropLowest(),
			"con":               roll4d6DropLowest(),
			"int":               roll4d6DropLowest(),
			"wis":               roll4d6DropLowest(),
			"cha":               roll4d6DropLowest(),
			"level":             1,
			"hp":                10,
			"ac":                10,
			"proficiency_bonus": 2,
		}
	case "ironsworn":
		return ironswornStats()
	case "vtm":
		return rollVtMV5Stats()
	case "coc":
		pow := rollNd(3, 6) * 5
		siz := (rollNd(2, 6) + 6) * 5
		con := rollNd(3, 6) * 5
		return map[string]any{
			"occupation": randPick([]string{"Antiquarian", "Artist", "Author", "Detective", "Doctor", "Engineer", "Journalist", "Military Officer", "Occultist", "Parapsychologist", "Police Inspector", "Professor", "Thief"}),
			"str":        rollNd(3, 6) * 5,
			"con":        con,
			"siz":        siz,
			"dex":        rollNd(3, 6) * 5,
			"app":        rollNd(3, 6) * 5,
			"pow":        pow,
			"int":        (rollNd(2, 6) + 6) * 5,
			"edu":        (rollNd(2, 6) + 6) * 5,
			"hp":         (con/10 + siz/10 + 1) / 1,
			"sanity":     pow,
			"luck":       rollNd(3, 6) * 5,
			"mp":         pow / 10,
			"age":        17 + rollNd(2, 6),
			"skills":     "",
			"inventory":  "",
			"notes":      "",
		}
	case "cyberpunk", "cyberpunk_red":
		emp := rollNd(2, 6) + 2
		return map[string]any{
			"role":        randPick([]string{"Rockerboy", "Solo", "Netrunner", "Tech", "Medtech", "Media", "Cop", "Corporate", "Fixer", "Nomad"}),
			"int":         rollNd(2, 6) + 2,
			"ref":         rollNd(2, 6) + 2,
			"cool":        rollNd(2, 6) + 2,
			"tech":        rollNd(2, 6) + 2,
			"lk":          rollNd(2, 6) + 2,
			"att":         rollNd(2, 6) + 2,
			"ma":          rollNd(2, 6) + 2,
			"emp":         emp,
			"body":        rollNd(2, 6) + 2,
			"humanity":    emp * 10,
			"eurodollars": rollNd(2, 6)*100 + 200,
			"skills":      "",
			"cyberware":   "",
			"gear":        "",
			"notes":       "",
		}
	case "shadowrun":
		bod := rollNd(1, 6)
		agi := rollNd(1, 6)
		rea := rollNd(1, 6)
		str := rollNd(1, 6)
		wil := rollNd(1, 6)
		log := rollNd(1, 6)
		intu := rollNd(1, 6)
		cha := rollNd(1, 6)
		return map[string]any{
			"metatype":       randPick([]string{"Human", "Elf", "Dwarf", "Ork", "Troll"}),
			"archetype":      randPick([]string{"Street Samurai", "Adept", "Decker", "Technomancer", "Rigger", "Mage", "Shaman", "Face", "Infiltrator", "Fixer"}),
			"priority":       "A/B/C/D/E",
			"body":           bod,
			"agility":        agi,
			"reaction":       rea,
			"strength":       str,
			"willpower":      wil,
			"logic":          log,
			"intuition":      intu,
			"charisma":       cha,
			"edge":           rollNd(1, 4),
			"essence":        6,
			"physical_limit": (str*2+bod+rea)/3 + 1,
			"mental_limit":   (log*2+intu+wil)/3 + 1,
			"social_limit":   (cha*2 + wil + 6) / 3,
			"nuyen":          rollNd(2, 6) * 100,
			"karma":          0,
			"reputation":     0,
			"notoriety":      0,
			"notes":          "",
		}
	case "wfrp":
		return map[string]any{
			"species":      randPick([]string{"Human", "Halfling", "Dwarf", "High Elf", "Wood Elf"}),
			"career":       randPick([]string{"Apothecary", "Engineer", "Lawyer", "Physician", "Scholar", "Wizard", "Agitator", "Artisan", "Beggar", "Investigator", "Merchant", "Rat Catcher", "Soldier", "Thief", "Entertainer", "Messenger", "Soldier", "Scout"}),
			"career_level": 1,
			"ws":           rollNd(2, 10) + 20,
			"bs":           rollNd(2, 10) + 20,
			"s":            rollNd(2, 10) + 20,
			"t":            rollNd(2, 10) + 20,
			"ag":           rollNd(2, 10) + 20,
			"i":            rollNd(2, 10) + 20,
			"dex":          rollNd(2, 10) + 20,
			"int":          rollNd(2, 10) + 20,
			"wp":           rollNd(2, 10) + 20,
			"fel":          rollNd(2, 10) + 20,
			"wounds":       10,
			"fate":         2,
			"fortune":      2,
			"resilience":   1,
			"resolve":      1,
			"xp":           0,
			"ambitions":    "",
			"notes":        "",
		}
	case "starwars":
		return starWarsStats()

	case "l5r":
		clan := randPick([]string{"Crab", "Crane", "Dragon", "Lion", "Mantis", "Phoenix", "Scorpion", "Unicorn"})
		familyByClan := map[string][]string{
			"Crab":     {"Hida", "Hiruma", "Kaiu", "Kuni", "Toritaka", "Yasuki"},
			"Crane":    {"Asahina", "Daidoji", "Doji", "Kakita"},
			"Dragon":   {"Agasha", "Hitomi", "Kitsuki", "Mirumoto", "Tamori"},
			"Lion":     {"Akodo", "Ikoma", "Kitsu", "Matsu"},
			"Mantis":   {"Kamoto", "Moshi", "Tsuruchi", "Yoritomo"},
			"Phoenix":  {"Agasha", "Asako", "Isawa", "Shiba"},
			"Scorpion": {"Bayushi", "Shosuro", "Soshi", "Yogo"},
			"Unicorn":  {"Horiuchi", "Iuchi", "Moto", "Shinjo", "Utaku"},
		}
		return map[string]any{
			"clan":        clan,
			"family":      randPick(familyByClan[clan]),
			"school":      clan + " School",
			"school_rank": 1,
			"air":         2,
			"earth":       2,
			"fire":        2,
			"water":       2,
			"void":        1,
			"endurance":   8,
			"composure":   8,
			"focus":       3,
			"vigilance":   2,
			"glory":       45,
			"honor":       45,
			"status":      30,
			"xp":          0,
			"notes":       "",
		}
	case "theonering":
		body := rollNd(1, 3) + 1
		heart := rollNd(1, 3) + 1
		wits := rollNd(1, 3) + 1
		return map[string]any{
			"culture":          randPick([]string{"Bardings", "Beornings", "Dwarves of Erebor", "Elves of Mirkwood", "Hobbits of the Shire", "Men of Bree", "Rangers of the North", "Woodmen of Wilderland"}),
			"calling":          randPick([]string{"Scholar", "Slayer", "Treasure Hunter", "Wanderer", "Warden"}),
			"body":             body,
			"heart":            heart,
			"wits":             wits,
			"endurance":        20 + body,
			"endurance_max":    20 + body,
			"hope":             8 + heart,
			"hope_max":         8 + heart,
			"shadow_points":    0,
			"shadow_scars":     0,
			"valour":           1,
			"wisdom":           1,
			"standing":         randPick([]string{"Emissary", "Exile", "Honoured", "Renowned", "Strider"}),
			"fellowship_score": 18,
			"notes":            "",
		}
	case "wrath_glory":
		return rollWrathGloryStats(archetype)
	case "blades":
		return bladesStats()

	case "paranoia":
		sector := randPick([]string{"ALF", "BRT", "DEN", "GRN", "MEL", "PLC", "RED", "VLT"})
		return map[string]any{
			"full_name":          "",
			"sector":             sector,
			"security_clearance": "INFRARED",
			"management_style":   randPick([]string{"Authoritarian", "Bureaucratic", "Paranoid", "Obsequious"}),
			"power_group":        randPick([]string{"Armed Forces", "CPU", "HPD&MC", "IntSec", "PLC", "R&D", "Tech Services"}),
			"secret_society":     randPick([]string{"Anti-Mutant", "Communists", "FCCC-P", "Free Enterprise", "Humanists", "Mystics", "Pro Tech", "Romantics"}),
			"violence":           rollNd(1, 6),
			"treachery":          rollNd(1, 6),
			"happiness":          rollNd(1, 6),
			"straight":           rollNd(1, 6),
			"moxie":              rollNd(1, 6),
			"credits":            rollNd(1, 6)*10 + 20,
			"clone_number":       1,
			"treason_points":     0,
			"notes":              "",
		}
	default:
		return map[string]any{}
	}
}

// rollWrathGloryStats generates a W&G character using the wgArchetypes table.
// Archetype starting abilities are pre-populated into the talents field.
// If archetypeName is empty or not found, a random archetype is chosen.
func rollWrathGloryStats(archetypeArgs ...string) map[string]any {
	archetypeName := ""
	if len(archetypeArgs) > 0 {
		archetypeName = archetypeArgs[0]
	}
	def, ok := wgArchetypes[archetypeName]
	if !ok {
		names := make([]string, 0, len(wgArchetypes))
		for k := range wgArchetypes {
			names = append(names, k)
		}
		archetypeName = randPick(names)
		def = wgArchetypes[archetypeName]
	}

	// Roll each attribute as minValue + rand.Intn(3), giving a range of
	// [min, min+2] — always meets prerequisites and adds variety.
	attr := func(minVal int) int {
		return minVal + rand.Intn(3)
	}
	str := attr(def.attrMin[0])
	agi := attr(def.attrMin[1])
	tgh := attr(def.attrMin[2])
	itl := attr(def.attrMin[3])
	wil := attr(def.attrMin[4])
	fel := attr(def.attrMin[5])
	ini := attr(def.attrMin[6])

	// Roll each skill, enforcing any prerequisite floor from the archetype def.
	skillVal := func(key string) int {
		floor := def.skillMin[key]
		rolled := wrathSkillRoll()
		if rolled < floor {
			return floor
		}
		return rolled
	}

	// Derived values per core rulebook.
	resolve := wil - 1
	if resolve < 1 {
		resolve = 1
	}
	speed, ok := wgSpeciesSpeed[def.species]
	if !ok {
		speed = 6
	}
	wounds := (def.tier * 2) + tgh
	shock := wil + def.tier

	// Starting Wealth Tier: Tier 1-2 start at 2, Tier 3-4 start at 3.
	wealthTier := 2
	if def.tier >= 3 {
		wealthTier = 3
	}

	// Pre-populate talents with archetype starting abilities.
	talents := strings.Join(def.abilities, "|")

	_ = str
	_ = itl
	_ = fel

	// Build keyword string: SPECIES, ARCHETYPE, FACTION keywords.
	keywords := strings.ToUpper(def.species) + ", " + strings.ToUpper(archetypeName) + ", " + strings.ToUpper(def.faction)

	return map[string]any{
		"archetype": archetypeName,
		"faction":   def.faction,
		"rank":      1,
		"keywords":  keywords,

		"strength":   str,
		"agility":    agi,
		"toughness":  tgh,
		"intellect":  itl,
		"willpower":  wil,
		"fellowship": fel,
		"initiative": ini,

		"ws":              skillVal("ws"),
		"bs":              skillVal("bs"),
		"athletics":       skillVal("athletics"),
		"awareness":       skillVal("awareness"),
		"cunning":         skillVal("cunning"),
		"deception":       skillVal("deception"),
		"fortitude":       skillVal("fortitude"),
		"insight":         skillVal("insight"),
		"intimidation":    skillVal("intimidation"),
		"investigation":   skillVal("investigation"),
		"leadership":      skillVal("leadership"),
		"medicae":         skillVal("medicae"),
		"persuasion":      skillVal("persuasion"),
		"pilot":           skillVal("pilot"),
		"psychic_mastery": skillVal("psychic_mastery"),
		"scholar":         skillVal("scholar"),
		"stealth":         skillVal("stealth"),
		"survival":        skillVal("survival"),
		"tech":            skillVal("tech"),

		// Derived stats per core rulebook formulas.
		"speed":         speed,
		"defence":       ini - 1,
		"resilience":    tgh + 1,
		"determination": tgh,
		"resolve":       resolve,
		"conviction":    wil,
		"influence":     fel - 1,

		// Maximum Wounds = (Tier×2) + Toughness
		// Maximum Shock  = Willpower + Tier
		"wounds":     wounds,
		"shock":      shock,
		"corruption": 0,

		"wrath":  0,
		"glory":  0,
		"ruin":   0,
		"wealth": wealthTier,
		"xp":     0,

		"talents": talents,
		"powers":  "",
		"notes":   "",
	}
}

// wrathSkillRoll returns a starting skill rating for W&G: 70% chance of 0,
// 25% chance of 1, 5% chance of 2 — representing an untrained recruit with
// a few areas of natural aptitude.
func wrathSkillRoll() int {
	r := rand.Intn(20)
	if r < 1 {
		return 2
	}
	if r < 6 {
		return 1
	}
	return 0
}

// roll4d6DropLowest rolls 4d6 and returns the sum of the top 3 (standard D&D method).
func roll4d6DropLowest() int {
	rolls := make([]int, 4)
	for i := range rolls {
		rolls[i] = rand.Intn(6) + 1
	}
	sort.Ints(rolls)
	return rolls[1] + rolls[2] + rolls[3]
}

// rollNd rolls n dice with the given number of sides and returns the total.
func rollNd(n, sides int) int {
	total := 0
	for i := 0; i < n; i++ {
		total += rand.Intn(sides) + 1
	}
	return total
}

// ironswornStats distributes 7 points among the five Ironsworn attributes (each 1–3).
func ironswornStats() map[string]any {
	attrs := [5]string{"edge", "heart", "iron", "shadow", "wits"}
	vals := [5]int{1, 1, 1, 1, 1} // base total = 5; distribute 2 more
	for remaining := 2; remaining > 0; {
		i := rand.Intn(5)
		if vals[i] < 3 {
			vals[i]++
			remaining--
		}
	}
	result := map[string]any{
		"health":   5,
		"spirit":   5,
		"supply":   5,
		"momentum": 2,
		"vows":     "",
		"bonds":    "",
		"assets":   "",
		"notes":    "",
	}
	for i, k := range attrs {
		result[k] = vals[i]
	}
	return result
}

// starWarsStats generates Edge of the Empire characteristics for a typical humanoid.
// All six characteristics start at 2; three randomly chosen are raised to 3.
func starWarsStats() map[string]any {
	keys := []string{"brawn", "agility", "intellect", "cunning", "willpower", "presence"}
	stats := map[string]int{
		"brawn": 2, "agility": 2, "intellect": 2,
		"cunning": 2, "willpower": 2, "presence": 2,
	}
	rand.Shuffle(len(keys), func(i, j int) { keys[i], keys[j] = keys[j], keys[i] })
	for _, k := range keys[:3] {
		stats[k]++
	}
	career := randPick([]string{"Bounty Hunter", "Colonist", "Explorer", "Hired Gun", "Mystic", "Smuggler", "Technician"})
	specializationByCareeer := map[string][]string{
		"Bounty Hunter": {"Assassin", "Gadgeteer", "Survivalist"},
		"Colonist":      {"Doctor", "Politico", "Scholar"},
		"Explorer":      {"Fringer", "Scout", "Trader"},
		"Hired Gun":     {"Bodyguard", "Marauder", "Mercenary Soldier"},
		"Mystic":        {"Advisor", "Magician", "Seer"},
		"Smuggler":      {"Charmer", "Gambler", "Pilot"},
		"Technician":    {"Mechanic", "Outlaw Tech", "Slicer"},
	}
	result := map[string]any{
		"species":          randPick([]string{"Human", "Twi'lek", "Rodian", "Wookiee", "Bothan", "Mon Calamari", "Trandoshan", "Duros", "Zabrak", "Togruta"}),
		"career":           career,
		"specialization":   randPick(specializationByCareeer[career]),
		"wounds_current":   0,
		"wounds_threshold": 10 + stats["brawn"],
		"strain_current":   0,
		"strain_threshold": 10 + stats["willpower"],
		"soak":             stats["brawn"],
		"defense_melee":    0,
		"defense_ranged":   0,
		"credits":          500,
		"obligation":       10 + rollNd(1, 10),
		"force_rating":     0,
		"notes":            "",
	}
	for k, v := range stats {
		result[k] = v
	}
	return result
}

// bladesStats distributes 4 points among the 12 Blades in the Dark action ratings (each 0–2).
func bladesStats() map[string]any {
	actions := []string{
		"hunt", "study", "survey", "tinker",
		"finesse", "prowl", "skirmish", "wreck",
		"attune", "command", "consort", "sway",
	}
	vals := make(map[string]int, len(actions))
	for _, a := range actions {
		vals[a] = 0
	}
	for remaining := 4; remaining > 0; {
		a := actions[rand.Intn(len(actions))]
		if vals[a] < 2 {
			vals[a]++
			remaining--
		}
	}
	result := map[string]any{
		"playbook":    randPick([]string{"Cutter", "Hound", "Leech", "Lurk", "Slide", "Spider", "Whisper"}),
		"heritage":    randPick([]string{"Akoros", "The Dagger Isles", "Iruvia", "Severos", "Skovlan", "Tycheros"}),
		"background":  randPick([]string{"Academic", "Labor", "Law", "Trade", "Military", "Underworld"}),
		"vice":        randPick([]string{"Faith", "Gambling", "Leisure", "Obligation", "Pleasure", "Stupor", "Weird"}),
		"stress":      0,
		"trauma":      0,
		"coin":        2,
		"stash":       0,
		"load":        3,
		"xp_insight":  0,
		"xp_prowess":  0,
		"xp_resolve":  0,
		"notes":       "",
	}
	for k, v := range vals {
		result[k] = v
	}
	return result
}

// rollVtMV5Stats generates a V5 starting vampire character.
// Attribute pools: primary (4,3,2), secondary (3,3,2), tertiary (2,2,2).
// One attribute dot is distributed to each pool's extra slot during play.
func rollVtMV5Stats() map[string]any {
	// Three attribute groups, each gets a shuffled pool of values
	primaries := [3][3]int{
		{4, 3, 2},
		{3, 3, 2},
		{2, 2, 2},
	}
	// Randomly assign which priority goes to which group
	order := []int{0, 1, 2}
	for i := range order {
		j := i + rand.Intn(len(order)-i)
		order[i], order[j] = order[j], order[i]
	}
	groups := [3][3]int{}
	for g := 0; g < 3; g++ {
		vals := primaries[order[g]]
		for i := range vals {
			j := i + rand.Intn(len(vals)-i)
			vals[i], vals[j] = vals[j], vals[i]
		}
		groups[g] = vals
	}

	str := groups[0][0]
	dex := groups[0][1]
	sta := groups[0][2]
	cha := groups[1][0]
	man := groups[1][1]
	com := groups[1][2]
	intel := groups[2][0]
	wit := groups[2][1]
	res := groups[2][2]

	clans := []string{
		"Brujah", "Gangrel", "Malkavian", "Nosferatu", "Toreador", "Tremere", "Ventrue", "Caitiff", "Thin-Blooded",
		"Banu Haqim", "Hecata", "Lasombra", "Ministry", "Ravnos", "Salubri", "Tzimisce",
	}
	predTypes := []string{
		"Alleycat", "Bagger", "Blood Leech", "Cleaner", "Consensualist", "Extortionist",
		"Graverobber", "Osiris", "Sandman", "Siren",
		"Farmer", "Montero", "Scene Queen", "Treasure Hunter", "Pursuer", "Witch Hunter",
	}
	sects := []string{"Camarilla", "Anarch", "Unaligned", "Sabbat (lapsed)"}

	stats := map[string]any{
		"character_type": "Vampire",
		"clan": randPick(clans), "predator_type": randPick(predTypes), "sect": randPick(sects),
		"generation": randPick([]string{"10th", "11th", "12th", "13th", "14th", "15th (Thin-Blooded)", "16th (Thin-Blooded)"}),
		"hunger": 1, "blood_potency": 1, "bane_severity": 1,
		"humanity": 7, "stains": 0, "xp": 0,
		"strength": str, "dexterity": dex, "stamina": sta,
		"charisma": cha, "manipulation": man, "composure": com,
		"intelligence": intel, "wits": wit, "resolve": res,
		"health_max": sta + 3, "health_superficial": 0, "health_aggravated": 0,
		"willpower_max": com + res, "willpower_superficial": 0, "willpower_aggravated": 0,
		"athletics": 0, "brawl": 0, "craft": 0, "drive": 0, "firearms": 0,
		"larceny": 0, "melee": 0, "stealth": 0, "survival": 0,
		"animal_ken": 0, "etiquette": 0, "insight": 0, "intimidation": 0,
		"leadership": 0, "performance": 0, "persuasion": 0, "streetwise": 0, "subterfuge": 0,
		"academics": 0, "awareness": 0, "finance": 0, "investigation": 0,
		"medicine": 0, "occult": 0, "politics": 0, "technology": 0,
		"animalism": 0, "auspex": 0, "blood_sorcery": 0, "celerity": 0,
		"dominate": 0, "fortitude": 0, "obfuscate": 0, "oblivion": 0,
		"potence": 0, "presence": 0, "protean": 0,
		"thin_blood_alchemy": 0, "bond_strength": 0,
		"domitor": "", "rituals": "", "ceremonies": "", "known_formulae": "", "loresheets": "",
		"skill_specialties": "", "merits_flaws": "", "ambition": "", "desire": "",
		"convictions": "", "touchstones": "", "notes": "",
	}
	ApplyVtMPredatorType(stats["predator_type"].(string), stats)
	return stats
}
