import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { GameContext } from '../types'
import { NarrativeStream } from './NarrativeStream'
import { PlayerComposer } from './PlayerComposer'
import { RightWorkspace } from './RightWorkspace'

afterEach(cleanup)

describe('extracted session modules', () => {
  it('preserves narrative empty and message accessible output', () => {
    const { rerender } = render(
      <NarrativeStream
        messages={[]}
        characterName="Player"
        activeMapId={null}
        activeMapImagePath={null}
        charactersList={[]}
      />,
    )
    expect(screen.getByText('The story has not yet begun.')).toBeInTheDocument()
    rerender(
      <NarrativeStream
        messages={[{ id: 1, session_id: 1, role: 'user', content: 'I listen.', created_at: '' }]}
        characterName="Nyx"
        activeMapId={null}
        activeMapImagePath={null}
        charactersList={[]}
      />,
    )
    expect(screen.getByText('Nyx speaks')).toBeInTheDocument()
    expect(screen.getByText('I listen.')).toBeInTheDocument()
  })

  it('preserves composer input, whisper, and send callbacks', () => {
    const onSend = vi.fn().mockResolvedValue(undefined)
    const setWhisperMode = vi.fn()
    render(
      <PlayerComposer
        characterId={null}
        hasSession
        input="Open the gate"
        onInputChange={vi.fn()}
        onSend={onSend}
        onSendText={vi.fn().mockResolvedValue(true)}
        sending={false}
        whisperMode={false}
        setWhisperMode={setWhisperMode}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Enable whisper mode' }))
    expect(setWhisperMode).toHaveBeenCalled()
    fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Enter' })
    expect(onSend).toHaveBeenCalledTimes(1)
  })

  it('preserves right-workspace navigation and active region semantics', () => {
    const ctx: GameContext = {
      campaign: null,
      character: null,
      session: null,
      recent_messages: [],
      active_combat: null,
    }
    render(
      <RightWorkspace
        aiEnabled={false}
        ctx={ctx}
        lastEvent={null}
        mobileDestination="world"
        onMobileDestinationChange={vi.fn()}
        rightTab="notes"
        setRightTab={vi.fn()}
      />,
    )
    expect(screen.getByRole('region')).toHaveAttribute('aria-labelledby', 'workspace-panel-control-notes')
    expect(screen.getByRole('button', { name: 'Notes' })).toHaveAttribute('aria-current', 'page')
  })
})
