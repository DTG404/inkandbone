package ruleset

// vtmPredatorTypeDef holds the mechanical grants for a VtM V5 Predator Type.
type vtmPredatorTypeDef struct {
	disciplines []disciplineGrant // exactly 2
	specialty   string            // "Skill (Specialty)"
	meritsFlaws string            // free-text description
}

type disciplineGrant struct {
	name  string
	bonus int
}

// vtmPredatorTypes maps each V5 Predator Type to its rulebook-defined grants.
var vtmPredatorTypes = map[string]vtmPredatorTypeDef{
	"Alleycat": {
		disciplines: []disciplineGrant{{"celerity", 1}, {"potence", 1}},
		specialty:   "Athletics:Brawling",
		meritsFlaws: "Merit: Prowler's Instinct / Flaw: Prey Exclusion (Homeless)",
	},
	"Bagger": {
		disciplines: []disciplineGrant{{"blood_sorcery", 1}, {"obfuscate", 1}},
		specialty:   "Streetwise:Black Market",
		meritsFlaws: "Merit: Iron Gullet / Flaw: Prey Exclusion (Bagged Blood)",
	},
	"Blood Leech": {
		disciplines: []disciplineGrant{{"animalism", 1}, {"obfuscate", 1}},
		specialty:   "Stealth:Stalking",
		meritsFlaws: "Flaw: Shunned, Prey Exclusion (Mortals)",
	},
	"Cleaner": {
		disciplines: []disciplineGrant{{"auspex", 1}, {"dominate", 1}},
		specialty:   "Investigation:Crime Scenes",
		meritsFlaws: "Merit: Retainer / Flaw: Obvious Predator",
	},
	"Consensualist": {
		disciplines: []disciplineGrant{{"auspex", 1}, {"presence", 1}},
		specialty:   "Medicine:Kindred Physiology",
		meritsFlaws: "Merit: Herd / Flaw: Prey Exclusion (Non-consenting)",
	},
	"Extortionist": {
		disciplines: []disciplineGrant{{"dominate", 1}, {"potence", 1}},
		specialty:   "Intimidation:Coercion",
		meritsFlaws: "Merit: Contacts / Flaw: Prey Exclusion (Vulnerable)",
	},
	"Graverobber": {
		disciplines: []disciplineGrant{{"fortitude", 1}, {"oblivion", 1}},
		specialty:   "Occult:Grave Rituals",
		meritsFlaws: "Flaw: Obvious Predator",
	},
	"Osiris": {
		disciplines: []disciplineGrant{{"blood_sorcery", 1}, {"presence", 1}},
		specialty:   "Academics:Occult Lore",
		meritsFlaws: "Merit: Fame / Flaw: Prey Exclusion (Faithful)",
	},
	"Sandman": {
		disciplines: []disciplineGrant{{"auspex", 1}, {"obfuscate", 1}},
		specialty:   "Stealth:Sneaking",
		meritsFlaws: "Flaw: Prey Exclusion (Sleeping)",
	},
	"Siren": {
		disciplines: []disciplineGrant{{"presence", 1}, {"potence", 1}},
		specialty:   "Persuasion:Seduction",
		meritsFlaws: "Merit: Looks (Beautiful) / Flaw: Prey Exclusion (Mortals in relationships)",
	},
	"Farmer": {
		disciplines: []disciplineGrant{{"animalism", 1}, {"protean", 1}},
		specialty:   "Animal Ken (Specific Animal)",
		meritsFlaws: "Merit: Iron Gullet / Flaw: Prey Exclusion (Humans)",
	},
	"Montero": {
		disciplines: []disciplineGrant{{"fortitude", 1}, {"protean", 1}},
		specialty:   "Survival (Hunting)",
		meritsFlaws: "Merit: Herd (Rural) / Flaw: Prey Exclusion (City Dwellers)",
	},
	"Scene Queen": {
		disciplines: []disciplineGrant{{"auspex", 1}, {"presence", 1}},
		specialty:   "Etiquette (High Society)",
		meritsFlaws: "Merit: Herd (Artistic Community) / Flaw: Prey Exclusion (Mundane)",
	},
	"Treasure Hunter": {
		disciplines: []disciplineGrant{{"dominate", 1}, {"obfuscate", 1}},
		specialty:   "Larceny (Security Systems)",
		meritsFlaws: "Merit: Contacts (Wealthy) / Flaw: Prey Exclusion (Poor)",
	},
	"Pursuer": {
		disciplines: []disciplineGrant{{"celerity", 1}, {"obfuscate", 1}},
		specialty:   "Athletics (Chase)",
		meritsFlaws: "Flaw: Prey Exclusion (Docile)",
	},
	"Witch Hunter": {
		disciplines: []disciplineGrant{{"auspex", 1}, {"fortitude", 1}},
		specialty:   "Occult (Vampire Lore)",
		meritsFlaws: "Flaw: Obvious Predator",
	},
}

// ApplyVtMPredatorType modifies a character stats map in-place with the grants
// for the named Predator Type. No-op if the type is not found.
func ApplyVtMPredatorType(predatorType string, stats map[string]any) {
	pt, ok := vtmPredatorTypes[predatorType]
	if !ok {
		return
	}
	for _, d := range pt.disciplines {
		current := 0
		if v, ok := stats[d.name]; ok {
			switch n := v.(type) {
			case int:
				current = n
			case float64:
				current = int(n)
			}
		}
		stats[d.name] = current + d.bonus
	}
	if pt.specialty != "" {
		existing, _ := stats["skill_specialties"].(string)
		if existing == "" {
			stats["skill_specialties"] = pt.specialty
		} else {
			stats["skill_specialties"] = existing + ", " + pt.specialty
		}
	}
	if pt.meritsFlaws != "" {
		existing, _ := stats["merits_flaws"].(string)
		if existing == "" {
			stats["merits_flaws"] = pt.meritsFlaws
		} else {
			stats["merits_flaws"] = existing + "; " + pt.meritsFlaws
		}
	}
}
