import { useState, useEffect, useRef, useMemo } from 'react'
import type { ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import { patchSession, createMapPin, fetchTalentDescription, reanalyzeSession, patchSettings } from './api'
import type { GameContext, Message, Session, XPSpendSuggestionsEvent } from './types'
import { CombatPanel } from './CombatPanel'
import { WorldNotesPanel } from './WorldNotesPanel'
import { DiceHistoryPanel } from './DiceHistoryPanel'
import { DiceRoller } from './DiceRoller'
import { MapPanel } from './MapPanel'
import { JournalPanel } from './JournalPanel'
import { CharacterSheetPanel } from './CharacterSheetPanel'
import { NPCRosterPanel } from './NPCRosterPanel'
import { ObjectivesPanel } from './ObjectivesPanel'
import { InventoryPanel } from './InventoryPanel'
import { OraclePanel } from './OraclePanel'
import { RelationshipsPanel } from './RelationshipsPanel'
import { FactionsPanel } from './FactionsPanel'
import { SecretsPanel } from './SecretsPanel'
import { NPCStatBlockPanel } from './NPCStatBlockPanel'
import { AdventuresPanel } from './AdventuresPanel'
import { CalendarPanel } from './CalendarPanel'
import { GMToolsPanel } from './GMToolsPanel'
import { HandoutsPanel } from './HandoutsPanel'
import { CompendiumPanel } from './CompendiumPanel'
import { XPSuggestionsPanel } from './XPSuggestionsPanel'
import { SessionTimeline } from './SessionTimeline'
import { XPLogPanel } from './XPLogPanel'
import { CharacterSelector } from './CharacterSelector'
import { setAmbientTrack } from './audio/ambient'
import { MacroBar } from './MacroBar'
import { wgTalentDescription } from './wgTalentData'
import './App.css'

// ── Turn Order Strip ────────────────────────────────────────

interface TurnOrderStripProps {
  combatants: GameContext['active_combat'] extends null ? never : NonNullable<GameContext['active_combat']>['combatants']
}

function TurnOrderStrip({ combatants }: TurnOrderStripProps) {
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
  mapImagePath: string
  defaultLabel: string
  onClose: () => void
}

function PinPlacementModal({ mapId, mapImagePath, defaultLabel, onClose }: PinPlacementModalProps) {
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
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="pin-modal-backdrop" onClick={(e) => { if (e.target === e.currentTarget) onClose() }}>
      <div className="pin-modal">
        <div className="pin-modal-header">
          <span>Place Map Pin</span>
          <button className="pin-modal-close" onClick={onClose}>×</button>
        </div>
        <p className="pin-modal-hint">Click on the map to place the pin</p>
        <div className="pin-modal-map-wrap">
          <img
            ref={imgRef}
            src={`/api/files/${mapImagePath}`}
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
      </div>
    </div>
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
function normalizeGMContent(text: string): string {
  return text.replace(/\s*(\*\*)?What do you do\??(\*\*)?\s*$/, '\n\n**What do you do?**')
}

function ProseJournal({
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
          mapImagePath={activeMapImagePath}
          defaultLabel={pinModal.content}
          onClose={() => setPinModal(null)}
        />
      )}
    </>
  )
}

// ── Scene Tag Picker ────────────────────────────────────────

const SCENE_TAGS = ['tavern', 'dungeon', 'forest', 'city', 'ocean', 'cave', 'castle', 'rain', 'night', 'battle', 'market', 'temple', 'ruins']

interface SceneTagPickerProps {
  session: Session
  onUpdate: (tags: string) => void
}

function SceneTagPicker({ session, onUpdate }: SceneTagPickerProps) {
  const activeTags = session.scene_tags ? session.scene_tags.split(',').filter(Boolean) : []

  async function toggleTag(tag: string) {
    const newTags = activeTags.includes(tag)
      ? activeTags.filter(t => t !== tag)
      : [...activeTags, tag]
    const tagsStr = newTags.join(',')
    try {
      await patchSession(session.id, { scene_tags: tagsStr })
      onUpdate(tagsStr)
      setAmbientTrack(newTags[0] ?? null)
    } catch (err) {
      console.error('Failed to update scene tags:', err)
    }
  }

  return (
    <div className="scene-tag-picker">
      {SCENE_TAGS.map(tag => (
        <button
          key={tag}
          className={`scene-tag${activeTags.includes(tag) ? ' active' : ''}`}
          onClick={() => toggleTag(tag)}
          title={tag}
        >
          {tag}
        </button>
      ))}
    </div>
  )
}

