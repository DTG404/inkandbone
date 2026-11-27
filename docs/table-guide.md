# At the table

Use **Manage** to select the campaign, character and session. A campaign holds the world across sessions. A session holds one stretch of play and its transcript. The ruleset determines the fields on the character sheet and the guidance supplied to the GM. [Play your first turn](../README.md#play-your-first-turn)

![Campaign management with the tabs for campaigns, characters and sessions](ui-manage-panel.png)

## Find something during play

| Where | What to look for |
| --- | --- |
| Character sheet | Attributes, resource tracks, inventory and ruleset-specific fields. Click a rollable skill or attribute to roll it. |
| Story | Read or search the transcript. Send the action your character takes, then watch the GM’s reply stream in. |
| Journal and world tools | Revisit a recap, look up an NPC or relationship, or keep a note for a detail you want later. |
| GM tools | Track objectives, manage encounters, prepare a pre-session brief or look for unresolved story threads. The oracle offers an action or theme when you need inspiration. |
| Maps | Generate a location map, add pins and move tokens. With the illustration worker configured, reveal named zones as the party explores. |
| Manage → Automation | Choose which background tasks run, including NPC extraction, stat updates, objectives, items, recaps and automatic maps. Updates can arrive after the GM finishes speaking. |

![Map view with locations marked for the session](ui-map.png)

Some mechanics belong to a particular system. The [Vampire V5 guide](games/vtm.md) covers Hunger and the Masquerade. The [Ironsworn guide](games/ironsworn.md) covers vows and momentum.

## Pick up a session

The action box saves an unsent draft for each session in this browser. If a connection drops during a turn, reopen the table and choose **Resume GM response**. An accepted action keeps its roll and completed answer when you reconnect.

After a break of half an hour, **Where were we?** opens with your character’s last known location, pending decision and known leads. You can also open it beside the action box at any time. **Ask the GM** uses the same public narration addressed to your character. Asking leaves the transcript, rolls, inventory, game clock and soundtrack alone. A detail your character hasn’t learned stays unknown.

Open **Changes** beneath a GM reply to inspect its recorded updates. Undo an individual update there. If someone has since edited that value, the table preserves their edit. When a rulebook passage was supplied for a reply, expand its source to read the saved heading and exact passage.

## Use a touch map

Open the map drawer and choose **Tokens**. Tap a character or NPC, then tap its position on the map. To move a placed token, tap it and then tap its destination. In the zone editor, tap two opposite corners, enter a name and choose **Add**. Use **Reveal** when the party discovers that zone.

## Move a campaign

In **Manage → Campaigns**, download the campaign archive. It includes private GM material, whispers and artwork. Keep it somewhere appropriate for the GM. Importing creates a separate campaign with its own IDs and transcript. The destination needs the referenced ruleset and matching installed rulebook sources. Select the restored campaign, character and session to continue play.

## Add a book to the compendium

You can play with a built-in sheet before adding any books. Import a book when you want passages from that edition or supplement available during play.

1. Open **Manage → Rulebooks** and select a campaign using the book’s ruleset. Campaigns using that ruleset share its sources.
2. Choose your PDF, plain text or Markdown file. Give it a distinct source label, such as *Core rules* or *Harbor gazetteer*. Reusing a label replaces the indexed text for that source.
3. Upload it and check the source name and indexed chunk count. Open **Compendium** and search for a distinctive term from the book to check the extracted text.

PDF import requires `pdftotext` on the server (`poppler-utils` on Ubuntu). For image-only scans, supply an OCR text export. Relevant excerpts may go to the selected GM provider during play. [Provider data flow](security.md#what-model-providers-receive)

## Sound at the table

The header has mute, volume and ambient playback controls. The browser synthesizes short effects for events such as a roll or combat starting. On a table connected to Syrinscape, the sound service uses the active session’s public narration to choose what the character can hear. The subscription supplies one shared playback session. [Syrinscape setup and playback checks](../pilots/syrinscape/README.md)
