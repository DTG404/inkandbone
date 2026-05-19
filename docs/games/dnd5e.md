# D&D 5th Edition — How to Play

You're an adventurer. There's a world in trouble, dungeons to explore, monsters to fight, treasure to find. The AI is your Dungeon Master — it narrates the world, voices the NPCs, runs combat, and adjudicates the rules.

## Your Character Sheet

Create through **⚙ Manage → Characters → + New Character**. Inkandbone auto-rolls abilities (4d6 drop lowest for each of the six), sets Level 1, HP 10, AC 10, Proficiency Bonus 2.

**The six ability scores** — STR, DEX, CON, INT, WIS, CHA — are what you roll against. Each is a pip rating (3-20). Your class determines which ones matter most.

**HP** is your health bar. Rest to recover it.

**Level** determines your power. When your XP reaches thresholds (300 → level 2, 900 → level 3, etc.), you can level up. Leveling increases your HP by 5, and your Proficiency Bonus auto-computes as `floor((level-1)/4)+2`.

**AC** is your Armor Class. Enemies need to roll this high to hit you.

**Proficiency Bonus** is computed from your level automatically. It applies to attacks with weapons you're proficient in, skills you're trained in, and saving throws.

**Skills, Inventory, Spells, Features** are free-text. List what your character has and can do.

## Playing

Open the browser. Get your campaign, character, and session active. Type what your character does:

> *I grip my sword and step through the doorway. "Who's there?" I call into the darkness.*

The AI describes what happens, asks for rolls when there's uncertainty, and builds the world around you.

When combat breaks out, the AI tracks initiative. The turn order strip appears at the top of the story column — each combatant listed with initiative order. The active combatant is highlighted. Dead ones (0 HP) are dimmed. On your turn, describe what you do:

> *I swing my longsword at the goblin in front of me.*

The AI resolves the attack, applies damage, and moves to the next turn.

## How the AI Runs D&D

The AI manages encounters, NPCs, exploration, traps, puzzles, and social interactions. It tracks your HP changes, spell slots (if you list them in your notes), and conditions.

**Dice rolls** use standard expressions: `1d20`, `2d6+3`, `1d8-1`. Type them inline in your message and the AI rolls them.

**XP and leveling** happen through story. When you defeat enemies or complete objectives, the automation detects the XP change and applies it. When you have enough XP for the next level, the ⬆ Advance button in the header lights up. Click it for AI-suggested advancements.

**Combat** is turn-based. The AI tracks initiative order, HP, conditions, and death saves. When you drop to 0 HP, the AI starts death saving throws.

## What a Session Looks Like

**The hook.** Someone needs something done. A villager at the tavern, a patron with a job, a rumor about ruins nearby.

**The journey.** Travel to the location. The AI describes the journey — the environment, encounters along the way, chances to gather information.

**The dungeon or encounter.** Explore. The AI describes what you see. You decide what to investigate, what to fight, what to avoid. One combat encounter tests your resources.

**The reward.** Treasure, information, a clue to a bigger threat, a level-up. The session ends with a thread pulling you toward the next one.

## Key Rules

- **Ability checks:** The AI sets a DC. You roll d20 + ability modifier + proficiency (if trained).
- **Advantage/Disadvantage:** Roll twice, take the higher or lower. The AI tells you when this applies.
- **Short rests:** Recover some HP using Hit Dice. Tell the AI when you rest.
- **Long rests:** Full HP recovery, spell slots back. The AI decides if it's safe.
- **Death saves:** At 0 HP, roll d20 each turn. 10+ is a success, 1 is two failures, 20 is conscious with 1 HP. Three successes = stable. Three failures = death.
- **Leveling up:** XP thresholds follow the standard 5e progression. Your AI will suggest what to increase.
