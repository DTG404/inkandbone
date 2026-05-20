#!/usr/bin/env node
import { chromium } from 'playwright';

const BASE = 'http://localhost:7432';
const API = BASE;
let passed = 0, failed = 0, errors = [];
const allGm = [], wsEvents = [];

function assert(c,m){if(c)passed++;else{failed++;errors.push(m);console.error('  X',m)}}
function sleep(ms){return new Promise(r=>setTimeout(r,ms))}
function parseSSE(s){
  return s.split('\n\n').filter(l=>l.startsWith('data: ')).map(l=>l.replace(/^data: /,'')).join('').replace(/\s+/g,' ').trim();
}

async function main(){
  console.log('\n=== DEEP MP E2E: 20 TURNS EACH ===\n');
  const br = await chromium.launch({headless:true});
  const pg = await br.newPage({viewport:{width:1440,height:900}});
  pg.on('websocket',ws=>{ws.on('framereceived',e=>{try{wsEvents.push(JSON.parse(e.payload))}catch{}})});

  // SEED
  const camp = await(await fetch(API+'/api/campaigns',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:'Deep MP',ruleset_id:3})})).json();
  const cid = camp.id;
  async function cch(name,data){
    const c = await(await fetch(API+'/api/campaigns/'+cid+'/characters',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name})})).json();
    await fetch(API+'/api/characters/'+c.id,{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({data_json:JSON.stringify(data)})});
    return c;
  }
  const z = await cch('Zay',{character_type:'Vampire',clan:'Brujah',generation:'12th',sect:'Anarch',predator_type:'Alleycat',hunger:2,humanity:7,health_max:5,hp:5,health_superficial:0,willpower_max:6,xp:0});
  const ny = await cch('Nyx',{character_type:'Vampire',clan:'Toreador',generation:'10th',sect:'Anarch',predator_type:'Siren',hunger:1,humanity:6,health_max:5,hp:5,health_superficial:0,willpower_max:6,xp:0});
  const zid = z.id, nyid = ny.id;
  const chk = await(await fetch(API+'/api/campaigns/'+cid+'/characters')).json();
  assert(JSON.parse(chk.find(x=>x.name==='Zay').data_json).clan==='Brujah','Zay clan');
  assert(JSON.parse(chk.find(x=>x.name==='Nyx').data_json).clan==='Toreador','Nyx clan');
  console.log('  Identity OK');

  const ss = await(await fetch(API+'/api/campaigns/'+cid+'/sessions',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({title:'Deep',date:'2026-05-19'})})).json();
  const sid = ss.id;
  await fetch(API+'/api/settings',{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({campaign_id:cid,character_id:zid,session_id:sid})});

  // BROWSER
  await pg.goto(BASE); await pg.waitForLoadState('networkidle'); await sleep(3000);
  const wsb = wsEvents.length;
  console.log('  Browser connected');

  // TURNS
  async function turn(chid,chn,act,tn){
    const r = await fetch(API+'/api/sessions/'+sid+'/messages',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({role:'user',content:act,character_id:chid})});
    if(!r.ok){console.error('  MSG FAIL',chn,'T'+tn,await r.text())}
    assert(r.status===201,chn+' T'+tn+': msg');
    await fetch(API+'/api/settings',{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({campaign_id:cid,character_id:chid,session_id:sid})});
    const g = await fetch(API+'/api/sessions/'+sid+'/gm-respond-stream',{method:'POST'});
    assert(g.ok,chn+' T'+tn+': GM');
    const txt = parseSSE(await g.text());
    assert(txt.length>80,chn+' T'+tn+': '+txt.length+'c');
    allGm.push({turn:tn,character:chn,narrative:txt});
    const last = txt.slice(-300);
    if(!last.includes('What do you do, '+chn)){
      if(!/\?\s*$/.test(txt.trim())){/* ok */}
    }
  }

  const za = [
    'I push through the Velvet Knot door and scan for Nyx.',
    'I drop into the booth and grab the bourbon.',
    'I am in. But you owe me an explanation later.',
    'I pocket the map. Midnight. Dont be late.',
    'I step into the tunnel, hand on the brick.',
    'Old or pre-Chicago? I ask the dark.',
    'Then lets see what woke up. I move forward.',
    'I swing first. No warning. The thing crumples.',
    'Fist to chest. The bone cracks.',
    'I grab broken glass and stab its ribs.',
    'I haul her arm and run.',
    'I flip rain-soaked pages. This is about us.',
    'We use it. Warn everyone on the list.',
    'Do it. Three hours till dawn.',
    'I drain ox-blood at a ghoul bar.',
    'We feed them bad intel.',
    'Safehouse cleared. Meet at the diner.',
    'We are at war now. Who fires first?',
    'The Tzimisce said the blood sings in us both.',
    'Then we make it expensive. I step into dawn.'
  ];
  const na = [
    'I rise as Zay enters. Contact went dark in the Warehouse district.',
    'Three nights. They were mapping sewer access.',
    'I slide the map across. Midnight. Stockyard tunnels.',
    'Midnight at the tunnel mouth. Auspex sharp.',
    'I follow. Something ahead. Not Sabbat. Older.',
    'A chamber. Recently opened. I think it just woke up.',
    'The altar has a figure. Pale. Tzimisce.',
    'I hit it with Presence. Again!',
    'I cross the room and drive silver into its shoulder.',
    'I grab the journal and vial. GO!',
    'We burst into rain. Decades of kindred data in this journal.',
    'Last page has blood writing. It means property.',
    'I know a Nosferatu who can copy every page.',
    'I toss bills on the bar. Next time I know a butcher.',
    'I order a drink I wont touch.',
    'I know a Malkavian for disinformation.',
    'I slide a burner phone across. Encrypted.',
    'I will compile a threat map. Keep moving.',
    'I go still. What if we are being collected?',
    'I watch him go. Tonight we plan. Tomorrow we hunt.'
  ];

  for(let i=0;i<20;i++){
    console.log('  Round '+(i+1));
    await turn(zid,'Zay',za[i],i+1);
    await turn(nyid,'Nyx',na[i],i+1);
  }

  // ANALYSIS
  console.log('\n=== ANALYSIS ===\n');
  const msgs = await(await fetch(API+'/api/sessions/'+sid+'/messages')).json();
  const zm = msgs.filter(m=>m.role==='user'&&m.character_id===zid);
  const nm = msgs.filter(m=>m.role==='user'&&m.character_id===nyid);
  assert(zm.length===20,'Zay 20: '+zm.length);
  assert(nm.length===20,'Nyx 20: '+nm.length);
  assert(msgs.length>=60,'Total >=60');

  const fin = await(await fetch(API+'/api/campaigns/'+cid+'/characters')).json();
  const fz = JSON.parse(fin.find(x=>x.name==='Zay').data_json||'{}');
  const fn = JSON.parse(fin.find(x=>x.name==='Nyx').data_json||'{}');
  assert(fz.clan==='Brujah','Zay clan: '+fz.clan);
  assert(fn.clan==='Toreador','Nyx clan: '+fn.clan);

  const during = wsEvents.slice(wsb);
  console.log('  WS events:',during.length,'expected_action:',during.filter(e=>e.type==='expected_action').length);
  console.log('  Zay: hunger='+fz.hunger+' XP='+(fz.xp||0));
  console.log('  Nyx: hunger='+fn.hunger+' XP='+(fn.xp||0));

  const avgL = Math.round(allGm.reduce((s,r)=>s+r.narrative.length,0)/allGm.length);
  const zN = allGm.filter(r=>r.character==='Zay'&&r.narrative.includes('Zay')).length;
  const nN = allGm.filter(r=>r.character==='Nyx'&&r.narrative.includes('Nyx')).length;
  console.log('  GM avg '+avgL+'c, Zay named '+zN+'/20, Nyx named '+nN+'/20');

  // Browser check
  await pg.reload(); await pg.waitForLoadState('networkidle'); await sleep(3000);
  const st = await pg.locator('.story-scroll').innerText().catch(()=>'');
  assert(/Zay/i.test(st),'Zay in story');
  assert(/Nyx/i.test(st),'Nyx in story');

  console.log('\n'+passed+' passed, '+failed+' failed');
  if(failed>0){errors.forEach(e=>console.log('  -',e));process.exit(1)}
  await br.close();
}
main().catch(e=>{console.error('FATAL:',e);process.exit(1)});
