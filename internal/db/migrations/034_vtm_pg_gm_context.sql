-- 034_vtm_pg_gm_context.sql: Append Player's Guide lore to VtM gm_context.
-- Adds: new clan banes + compulsions, Blood Potency table, character type narration,
-- Oblivion Stain triggers, new predator type flavor.
UPDATE rulesets SET gm_context = gm_context || '

PLAYER''S GUIDE CLANS (7 additional clans):

CLAN BANES:
- Banu Haqim: Bane of Judgment — must spend vitae equal to Bane Severity when they harm an innocent, taking aggravated damage if they cannot.
- Hecata: Bane of the Living Grave — cannot feed from the dead or those with less than half their blood remaining, and must feed from the living only.
- Lasombra: Bane of the Abyss — cast no reflection, do not appear on cameras or recording devices, animals shy from them, +Bane Severity to Masquerade breaches caused by their nature.
- Ministry: Bane of the Unbound — when facing temptation relevant to their Vice, must succeed at a Resolve + Composure roll (difficulty = Bane Severity) or act on it.
- Ravnos: Bane of Rötschreck — sunlight triggers Rötschreck automatically at Bane Severity difficulty, fire is +Bane Severity difficulty to resist, and Ravnos burn faster than other Kindred.
- Salubri: Bane of the Third Eye — a third eye opens on their brow whenever they use Disciplines, Auspex use is visible to anyone with Auspex 1+, and mortals who see it must succeed at a Composure + Occult roll or flee.
- Tzimisce: Bane of Old Soil — must sleep with at least 2 lbs of earth from their homeland or haven, and failing to do so means rising each night at maximum Hunger for the first action.

PLAYER''S GUIDE CLAN COMPULSIONS:
- Banu Haqim: Judgment — must condemn or expose a witnessed crime, cannot let wrongdoing pass without response.
- Hecata: Morbidity — fixated on death, must examine, contemplate, or discuss death before any other action.
- Lasombra: Ruthlessness — must dominate or destroy opposition, compromise is failure.
- Ministry: Transgression — must tempt or corrupt someone present before acting on any other goal.
- Ravnos: Tempted — seized by aimless thrill-seeking, steals, lies, wanders, or courts danger for amusement.
- Salubri: Affective — compelled to heal and protect, cannot act violently without offering mercy first.
- Tzimisce: Covetous — must claim, mark, or control territory, objects, or people before other actions.

BLOOD POTENCY (BP) SUMMARY:
- BP 0 (Thin-Blood): No Disciplines except Thin-Blood Alchemy, Resonance is muted, cannot create ghouls or Embrace, Hunger maxes at 4.
- BP 1-2 (Neonate): All mortal blood acceptable, Bane Severity 1-2, standard Discipline access.
- BP 3-4 (Ancilla): Mortal blood only partially satisfying — needs 2+ Rouse Checks worth of blood per feeding, Bane Severity 2-3, +1 to Discipline power rolls.
- BP 5 (Elder): Only vampire vitae or 3+ mortals per Rouse Check, Bane Severity 3, +2 to Discipline power rolls, slumber is deeper.
- BP 6+ (Methuselah): Sustained only by Kindred blood of significant potency, Bane Severity 3+, near-mythic power.

CHARACTER TYPE NARRATION:
- Vampire: Standard Kindred rules apply. Hunger, Rouse Checks, Frenzy, Masquerade, Clan Compulsions all active.
- Thin-Blooded: Hunger maxes at 4 (not 5). No clan Disciplines — use Thin-Blood Alchemy instead (improvised, alchemical powers based on Resonance). More mortal in some ways: can walk in sunlight longer (Bane Severity aggravated at Dawn, not full sun). Blood Potency 0.
- Ghoul: Mortal servant granted vitae by a domitor. No Hunger track. Has one Discipline at 1 dot (their domitor''s gift). Bond Strength 1-3 (blood bond, 3 is fully bound). Ghouls age normally without monthly vitae, and withdrawal causes painful degeneration. The domitor''s clan resonance affects the ghoul''s emotions.
- Mortal: A fully human character. No Disciplines, no Hunger. Can be an ally, a Touchstone, or an antagonist. Treat mortal wounds as aggravated damage when narrating severe injury.

OBLIVION STAINS:
Whenever a character uses Oblivion (Disciplines: Oblivion), there is a risk of Stains — permanent marks of corruption on the soul:
- Actively causing the death of an innocent with Oblivion power: 1-3 Stains (severity by circumstances).
- Using Oblivion in a way that degrades or humiliates the dead: 1 Stain.
- Invoking Oblivion in the presence of a Touchstone without their understanding: 1 Stain.
Stains add to the character''s Humanity degradation track. Narrate Stains as a creeping darkness behind the eyes, a faint smell of the grave, or shadows that move wrong.'
WHERE name = 'vtm';
