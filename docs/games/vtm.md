# Vampire: The Masquerade (V5) — How to Play

You are a vampire. Kindred. One of the Damned. You hunt, you feed, you struggle against the Beast inside you, and you navigate the brutal politics of an immortal society hidden within the mortal world. The AI is your Storyteller.

V5 replaces blood points with Hunger. You don't count how much blood you have — you track how desperately you need it. Hunger colors every roll, every Discipline activation, every moment of stress.

## The Character Sheet

Create through **⚙ Manage → Characters → + New Character**. Inkandbone generates all V5 fields: attribute priorities distributed, skills zeroed, Health (Stamina+3) and Willpower (Composure+Resolve) tracks set.

**Hunger** (0-5) is red squares at the top of your sheet. At 1-2, you're controlled. At 3, you're hungry. At 4-5, you're dangerous — frenzy risk is critical, and Hunger dice dominate your pools. Vampires max at 5. Thin-Blooded max at 4. Mortals and Ghouls don't have Hunger.

**The nine attributes** split into Physical (Strength, Dexterity, Stamina), Social (Charisma, Manipulation, Composure), and Mental (Intelligence, Wits, Resolve). Each is rated 1-5. You get one primary group at 4/3/2, secondary at 3/3/2, tertiary at 2/2/2.

**The 27 skills** split the same way: Physical (Athletics through Survival), Social (Animal Ken through Subterfuge), and Mental (Academics through Technology). Each rated 0-5.

**Disciplines** are your supernatural powers: Animalism, Auspex, Blood Sorcery, Celerity, Dominate, Fortitude, Obfuscate, Oblivion, Potence, Presence, Protean. Thin-Blooded get Alchemy instead.

**Health** is a damage track — superficial (/) fills from the left, aggravated (X) fills from the right. Click boxes to toggle. **Willpower** works the same way.

**Humanity** is your moral compass (7 is default). **Stains** accumulate when you violate your Convictions. At 10 Stains, you make a Remorse check or risk degeneration.

**Clan** determines your curse, your in-clan Disciplines, and your place in Kindred society. **Predator Type** determines your feeding style and starting Discipline spread.

## Hunger and Rouse Checks

Hunger is the core V5 mechanic. Every time you activate a Discipline, you make a **Rouse Check**: roll 1 die. If it exceeds your Hunger, nothing happens. If it's equal to or below, Hunger increases by 1.

The AI detects `/rouse` in your messages and performs Rouse Checks automatically.

When you make a roll where your vampiric nature matters, a number of dice equal to your Hunger become **Hunger dice** — rolled separately and tracked. If a Hunger die shows 1 on a failed roll, it's a **Bestial Failure**. If a Hunger die shows 10 on a successful roll, it's a **Messy Critical** — you succeed, but the Beast leaks through, and your Clan Compulsion triggers.

## How the AI Runs V5

The AI handles everything V5-specific automatically. It tracks your Hunger, performs Rouse Checks, identifies Hunger dice in your pools, catches Messy Criticals and Bestial Failures, triggers Clan Compulsions, and narrates the Beast.

When the Storyteller describes a new night falling, the **Chronicle Night** counter in the header advances by 1. Phrases like "dusk falls", "nightfall", or "darkness descends" trigger this.

The AI scans narration for **Masquerade breaches** — witnessed feeding, cameras, supernatural displays. Each breach decrements the session's Masquerade Integrity (0-10, starts at 10). Low Masquerade means the Second Inquisition gets closer.

## Playing

Open the browser. Get your coterie and session active. Elysium awaits:

> *I step into the converted theater, the weight of unlife settled into my bones. Madame Beaumont hasn't arrived yet. I scan the room — Toreador pose near the stage, a Nosferatu lurks in the shadows of the balcony, a Ventrue holds court at the bar. I need information about my sire's disappearance.*

The AI describes the scene, the NPCs, the tensions. When you act — Manipulation + Subterfuge to gather information, Charisma + Persuasion to sway an elder — roll your pool, and the AI counts successes.

When you activate a power, type `/rouse` and the AI checks.

When you feed, describe it. The AI determines Hunger reduction: a sip (a victim later, no risk) reduces by 1, a proper feed reduces by 2, a desperate gullet feed reduces by 3.

## The Chronicle

**Elysium:** Meet the local Kindred. Learn the power structure. Don't offend anyone with higher status than you.

**The Assignment:** An elder gives you a task — find a missing Nosferatu, retrieve a stolen artifact, investigate a rumor. Something they don't want to handle personally.

**The Hunt:** The task pushes you into mortal society. You'll need to feed, maintain the Masquerade, use your Disciplines, and make choices that test your Humanity.

**The Reckoning:** The task resolves. Your standing changes. Your Hunger is higher or lower. Your Humanity may have slipped. The Chronicle continues.

## Key Rules

- **Dice pools:** Attribute + Skill in d10s. Count 6+ as successes. Difficulty can reduce effective dice.
- **Rouse Checks:** 1 die vs Hunger. Equal or under = Hunger +1.
- **Hunger dice:** Your Hunger rating in dice are marked. If a Hunger die shows 1 on a failure = Bestial Failure. Shows 10 on a success = Messy Critical + Clan Compulsion.
- **Frenzy:** When fire, extreme Hunger (4-5), or supernatural fear triggers it, roll Composure + Resolve.
- **Feeding:** Describe how you feed. Sip = -1 Hunger. Proper feed = -2. Deep feed = -3. Never below 0.
- **XP:** Attributes cost new dots × 5. Skills cost new dots × 3. In-clan Disciplines cost new dots × 5. Out-of-clan cost × 7.
