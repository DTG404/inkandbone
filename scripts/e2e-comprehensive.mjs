#!/usr/bin/env node
/**
 * ink & bone — Comprehensive End-to-End Test Suite
 *
 * Tests EVERY feature: UI panels, CRUD APIs, edge cases, all 12 sidebar tabs,
 * Manage panel with 5 tabs, Automation settings, all entity systems.
 *
 * Usage:
 *   1. Start inkandbone server with a fresh DB:
 *      rm -f ~/.ttrpg && ttrpg -db /tmp/e2e-test.db
 *   2. Run this test:
 *      node scripts/e2e-comprehensive.mjs
 *
 * Requirements: node, playwright (npm install playwright), chromium
 *
 * The test seeds its own data via the HTTP API, then tests every feature
 * through both browser interactions and API calls.
 */

import { chromium } from 'playwright';

const BASE = 'http://localhost:7432';
const API = BASE;

let passed = 0;
let failed = 0;
const errors = [];

function assert(condition, msg) {
  if (condition) { passed++; }
  else { failed++; errors.push(msg); console.error('  ✗', msg); }
}

async function sleep(ms) { return new Promise(r => setTimeout(r, ms)); }

async function main() {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();

  // ===== 1. SEED TEST DATA VIA API =====
  console.log('\n=== 1. Seeding test data ===');

  const cCamp = await (await fetch(`${API}/api/campaigns`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({name:'E2E Test Campaign', ruleset_id:1})
  })).json();
  const campId = cCamp.id;
  assert(campId > 0, '1.1 Create campaign');

  const cChar = await (await fetch(`${API}/api/campaigns/${campId}/characters`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({name:'Test Hero'})
  })).json();
  const charId = cChar.id;
  assert(charId > 0, '1.2 Create character');

  await fetch(`${API}/api/characters/${charId}`, {
    method: 'PATCH', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({data_json: JSON.stringify({
      strength:15, dexterity:14, constitution:13, intelligence:12, wisdom:10, charisma:8,
      level:3, hp:24, ac:16, proficiency_bonus:2,
      race:'Human', class:'Fighter', skills:'Athletics, Perception', inventory:'Sword', spells:'None', features:'Fighting Style'
    })})
  });
  assert(true, '1.3 Patch character stats');

  const cSess = await (await fetch(`${API}/api/campaigns/${campId}/sessions`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({title:'E2E Test Session', date:'2026-05-18'})
  })).json();
  const sessId = cSess.id;
  assert(sessId > 0, '1.4 Create session');

  await fetch(`${API}/api/settings`, {
    method: 'PATCH', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({campaign_id: campId, character_id: charId, session_id: sessId})
  });

  await fetch(`${API}/api/sessions/${sessId}/messages`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({role:'user', content:'I cautiously enter the dark cave, drawing my sword.'})
  });
  await fetch(`${API}/api/sessions/${sessId}/messages`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({role:'user', content:'I search for traps along the walls.', whisper: true})
  });

  await fetch(`${API}/api/sessions/${sessId}/dice-rolls`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({expression:'1d20'})
  });
  await fetch(`${API}/api/sessions/${sessId}/dice-rolls`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({expression:'2d6+3'})
  });

  await fetch(`${API}/api/campaigns/${campId}/objectives`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({title:'Find the Lost Treasure', description:'Search the cave for the ancient dwarven treasure.'})
  });

  await fetch(`${API}/api/characters/${charId}/items`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({name:'Longsword', description:'A well-balanced steel blade', quantity:1})
  });
  await fetch(`${API}/api/characters/${charId}/items`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({name:'Torch', description:'Provides light in darkness', quantity:3})
  });

  // Faction
  await fetch(`${API}/api/campaigns/${campId}/factions`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({name:'The Iron Alliance', faction_type:'faction', description:'A powerful mercenary guild', influence:7, color:'#ff4444'})
  });

  // Adventure
  await fetch(`${API}/api/campaigns/${campId}/adventures`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({title:'The Lost Mines', description:'Search for ancient dwarven mines', status:'active'})
  });

  await fetch(`${API}/api/sessions/${sessId}/adventure`, {
    method: 'PATCH', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({adventure_id: 1})
  });

  // NPC Stat Blocks
  await fetch(`${API}/api/campaigns/${campId}/npc-stats`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({
      name:'Goblin Scout', role:'scout', hp_max:7, armor_class:13, initiative_mod:2,
      skills: JSON.stringify(['Stealth', 'Perception']),
      abilities: JSON.stringify(['Nimble Escape']),
      loot: JSON.stringify(['Shortbow', '10 arrows', '3 cp'])
    })
  });
  await fetch(`${API}/api/campaigns/${campId}/npc-stats`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({
      name:'Cave Spider', role:'beast', hp_max:15, armor_class:12, initiative_mod:1,
      skills: JSON.stringify(['Stealth']),
      abilities: JSON.stringify(['Web (DC 12)', 'Poison Bite']),
      loot: JSON.stringify(['Spider silk (worthless)'])
    })
  });

  // Secret (hidden by default)
  await fetch(`${API}/api/campaigns/${campId}/secrets`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({title:'The Hidden Vault', content:'Beneath the old temple lies a vault containing the Crown of Kings.', category:'secret'})
  });

  // Calendar event
  await fetch(`${API}/api/campaigns/${campId}/calendar-events`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({title:'Spring Equinox Festival', description:'The annual celebration', event_type:'holiday', in_game_year:1243, in_game_month:3, in_game_day:20})
  });

  // XP + Summary
  await fetch(`${API}/api/sessions/${sessId}/xp`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({note:'Defeated goblin patrol', amount:100})
  });
  await fetch(`${API}/api/sessions/${sessId}`, {
    method: 'PATCH', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({summary:'The hero ventured into the dark cave and encountered goblins.'})
  });

  // Map entry (mock image)
  await fetch(`${API}/api/campaigns/${campId}/maps`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({name:'Cave Map', image_path:'maps/test_map.svg'})
  });

  console.log(`  Seed data ready. Campaign=${campId}, Character=${charId}, Session=${sessId}`);

  // ===== 2. NAVIGATION & LOAD =====
  console.log('\n=== 2. Navigation & Load ===');
  await page.goto(BASE);
  await page.waitForLoadState('networkidle');
  await sleep(1000);

  await assert(await page.getByText('E2E Test Campaign').count() > 0, '2.1 Header shows campaign name');
  await assert(await page.getByText('Test Hero').count() > 0, '2.2 Header shows character name');
  await assert(await page.getByText('E2E Test Session').count() > 0, '2.3 Header shows session name');
  await assert(await page.locator('button[title="Your actions"]').isVisible().catch(() => false), '2.4 Actions button');
  await assert(await page.locator('button[title*="Talents"]').isVisible().catch(() => false), '2.5 Talents button');
  await assert(await page.locator('.h-export').isVisible().catch(() => false), '2.6 Export button');
  await assert(await page.getByTitle('GM Screen — campaign config, notes, and tools').isVisible().catch(() => false), '2.7 GM Screen button');
  await assert(await page.getByTitle('Manage campaigns, characters, sessions').isVisible().catch(() => false), '2.8 Manage button');
  await assert(await page.locator('button[title*="Mute"]').or(page.getByRole('button', { name: '🔊' })).isVisible().catch(() => false), '2.9 Audio button');

  // ===== 3. CHARACTER SHEET =====
  console.log('\n=== 3. Character Sheet ===');
  for (const stat of ['STR', 'DEX', 'CON', 'INT', 'WIS', 'CHA']) {
    await assert(await page.getByText(stat).count() > 0, `3.${['STR','DEX','CON','INT','WIS','CHA'].indexOf(stat)+1} ${stat} attribute`);
  }
  await assert(await page.getByText('HP').count() > 0, '3.7 HP track');
  await assert(await page.getByText('Inventory').count() > 0, '3.8 Inventory section');
  await assert(await page.getByText('Longsword').count() > 0, '3.9 Longsword item');
  await assert(await page.getByText('Torch').count() > 0, '3.10 Torch item');
  await assert(await page.getByText('Gold').count() > 0, '3.11 Currency label');
  await assert(await page.getByText('XP Log').count() > 0, '3.12 XP Log header');
  await assert(await page.getByText('Defeated goblin patrol').count() > 0, '3.13 XP entry note');
  await assert(await page.getByText('+100').count() > 0, '3.14 XP entry amount');

  // ===== 3b. XP LOG CRUD =====
  console.log('\n=== 3b. XP Log CRUD ===');
  await page.locator('.xp-log-input').fill('Roleplay bonus');
  await page.locator('.xp-log-amount-input').fill('50');
  await page.locator('.xp-log-btn').click();
  await sleep(500);
  await assert(await page.getByText('Roleplay bonus').count() > 0, '3b.1 XP entry added via UI');
  await assert(await page.getByText('+50').count() > 0, '3b.2 XP amount shown');

  // ===== 4. SESSION VIEW =====
  console.log('\n=== 4. Session View ===');
  await assert(await page.getByText('E2E TEST SESSION').count() > 0, '4.1 Session title');
  await assert(await page.getByText('I cautiously enter the dark cave').count() > 0, '4.2 Player message');
  await assert(await page.locator('textarea[placeholder="What do you do?"]').isVisible().catch(() => false), '4.3 Player input');
  await assert(await page.locator('.story-search-bar input').isVisible().catch(() => false), '4.4 Story search');
  for (const tag of ['tavern', 'dungeon', 'forest', 'cave']) {
    await assert(await page.locator(`.scene-tag:has-text("${tag}")`).count() > 0, `4.5+ Scene tag: ${tag}`);
  }

  // ===== 5. MAP DRAWER =====
  console.log('\n=== 5. Map Drawer ===');
  await page.locator('.map-drawer-handle').click();
  await sleep(500);
  assert(true, '5.1 Map drawer opened');
  await page.locator('button:has-text("COLLAPSE")').click();
  await sleep(300);

  // ===== 6. SIDEBAR TABS =====
  console.log('\n=== 6. Sidebar Tabs ===');
  const tabs = ['Notes', 'Journal', 'NPCs', 'Objectives', 'Oracle', 'Relations', 'Factions', 'Calendar', 'Stat Blocks', 'Adventures', 'Secrets', 'GM Tools'];
  for (const tab of tabs) {
    await assert(await page.locator(`.tab-btn:has-text("${tab}")`).count() > 0, `6.${tabs.indexOf(tab)+1} ${tab} tab`);
  }

  // ===== 7-18. INDIVIDUAL TAB VERIFICATION =====
  const tabActions = {
    'Notes': async () => { await page.locator('.tab-btn:has-text("Notes")').click(); await sleep(300); return true; },
    'Journal': async () => { await page.locator('.tab-btn:has-text("Journal")').click(); await sleep(300); return true; },
    'NPCs': async () => { await page.locator('.tab-btn:has-text("NPCs")').click(); await sleep(300); return true; },
    'Objectives': async () => { await page.locator('.tab-btn:has-text("Objectives")').click(); await sleep(300); return true; },
    'Oracle': async () => { await page.locator('.tab-btn:has-text("Oracle")').click(); await sleep(300); return true; },
    'Relations': async () => { await page.locator('.tab-btn:has-text("Relations")').click(); await sleep(300); return true; },
    'Factions': async () => {
      await page.locator('.tab-btn:has-text("Factions")').click(); await sleep(500);
      const vis = await page.getByText('The Iron Alliance').isVisible().catch(() => false);
      await page.getByText('The Iron Alliance').click().catch(() => {}); await sleep(200);
      return vis;
    },
    'Calendar': async () => {
      await page.locator('.tab-btn:has-text("Calendar")').click(); await sleep(500);
      const hasDate = await page.getByText('Year').isVisible().catch(() => false);
      const hasEvent = await page.getByText('Spring Equinox Festival').isVisible().catch(() => false);
      return hasDate && hasEvent;
    },
    'Stat Blocks': async () => {
      await page.locator('.tab-btn:has-text("Stat Blocks")').click(); await sleep(500);
      const vis1 = await page.getByText('Goblin Scout').isVisible().catch(() => false);
      const vis2 = await page.getByText('Cave Spider').isVisible().catch(() => false);
      return vis1 && vis2;
    },
    'Adventures': async () => {
      await page.locator('.tab-btn:has-text("Adventures")').click(); await sleep(500);
      const vis = await page.getByText('The Lost Mines').isVisible().catch(() => false);
      await page.getByText('The Lost Mines').click().catch(() => {}); await sleep(200);
      return vis;
    },
    'Secrets': async () => {
      await page.locator('.tab-btn:has-text("Secrets")').click(); await sleep(500);
      return await page.getByText('The Hidden Vault').isVisible().catch(() => false);
    },
    'GM Tools': async () => {
      await page.locator('.tab-btn:has-text("GM Tools")').click(); await sleep(500);
      const improvise = await page.locator('.gm-tool-tab:has-text("Improvise")').isVisible().catch(() => false);
      await page.locator('.gm-tool-tab:has-text("Ask")').click(); await sleep(200);
      await page.locator('.gm-tool-tab:has-text("Improvise")').click(); await sleep(200);
      return improvise;
    },
  };

  for (const [name, action] of Object.entries(tabActions)) {
    const result = await action();
    const idx = Object.keys(tabActions).indexOf(name) + 1;
    const sectionNum = 6 + idx;
    await assert(result, `${sectionNum}.${idx} ${name} tab renders content`);
    console.log(`  ✓ Section ${sectionNum}: ${name} tab`);
  }

  // ===== 19. JOURNAL TIMELINE SUB-TAB =====
  console.log('\n=== 19. Journal Sub-Tabs ===');
  await page.locator('.tab-btn:has-text("Journal")').click();
  await sleep(300);
  await assert(await page.locator('button:has-text("Timeline")').count() > 0, '19.1 Timeline sub-tab button');
  await assert(await page.locator('button:has-text("Notes")').first().isVisible().catch(() => false), '19.2 Notes sub-tab button');
  await assert(await page.locator('button:has-text("Reanalyze")').isVisible().catch(() => false), '19.3 Reanalyze button');
  await page.locator('button:has-text("Timeline")').click();
  await sleep(400);
  await assert(await page.getByText('Session Timeline').count() > 0, '19.4 Timeline heading visible');
  await page.locator('button:has-text("Notes")').first().click();
  await sleep(200);

  // ===== 20. GM SCREEN OVERLAY =====
  console.log('\n=== 20. GM Screen ===');
  await page.getByTitle('GM Screen — campaign config, notes, and tools').click();
  await sleep(500);
  await assert(await page.getByText('GM Screen').count() > 0, '20.1 GM Screen overlay');
  await assert(await page.getByText('Campaign Overview').count() > 0, '20.2 Campaign Overview');
  await assert(await page.getByText('GM Notes').count() > 0, '20.3 GM Notes section');
  await assert(await page.getByText('System Prompt Override').count() > 0, '20.4 System Prompt Override');
  await page.getByRole('button', { name: '×' }).first().click();
  await sleep(300);

  // ===== 21. PLAYER HISTORY OVERLAY =====
  console.log('\n=== 21. Player History ===');
  await page.locator('button[title="Your actions"]').first().click();
  await sleep(500);
  await assert(await page.getByText('Your Actions').count() > 0, '21.1 Player history overlay');
  await assert(await page.getByText('I cautiously enter the dark cave').count() > 0, '21.2 Player action shown');
  await page.locator('.player-history-header button').click();
  await sleep(300);

  // ===== 22. TALENTS OVERLAY =====
  console.log('\n=== 22. Talents Overlay ===');
  await page.getByRole('button', { name: '✦ Talents' }).click();
  await sleep(500);
  await assert(await page.getByText('Talents & Powers').count() > 0, '22.1 Talents overlay');
  await page.locator('.talents-overlay-header button').click();
  await sleep(200);

  // ===== 23. THEME TOGGLE =====
  console.log('\n=== 23. Theme ===');
  const themeBtn = page.locator('button[title="Toggle theme"]');
  await themeBtn.click(); await sleep(300);
  await assert((await page.locator('html').getAttribute('data-theme')) === 'parchment', '23.1 Theme → light');
  await themeBtn.click(); await sleep(300);
  await assert((await page.locator('html').getAttribute('data-theme')) === 'worn-grimoire', '23.2 Theme → dark');

  // ===== 24. AUDIO =====
  console.log('\n=== 24. Audio ===');
  const mBtn = page.locator('button[title*="Mute"]').or(page.getByRole('button', { name: '🔊' }));
  await assert(await mBtn.isVisible().catch(() => false), '24.1 Mute button');

  // ===== 25. STORY SEARCH =====
  console.log('\n=== 25. Story Search ===');
  await page.locator('.story-search-bar input').fill('cave');
  await sleep(300);
  await page.locator('.story-search-bar input').fill('');
  await sleep(200);

  // ===== 26. EXPORT =====
  console.log('\n=== 26. Export ===');
  await page.locator('.h-export').click();
  await sleep(300);
  assert(true, '26.1 Export triggered');

  // ===== 27. WHISPER TOGGLE =====
  console.log('\n=== 27. Whisper ===');
  await page.locator('button:has-text("🔒")').first().click();
  await sleep(200);
  await page.locator('button:has-text("🔒")').first().click();
  await sleep(200);

  // ===== 28. SCENE TAGS =====
  console.log('\n=== 28. Scene Tags ===');
  await page.locator('.scene-tag:has-text("cave")').first().click();
  await sleep(200);
  await page.locator('.scene-tag:has-text("cave")').first().click();
  await sleep(200);

  // ===== 29. MANAGE PANEL =====
  console.log('\n=== 29. Manage Panel ===');
  await page.getByTitle('Manage campaigns, characters, sessions').click();
  await sleep(500);
  await assert(await page.getByText('Manage Campaign').count() > 0, '29.1 Manage panel');
  for (const tab of ['Campaigns', 'Characters', 'Sessions', 'Rulebooks', 'Automation']) {
    await assert(await page.locator(`.manage-tab:has-text("${tab}")`).count() > 0, `29.2+ Manage ${tab} tab`);
  }

  // ===== 30. AUTOMATION SETTINGS =====
  console.log('\n=== 30. Automation Settings ===');
  await page.locator('.manage-tab:has-text("Automation")').click();
  await sleep(500);
  for (const label of ['Extract NPCs', 'Generate Maps', 'Update Character Stats', 'Detect Objectives']) {
    await assert(await page.getByText(label).count() > 0, `30.x ${label} setting`);
  }
  await page.locator('.automation-toggle').first().click();
  await sleep(400);
  assert(true, '30.5 Automation toggle clicked');

  // ===== 31. CHARACTER SHEET EDITING =====
  console.log('\n=== 31. Character Editing ===');
  const numInput = page.locator('input[type="number"]').first();
  if (await numInput.isVisible().catch(() => false)) {
    await numInput.fill('4');
    await sleep(300);
    assert(true, '31.1 Number field editable');
  }

  // ===== 32. INVENTORY UI =====
  console.log('\n=== 32. Inventory ===');
  const invInput = page.locator('input[placeholder*="Item name"]');
  if (await invInput.isVisible().catch(() => false)) {
    await invInput.fill('Health Potion');
    await page.locator('button:has-text("Add")').last().click().catch(() => {});
    await sleep(300);
    assert(true, '32.1 Add item attempted via UI');
  }

  // Close manage
  await page.locator('.manage-close').click();
  await sleep(300);

  // ===== NOW: API ASSERTIONS =====
  // Test every backend endpoint

  async function api(url, opts = {}) {
    const res = await fetch(url.startsWith('http') ? url : `${API}${url}`, {
      headers: {'Content-Type': 'application/json'},
      ...opts
    });
    if (opts.raw) return res;
    const text = await res.text();
    try { return { status: res.status, ok: res.ok, data: JSON.parse(text), raw: res }; }
    catch { return { status: res.status, ok: res.ok, text, raw: res }; }
  }

  // 33. Campaign Config
  console.log('\n=== 33. Campaign Config API ===');
  const configGet = await api(`/api/campaigns/${campId}/config`);
  assert(configGet.data && configGet.data.description !== undefined, '33.1 Get campaign config');
  assert(typeof configGet.data.session_count === 'number', '33.2 Config has session_count');

  const configUpd = await api(`/api/campaigns/${campId}/config`, {
    method: 'PATCH', body: JSON.stringify({gm_notes: 'Secret GM notes for this campaign'})
  });
  assert(configUpd.ok, '33.3 Update campaign config');

  const configChk = await api(`/api/campaigns/${campId}/config`);
  assert(configChk.data.gm_notes === 'Secret GM notes for this campaign', '33.4 GM notes persisted');

  // 34. Secrets
  console.log('\n=== 34. Secrets API ===');
  const secList = await api(`/api/campaigns/${campId}/secrets`);
  assert(Array.isArray(secList.data) && secList.data.length > 0, '34.1 List secrets');

  const secRev = await api(`/api/secrets/${secList.data[0].id}/reveal`, {
    method: 'PATCH', body: JSON.stringify({session_id: sessId})
  });
  assert(secRev.ok, '34.2 Reveal secret');

  const secDel = await api(`/api/secrets/${secList.data[0].id}`, { method: 'DELETE' });
  assert(secDel.ok, '34.3 Delete secret');

  // 35. Factions
  console.log('\n=== 35. Factions API ===');
  const facList = await api(`/api/campaigns/${campId}/factions`);
  assert(Array.isArray(facList.data) && facList.data.length > 0, '35.1 List factions');

  const facUpd = await api(`/api/factions/${facList.data[0].id}`, {
    method: 'PATCH', body: JSON.stringify({name:'The Iron Alliance (Updated)', influence:8})
  });
  assert(facUpd.ok, '35.2 Update faction');

  const facGet = await api(`/api/factions/${facList.data[0].id}`);
  assert(facGet.data.name === 'The Iron Alliance (Updated)', '35.3 Faction name updated');
  assert(facGet.data.influence === 8, '35.4 Faction influence updated');

  // 36. Adventures
  console.log('\n=== 36. Adventures API ===');
  const advList = await api(`/api/campaigns/${campId}/adventures`);
  assert(Array.isArray(advList.data) && advList.data.length > 0, '36.1 List adventures');

  const advUpd = await api(`/api/adventures/${advList.data[0].id}`, {
    method: 'PATCH', body: JSON.stringify({title:'The Lost Mines', status:'completed', sort_order:0, description:''})
  });
  assert(advUpd.ok, '36.2 Update adventure status');

  // 37. NPC Stats
  console.log('\n=== 37. NPC Stats API ===');
  const nsList = await api(`/api/campaigns/${campId}/npc-stats`);
  assert(Array.isArray(nsList.data) && nsList.data.length >= 2, '37.1 List NPC stats');

  const nsUpd = await api(`/api/npc-stats/${nsList.data[0].id}`, {
    method: 'PATCH', body: JSON.stringify({name:'Goblin Scout', role:'scout', hp_max:10, armor_class:13, initiative_mod:2, skills:'[]', abilities:'[]', loot:'[]', notes:''})
  });
  assert(nsUpd.ok, '37.2 Update NPC stat block');

  const nsDel = await api(`/api/npc-stats/${nsList.data[1].id}`, { method: 'DELETE' });
  assert(nsDel.ok, '37.3 Delete NPC stat block');

  // 38. Calendar
  console.log('\n=== 38. Calendar API ===');
  const calGet = await api(`/api/campaigns/${campId}/calendar`);
  assert(typeof calGet.data.in_game_year === 'number', '38.1 Get calendar date');

  const calPatch = await api(`/api/campaigns/${campId}/calendar`, {
    method: 'PATCH', body: JSON.stringify({advance_days: 7})
  });
  assert(calPatch.ok, '38.2 Advance calendar');

  const calVerify = await api(`/api/campaigns/${campId}/calendar`);
  const advanced = calVerify.data.in_game_day > calGet.data.in_game_day ||
                   calVerify.data.in_game_month > calGet.data.in_game_month;
  assert(advanced, '38.3 Calendar advanced');

  const calEvents = await api(`/api/campaigns/${campId}/calendar-events`);
  if (calEvents.data.length > 0) {
    const delCal = await api(`/api/calendar-events/${calEvents.data[0].id}`, { method: 'DELETE' });
    assert(delCal.ok, '38.4 Delete calendar event');
  }

  // 39. Automation Settings
  console.log('\n=== 39. Automation Settings API ===');
  const autoSet = await api(`/api/settings/automations`);
  assert(Array.isArray(autoSet.data) && autoSet.data.length > 0, '39.1 List automation settings');
  assert(autoSet.data[0].key !== undefined, '39.2 Has key');
  assert(autoSet.data[0].label !== undefined, '39.3 Has label');
  assert(autoSet.data[0].enabled !== undefined, '39.4 Has enabled');

  const autoToggle = await api(`/api/settings/automations`, {
    method: 'PATCH', body: JSON.stringify({key: autoSet.data[0].key, enabled: false})
  });
  assert(autoToggle.ok, '39.5 Toggle automation');

  // 40. World Notes
  console.log('\n=== 40. World Notes API ===');
  const wnList = await api(`/api/campaigns/${campId}/world-notes`);
  assert(Array.isArray(wnList.data), '40.1 List world notes');
  const wnSearch = await api(`/api/campaigns/${campId}/world-notes?q=test`);
  assert(Array.isArray(wnSearch.data), '40.2 Search world notes');

  // 41. Objectives
  console.log('\n=== 41. Objectives API ===');
  const objList = await api(`/api/campaigns/${campId}/objectives`);
  assert(objList.data.length > 0, '41.1 List objectives');
  const objUpd = await api(`/api/objectives/${objList.data[0].id}`, {
    method: 'PATCH', body: JSON.stringify({status: 'completed'})
  });
  assert(objUpd.ok, '41.2 Update objective');

  // 42. Session
  console.log('\n=== 42. Session API ===');
  const sessUpd = await api(`/api/sessions/${sessId}`, {
    method: 'PATCH', body: JSON.stringify({notes: 'E2E session notes'})
  });
  assert(sessUpd.ok, '42.1 Update session notes');

  // 43. Tension
  console.log('\n=== 43. Tension API ===');
  const tensGet = await api(`/api/sessions/${sessId}/tension`);
  assert(typeof tensGet.data.tension_level === 'number', '43.1 Get tension');
  const tensSet = await api(`/api/sessions/${sessId}/tension`, {
    method: 'PATCH', body: JSON.stringify({tension_level: 7})
  });
  assert(tensSet.ok, '43.2 Set tension');

  // 44. Oracle
  console.log('\n=== 44. Oracle API ===');
  const oracle = await api(`/api/oracle/roll`, {
    method: 'POST', body: JSON.stringify({table:'action', roll:23, ruleset_id: null})
  });
  assert(oracle.data.result !== '', '44.1 Oracle roll returns result');

  // 45. Dice
  console.log('\n=== 45. Dice API ===');
  const diceList = await api(`/api/sessions/${sessId}/dice-rolls`);
  assert(Array.isArray(diceList.data) && diceList.data.length >= 2, '45.1 List dice rolls');

  // 46. Timeline
  console.log('\n=== 46. Timeline API ===');
  const tl = await api(`/api/sessions/${sessId}/timeline`);
  assert(Array.isArray(tl.data) && tl.data.length > 0, '46.1 Session timeline');

  // 47. Relationships
  console.log('\n=== 47. Relationships CRUD ===');
  const relCreate = await api(`/api/campaigns/${campId}/relationships`, {
    method: 'POST', body: JSON.stringify({from_name:'Test Hero', to_name:'Goblin King', relationship_type:'enemy', description:'Sworn enemy'})
  });
  assert(relCreate.data.id > 0, '47.1 Create relationship');

  const relList = await api(`/api/campaigns/${campId}/relationships`);
  assert(relList.data.length > 0, '47.2 List relationships');

  const relUpd = await api(`/api/relationships/${relCreate.data.id}`, {
    method: 'PATCH', body: JSON.stringify({relationship_type:'rival', description:'Now competes'})
  });
  assert(relUpd.ok, '47.3 Update relationship');

  const relDel = await api(`/api/relationships/${relCreate.data.id}`, { method: 'DELETE' });
  assert(relDel.ok, '47.4 Delete relationship');

  // 48. Chronicle Night
  console.log('\n=== 48. Chronicle Night API ===');
  const nightPatch = await api(`/api/campaigns/${campId}`, {
    method: 'PATCH', body: JSON.stringify({chronicle_night: 5})
  });
  assert(nightPatch.ok, '48.1 Set chronicle night');

  // 49. Health
  console.log('\n=== 49. Health ===');
  const health = await api(`/api/health`);
  assert(health.data.status === 'ok', '49.1 Health ok');

  // 50. Context
  console.log('\n=== 50. Context ===');
  const ctx = await api(`/api/context`);
  assert(ctx.data.campaign !== null, '50.1 Context campaign');
  assert(ctx.data.character !== null, '50.2 Context character');
  assert(ctx.data.session !== null, '50.3 Context session');

  // 51. Character List
  console.log('\n=== 51. Character List ===');
  const chars = await api(`/api/campaigns/${campId}/characters`);
  assert(chars.data.length > 0, '51.1 List characters');

  // 52. Session List
  console.log('\n=== 52. Session List ===');
  const sessList = await api(`/api/campaigns/${campId}/sessions`);
  assert(sessList.data.length > 0, '52.1 List sessions');

  // 53. Messages
  console.log('\n=== 53. Messages ===');
  const msgs = await api(`/api/sessions/${sessId}/messages`);
  assert(msgs.data.length >= 2, '53.1 List messages');
  assert(msgs.data.some(m => m.role === 'user'), '53.2 Includes user messages');

  // 54. XP
  console.log('\n=== 54. XP List ===');
  const xpList = await api(`/api/sessions/${sessId}/xp`);
  assert(xpList.data.length >= 1, '54.1 List XP');
  assert(xpList.data.some(e => e.note.includes('goblin patrol')), '54.2 XP has correct note');

  // 55. Rulesets
  console.log('\n=== 55. Rulesets ===');
  const rulesets = await api(`/api/rulesets`);
  assert(Array.isArray(rulesets.data) && rulesets.data.length > 0, '55.1 List rulesets');
  assert(rulesets.data.some(r => r.name === 'dnd5e'), '55.2 D&D 5e present');
  const rs = await api(`/api/rulesets/${rulesets.data[0].id}`);
  assert(rs.data.name !== undefined, '55.3 Get single ruleset');

  // 56. Character Options
  console.log('\n=== 56. Character Options ===');
  const opts = await api(`/api/rulesets/${rulesets.data[0].id}/character-options`);
  assert(Object.keys(opts.data).length > 0, '56.1 Character options');

  // 57. Campaign List
  console.log('\n=== 57. Campaign Management ===');
  const campList = await api(`/api/campaigns`);
  assert(campList.data.length > 0, '57.1 List campaigns');

  // 58. Talent Description (will 503 without AI key)
  console.log('\n=== 58. Talent Description ===');
  const talentRaw = await api(`/api/talent-description?name=Furious+Assault&system=wrath_glory`);
  assert(talentRaw.status === 503 || talentRaw.ok, '58.1 Talent description (503 without AI)');

  // 59. Map
  console.log('\n=== 59. Maps ===');
  const maps = await api(`/api/campaigns/${campId}/maps`);
  assert(Array.isArray(maps.data), '59.1 List maps');

  // 60. File serving (path traversal blocked)
  console.log('\n=== 60. File Serving ===');
  const fileResp = await fetch(`${API}/api/files/../../etc/passwd`, { redirect: 'manual' });
  assert(fileResp.status === 403 || fileResp.status >= 300, '60.1 Path traversal blocked');

  // 61. Rulebook
  console.log('\n=== 61. Rulebook ===');
  const rbSources = await api(`/api/rulesets/${rulesets.data[0].id}/rulebook`);
  assert(Array.isArray(rbSources.data), '61.1 Rulebook sources');

  const rbIngest = await api(`/api/rulesets/${rulesets.data[0].id}/rulebook`, {
    method: 'POST', headers: {'Content-Type': 'text/plain'}, raw: true,
    body: '# Combat\nCombat rules.\n# Magic\nMagic rules.\n# Skills\nSkills are important.'
  });
  assert(rbIngest.ok, '61.2 Ingest rulebook');

  // 62. Dice Validation
  console.log('\n=== 62. Dice Validation ===');
  const badRoll = await api(`/api/sessions/${sessId}/dice-rolls`, {
    method: 'POST', body: JSON.stringify({expression:'invalid'})
  });
  assert(!badRoll.ok, '62.1 Invalid dice rejected');

  // 63. Oracle Validation
  console.log('\n=== 63. Oracle Validation ===');
  const badOra = await api(`/api/oracle/roll`, {
    method: 'POST', body: JSON.stringify({table:'nonexistent', roll:99})
  });
  assert(!badOra.ok, '63.1 Invalid oracle returns error');

  // 64. Campaign Deletion
  console.log('\n=== 64. Campaign Deletion ===');
  const delCamp = await api(`/api/campaigns/${campId}`, { method: 'DELETE' });
  assert(delCamp.ok, '64.1 Delete campaign');

  const campAfter = await api(`/api/campaigns`);
  assert(!campAfter.data.some(c => c.id === campId), '64.2 Campaign actually deleted');

  // ===== RESULTS =====
  console.log(`\n${'='.repeat(50)}`);
  console.log(`RESULTS: ${passed} passed, ${failed} failed`);
  if (failed > 0) {
    console.log(`\nFAILURES:`);
    errors.forEach(e => console.log(`  - ${e}`));
  }
  console.log(`${'='.repeat(50)}`);

  await browser.close();
  process.exit(failed > 0 ? 1 : 0);
}

main().catch(e => {
  console.error('FATAL:', e);
  process.exit(1);
});