// ── Session View ────────────────────────────────────────────

export interface SessionViewProps {
  ctx: GameContext
  messages: Message[]
  displayMessages: Message[]
  searchQuery: string
  setSearchQuery: (q: string) => void
  aiEnabled: boolean
  gmResponding: boolean
  streamingText: string
  generatingMap: boolean
  mapOpen: boolean
  setMapOpen: React.Dispatch<React.SetStateAction<boolean>>
  activeMapId: number | null
  activeMapImagePath: string | null
  setActiveMapId: (id: number | null) => void
  setActiveMapImagePath: (path: string | null) => void
  rightTab: string
  setRightTab: React.Dispatch<React.SetStateAction<string>>
  showPlayerHistory: boolean
  setShowPlayerHistory: (show: boolean) => void
  showTalentsPanel: boolean
  setShowTalentsPanel: (show: boolean) => void
  input: string
  setInput: (val: string) => void
  sending: boolean
  whisperMode: boolean
  setWhisperMode: React.Dispatch<React.SetStateAction<boolean>>
  rulesetName: string | null
  xpSuggestionsEvent: XPSpendSuggestionsEvent | null
  xpPanelDismissed: boolean
  setXpPanelDismissed: (dismissed: boolean) => void
  setXPSuggestionsEvent: (event: XPSpendSuggestionsEvent | null) => void
  aiTalentDescs: Record<string, string>
  setAiTalentDescs: React.Dispatch<React.SetStateAction<Record<string, string>>>
  handleSend: () => Promise<void>
  onSendText: (text: string) => Promise<void>
  handleGenerateMap: () => Promise<void>
  handleSpendXP: (characterId: number, field: string, newValue: number) => Promise<void>
  lastEvent: unknown
  setCtx: React.Dispatch<React.SetStateAction<GameContext | null>>
  typingNames: string[]
  charactersList: { id: number; name: string }[]
  selectedCharacterId: number | null
  onCharacterSelect: (id: number) => void
}

