'use strict';
const $ = id => document.getElementById(id);
const root = '/sound-test/';
let csrf, current, engine, connected = false, muted = localStorage.getItem('inkandbone_audio_muted') === 'true';
const volume = Number(localStorage.getItem('inkandbone_audio_volume') ?? 0.7);
$('volume').value = Number.isFinite(volume) ? volume : 0.7;

async function api(path, version) {
  const options = version === undefined ? {} : {method:'POST', headers:{'Content-Type':'application/json','X-CSRF-Token':csrf}, body:JSON.stringify({version})};
  const response = await fetch(root + path, options);
  const result = await response.json();
  if (!response.ok) throw new Error(result.error || 'Could not connect to the rehearsal. Reload to try again.');
  return result;
}
function status(msg) { $('status').textContent = msg; }
function report(error) { status(error.message || 'Sound connection unavailable.'); }
function line(text) { const p = document.createElement('p'); p.textContent = text; $('scene').append(p); }
function renderAction() {
  $('action').textContent = current.next || 'Revisit the rehearsal';
  $('action').hidden = false;
  $('action').disabled = false;
}
function audioLevel() {
  if (engine) engine.player.audioSystem.setLocalVolume(muted ? 0 : Number($('volume').value));
  $('mute').setAttribute('aria-pressed', String(muted));
  $('mute').textContent = muted ? 'Unmute' : 'Mute';
}
$('mute').onclick = () => { muted = !muted; localStorage.setItem('inkandbone_audio_muted', String(muted)); audioLevel(); };
$('volume').oninput = () => { localStorage.setItem('inkandbone_audio_volume', $('volume').value); audioLevel(); };
function loadScript(src, integrity) {
  return new Promise((resolve,reject) => {
    const s = document.createElement('script'); s.src = src; s.integrity = 'sha384-' + integrity; s.crossOrigin = 'anonymous';
    s.onload = resolve; s.onerror = () => reject(new Error('The Syrinscape player could not load.')); document.head.append(s);
  });
}
function sleep(ms) { return new Promise(resolve => setTimeout(resolve,ms)); }
async function afterPaint() { await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))); }
let busy = false;
$('action').onclick = async () => {
  if (busy) return;
  busy = true; $('action').disabled = true;
  try {
    const restart = !current.next;
    current = await api(restart ? 'restart' : 'advance', current.version);
    $('scene').replaceChildren();
    if (current.scene) {
      const [setup, perception, after] = current.scene.lines;
      line(setup); await sleep(800); line(perception); await afterPaint();
      await api('heard', current.version);
      line(after);
    }
    status(connected ? 'Syrinscape connected' : 'Joining the sound session…');
    renderAction();
  } catch (err) { report(err); }
  finally { busy = false; }
};
async function open() {
  const info = await (await fetch('/api/auth/session')).json();
  $('unlock').hidden = info.authenticated;
  if (!info.authenticated) { status('Unlock your Ink & Bone table to enter the rehearsal.'); return; }
  csrf = info.csrf_token;
  current = await api('state');
  const sessionId = current.listenerSession;
  if (current.scene) current.scene.lines.forEach(line);
  status('Connecting to Syrinscape…');
  const dist = 'https://syrinscape.com/static/dist/13f6592389d6bd2903c55a45ed525780c625bff2/';
  await loadScript(dist+'integration.5f52f359c6411630754e.js','VzpmchxZ+QqsbQSD8BPwmdjUDE4drHq/AMl68hIDI/ArpZvU31uBiHJ7cI8BAjH2');
  await loadScript(dist+'player.3d021545a255e950555d.js','v5YeEsUY9GX0ckicEaZ+LQ9uLfQqAYG+KkqvSp3WbKjO4gZTikSGiFZIUZ5N3gBj');
  engine = window.syrinscape;
  await engine.player.init({
    configure: async () => { engine.config.sessionId = sessionId; },
    onActive: () => {
      connected = true; audioLevel(); status('Syrinscape connected');
      if (current.pos >= 0 && !busy) api('resume', current.version).catch(report);
    },
    onInactive: () => { connected = false; status('Sound will join when you interact with the table.'); },
  });
  $('controls').hidden = false; renderAction();
}
$('unlock').onsubmit = async ev => {
  ev.preventDefault();
  const secret = $('unlock').elements.secret.value; $('unlock').reset();
  try {
    const r = await fetch('/api/auth/login', {method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({secret})});
    if (!r.ok) throw new Error('Unlock failed.');
    await open();
  } catch(err) { report(err); }
};
open().catch(report);
