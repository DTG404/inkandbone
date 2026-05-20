#!/usr/bin/env node
/**
 * Deep Multiplayer E2E — 20 turns each for 2 characters (40 total GM responses).
 */

import { chromium } from 'playwright';
const BASE = 'http://localhost:7432';
const API = BASE;
let passed = 0, failed = 0, errors = [];
const allGmResponses = [], wsEvents = [];
function assert(c,m){if(c)passed++;else{failed++;errors.push(m);console.error('  ✗',m)}}
async function sleep(ms){return new Promise(r=>setTimeout(r,ms))}
function parseSSE(sse){return sse.split('\n\n').filter(l=>l.startsWith('data: ')).map(l=>l.replace(/^data: /,'')).join('').replace(/\s+/g,' ').trim()}

async function main() {
  console.log('\n═══════════════════════════════════════');
  console.log('  DEEP MULTIPLAYER E2E — 20 EACH');
  console.log('═══════════════════════════════════════\n');

  // ==== 1. SEED ====
  console.log('=== 1. Seeding ===\n');
  const cCamp = await (await fetch(API+'/api/campaigns',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:'Deep MP',ruleset_id:3})})).json();
  const campId = cCamp.id;
  assert(campId > 0, 'Campaign');

  async function createChar(name, data) {
    const c = await (await fetch(API+'/api/campaigns/'+campId+'/characters',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name})})).json();
    await fetch(API+'/api/characters/'+c.id,{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({data_json:JSON.stringify(data)})});
    return c;
  }

  const cZay = await createChar('Zay', {
    character_type:'Vampire',clan:'Brujah',generation:'12th',sect:'Anarch',predator_type:'Alleycat',
    hunger:2,blood_potency:1,humanity:7,stains:0,strength:4,dexterity:3,stamina:3,
    charisma:2,manipulation:2,composure:3,intelligence:2,wits:3,resolve:2,
    athletics:3,brawl:4,craft:1,drive:1,firearms:1,melee:2,stealth:2,survival:1,
    animalken:0,etiquette:1,insight:2,intimidation:4,leadership:2,performance:0,
    persuasion:1,streetwise:3,subterfuge:1,academics:1,awareness:3,finance:1,
    investigation:1,medicine:0,occult:1,politics:1,science:0,technology:1,
    presence:2,potence:3,celerity:1,health_max:5,willpower_max:6,
    health_superficial:0,health_aggravated:0,willpower_superficial:0,xp:0,
    ambition:'Burn Camarilla',desire:'Find sire killer',
    convictions:'Strength is truth',touchstones:'My pack-brother',
    merits_flaws:'Iron Will'
  });
  const zayId = cZay.id; assert(zayId > 0, 'Zay');

  const cNyx = await createChar('Nyx', {
    character_type:'Vampire',clan:'Toreador',generation:'10th',sect:'Anarch',predator_type:'Siren',
    hunger:1,blood_potency:2,humanity:6,stains:0,strength:2,dexterity:3,stamina:2,
    charisma:4,manipulation:3,composure:3,intelligence:2,wits:4,resolve:2,
    athletics:1,brawl:1,craft:2,drive:1,firearms:1,melee:2,stealth:2,survival:1,
    animalken:1,etiquette:3,insight:3,intimidation:2,leadership:2,performance:3,
    persuasion:3,streetwise:3,subterfuge:3,academics:2,awareness:3,finance:2,
    investigation:2,medicine:1,occult:1,politics:2,science:1,technology:1,
    presence:3,auspex:2,celerity:1,health_max:5,willpower_max:6,
    health_superficial:0,health_aggravated:0,willpower_superficial:0,xp:0,
    ambition:'Info network',desire:'Find sires kin',
    convictions:'Beauty is armour',touchstones:'The Velvet Knot',
    merits_flaws:'Haven'
  });
  const nyxId = cNyx.id; assert(nyxId > 0, 'Nyx');

  // Verify clan at creation
  const verify = await (await fetch(API+'/api/campaigns/'+campId+'/characters')).json();
  const vz = JSON.parse(verify.find(x=>x.name==='Zay').data_json||'{}');
  const vn = JSON.parse(verify.find(x=>x.name==='Nyx').data_json||'{}');
  assert(vz.clan==='Brujah', 'Zay clan OK: '+vz.clan);
  assert(vn.clan==='Toreador', 'Nyx clan OK: '+vn.clan);

  const sess = await (await fetch(API+'/api/campaigns/'+campId+'/sessions',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({title:'Blood Jazz',date:'2026-05-19'})})).json();
  const sessId = sess.id; assert(sessId > 0, 'Session');

  // ==== 2. BROWSER (captures WS events) ====
  console.log('\n=== 2. Browser ===\n');
  const browser = await chromium.launch({headless:true});
  const page = await browser.newPage({viewport:{width:1440,height:900}});
  page.on('websocket',ws=>{ws.on('framereceived',e=>{try{wsEvents.push(JSON.parse(e.payload))}catch{}})});
  await page.goto(BASE); await page.waitForLoadState('networkidle'); await sleep(2000);
  const wsBefore = wsEvents.length;

  await fetch(API+'/api/settings',{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({campaign_id:campId,character_id:zayId,session_id:sessId})});

  // ==== 3. 20 TURNS EACH ====
  console.log('\n=== 3. 20 turns each ===\n');

  async function turn(charId, charName, action, tn) {
    await fetch(API+'/api/settings',{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({campaign_id:campId,character_id:charId,session_id:sessId})});
    const r = await (await fetch(API+'/api/sessions/'+sessId+'/messages',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({role:'user',content:action,character_id:charId})}));
    assert(r.status===201, charName+' T'+tn+': msg');
    if (!r.ok) { const t=await r.text(); console.error(`  ${charName} T${tn}: msg FAILED ${r.status}: ${t.slice(0,100)}`); }
    await fetch(API+'/api/sessions/'+sessId+'/typing',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({character_id:charId,status:'thinking'})});
    const g = await (await fetch(API+'/api/sessions/'+sessId+'/gm-respond-stream',{method:'POST'}));
    assert(g.ok, charName+' T'+tn+': GM');
    const raw = await g.text();
    const n = parseSSE(raw);
    assert(n.length>80, charName+' T'+tn+': '+n.length+'c');
    // Check for "What do you do, [Name]?" in last 200 chars
    const last = n.slice(-200);
    const hasPrompt = last.includes('What do you do, '+charName) || last.includes('what do you do, '+charName) || /\?\s*$/.test(n.trim());
    if (!hasPrompt) console.log(`  ⚠ ${charName} T${tn}: no name-prompt in last 200 chars`);
    const nameIn = n.includes(charName);
    if (!nameIn && tn>3) console.log(`  ⚠ ${charName} T${tn}: name not in text`);
    await fetch(API+'/api/sessions/'+sessId+'/typing',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({character_id:charId,status:'done'})});
    allGmResponses.push({turn:tn,character:charName,narrative:n});
  }

  const story = [
    [1, `I push through the Velvet Knot door.`, `I rise as Zay enters. "Contact went dark."`],
    [2, `I grab the bourbon. "Sabbat?"`, `He nods. "Three nights. Warehouse District."`],
    [3, `"I'm in. You're coming too."`, `"Midnight. Stockyard tunnels."`],
    [4, `I pocket the map. "Midnight."`, `Midnight at the tunnel, Auspex sharp.`],
    [5, `I step in. "Stay close."`, `I follow, mirror ready. "Something ahead."`],
    [6, `"Old or pre-city?"`, `"Old chamber. Recently opened."`],
    [7, `"Let's see." I move forward.`, `The altar has a figure. "Tzimisce."`],
    [8, `I swing at it. No hesitation.`, `I hit it with Presence. "Again!"`],
    [9, `Fist to chest. Bone cracks.`, `I drive silver into its shoulder.`],
    [10, `"Nyx, find anything!"`, `I grab a journal and vial. "GO!"`],
    [11, `I haul Nyx out. It's healing.`, `We burst into rain. "Journal tracks kindred."`],
    [12, `I flip pages. "About us."`, `"Blood-writing symbol. Property."`],
    [13, `"Warn everyone listed."`, `"I know a Nosferatu who can copy it."`],
    [14, `"Do it. Three hours till dawn."`, `I toss money. "Let's move."`],
    [15, `I drain ox-blood at a ghoul bar.`, `I order a drink. "Page 34 is your pack."`],
    [16, `"Feed them bad intel."`, `"Malkavian contact for disinformation."`],
    [17, `"Safehouse is packed. Diner at dawn."`, `I slide a burner phone. "Encrypted."`],
    [18, `"We're at war now."`, `"Threat map. Keep moving. Don't stay twice."`],
    [19, `"Blood sings in both of us," I say.`, `I go still. "What if we're being collected?"`],
    [20, `"Make it expensive." I step into dawn.`, `I watch him go. "Tonight we plan."`]
  ];

  for (let i = 0; i < 20; i++) {
    const s = story[i];
    console.log(`  Round ${i+1}:`);
    await turn(zayId, 'Zay', s[0], i+1);
    await turn(nyxId, 'Nyx', s[1], i+1);
  }

  // ==== 4. ANALYSIS ====
  console.log('\n═══ ANALYSIS ═══\n');

  const msgs = await (await fetch(API+'/api/sessions/'+sessId+'/messages')).json();
  const userM = msgs.filter(m=>m.role==='user');
  const zM = userM.filter(m=>m.character_id===zayId);
  const nM = userM.filter(m=>m.character_id===nyxId);
  assert(zM.length===20, 'Zay 20 msgs: '+zM.length);
  assert(nM.length===20, 'Nyx 20 msgs: '+nM.length);
  assert(msgs.length>=60, '≥60 total');

  // Identity check
  const chars = await (await fetch(API+'/api/campaigns/'+campId+'/characters')).json();
  const fz = JSON.parse(chars.find(c=>c.name==='Zay').data_json||'{}');
  const fn = JSON.parse(chars.find(c=>c.name==='Nyx').data_json||'{}');
  assert(fz.clan==='Brujah', 'Zay clan: Brujah (got '+fz.clan+')');
  assert(fn.clan==='Toreador', 'Nyx clan: Toreador (got '+fn.clan+')');

  console.log('  Zay hunger: '+fz.hunger+' XP: '+(fz.xp||0)+' HP: '+fz.health_superficial);
  console.log('  Nyx hunger: '+fn.hunger+' XP: '+(fn.xp||0)+' HP: '+fn.health_superficial);

  // WS events
  const during = wsEvents.slice(wsBefore);
  const expAct = during.filter(e=>e.type==='expected_action');
  const typing = during.filter(e=>e.type==='typing');
  console.log('  WS events: '+during.length+' expected_action: '+expAct.length+' typing: '+typing.length);
  assert(expAct.length>=20, '≥20 expected_action');
  assert(typing.length>=20, '≥20 typing');

  // Names in narration
  const zayNamed = allGmResponses.filter(r=>r.character==='Zay'&&r.narrative.includes('Zay')).length;
  const nyxNamed = allGmResponses.filter(r=>r.character==='Nyx'&&r.narrative.includes('Nyx')).length;
  console.log('  Zay in GM text: '+zayNamed+'/20');
  console.log('  Nyx in GM text: '+nyxNamed+'/20');

  // Browser
  await page.reload(); await page.waitForLoadState('networkidle'); await sleep(3000);
  const st = await page.locator('.story-scroll').innerText().catch(()=>'');
  assert(/Zay/i.test(st),'Zay in story');
  assert(/Nyx/i.test(st),'Nyx in story');

  console.log(`\n═══════════════════════════════════════`);
  console.log(`  ${passed} passed, ${failed} failed`);
  if (failed) errors.forEach(e=>console.log('  -',e));
  console.log(`═══════════════════════════════════════\n`);
  await browser.close();
  process.exit(failed?1:0);
}
main().catch(e=>{console.error('FATAL:',e);process.exit(1)});
