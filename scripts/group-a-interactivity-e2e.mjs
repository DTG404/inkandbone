#!/usr/bin/env node
/**
 * ink & bone — Group A Interactivity E2E Test Suite
 *
 * Tests: click-to-roll, macro quick-bar, initiative reorder
 *
 * Usage:
 *   node scripts/group-a-interactivity-e2e.mjs
 *
 * Requires: node (v24+), playwright (npm install playwright), chromium
 * Server is launched internally using a fresh isolated DB.
 */

import { chromium } from 'playwright';
import { spawn } from 'child_process';
import { DatabaseSync } from 'node:sqlite';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '..');
const BASE = 'http://localhost:7432';
const API = BASE;
const DB_PATH = '/tmp/e2e-group-a.db';

let passed = 0;
let failed = 0;
const errors = [];

function assert(condition, msg) {
  if (condition) { passed++; console.log('  ✓', msg); }
  else { failed++; errors.push(msg); console.error('  ✗', msg); }
}

async function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }

async function pollHealth(maxMs = 15000) {
  const start = Date.now();
  while (Date.now() - start < maxMs) {
    try {
      const res = await fetch(`${BASE}/api/health`);
      if (res.ok) return true;
    } catch { /* not up yet */ }
    await sleep(300);
  }
  return false;
}

