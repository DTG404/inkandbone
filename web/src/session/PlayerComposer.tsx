import { useMobileWorkspace } from '../layout/useMobileWorkspace'
import type { Dispatch, SetStateAction } from 'react'
import { MacroBar } from '../MacroBar'

export interface PlayerComposerProps {
  characterId: number | null
  hasSession: boolean
  input: string
  onInputChange: (value: string) => void
  recoverTurn?: () => void
  retrySubmission?: boolean
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
  recoverTurn, retrySubmission, onSend,
  onSendText,
  sending,
  whisperMode,
  setWhisperMode,
}: PlayerComposerProps) {
  const phone = useMobileWorkspace()
  const awaitingRecovery = !!recoverTurn && (!whisperMode || retrySubmission)
  return (
    <>
      {recoverTurn && !sending && <div role="status">
        <p>{retrySubmission ? 'Your submission is awaiting confirmation.' : 'Your action is saved and awaits a GM response.'}</p>
        <button type="button" onClick={recoverTurn}>{retrySubmission ? 'Retry submission' : 'Resume GM response'}</button>
      </div>}
      <MacroBar
        characterId={characterId}
        onFire={(text) => { void onSendText(text) }}
        disabled={sending || !!recoverTurn || !hasSession}
      />

      <div className="player-input-bar">
        <button
          type="button"
          className={`whisper-toggle${whisperMode ? ' active' : ''}`}
          onClick={() => setWhisperMode((value) => !value)}
          aria-pressed={whisperMode}
          aria-label={whisperMode ? 'Disable whisper mode' : 'Enable whisper mode'}
          title={whisperMode ? 'Whisper mode on — GM will not respond' : 'Enable whisper mode'}
        >
          {phone ? 'Whisper' : '🔒'}
        </button>
        <textarea
          className={`player-input-field${whisperMode ? ' whisper-active' : ''}`}
          placeholder={whisperMode ? 'Whisper (private, no GM response)…' : 'What do you do?'}
          value={input}
          disabled={sending || !hasSession}
          onChange={(event) => onInputChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && !event.shiftKey && !awaitingRecovery) {
              event.preventDefault()
              void onSend()
            }
          }}
          rows={3}
        />
        <button
          type="button"
          className="player-input-send"
          disabled={sending || awaitingRecovery || !input.trim() || !hasSession}
          onClick={() => { void onSend() }}
        >
          {sending ? '…' : phone ? 'Send' : '↵'}
        </button>
      </div>
    </>
  )
}
