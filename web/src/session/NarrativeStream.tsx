import { useMemo, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import { createMapPin, mapAssetURL } from '../api'
import type { GameContext, Message } from '../types'
import { Dialog } from '../ui/Dialog'
import { IconButton } from '../ui/IconButton'
import { useToast } from '../ui/ToastProvider'

// ── Turn Order Strip ────────────────────────────────────────

interface TurnOrderStripProps {
  combatants: GameContext['active_combat'] extends null ? never : NonNullable<GameContext['active_combat']>['combatants']
}

export function TurnOrderStrip({ combatants }: TurnOrderStripProps) {
  return (
    <div className="turn-strip">
      {combatants.map((c, idx) => {
        const isDead = c.hp_current <= 0
        const isActive = idx === 0
        return (
          <div
            key={c.id}
            className={`turn-chip${isActive ? ' active-turn' : ''}${isDead ? ' dead' : ''}`}
          >
            {c.name} ({c.initiative})
          </div>
        )
      })}
    </div>
  )
}

// ── Pin Placement Modal ─────────────────────────────────────

interface PinPlacementModalProps {
  mapId: number
  defaultLabel: string
  onClose: () => void
}

function PinPlacementModal({ mapId, defaultLabel, onClose }: PinPlacementModalProps) {
  const toast = useToast()
  const [label, setLabel] = useState(defaultLabel.slice(0, 60))
  const [note, setNote] = useState('')
  const [pos, setPos] = useState<{ x: number; y: number } | null>(null)
  const [saving, setSaving] = useState(false)
  const imgRef = useRef<HTMLImageElement>(null)

  function handleImageClick(e: React.MouseEvent<HTMLImageElement>) {
    const rect = imgRef.current?.getBoundingClientRect()
    if (!rect) return
    setPos({
      x: (e.clientX - rect.left) / rect.width,
      y: (e.clientY - rect.top) / rect.height,
    })
  }

  async function handleSubmit() {
    if (!pos) return
    setSaving(true)
    try {
      await createMapPin(mapId, { x: pos.x, y: pos.y, label, note, color: '#c9a84c' })
      onClose()
    } catch (err) {
      console.error(err)
      toast.error('Could not place map pin.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Dialog open title="Place Map Pin" onClose={onClose} className="pin-modal">
        <div className="pin-modal-header">
          <span>Place Map Pin</span>
          <IconButton className="pin-modal-close" label="Close map pin dialog" icon="×" onClick={onClose} />
        </div>
        <p className="pin-modal-hint">Click on the map to place the pin</p>
        <div className="pin-modal-map-wrap">
          <img
            ref={imgRef}
            src={mapAssetURL(mapId)}
            alt="Map"
            className="pin-modal-map"
            onClick={handleImageClick}
          />
          {pos && (
            <div
              className="pin-modal-marker"
              style={{ left: `${pos.x * 100}%`, top: `${pos.y * 100}%` }}
            >
              ✦
            </div>
          )}
        </div>
        <input
          className="pin-modal-input"
          value={label}
          onChange={(e) => setLabel(e.target.value)}
          placeholder="Label…"
        />
        <textarea
          className="pin-modal-textarea"
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="Note…"
          rows={3}
        />
        <button
          className="pin-modal-submit"
          onClick={handleSubmit}
          disabled={!pos || saving || !label.trim()}
        >
          {saving ? 'Saving…' : 'Place Pin'}
        </button>
    </Dialog>
  )
}

// ── Prose Journal ───────────────────────────────────────────

function highlightText(text: string, query: string): ReactNode {
  if (!query) return text
  const lower = text.toLowerCase()
  const lowerQ = query.toLowerCase()
  const parts: ReactNode[] = []
  let start = 0
  let idx = lower.indexOf(lowerQ, start)
  while (idx !== -1) {
    if (idx > start) parts.push(text.slice(start, idx))
    parts.push(<mark key={idx}>{text.slice(idx, idx + query.length)}</mark>)
    start = idx + query.length
    idx = lower.indexOf(lowerQ, start)
  }
  if (start < text.length) parts.push(text.slice(start))
  return <>{parts}</>
}

interface ProseJournalProps {
  messages: Message[]
  characterName: string
  searchQuery?: string
  activeMapId: number | null
  activeMapImagePath: string | null
  charactersList: { id: number; name: string }[]
}

// Ensure "What do you do?" at the end of GM responses is always its own paragraph
// and rendered bold+italic gold to stand out as the player prompt cue.
export function normalizeGMContent(text: string): string {
  return text.replace(/\s*(\*\*)?What do you do\??(\*\*)?\s*$/, '\n\n**What do you do?**')
}

export function NarrativeStream({
  messages,
  characterName,
  searchQuery = '',
  activeMapId,
  activeMapImagePath,
  charactersList = [],
}: ProseJournalProps) {
  const [pinModal, setPinModal] = useState<{ content: string } | null>(null)
  const charNameMap = useMemo(() => {
    const map: Record<number, string> = {}
    for (const c of charactersList) map[c.id] = c.name
    return map
  }, [charactersList])

  if (messages.length === 0) {
    return <p className="empty">The story has not yet begun.</p>
  }

  const nodes: ReactNode[] = []
  messages.forEach((m, i) => {
    if (m.role === 'assistant') {
      nodes.push(
        <div key={m.id} className="prose-gm prose-gm-wrap">
          <ReactMarkdown>{normalizeGMContent(m.content)}</ReactMarkdown>
          {activeMapId !== null && activeMapImagePath !== null && (
            <button
              className="prose-pin-btn"
              title="Place as map pin"
              onClick={() => setPinModal({ content: m.content.replace(/[#*_`[\]]/g, '').slice(0, 60) })}
            >
              📍
            </button>
          )}
        </div>
      )
    } else {
      const isWhisper = m.whisper === true
      const speakerName = m.character_id != null && charNameMap[m.character_id]
        ? charNameMap[m.character_id]
        : characterName
      nodes.push(
        <div key={m.id} className={`prose-player${isWhisper ? ' prose-player--whisper' : ''}`}>
          <div className="prose-player-label">{speakerName} speaks</div>
          <p className="prose-player-text">
            {searchQuery ? highlightText(m.content, searchQuery) : m.content}
          </p>
        </div>
      )
      if (i < messages.length - 1) {
        nodes.push(
          <div key={`div-${m.id}`} className="prose-divider">◆</div>
        )
      }
    }
  })

  return (
    <>
      {nodes}
      {pinModal && activeMapId !== null && activeMapImagePath !== null && (
        <PinPlacementModal
          mapId={activeMapId}
          defaultLabel={pinModal.content}
          onClose={() => setPinModal(null)}
        />
      )}
    </>
  )
}

// ── Scene Tag Picker ────────────────────────────────────────
