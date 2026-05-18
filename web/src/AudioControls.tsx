import { useState, useEffect } from 'react';
import { setAmbientVolume, setAmbientMuted, pauseAmbient, resumeAmbient } from './audio/ambient';

const STORAGE_KEY_MUTED = 'inkandbone_audio_muted';
const STORAGE_KEY_VOLUME = 'inkandbone_audio_volume';
const STORAGE_KEY_PAUSED = 'inkandbone_ambient_paused';

// Exported state accessor for sounds.ts and other consumers
export function getAudioMuted(): boolean {
  return localStorage.getItem(STORAGE_KEY_MUTED) === 'true';
}

export function getAudioVolume(): number {
  const v = localStorage.getItem(STORAGE_KEY_VOLUME);
  return v ? parseFloat(v) : 0.7;
}

export default function AudioControls() {
  const [muted, setMuted] = useState<boolean>(() => getAudioMuted());
  const [volume, setVolume] = useState<number>(() => getAudioVolume());
  const [ambientPaused, setAmbientPaused] = useState<boolean>(
    () => localStorage.getItem(STORAGE_KEY_PAUSED) === 'true'
  );

  useEffect(() => {
    setAmbientMuted(muted);
    localStorage.setItem(STORAGE_KEY_MUTED, String(muted));
  }, [muted]);

  useEffect(() => {
    setAmbientVolume(volume);
    localStorage.setItem(STORAGE_KEY_VOLUME, String(volume));
  }, [volume]);

  // Sync ambient module's paused flag on mount only — re-running on ambientPaused
  // changes would fight with user toggling the pause button.
  useEffect(() => {
    if (ambientPaused) {
      pauseAmbient();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function toggleAmbientPause() {
    const next = !ambientPaused;
    setAmbientPaused(next);
    localStorage.setItem(STORAGE_KEY_PAUSED, String(next));
    if (next) {
      pauseAmbient();
    } else {
      resumeAmbient();
    }
  }

  return (
    <div className="audio-controls">
      <button
        onClick={() => setMuted(m => !m)}
        title={muted ? 'Unmute audio' : 'Mute audio'}
        className="audio-mute-btn"
      >
        {muted ? '🔇' : '🔊'}
      </button>
      <input
        type="range"
        min={0}
        max={1}
        step={0.05}
        value={volume}
        onChange={e => setVolume(parseFloat(e.target.value))}
        disabled={muted}
        title="Volume"
        className="audio-volume-slider"
      />
      <button
        onClick={toggleAmbientPause}
        disabled={muted}
        title={ambientPaused ? 'Resume ambient music' : 'Pause ambient music'}
        className="audio-pause-btn"
      >
        {ambientPaused ? '▶' : '⏸'}
      </button>
    </div>
  );
}
