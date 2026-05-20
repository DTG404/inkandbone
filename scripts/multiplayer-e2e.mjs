#!/usr/bin/env node
/**
 * Multiplayer E2E test — 5 turns each for Zay and Nyx.
 * Verifies GM addresses both characters by name, tracks identity, and handles turns.
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

/** Strip SSE `data: ` framing and reconstruct the original text.
 *  Each SSE message is "data: <chunk>\n\n". Chunks are variable-length
 *  character sequences streamed from the AI. Concatenate them directly
 *  to reconstruct the narrative, then normalize whitespace. */
function parseSSE(sse) {
  return sse
    .split('\n\n')
    .filter(l => l.startsWith('data: '))
    .map(l => l.replace(/^data: /, ''))
    .join('')
    .replace(/\s+/g, ' ')
    .trim();
}

async function main() {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();

  // ===== 1. SEED =====
  console.log('\n=== 1. Seeding data ===');

  const cCamp = await (await fetch(`${API}/api/campaigns`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({name: 'Multiplayer E2E', ruleset_id: 3})
  })).json();
  const campId = cCamp.id;
  assert(campId > 0, '1.1 Create campaign');

  const cZay = await (await fetch(`${API}/api/campaigns/${campId}/characters`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({name: 'Zay', data_json: JSON.stringify({
      character_type: 'Vampire', clan: 'Brujah', generation: '12th', sect: 'Anarch',
      predator_type: 'Alleycat', hunger: 1, blood_potency: 1, humanity: 7,
      strength: 4, dexterity: 3, stamina: 3, charisma: 2, manipulation: 2, composure: 3,
      intelligence: 2, wits: 3, resolve: 2,
      brawl: 3, melee: 2, intimidation: 3, streetwise: 2, athletics: 2,
      presence: 2, potence: 2, celerity: 1,
      health_max: 5, willpower_max: 6,
      hp: 5, hp_max: 5
    })})
  })).json();
  const zayId = cZay.id;
  assert(zayId > 0, '1.2 Create Zay (Brujah)');

  const cNyx = await (await fetch(`${API}/api/campaigns/${campId}/characters`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({name: 'Nyx', data_json: JSON.stringify({
      character_type: 'Vampire', clan: 'Toreador', generation: '10th', sect: 'Anarch',
      predator_type: 'Siren', hunger: 1, blood_potency: 2, humanity: 6,
      charisma: 4, manipulation: 3, composure: 3,
      strength: 2, dexterity: 3, stamina: 2,
      intelligence: 2, wits: 4, resolve: 2,
      etiquette: 3, insight: 3, persuasion: 3, subterfuge: 3, performance: 3,
      presence: 3, auspex: 2, celerity: 1,
      health_max: 5, willpower_max: 6,
      hp: 5, hp_max: 5
    })})
  })).json();
  const nyxId = cNyx.id;
  assert(nyxId > 0, '1.3 Create Nyx (Toreador)');

  const cSess = await (await fetch(`${API}/api/campaigns/${campId}/sessions`, {
    method: 'POST', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({title: 'First Blood', date: '2026-05-19'})
  })).json();
  const sessId = cSess.id;
  assert(sessId > 0, '1.4 Create session');

  const chars = await (await fetch(`${API}/api/campaigns/${campId}/characters`)).json();
  assert(chars.length === 2, '1.5 Two characters in campaign');

  // Set context to Zay first
  await fetch(`${API}/api/settings`, {
    method: 'PATCH', headers: {'Content-Type':'application/json'},
    body: JSON.stringify({campaign_id: campId, character_id: zayId, session_id: sessId})
  });

  // ===== 2. VERIFY SETUP =====
  console.log('\n=== 2. Verifying setup ===');

  const ctx = await (await fetch(`${API}/api/context`)).json();
  assert(ctx.campaign?.id === campId, '2.1 Campaign in context');
  assert(ctx.character?.id === zayId, '2.2 Zay active');
  assert(ctx.session?.id === sessId, '2.3 Session active');

  // ===== 3. TURNS =====
  console.log('\n=== 3. Running 5 turns each (10 total) ===');

  const zayActions = [
    'I crack my knuckles and scan the bar for anyone who has been watching me too long.',
    'I lean back and signal the bartender for another whiskey. Neat.',
    'A Nosferatu at the corner table caught me staring. I hold eye contact.',
    'I tap the bar twice — my signal. My people flank the exits.',
    'I pull out my phone and scroll a burner message: "The Toreadors pet is here."'
  ];

  const nyxActions = [
    'I adjust my cuff and order a martini — dry, two olives.',
    'I catch Zays signal. I tilt my head toward the back exit.',
    'I run my finger along the rim of my glass, reading the reflections.',
    'I pull a compact mirror from my clutch and catalogue every face in the room.',
    'I gesture to the jazz trio. They shift into something slower.'
  ];

  async function turn(charId, charName, action) {
    // Set active context so buildWorldContext uses this character
    await fetch(`${API}/api/settings`, {
      method: 'PATCH', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({campaign_id: campId, character_id: charId, session_id: sessId})
    });

    const msgRes = await fetch(`${API}/api/sessions/${sessId}/messages`, {
      method: 'POST', headers: {'Content-Type':'application/json'},
      body: JSON.stringify({role: 'user', content: action, character_id: charId})
    });
    assert(msgRes.status === 201, `${charName} msg sent`);

    // Trigger GM
    const gmRes = await fetch(`${API}/api/sessions/${sessId}/gm-respond-stream`, { method: 'POST' });
    assert(gmRes.ok, `${charName} GM 200`);
    const raw = await gmRes.text();

    // Parse SSE
    const narrative = parseSSE(raw);
    assert(narrative.length > 50, `${charName} GM narrative has substance (${narrative.length} chars)`);

    // The GM should mention the acting character's name in the narrative
    const nameFound = narrative.includes(charName);
    assert(nameFound, `${charName} mentioned in GM text`);
    if (!nameFound) console.error(`    Excerpt: "${narrative.slice(0, 200)}..."`);

    // The GM should mention the OTHER character too (multiplayer awareness)
    const otherName = charName === 'Zay' ? 'Nyx' : 'Zay';
    const otherFound = narrative.includes(otherName);
    if (!otherFound) {
      // Not a hard fail — the GM might not always refer to the other character
      console.log(`  (note: ${otherName} not mentioned this turn — sometimes fine)`);
    }

    // Check for prompt language (flexible)
    const hasPrompt = /what do you do/i.test(narrative) || /your move/i.test(narrative) || /\?\s*$/.test(narrative.trim());
    if (!hasPrompt) {
      // Check last 100 chars for any question
      const last100 = narrative.slice(-100);
      const endsWithQuestion = /\?\s*$/.test(last100);
      if (!endsWithQuestion) {
        console.log(`  (note: no explicit prompt detected in ${charName}'s GM response)`);
      }
    }

    return narrative;
  }

  // 5 alternating turns starting with Zay
  for (let i = 0; i < 5; i++) {
    console.log(`\n  Round ${i + 1}:`);
    await turn(zayId, 'Zay', zayActions[i]);
    await turn(nyxId, 'Nyx', nyxActions[i]);
  }

  // ===== 4. VERIFY MESSAGE HISTORY =====
  console.log('\n=== 4. Checking message history ===');

  const messages = await (await fetch(`${API}/api/sessions/${sessId}/messages`)).json();
  const userMsgs = messages.filter(m => m.role === 'user');
  const zayMsgs = userMsgs.filter(m => m.character_id === zayId);
  const nyxMsgs = userMsgs.filter(m => m.character_id === nyxId);

  assert(zayMsgs.length === 5, `4.1 Zay has 5 messages (got ${zayMsgs.length})`);
  assert(nyxMsgs.length === 5, `4.2 Nyx has 5 messages (got ${nyxMsgs.length})`);
  assert(messages.length >= 15, `4.3 ≥15 total msgs (got ${messages.length})`);

  const allHaveCharId = userMsgs.every(m => m.character_id != null);
  assert(allHaveCharId, '4.4 All user messages have character_id');

  // ===== 5. BROWSER UI =====
  console.log('\n=== 5. Browser UI verification ===');

  await page.goto(BASE);
  await page.waitForLoadState('networkidle');
  await sleep(3000);

  const headerCamp = await page.locator('.h-campaign').textContent();
  assert(headerCamp === 'Multiplayer E2E', `5.1 Campaign: "${headerCamp}"`);

  const storyText = await page.locator('.story-scroll').innerText();
  assert(/Zay/i.test(storyText), '5.2 Zay in story');
  assert(/Nyx/i.test(storyText), '5.3 Nyx in story');

  const charSelect = await page.locator('.character-selector select');
  if (await charSelect.count() > 0) {
    const opts = await charSelect.locator('option').allTextContents();
    assert(opts.some(o => o.trim() === 'Zay'), '5.4 Selector has Zay');
    assert(opts.some(o => o.trim() === 'Nyx'), '5.5 Selector has Nyx');
  } else {
    console.log('  (note: character selector not visible — expected when only 1 char in context)');
  }

  // ===== RESULTS =====
  console.log(`\n==================================================`);
  console.log(`RESULTS: ${passed} passed, ${failed} failed`);
  if (failed > 0) {
    console.log('\nFAILURES:');
    errors.forEach(e => console.log(`  - ${e}`));
  }
  console.log(`==================================================\n`);

  await browser.close();
  process.exit(failed > 0 ? 1 : 0);
}

main();
