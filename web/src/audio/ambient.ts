const YT_BASE = 'https://www.youtube.com/embed/?list=search&q=';

let currentIframeId: string | null = null;
let currentTag: string | null = null;
let masterVolume = 1.0;
let muted = false;

export function setAmbientVolume(volume: number): void {
  masterVolume = Math.max(0, Math.min(1, volume));
  const iframe = document.getElementById('yt-ambient') as HTMLIFrameElement | null;
  if (!iframe?.contentWindow) return;
  const vol = Math.round(masterVolume * 100);
  try {
    iframe.contentWindow.postMessage(
      JSON.stringify({ event: 'command', func: 'setVolume', args: [vol] }),
      '*'
    );
  } catch { /* cross-origin restrictions may apply */ }
}

export function setAmbientMuted(isMuted: boolean): void {
  muted = isMuted;
  const iframe = document.getElementById('yt-ambient') as HTMLIFrameElement | null;
  if (!iframe?.contentWindow) return;
  try {
    iframe.contentWindow.postMessage(
      JSON.stringify({ event: 'command', func: isMuted ? 'mute' : 'unMute', args: [] }),
      '*'
    );
  } catch { /* cross-origin restrictions may apply */ }
}

export function pauseAmbient(): void {
  const el = document.getElementById('yt-ambient');
  if (el) el.remove();
}

export function resumeAmbient(): void {
  if (currentTag) {
    const tag = currentTag;
    currentTag = null;
    setAmbientTrack(tag);
  }
}

export async function setAmbientTrack(tag: string | null): Promise<void> {
  if (tag === currentTag) return;
  currentTag = tag;

  const oldEl = document.getElementById('yt-ambient');
  if (oldEl) oldEl.remove();

  if (!tag) return;

  const iframe = document.createElement('iframe');
  iframe.id = 'yt-ambient';
  iframe.style.display = 'none';
  iframe.src = `${YT_BASE}${encodeURIComponent(tag + ' ambient music rpg')}&autoplay=1&loop=1`;
  iframe.allow = 'autoplay';
  document.body.appendChild(iframe);
}

export function stopAmbient(): void {
  const el = document.getElementById('yt-ambient');
  if (el) el.remove();
  currentTag = null;
}