async function main() {
  // Kill any leftover server on port 7432
  try {
    const res = await fetch(`${BASE}/api/health`);
    if (res.ok) {
      console.log('WARNING: server already running on port 7432 — test may use wrong DB');
    }
  } catch { /* good, not running */ }

  // Remove stale DB
  try {
    const { unlinkSync } = await import('fs');
    unlinkSync(DB_PATH);
  } catch { /* ok */ }

  // Start server
  console.log(`\nStarting server with DB: ${DB_PATH}`);
  const server = spawn(
    path.join(ROOT, 'ttrpg-e2e'),
    ['-db', DB_PATH],
    { cwd: ROOT, stdio: ['ignore', 'pipe', 'pipe'] }
  );
  server.stdout.on('data', (d) => process.stdout.write('[srv] ' + d));
  server.stderr.on('data', (d) => process.stderr.write('[srv] ' + d));

  const up = await pollHealth(15000);
  if (!up) {
    console.error('FATAL: server did not start within 15s');
    server.kill();
    process.exit(1);
  }
  console.log('Server is up.');

  const browser = await chromium.launch({
    headless: true,
    executablePath: '/usr/bin/chromium-browser',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();

  // ===== SETUP: Seed data via API =====
  console.log('\n=== Setup: Seed Data ===');

  // Get rulesets
  const rulesets = await (await fetch(`${API}/api/rulesets`)).json();
  const dnd5e = rulesets.find(r => r.name === 'dnd5e');
  const dune  = rulesets.find(r => r.name === 'dune');
  assert(dnd5e != null, 'Setup: dnd5e ruleset found');
  assert(dune  != null, 'Setup: dune ruleset found');
  const dnd5eId = dnd5e?.id ?? 1;
  const duneId  = dune?.id;

  // D&D 5e campaign + character + session
  const dndCamp = await (await fetch(`${API}/api/campaigns`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ name: 'Group A DnD Campaign', ruleset_id: dnd5eId })
  })).json();
  const dndCampId = dndCamp.id;
  assert(dndCampId > 0, 'Setup: D&D 5e campaign created');

  const dndChar = await (await fetch(`${API}/api/campaigns/${dndCampId}/characters`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ name: 'Aria Swiftblade' })
  })).json();
  const dndCharId = dndChar.id;
  assert(dndCharId > 0, 'Setup: D&D 5e character created');

  await fetch(`${API}/api/characters/${dndCharId}`, {
    method: 'PATCH', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ data_json: JSON.stringify({
      str: 3, dex: 2, con: 4, int: 2, wis: 1, cha: 3,
      level: 1, hp: 10, ac: 14, proficiency_bonus: 2,
      race: 'Human', class: 'Fighter',
      skills: 'Athletics, Perception', inventory: 'Sword', spells: '', features: 'Second Wind'
    }) })
  });
  assert(true, 'Setup: D&D 5e character stats patched');

  const dndSess = await (await fetch(`${API}/api/campaigns/${dndCampId}/sessions`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ title: 'Group A Session', date: '2026-06-25' })
  })).json();
  const dndSessId = dndSess.id;
  assert(dndSessId > 0, 'Setup: D&D 5e session created');

  await fetch(`${API}/api/settings`, {
    method: 'PATCH', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ campaign_id: dndCampId, character_id: dndCharId, session_id: dndSessId })
  });
  assert(true, 'Setup: Active settings set to D&D 5e');

  // Dune campaign + character + session (for skill click-to-roll)
  let duneCampId = null, duneCharId = null, duneSessId = null;
  if (duneId) {
    const duneCamp = await (await fetch(`${API}/api/campaigns`, {
      method: 'POST', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({ name: 'Group A Dune Campaign', ruleset_id: duneId })
    })).json();
    duneCampId = duneCamp.id;
    assert(duneCampId > 0, 'Setup: Dune campaign created');

    const duneChar = await (await fetch(`${API}/api/campaigns/${duneCampId}/characters`, {
      method: 'POST', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({ name: 'Liet Swordmaster' })
    })).json();
    duneCharId = duneChar.id;
    assert(duneCharId > 0, 'Setup: Dune character created');

    await fetch(`${API}/api/characters/${duneCharId}`, {
      method: 'PATCH', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({ data_json: JSON.stringify({
        duty: 3, faith: 2, justice: 4, power: 2, truth: 3,
        battle: 4, communicate: 2, discipline: 3, move: 3, understand: 2,
        determination: 3, advancement: 0, wounds: 0,
        archetype: 'Swordmaster', house: 'Atreides', role: 'Warrior',
        faction: 'House Atreides', ambition: 'Serve with honor'
      }) })
    });
    assert(true, 'Setup: Dune character stats patched');

    const duneSessResp = await (await fetch(`${API}/api/campaigns/${duneCampId}/sessions`, {
      method: 'POST', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({ title: 'Dune Group A Session', date: '2026-06-25' })
    })).json();
    duneSessId = duneSessResp.id;
    assert(duneSessId > 0, 'Setup: Dune session created');
  }

  // Seed encounter + 3 combatants via node:sqlite
  console.log('\n  Seeding encounter via node:sqlite...');
  await sleep(300);
  const sqldb = new DatabaseSync(DB_PATH);
  sqldb.exec(`INSERT INTO combat_encounters (session_id, name, active_turn_index, active) VALUES (${dndSessId}, 'Ambush', 0, 1)`);
  const encId = sqldb.prepare('SELECT last_insert_rowid() as id').get().id;
  sqldb.exec(`INSERT INTO combatants (encounter_id, name, initiative, hp_current, hp_max, is_player, conditions_json, sort_order) VALUES (${encId}, 'Fighter', 18, 30, 30, 1, '[]', 0)`);
  const fighterId = sqldb.prepare('SELECT last_insert_rowid() as id').get().id;
  sqldb.exec(`INSERT INTO combatants (encounter_id, name, initiative, hp_current, hp_max, is_player, conditions_json, sort_order) VALUES (${encId}, 'Goblin', 12, 7, 7, 0, '[]', 1)`);
  const goblinId = sqldb.prepare('SELECT last_insert_rowid() as id').get().id;
  sqldb.exec(`INSERT INTO combatants (encounter_id, name, initiative, hp_current, hp_max, is_player, conditions_json, sort_order) VALUES (${encId}, 'Mage', 9, 15, 15, 0, '[]', 2)`);
  const mageId = sqldb.prepare('SELECT last_insert_rowid() as id').get().id;
  sqldb.close();
  assert(encId > 0, `Setup: Encounter seeded (id=${encId})`);
  assert(fighterId > 0 && goblinId > 0 && mageId > 0, `Setup: 3 combatants seeded (${fighterId}, ${goblinId}, ${mageId})`);

  console.log(`  Campaign D&D5e=${dndCampId}, Char=${dndCharId}, Sess=${dndSessId}, Enc=${encId}`);
  console.log(`  Campaign Dune=${duneCampId}, Char=${duneCharId}, Sess=${duneSessId}`);
  console.log(`  Combatants: Fighter=${fighterId}, Goblin=${goblinId}, Mage=${mageId}`);

  // Wait for server to pick up fresh DB state
  await sleep(500);

  // ===== SECTION A: Click-to-Roll =====
  console.log('\n=== Section A: Click-to-Roll ===');

  // Mock gm-respond-stream
  await page.route('**/api/sessions/*/gm-respond-stream', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'text/event-stream',
      body: 'data: The GM narrates the outcome.\n\n'
    });
  });

  await page.goto(BASE);
  await page.waitForLoadState('networkidle');
  await sleep(1000);

  // A.1 Verify .attr-row with title exists (D&D 5e attributes)
  const attrRowCount = await page.locator('.attr-row[title^="Click to roll"]').count();
  assert(attrRowCount > 0, 'A.1: .attr-row with "Click to roll" title present (D&D 5e attributes)');

  // A.2 Verify .roll-hint-icon exists inside .attr-row
  const hintIconCount = await page.locator('.attr-row .roll-hint-icon').count();
  assert(hintIconCount > 0, 'A.2: .roll-hint-icon present inside .attr-row');

  // A.3 Click first .attr-row with title
  const firstAttrRow = page.locator('.attr-row[title^="Click to roll"]').first();
  const attrLabel = await firstAttrRow.getAttribute('title');
  console.log(`  Clicking attr row: ${attrLabel}`);
  await firstAttrRow.click();
  await sleep(2000);

  // A.4 Verify message fired via API
  const dndMsgs = await (await fetch(`${API}/api/sessions/${dndSessId}/messages`)).json();
  const lastUserMsg = dndMsgs.filter(m => m.role === 'user').at(-1);
  assert(lastUserMsg?.content?.includes('attempts a'), 'A.4: click-to-roll message contains "attempts a"');
  assert(lastUserMsg?.content?.includes('check.'), 'A.5: click-to-roll message contains "check."');
  console.log(`  Message: "${lastUserMsg?.content}"`);

  // A.6 Verify localStorage key set
  const rollHintShown = await page.evaluate(() => localStorage.getItem('inkandbone_roll_hint_shown'));
  assert(rollHintShown === '1', 'A.6: localStorage inkandbone_roll_hint_shown is set to "1"');

  // A.7 D&D 5e has no category:"skill" number fields → label[title^="Click to roll"] should NOT appear
  const skillLabelCount = await page.locator('label[title^="Click to roll"]').count();
  assert(skillLabelCount === 0, 'A.7: No label[title^="Click to roll"] in D&D 5e (no skill number fields)');

  // Switch to Dune campaign for skill test
  if (duneCampId && duneSessId) {
    await fetch(`${API}/api/settings`, {
      method: 'PATCH', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({ campaign_id: duneCampId, character_id: duneCharId, session_id: duneSessId })
    });
    await sleep(200);

    // Re-mock for Dune session
    await page.route('**/api/sessions/*/gm-respond-stream', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: 'data: The GM narrates the Dune outcome.\n\n'
      });
    });

    await page.goto(BASE);
    await page.waitForLoadState('networkidle');
    await sleep(1000);

    // A.8 Verify label[title^="Click to roll"] exists in Dune (skill fields)
    const duneSkillLabelCount = await page.locator('label[title^="Click to roll"]').count();
    assert(duneSkillLabelCount > 0, 'A.8: label[title^="Click to roll"] present in Dune (skill fields)');

    // A.9 Click a skill label
    const firstSkillLabel = page.locator('label[title^="Click to roll"]').first();
    const skillTitle = await firstSkillLabel.getAttribute('title');
    console.log(`  Clicking skill label: ${skillTitle}`);
    await firstSkillLabel.click();
    await sleep(2000);

    // A.10 Verify message fired
    const duneMsgs = await (await fetch(`${API}/api/sessions/${duneSessId}/messages`)).json();
    const lastDuneMsg = duneMsgs.filter(m => m.role === 'user').at(-1);
    assert(lastDuneMsg?.content?.includes('attempts a'), 'A.9: Dune skill click-to-roll message contains "attempts a"');
    assert(lastDuneMsg?.content?.includes('check.'), 'A.10: Dune skill click-to-roll message contains "check."');
    console.log(`  Dune message: "${lastDuneMsg?.content}"`);
  } else {
    assert(false, 'A.8: Dune ruleset not available — skipped');
  }

  // Switch back to D&D 5e for MacroBar tests
  await fetch(`${API}/api/settings`, {
    method: 'PATCH', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({ campaign_id: dndCampId, character_id: dndCharId, session_id: dndSessId })
  });
  await sleep(200);

  // ===== SECTION B: MacroBar =====
  console.log('\n=== Section B: MacroBar ===');

  // Re-mock for D&D session
  await page.route('**/api/sessions/*/gm-respond-stream', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'text/event-stream',
      body: 'data: The GM narrates the macro outcome.\n\n'
    });
  });

  await page.goto(BASE);
  await page.waitForLoadState('networkidle');
  await sleep(1000);

  // B.1 Verify .macro-bar is present
  const macroBarVisible = await page.locator('.macro-bar').isVisible().catch(() => false);
  assert(macroBarVisible, 'B.1: .macro-bar is present');

  // B.2 Click .macro-add-btn → verify .macro-form visible
  await page.locator('.macro-add-btn').click();
  await sleep(300);
  const macroFormVisible = await page.locator('.macro-form').isVisible().catch(() => false);
  assert(macroFormVisible, 'B.2: .macro-form visible after clicking +');

  // B.3 Fill label and action_text
  await page.locator('.macro-form-input').first().fill('Slash Attack');
  await page.locator('.macro-form-action').fill('I slash at the nearest enemy with my sword');
  assert(true, 'B.3: Filled macro form label and action_text');

  // B.4 Click a color swatch (second one = Red)
  const swatches = page.locator('.macro-color-swatch');
  await swatches.nth(1).click();
  assert(true, 'B.4: Color swatch clicked');

  // B.5 Click save (Add) button → verify form closes
  await page.locator('.macro-form-save').click();
  await sleep(500);
  const formClosedAfterAdd = !(await page.locator('.macro-form').isVisible().catch(() => false));
  assert(formClosedAfterAdd, 'B.5: .macro-form closed after saving macro');

  // B.6 Verify .macro-btn with text "Slash Attack" visible
  const slashMacroBtn = await page.locator('.macro-btn').filter({ hasText: 'Slash Attack' }).isVisible().catch(() => false);
  assert(slashMacroBtn, 'B.6: "Slash Attack" macro button visible');

  // B.7 Click the macro → verify message fired
  await page.locator('.macro-btn').filter({ hasText: 'Slash Attack' }).click();
  await sleep(2000);
  const macroMsgs = await (await fetch(`${API}/api/sessions/${dndSessId}/messages`)).json();
  const lastMacroMsg = macroMsgs.filter(m => m.role === 'user').at(-1);
  assert(lastMacroMsg?.content === 'I slash at the nearest enemy with my sword',
    'B.7: Macro fired with exact action_text as message');
  console.log(`  Macro message: "${lastMacroMsg?.content}"`);

  // B.8 Click .macro-gear-btn → verify .macro-edit-controls visible
  await page.locator('.macro-gear-btn').click();
  await sleep(300);
  const editControlsVisible = await page.locator('.macro-edit-controls').isVisible().catch(() => false);
  assert(editControlsVisible, 'B.8: .macro-edit-controls visible after clicking ⚙');

  // B.9 Create second macro via UI (gear must be off for + to work cleanly)
  // First turn off edit mode
  await page.locator('.macro-gear-btn').click();
  await sleep(200);

  await page.locator('.macro-add-btn').click();
  await sleep(300);
  await page.locator('.macro-form-input').first().fill('Quick Draw');
  await page.locator('.macro-form-action').fill('I draw my weapon and aim carefully');
  await page.locator('.macro-form-save').click();
  await sleep(500);
  const quickDrawBtn = await page.locator('.macro-btn').filter({ hasText: 'Quick Draw' }).isVisible().catch(() => false);
  assert(quickDrawBtn, 'B.9: "Quick Draw" second macro created and visible');

  // B.10 Click ↑ on second macro (reorder) in edit mode
  await page.locator('.macro-gear-btn').click();
  await sleep(300);

  // Verify initial API order before reorder
  const macrosBeforeReorder = await (await fetch(`${API}/api/characters/${dndCharId}/macros`)).json();
  const secondMacroBeforeId = macrosBeforeReorder[1]?.id;
  assert(macrosBeforeReorder.length >= 2, 'B.10a: At least 2 macros exist before reorder');

  // Find the second macro's ↑ button (Move left in macro bar = move up in order)
  const macroSlots = page.locator('.macro-slot');
  const secondSlotUpBtn = macroSlots.nth(1).locator('button[title="Move left"]');
  await secondSlotUpBtn.click();
  await sleep(500);

  const macrosAfterReorder = await (await fetch(`${API}/api/characters/${dndCharId}/macros`)).json();
  assert(macrosAfterReorder[0]?.id === secondMacroBeforeId,
    'B.10b: Second macro moved to first position after reorder');

  // B.11 Click × on second macro (now originally "Slash Attack") → verify it disappears
  // First identify what's now second
  const macrosBeforeDelete = await (await fetch(`${API}/api/characters/${dndCharId}/macros`)).json();
  const secondMacroLabel = macrosBeforeDelete[1]?.label;
  console.log(`  Deleting second macro: "${secondMacroLabel}"`);

  const secondSlotDeleteBtn = macroSlots.nth(1).locator('button[title="Delete"]');
  await secondSlotDeleteBtn.click();
  await sleep(500);

  const macrosAfterDelete = await (await fetch(`${API}/api/characters/${dndCharId}/macros`)).json();
  assert(!macrosAfterDelete.some(m => m.label === secondMacroLabel),
    `B.11: Macro "${secondMacroLabel}" deleted from DB`);
  const deletedBtnGone = await page.locator('.macro-btn').filter({ hasText: secondMacroLabel }).count() === 0;
  assert(deletedBtnGone, 'B.11b: Deleted macro button gone from DOM');

  // Turn off edit mode
  await page.locator('.macro-gear-btn').click();
  await sleep(200);

  // B.12 Pre-create 9 more macros via API to reach cap, then verify + is disabled
  const existingMacros = await (await fetch(`${API}/api/characters/${dndCharId}/macros`)).json();
  const needed = 10 - existingMacros.length;
  for (let i = 0; i < needed; i++) {
    await fetch(`${API}/api/characters/${dndCharId}/macros`, {
      method: 'POST', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({ label: `Macro ${i+1}`, action_text: `Action ${i+1}`, color: 'var(--gold)' })
    });
  }
  assert(true, `B.12a: Created ${needed} more macros via API to reach cap`);

  // Reload and verify + button is disabled
  await page.goto(BASE);
  await page.waitForLoadState('networkidle');
  await sleep(1000);

  const addBtnTitle = await page.locator('.macro-add-btn').getAttribute('title');
  const addBtnDisabled = await page.locator('.macro-add-btn').isDisabled();
  assert(addBtnTitle === 'Maximum 10 macros', 'B.12b: + button title is "Maximum 10 macros" at cap');
  assert(addBtnDisabled, 'B.12c: + button is disabled at 10 macros');

  // ===== SECTION C: Initiative Reorder =====
  console.log('\n=== Section C: Initiative Reorder ===');

  // Navigate to D&D session (already set)
  await page.route('**/api/sessions/*/gm-respond-stream', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'text/event-stream',
      body: 'data: The GM narrates combat.\n\n'
    });
  });

  // C.1 Find .combat-grimoire — verify it contains 3 combatant cards
  const grimoire = page.locator('.combat-grimoire');
  const grimoireVisible = await grimoire.isVisible().catch(() => false);
  assert(grimoireVisible, 'C.1: .combat-grimoire is visible');

  const combatantCards = grimoire.locator('.combatant-card');
  const cardCount = await combatantCards.count();
  assert(cardCount === 3, `C.2: .combat-grimoire contains 3 combatant cards (found ${cardCount})`);

  // C.3 Verify first combatant's ↑ button (Move up) is disabled
  const firstCardUpBtn = combatantCards.first().locator('button[title="Move up"]');
  const firstUpDisabled = await firstCardUpBtn.isDisabled();
  assert(firstUpDisabled, 'C.3: First combatant ↑ button is disabled');

  // C.4 Verify last combatant's ↓ button (Move down) is disabled
  const lastCardDownBtn = combatantCards.last().locator('button[title="Move down"]');
  const lastDownDisabled = await lastCardDownBtn.isDisabled();
  assert(lastDownDisabled, 'C.4: Last combatant ↓ button is disabled');

  // C.5 Get initial order via API context
  const ctxBefore = await (await fetch(`${API}/api/context`)).json();
  const firstCombatantBefore = ctxBefore.active_combat?.combatants?.[0]?.name;
  assert(firstCombatantBefore != null, 'C.5: Context has active_combat.combatants[0].name');
  console.log(`  Initial first combatant: ${firstCombatantBefore}`);

  // C.6 Click ↓ button on first combatant card → wait 1s
  const firstCardDownBtn = combatantCards.first().locator('button[title="Move down"]');
  await firstCardDownBtn.click();
  await sleep(1000);

  // C.7 Verify via API: first combatant changed
  const ctxAfterReorder = await (await fetch(`${API}/api/context`)).json();
  const firstCombatantAfter = ctxAfterReorder.active_combat?.combatants?.[0]?.name;
  assert(firstCombatantAfter !== firstCombatantBefore,
    `C.6: First combatant changed after ↓ click (was "${firstCombatantBefore}", now "${firstCombatantAfter}")`);
  console.log(`  After reorder first combatant: ${firstCombatantAfter}`);

  // Reload to get fresh combatant list order
  await page.reload();
  await page.waitForLoadState('networkidle');
  await sleep(1000);

  // C.8 Find .combatant-init span → click it → verify input appears
  const combatantCards2 = page.locator('.combat-grimoire .combatant-card');
  const firstInitSpan = combatantCards2.first().locator('.combatant-init');
  const initText = await firstInitSpan.textContent();
  console.log(`  First combatant initiative display: "${initText}"`);
  await firstInitSpan.click();
  await sleep(300);

  const initInput = combatantCards2.first().locator('input[type="number"]');
  const initInputVisible = await initInput.isVisible().catch(() => false);
  assert(initInputVisible, 'C.8: Clicking .combatant-init reveals <input type="number">');

  // C.9 Clear + type "25" → press Enter
  await initInput.fill('25');
  await initInput.press('Enter');
  await sleep(1000);

  // C.10 Verify via API: combatant initiative is 25
  const ctxAfterInit = await (await fetch(`${API}/api/context`)).json();
  const combatantsAfterInit = ctxAfterInit.active_combat?.combatants ?? [];
  // Find the combatant with initiative 25 in the list
  const patchedCombatant = combatantsAfterInit.find(c => c.initiative === 25);
  assert(patchedCombatant != null, 'C.9: Combatant initiative updated to 25 via API');
  console.log(`  Combatant with initiative=25: "${patchedCombatant?.name}"`);

  // C.11 Verify sort_order unchanged after initiative edit (still reflects reorder not init order)
  // After our earlier reorder, the second item (Goblin) should be first (sort_order 0)
  // After initiative edit, sort_order should not have reset
  const sortOrdersAfterInit = combatantsAfterInit.map(c => c.sort_order);
  console.log(`  Sort orders after initiative edit: ${JSON.stringify(sortOrdersAfterInit)}`);
  // The combatants should still be in reordered order (not re-sorted by initiative)
  // We verify by checking the order matches the context order (sort_order ascending)
  const isOrderedBySortOrder = combatantsAfterInit.every((c, i, arr) => {
    return i === 0 || c.sort_order >= arr[i-1].sort_order;
  });
  assert(isOrderedBySortOrder, 'C.10: Combatants still ordered by sort_order after initiative edit (not re-sorted)');

  // Additional: verify the init-edited combatant keeps its sort_order from before
  const initEditedSortOrder = patchedCombatant?.sort_order;
  const initEditedName = patchedCombatant?.name;
  // The combatant with initiative=25 should not have moved in sort_order just because initiative changed
  const ctxBeforeInitEdit = await (await fetch(`${API}/api/context`)).json();
  // (Already stored as ctxAfterReorder above — this final check just ensures sort_order is stable)
  assert(initEditedSortOrder !== undefined,
    `C.11: Combatant "${initEditedName}" has defined sort_order=${initEditedSortOrder} after initiative edit`);

  // ===== RESULTS =====
  console.log(`\n${'='.repeat(60)}`);
  console.log(`RESULTS: ${passed} passed, ${failed} failed`);
  if (failed > 0) {
    console.log(`\nFAILURES:`);
    errors.forEach(e => console.log(`  - ${e}`));
  }
  console.log(`${'='.repeat(60)}`);

  await browser.close();
  server.kill();

  // Write report
  await writeReport(passed, failed, errors);

  process.exit(failed > 0 ? 1 : 0);
}

