import type { Dispatch, SetStateAction } from 'react'
import { MacroBar } from '../MacroBar'

export interface PlayerComposerProps {
  characterId: number | null
  hasSession: boolean
  input: string
  onInputChange: (value: string) => void
  onSend: () => Promise<void>
  onSendText: (text: string) => Promise<boolean>
  sending: boolean
  whisperMode: boolean
  setWhisperMode: Dispatch<SetStateAction<boolean>>
}

export function PlayerComposer({
  characterId,
  hasSession,
  input,
  onInputChange,
  onSend,
  onSendText,
  sending,
  whisperMode,
  setWhisperMode,
}: PlayerComposerProps) {
  return (
    <>
      <MacroBar
        characterId={characterId}
        onFire={(text) => { void onSendText(text) }}
        disabled={sending || !hasSession}
      />

      <div className="player-input-bar">
        <button
          type="button"
          className={`whisper-toggle${whisperMode ? ' active' : ''}`}
          onClick={() => setWhisperMode((value) => !value)}
          aria-label={whisperMode ? 'Disable whisper mode' : 'Enable whisper mode'}
          title={whisperMode ? 'Whisper mode on — GM will not respond' : 'Enable whisper mode'}
        >
          🔒
        </button>
        <textarea
          className={`player-input-field${whisperMode ? ' whisper-active' : ''}`}
          placeholder={whisperMode ? 'Whisper (private, no GM response)…' : 'What do you do?'}
          value={input}
          disabled={sending || !hasSession}
          onChange={(event) => onInputChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && !event.shiftKey) {
              event.preventDefault()
              void onSend()
            }
          }}
          rows={3}
        />
        <button
          type="button"
          className="player-input-send"
          disabled={sending || !input.trim() || !hasSession}
          onClick={() => { void onSend() }}
        >
          {sending ? '…' : '↵'}
        </button>
      </div>
    </>
  )
}