export function SessionView({
  ctx,
  messages,
  displayMessages,
  searchQuery,
  setSearchQuery,
  aiEnabled,
  gmResponding,
  streamingText,
  generatingMap,
  mapOpen,
  setMapOpen,
  activeMapId,
  activeMapImagePath,
  setActiveMapId,
  setActiveMapImagePath,
  rightTab,
  setRightTab,
  showPlayerHistory,
  setShowPlayerHistory,
  showTalentsPanel,
  setShowTalentsPanel,
  input,
  setInput,
  sending,
  whisperMode,
  setWhisperMode,
  rulesetName,
  xpSuggestionsEvent,
  xpPanelDismissed,
  setXpPanelDismissed,
  setXPSuggestionsEvent,
  aiTalentDescs,
  setAiTalentDescs,
  handleSend,
  onSendText,
  handleGenerateMap,
  handleSpendXP,
  lastEvent,
  setCtx,
  charactersList,
  selectedCharacterId,
  onCharacterSelect,
  typingNames,
}: SessionViewProps) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const [journalSubTab, setJournalSubTab] = useState<'notes' | 'timeline'>('notes')

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages, streamingText])

  // When the talents panel opens, fetch AI descriptions for any talent/power
  // that has no static description.
  useEffect(() => {
    if (!showTalentsPanel || !ctx?.character) return
    let charData: Record<string, unknown> = {}
    try { charData = JSON.parse(ctx.character.data_json || '{}') } catch { /* ignore */ }
    const system = ctx?.campaign?.ruleset_id ? 'wrath_glory' : 'wrath_glory' // best effort
    const allNames: string[] = []
    const talentsStr = String(charData.talents ?? '').trim()
    const powersStr = String(charData.powers ?? '').trim()
    for (const s of [talentsStr, powersStr]) {
      if (s) s.split(/[|\n]/).map(t => t.trim().replace(/^[-•]\s*/, '')).filter(Boolean).forEach(n => allNames.push(n))
    }
    const unknown = allNames.filter(n => !wgTalentDescription(n) && !aiTalentDescs[n])
    if (unknown.length === 0) return
    unknown.forEach(name => {
      fetchTalentDescription(name, system).then(desc => {
        if (desc) setAiTalentDescs(prev => ({ ...prev, [name]: desc }))
      })
    })
  }, [showTalentsPanel, ctx?.character]) // eslint-disable-line react-hooks/exhaustive-deps

  const sessionTitle = ctx.session?.title?.toUpperCase() ?? ''
  const sessionDate = ctx.session?.date
    ? new Date(ctx.session.date).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })
    : ''

  return (
    <div className="grimoire-body">

      {/* Player History Overlay */}
      {showPlayerHistory && (
        <div className="player-history-overlay">
          <div className="player-history-header">
            <span>Your Actions</span>
            <button onClick={() => setShowPlayerHistory(false)}>×</button>
          </div>
          <div className="player-history-list">
            {messages.filter(m => m.role === 'user' && !m.whisper).map(m => (
              <div key={m.id} className="player-history-item">
                <p>{m.content}</p>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Talents & Powers Overlay */}
      {showTalentsPanel && ctx.character && (() => {
        let charData: Record<string, unknown> = {}
        try { charData = JSON.parse(ctx.character.data_json || '{}') } catch { /* ignore */ }

        if (rulesetName === 'vtm') {
          // VtM: show disciplines with descriptions, merits/flaws, convictions, touchstones
          const vtmDisciplineKeys = [
            'animalism', 'auspex', 'blood_sorcery', 'celerity', 'dominate',
            'fortitude', 'obfuscate', 'oblivion', 'potence', 'presence', 'protean',
          ]
          const vtmDisciplineDesc: Record<string, string> = {
            animalism: 'Command and communicate with beasts. Soothe or inflame animal rage. At higher levels, tap the Beast within other Kindred.',
            auspex: 'Heightened senses, aura perception, and telepathy. Pierce illusions and sense the supernatural beyond mortal limits.',
            blood_sorcery: 'Ritae and blood magic drawn from stolen Tremere sorcery. Curse, ward, and reshape vitae with ritualistic precision.',
            celerity: 'Supernatural speed and reflexes. Move faster than the eye can follow, act multiple times in a single moment.',
            dominate: 'Compel mortals and Kindred with a word or gaze. Issue commands, rewrite memories, and shatter the will of the weak.',
            fortitude: 'Superhuman resilience. Shrug off blows, endure fire and sunlight longer, and ignore pain that would break lesser beings.',
            obfuscate: 'Cloak your presence, alter your appearance, or vanish entirely from mortal senses. The perfect predator is never seen.',
            oblivion: 'Wield shadows and death itself. Communicate with the dead, conjure darkness, and rend souls from their moorings.',
            potence: 'Superhuman strength. Crush, lift, and destroy with a touch. Your blows land with the force of catastrophe.',
            presence: 'Supernatural charisma and emotional control. Inspire awe, fear, or adoration in mortals and Kindred alike.',
            protean: 'Reshape your body at will. Grow claws, meld into earth, turn to mist, or take the form of a beast of the night.',
          }
          const activeDisciplines = vtmDisciplineKeys
            .map(k => ({ key: k, rating: Number(charData[k] ?? 0) }))
            .filter(d => d.rating > 0)
          const meritsFlaws = String(charData.merits_flaws ?? '').trim()
          const convictions = String(charData.convictions ?? '').trim()
          const touchstones = String(charData.touchstones ?? '').trim()
          const bloodPotency = Number(charData.blood_potency ?? 1)

          return (
            <div className="talents-overlay">
              <div className="talents-overlay-header">
                <span>Disciplines &amp; Powers — {ctx.character.name}</span>
                <button onClick={() => setShowTalentsPanel(false)}>×</button>
              </div>
              <div className="talents-overlay-body">
                <div className="talents-section">
                  <div className="talents-section-title">Disciplines</div>
                  <div className="talents-entry" style={{ marginBottom: '0.5rem', opacity: 0.7, fontSize: '0.8rem' }}>
                    Blood Potency {bloodPotency} — in-clan disciplines cost {bloodPotency > 0 ? 'new dots × 5' : '5'} XP; out-of-clan cost new dots × 7 XP
                  </div>
                  {activeDisciplines.length > 0
                    ? activeDisciplines.map(({ key, rating }) => (
                        <div key={key} className="talents-entry">
                          <div className="talents-entry-name">
                            {key.replace(/_/g, ' ').replace(/\b\w/g, c => c.toUpperCase())}
                            <span className="talents-rank-badge">{'●'.repeat(rating)}{'○'.repeat(Math.max(0, 5 - rating))}</span>
                          </div>
                          <div className="talents-entry-desc">{vtmDisciplineDesc[key]}</div>
                        </div>
                      ))
                    : <div className="talents-empty">No disciplines learned yet.</div>
                  }
                </div>
                {meritsFlaws && (
                  <div className="talents-section">
                    <div className="talents-section-title">Merits &amp; Flaws</div>
                    {meritsFlaws.split(/[,\n|]/).map(s => s.trim()).filter(Boolean).map((entry, i) => {
                      const isFlaw = /flaw/i.test(entry)
                      const isMerit = /merit/i.test(entry)
                      const label = entry.replace(/^(merit|flaw)\s*[:—-]\s*/i, '')
                      return (
                        <div key={i} className="talents-entry">
                          <div className="talents-entry-name" style={{ color: isFlaw ? 'var(--crimson)' : isMerit ? 'var(--gold)' : undefined }}>
                            {isFlaw ? '⚠ ' : isMerit ? '★ ' : ''}{label}
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )}
                {(convictions || touchstones) && (
                  <div className="talents-section">
                    <div className="talents-section-title">Humanity Anchors</div>
                    {convictions && (
                      <div className="talents-entry">
                        <div className="talents-entry-name">Convictions</div>
                        <div className="talents-entry-desc">{convictions}</div>
                      </div>
                    )}
                    {touchstones && (
                      <div className="talents-entry">
                        <div className="talents-entry-name">Touchstones</div>
                        <div className="talents-entry-desc">{touchstones}</div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          )
        }

        // W&G (and other systems): original talents & powers panel
        const talents = String(charData.talents ?? '').trim()
        const powers = String(charData.powers ?? '').trim()
        const talentRanks = (charData.talent_ranks ?? {}) as Record<string, number>
        return (
          <div className="talents-overlay">
            <div className="talents-overlay-header">
              <span>Talents &amp; Powers — {ctx.character.name}</span>
              <button onClick={() => setShowTalentsPanel(false)}>×</button>
            </div>
            <div className="talents-overlay-body">
              <div className="talents-section">
                <div className="talents-section-title">Talents</div>
                {talents
                  ? talents.split(/[|\n]/).map(s => s.trim()).filter(Boolean).map((t, i) => {
                      const name = t.replace(/^[-•]\s*/, '')
                      const rank = talentRanks[name] ?? 1
                      const desc = wgTalentDescription(name) || aiTalentDescs[name] || ''
                      return (
                        <div key={i} className="talents-entry">
                          <div className="talents-entry-name">
                            {name}{rank > 1 && <span className="talents-rank-badge">Rank {rank}</span>}
                          </div>
                          {desc
                            ? <div className="talents-entry-desc">{desc}</div>
                            : <div className="talents-entry-desc talents-entry-loading">Loading description…</div>
                          }
                        </div>
                      )
                    })
                  : <div className="talents-empty">No talents recorded.</div>
                }
              </div>
              {powers && (
                <div className="talents-section">
                  <div className="talents-section-title">Psychic Powers</div>
                  {powers.split(/[|\n]/).map(s => s.trim()).filter(Boolean).map((p, i) => {
                    const name = p.replace(/^[-•]\s*/, '')
                    const desc = wgTalentDescription(name) || aiTalentDescs[name] || ''
                    return (
                      <div key={i} className="talents-entry">
                        <div className="talents-entry-name">{name}</div>
                        {desc
                          ? <div className="talents-entry-desc">{desc}</div>
                          : <div className="talents-entry-desc talents-entry-loading">Loading description…</div>
                        }
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          </div>
        )
      })()}

      {/* Left Sidebar */}
      <aside className="sidebar-left">
        <CharacterSelector
          characters={charactersList}
          selectedId={selectedCharacterId}
          onSelect={onCharacterSelect}
        />
        <CharacterSheetPanel
          character={ctx?.character ?? null}
          rulesetId={ctx?.campaign?.ruleset_id ?? null}
          lastEvent={lastEvent}
          onRollField={ctx.character && ctx.session
            ? (label) => {
                void onSendText(`${ctx.character!.name} attempts a ${label} check.`)
              }
            : undefined}
          afterTracks={ctx.session ? (
            <>
              <DiceRoller sessionId={ctx.session.id} />
              <DiceHistoryPanel sessionId={ctx.session.id} lastEvent={lastEvent} />
            </>
          ) : undefined}
        />
        <hr className="sidebar-rule" />
        {ctx.character && (
          <InventoryPanel
            characterId={ctx.character.id}
            characterCurrencyBalance={ctx.character.currency_balance ?? 0}
            characterCurrencyLabel={ctx.character.currency_label ?? 'Gold'}
            lastEvent={lastEvent}
          />
        )}
        <hr className="sidebar-rule" />
        <XPLogPanel sessionId={ctx?.session?.id ?? null} lastEvent={lastEvent} />
      </aside>

      {/* Center Column */}
      <main className="story-center">
        {ctx.active_combat && (
          <TurnOrderStrip combatants={ctx.active_combat.combatants} />
        )}

        <div className="story-search-bar">
          <input
            type="search"
            placeholder="Search story…"
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
          />
          {searchQuery && (
            <button onClick={() => setSearchQuery('')}>×</button>
          )}
        </div>

        <div className="story-scroll" ref={scrollRef}>
          {sessionTitle && (
            <>
              <div className="session-title">✦ {sessionTitle} ✦</div>
              {sessionDate && <div className="session-date">{sessionDate}</div>}
              {ctx.session && rulesetName !== 'vtm' && (
                <SceneTagPicker
                  session={ctx.session}
                  onUpdate={(tags) => {
                    setCtx(prev => prev && prev.session
                      ? { ...prev, session: { ...prev.session, scene_tags: tags } }
                      : prev
                    )
                  }}
                />
              )}
            </>
          )}
          {ctx.active_combat && <CombatPanel combat={ctx.active_combat} />}
          <ProseJournal
            messages={displayMessages}
            characterName={ctx.character?.name ?? 'Player'}
            searchQuery={searchQuery}
            activeMapId={activeMapId}
            activeMapImagePath={activeMapImagePath}
            charactersList={charactersList}
          />
          {streamingText && (
            <div className="prose-gm streaming">
              <ReactMarkdown>{normalizeGMContent(streamingText)}</ReactMarkdown>
            </div>
          )}
          {gmResponding && !streamingText && (
            <p className="gm-thinking">▸ The GM is narrating…</p>
          )}
        </div>

        {typingNames.length > 0 && (
          <p className="typing-indicator">⏳ {typingNames.join(' & ')} {typingNames.length === 1 ? 'is' : 'are'} thinking…</p>
        )}

        <MacroBar
          characterId={ctx.character?.id ?? null}
          onFire={(text) => { void onSendText(text) }}
          disabled={sending || !ctx.session}
        />

        <div className="player-input-bar">
          <button
            type="button"
            className={`whisper-toggle${whisperMode ? ' active' : ''}`}
            onClick={() => setWhisperMode((v) => !v)}
            title={whisperMode ? 'Whisper mode on — GM will not respond' : 'Enable whisper mode'}
          >
            🔒
          </button>
          <textarea
            className={`player-input-field${whisperMode ? ' whisper-active' : ''}`}
            placeholder={whisperMode ? 'Whisper (private, no GM response)…' : 'What do you do?'}
            value={input}
            disabled={sending || !ctx.session}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                handleSend()
              }
            }}
            rows={3}
          />
          <button
            type="button"
            className="player-input-send"
            disabled={sending || !input.trim() || !ctx.session}
            onClick={handleSend}
          >
            {sending ? '…' : '↵'}
          </button>
        </div>

        <div className="map-drawer">
          <div className="map-drawer-handle-row">
            <button
              type="button"
              className="map-drawer-handle"
              onClick={() => setMapOpen((o) => !o)}
            >
              {mapOpen
                ? '[ ▴ COLLAPSE ]'
                : `[ ${ctx.campaign?.name?.toUpperCase() ?? 'THE IRONLANDS'} ▾ ]`}
            </button>
            {aiEnabled && (
              <button
                type="button"
                className="map-generate-btn"
                onClick={handleGenerateMap}
                disabled={generatingMap}
                title="Generate a map with AI"
              >
                {generatingMap ? '…' : '✦ Generate Map'}
              </button>
            )}
          </div>
          <div className={`map-drawer-content${mapOpen ? ' open' : ''}`}>
            <div className="map-drawer-inner">
              <MapPanel
                campaignId={ctx?.campaign?.id ?? null}
                lastEvent={lastEvent}
                onActiveMapChange={(mapId, imagePath) => {
                  setActiveMapId(mapId)
                  setActiveMapImagePath(imagePath)
                }}
              />
            </div>
          </div>
        </div>
      </main>

      <XPSuggestionsPanel
        event={xpPanelDismissed ? null : xpSuggestionsEvent}
        onDismiss={() => { setXPSuggestionsEvent(null); setXpPanelDismissed(false) }}
        onHide={() => setXpPanelDismissed(true)}
        onSpend={handleSpendXP}
      />

      {/* Right Sidebar */}
      <aside className="sidebar-right">
        <div className="tab-bar">
          <button
            className={`tab-btn${rightTab === 'handouts' ? ' active' : ''}`}
            onClick={() => setRightTab('handouts')}
          >
            Handouts
          </button>
          <button
            className={`tab-btn${rightTab === 'compendium' ? ' active' : ''}`}
            onClick={() => setRightTab('compendium')}
          >
            Compendium
          </button>
          <button
            className={`tab-btn${rightTab === 'notes' ? ' active' : ''}`}
            onClick={() => setRightTab('notes')}
          >
            Notes
          </button>
          <button
            className={`tab-btn${rightTab === 'journal' ? ' active' : ''}`}
            onClick={() => setRightTab('journal')}
          >
            Journal
          </button>
          <button
            className={`tab-btn${rightTab === 'npcs' ? ' active' : ''}`}
            onClick={() => setRightTab('npcs')}
          >
            NPCs
          </button>
          <button
            className={`tab-btn${rightTab === 'objectives' ? ' active' : ''}`}
            onClick={() => setRightTab('objectives')}
          >
            Objectives
          </button>
          <button
            className={`tab-btn${rightTab === 'oracle' ? ' active' : ''}`}
            onClick={() => setRightTab('oracle')}
          >
            Oracle
          </button>
          <button
            className={`tab-btn${rightTab === 'relationships' ? ' active' : ''}`}
            onClick={() => setRightTab('relationships')}
          >
            Relations
          </button>
          <button
            className={`tab-btn${rightTab === 'factions' ? ' active' : ''}`}
            onClick={() => setRightTab('factions')}
          >
            Factions
          </button>
          <button
            className={`tab-btn${rightTab === 'calendar' ? ' active' : ''}`}
            onClick={() => setRightTab('calendar')}
          >
            Calendar
          </button>
          <button
            className={`tab-btn${rightTab === 'npcstats' ? ' active' : ''}`}
            onClick={() => setRightTab('npcstats')}
          >
            Stat Blocks
          </button>
          <button
            className={`tab-btn${rightTab === 'adventures' ? ' active' : ''}`}
            onClick={() => setRightTab('adventures')}
          >
            Adventures
          </button>
          <button
            className={`tab-btn${rightTab === 'secrets' ? ' active' : ''}`}
            onClick={() => setRightTab('secrets')}
          >
            Secrets
          </button>
          <button
            className={`tab-btn${rightTab === 'gmtools' ? ' active' : ''}`}
            onClick={() => setRightTab('gmtools')}
          >
            GM Tools
          </button>
        </div>
        <div className="tab-content">
          {rightTab === 'handouts' && ctx.campaign && (
            <HandoutsPanel campaignId={ctx.campaign.id} lastEvent={lastEvent} />
          )}
          {rightTab === 'compendium' && ctx.campaign && (
            <CompendiumPanel rulesetId={ctx.campaign.ruleset_id} />
          )}
          {rightTab === 'notes' && ctx.campaign && (
            <WorldNotesPanel
              campaignId={ctx.campaign.id}
              lastEvent={lastEvent}
              aiEnabled={aiEnabled}
            />
          )}
          {rightTab === 'journal' && (
            <div className="journal-container">
              <div className="journal-subtabs">
                <button className={`journal-subtab${journalSubTab === 'notes' ? ' active' : ''}`} onClick={() => setJournalSubTab('notes')}>Notes</button>
                <button className={`journal-subtab${journalSubTab === 'timeline' ? ' active' : ''}`} onClick={() => setJournalSubTab('timeline')}>Timeline</button>
              </div>
              <button
                className="journal-reanalyze-btn"
                onClick={async () => {
                  if (!ctx?.session?.id) return
                  try {
                    await reanalyzeSession(ctx.session.id)
                  } catch (e) {
                    console.error(e)
                  }
                }}
                title="Re-analyze session for objectives and NPCs"
              >
                ↻ Reanalyze
              </button>
              {journalSubTab === 'notes' ? (
                <JournalPanel
                  session={ctx?.session ?? null}
                  campaignId={ctx?.campaign?.id ?? null}
                  lastEvent={lastEvent}
                  aiEnabled={aiEnabled}
                />
              ) : ctx?.session?.id != null ? (
                <SessionTimeline sessionId={ctx.session.id} lastEvent={lastEvent} />
              ) : null}
            </div>
          )}
          {rightTab === 'npcs' && (
            <NPCRosterPanel
              sessionId={ctx?.session?.id ?? null}
              lastEvent={lastEvent}
            />
          )}
          {rightTab === 'objectives' && (
            <ObjectivesPanel campaignId={ctx?.campaign?.id ?? null} sessionId={ctx?.session?.id ?? null} lastEvent={lastEvent} />
          )}
          {rightTab === 'oracle' && ctx.session && (
            <OraclePanel sessionId={ctx.session.id} />
          )}
          {rightTab === 'relationships' && ctx.campaign && (
            <RelationshipsPanel campaignId={ctx.campaign.id} />
          )}
          {rightTab === 'factions' && ctx.campaign && (
            <FactionsPanel campaignId={ctx.campaign.id} />
          )}
          {rightTab === 'calendar' && ctx.campaign && (
            <CalendarPanel
              campaignId={ctx.campaign.id}
              sessionId={ctx.session?.id ?? null}
              lastEvent={lastEvent}
            />
          )}
          {rightTab === 'npcstats' && ctx.campaign && (
            <NPCStatBlockPanel campaignId={ctx.campaign.id} />
          )}
          {rightTab === 'adventures' && ctx.campaign && (
            <AdventuresPanel
              campaignId={ctx.campaign.id}
              onSessionClick={async (sessionId: number) => {
                await patchSettings({ session_id: sessionId })
              }}
              lastEvent={lastEvent}
            />
          )}
          {rightTab === 'secrets' && ctx.campaign && (
            <SecretsPanel
              campaignId={ctx.campaign.id}
              sessionId={ctx?.session?.id ?? null}
              lastEvent={lastEvent}
            />
          )}
          {rightTab === 'gmtools' && (
            <GMToolsPanel
              sessionId={ctx?.session?.id ?? null}
              campaignId={ctx?.campaign?.id ?? null}
              aiEnabled={aiEnabled}
            />
          )}
        </div>
      </aside>

    </div>
  )
}