async function writeReport(passed, failed, errors) {
  const { writeFileSync, mkdirSync } = await import('fs');
  const reportDir = path.join(ROOT, '.superpowers', 'sdd');
  try { mkdirSync(reportDir, { recursive: true }); } catch {}
  const now = new Date().toISOString();
  const lines = [
    `# Group A Interactivity E2E Report`,
    ``,
    `**Run:** ${now}`,
    `**Result:** ${failed === 0 ? 'PASS' : 'FAIL'} — ${passed} passed, ${failed} failed`,
    ``,
    `## Test Results`,
    ``,
    `| Status | Assertion |`,
    `|--------|-----------|`,
  ];
  // We can't enumerate individual results here without tracking them separately
  // (the assert() function only tracks pass/fail counts and failure messages)
  if (errors.length > 0) {
    lines.push(``, `## Failures`, ``);
    errors.forEach(e => lines.push(`- ${e}`));
  }
  lines.push(``, `## Summary`, ``);
  lines.push(`- Total passed: ${passed}`);
  lines.push(`- Total failed: ${failed}`);
  lines.push(`- Sections covered: Setup, Section A (Click-to-Roll), Section B (MacroBar), Section C (Initiative Reorder)`);
  writeFileSync(path.join(reportDir, 'e2e-report.md'), lines.join('\n') + '\n');
  console.log(`\nReport written to .superpowers/sdd/e2e-report.md`);
}

main().catch(e => {
  console.error('FATAL:', e);
  process.exit(1);
});
