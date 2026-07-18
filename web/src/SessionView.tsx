import { useState, useEffect, useRef } from 'react'
import ReactMarkdown from 'react-markdown'
import { patchSession, fetchTalentDescription, fetchNPCs } from './api'
import type { GameContext, Message, Session, XPSpendSuggestionsEvent, SessionNPC } from './types'
import { CombatPanel } from './CombatPanel'
import { DiceHistoryPanel } from './DiceHistoryPanel'
import { DiceRoller } from './DiceRoller'
import { MapPanel } from './MapPanel'
import { CharacterSheetPanel } from './CharacterSheetPanel'
import { InventoryPanel } from './InventoryPanel'
import { wsEvent } from './wsEvents'
import { XPSuggestionsPanel } from './XPSuggestionsPanel'
import { XPLogPanel } from './XPLogPanel'
import { CharacterSelector } from './CharacterSelector'
import { setAmbientTrack } from './audio/ambient'
import { wgTalentDescription } from './wgTalentData'
import { WorkspaceShell } from './layout/WorkspaceShell'
import { WorkspaceNavigation } from './navigation/WorkspaceNavigation'
import { PANEL_DEFINITIONS, type MobileDestination, type PanelID } from './navigation/panelRegistry'
import { Dialog } from './ui/Dialog'
import { IconButton } from './ui/IconButton'
import { useToast } from './ui/ToastProvider'
import { NarrativeStream, TurnOrderStrip, normalizeGMContent } from './session/NarrativeStream'
import { PlayerComposer } from './session/PlayerComposer'
import { RightWorkspace } from './session/RightWorkspace'

const SCENE_TAGS = ['tavern', 'dungeon', 'forest', 'city', 'ocean', 'cave', 'castle', 'rain', 'night', 'battle', 'market', 'temple', 'ruins']

interface SceneTagPickerProps {
  session: Session
  onUpdate: (tags: string) => void
}

function SceneTagPicker({ session, onUpdate }: SceneTagPickerProps) {
  const toast = useToast()
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
      toast.error('Could not update scene tags.')
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
  rightTab: PanelID
  setRightTab: React.Dispatch<React.SetStateAction<PanelID>>
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
  onSendText: (text: string) => Promise<boolean>
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
  const [mobileDestination, setMobileDestination] = useState<MobileDestination>('story')
  const [sessionNpcs, setSessionNpcs] = useState<SessionNPC[]>([])
  const [pendingHandout, setPendingHandout] = useState<{
    title: string
    content: string
    category: string
  } | null>(null)

  useEffect(() => {
    if (!ctx.session) return
    fetchNPCs(ctx.session.id).then(setSessionNpcs).catch(() => setSessionNpcs([])) // Background roster load retries on session change.
  }, [ctx.session?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages, streamingText])

  useEffect(() => {
    const ev = wsEvent(lastEvent)
    if (ev?.type !== 'secret_revealed') return
    if (ev.payload.session_id !== ctx.session?.id) return
    const { title, content, category } = ev.payload
    if (typeof title !== 'string' || typeof content !== 'string' || typeof category !== 'string') return
    setPendingHandout({ title, content, category })
  }, [lastEvent, ctx.session?.id])

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
  const changeMobileDestination = (destination: MobileDestination) => {
    setMobileDestination(destination)
    const currentGroup = PANEL_DEFINITIONS.find((panel) => panel.id === rightTab)?.group
    if (destination === 'gm' && currentGroup !== 'GM') setRightTab('objectives')
    if (destination === 'world' && currentGroup === 'GM') setRightTab('notes')
  }

  return (
    <>
    {pendingHandout && (
      <Dialog open title={pendingHandout.title} onClose={() => setPendingHandout(null)} className="handout-modal">
          <div className="handout-modal-header">
            <span className="handout-modal-category">{pendingHandout.category}</span>
            <IconButton className="handout-modal-close" label="Close handout" icon="×" onClick={() => setPendingHandout(null)} />
          </div>
          <div className="handout-modal-body">
            <p className="handout-modal-content">{pendingHandout.content}</p>
          </div>
      </Dialog>
    )}
      {/* Player History Overlay */}
      {showPlayerHistory && (
        <Dialog open title="Your Actions" onClose={() => setShowPlayerHistory(false)} className="player-history-overlay">
          <div className="player-history-header">
            <span>Your Actions</span>
            <IconButton label="Close action history" icon="×" onClick={() => setShowPlayerHistory(false)} />
          </div>
          <div className="player-history-list">
            {messages.filter(m => m.role === 'user' && !m.whisper).map(m => (
              <div key={m.id} className="player-history-item">
                <p>{m.content}</p>
              </div>
            ))}
          </div>
        </Dialog>
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
            <Dialog open title={`Disciplines & Powers — ${ctx.character.name}`} onClose={() => setShowTalentsPanel(false)} className="talents-overlay">
              <div className="talents-overlay-header">
                <span>Disciplines &amp; Powers — {ctx.character.name}</span>
                <IconButton label="Close disciplines and powers" icon="×" onClick={() => setShowTalentsPanel(false)} />
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
            </Dialog>
          )
        }

        // W&G (and other systems): original talents & powers panel
        const talents = String(charData.talents ?? '').trim()
        const powers = String(charData.powers ?? '').trim()
        const talentRanks = (charData.talent_ranks ?? {}) as Record<string, number>
        return (
          <Dialog open title={`Talents & Powers — ${ctx.character.name}`} onClose={() => setShowTalentsPanel(false)} className="talents-overlay">
            <div className="talents-overlay-header">
              <span>Talents &amp; Powers — {ctx.character.name}</span>
              <IconButton label="Close talents and powers" icon="×" onClick={() => setShowTalentsPanel(false)} />
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
          </Dialog>
        )
      })()}

      <WorkspaceShell
        mobileDestination={mobileDestination}
        mobileNav={(
          <WorkspaceNavigation
            activePanel={rightTab}
            onPanelChange={setRightTab}
            mobileDestination={mobileDestination}
            onMobileDestinationChange={changeMobileDestination}
            mobile
          />
        )}
        left={(
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
        )}

        story={(
          <>
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
          <NarrativeStream
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

        <PlayerComposer
          characterId={ctx.character?.id ?? null}
          hasSession={ctx.session != null}
          input={input}
          onInputChange={setInput}
          onSend={handleSend}
          onSendText={onSendText}
          sending={sending}
          whisperMode={whisperMode}
          setWhisperMode={setWhisperMode}
        />

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
                characters={ctx.character ? [ctx.character] : []}
                sessionNpcs={sessionNpcs}
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
          </>
        )}

        right={(
          <RightWorkspace
            aiEnabled={aiEnabled}
            ctx={ctx}
            lastEvent={lastEvent}
            mobileDestination={mobileDestination}
            onMobileDestinationChange={changeMobileDestination}
            rightTab={rightTab}
            setRightTab={setRightTab}
          />
        )}
      />
    </>
  )
}
