# Ink & Bone

A self-hosted AI Game Master for 14 tabletop RPG systems. You type in your browser. The AI GM narrates. Your dashboard updates live.

![A session in the Worn Grimoire dashboard, with a character sheet beside the story transcript and campaign tools](docs/ui-screenshot.png)

## Play your first turn

Open your table in a browser. If you’re setting it up yourself, start with [Run a local table](#run-a-local-table).

1. Choose **Open campaign management** on the welcome screen, or **Manage** from the dashboard.
2. In **Campaigns**, give the campaign a name and choose a ruleset. Try **Ironsworn** for the example below. Click **Create Campaign**, then **Set Active** on its row.
3. Reopen **Manage** as needed. Under **Characters**, create a character for that campaign and set it active. Under **Sessions**, create a session and set it active.
4. In **What do you do?**, describe one action your character takes. The GM’s reply streams into the story.

An example turn in Ironsworn might go like this:

> **You:** I offer the ferryman a day’s rations if he’ll take me across before sundown.
>
> **GM:** He tucks the food under his coat and frees the rope. Halfway across, he rests his oars. “If they ask at the gate, you came over the ridge. Understand?”

The crossing gives you a choice for your next turn. It also costs something you can track: if your Supply was 5 and you spend 1, check that the sheet now reads 4. When you reopen the session, the transcript and sheet are there to pick up where you left off.

## Run a local table

You’ll need Git, Make, Go 1.26.11, and Node.js 24.18.0 with npm. This builds one executable with the browser UI inside. [Development and verification](docs/development.md)

```bash
git clone https://github.com/DTG404/inkandbone.git
cd inkandbone
make build
```

If you already run Ollama locally, use a chat model you’ve pulled. In Bash, enter its name exactly as it appears in `ollama list`:

```bash
read -rp 'Installed Ollama model: ' model
OLLAMA_MODEL="$model" ./ttrpg
```

Open **http://127.0.0.1:7432** and leave the process running while you play. Your campaign database defaults to `~/.ttrpg/ttrpg.db`. If you prefer DeepSeek, Anthropic, OpenRouter, or a model on another machine, follow [AI provider setup](docs/providers.md). To keep a private table online after logout or reboot, follow [Host a private table](docs/hosting.md).

## What comes to the table

| During play | What you can do |
| --- | --- |
| Character and combat | Roll from the sheet, track initiative and wounds, and manage equipment. The sheet follows your chosen ruleset. |
| Campaign journal | Follow objectives, revisit NPCs and relationships, or read a session recap before your next move. |
| Rulebooks | Start with a built-in sheet. Import your own PDF or text when you want the GM to consult a particular book. [Player guide](docs/table-guide.md) |
| Maps | Generate a location map and place pins. The optional signed-in map worker adds tactical illustrations with named zones you can reveal. [Illustrated map setup](deploy/map-worker/README.md) |
| Sound | Hear browser sound effects for rolls and combat. Connect a Syrinscape subscription for automatic transitions based on what the GM describes your character perceiving. [Syrinscape setup](pilots/syrinscape/README.md) |

## Choose a ruleset

Each link opens the play guide for that system.

| | |
| --- | --- |
| [Blades in the Dark](docs/games/blades.md) | [Paranoia](docs/games/paranoia.md) |
| [Call of Cthulhu](docs/games/coc.md) | [Shadowrun](docs/games/shadowrun.md) |
| [Cyberpunk Red](docs/games/cyberpunk.md) | [Star Wars: Edge of the Empire](docs/games/starwars.md) |
| [D&D 5th Edition](docs/games/dnd5e.md) | [The One Ring](docs/games/theonering.md) |
| [Dune: Adventures in the Imperium](docs/games/dune.md) | [Vampire: The Masquerade V5](docs/games/vtm.md) |
| [Ironsworn](docs/games/ironsworn.md) | [Warhammer Fantasy Roleplay](docs/games/wfrp.md) |
| [Legend of the Five Rings](docs/games/l5r.md) | [Wrath & Glory](docs/games/wrath_glory.md) |

## Guides and reference

- [Player guide](docs/table-guide.md) for the dashboard, automations and compendium.
- [API and database reference](docs/technical-reference.md) for integrations and custom rulesets.
- [Security and provider data flow](docs/security.md) and [backup and recovery](docs/data-recovery.md) for table operators.
- [Development](docs/development.md) and [browser test coverage](docs/testing/e2e-coverage-matrix.md) for contributors.

[MIT license](LICENSE)
