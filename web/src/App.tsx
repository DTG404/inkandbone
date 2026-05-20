import { useState, useEffect, useCallback, useRef } from 'react'
import { useWebSocket } from './useWebSocket'
import { fetchContext, sendMessage, gmRespondStream, generateMap, fetchRuleset, suggestAdvances } from './api'
import type { GameContext, Message, XPSpendSuggestionsEvent } from './types'
import { ManagePanel } from './ManagePanel'
import { GMScreenPanel } from './GMScreenPanel'
import AudioControls, { getAudioMuted } from './AudioControls'
import { playDiceRoll, playNotification, playCombatStart } from './audio/sounds'
import { setAmbientTrack } from './audio/ambient'
import { SessionView } from './SessionView'
import { CharacterSelector } from './CharacterSelector'
import './App.css'

const WS_URL = `ws://${window.location.host}/ws`

// Minimum XP required to afford any advancement per ruleset.
// Derived from XPCostFor minimums in internal/ruleset/advancement.go.
const MIN_XP_TO_ADVANCE: Record<string, number> = {
  vtm: 3,           // skill dot 1 costs 3
  wrath_glory: 8,   // skill/attr to rating 2 costs 8
  shadowrun: 5,     // specialization costs 5
  wfrp: 10,         // flat 10 per advance
  cyberpunk_red: 10, // skill to rating 1 costs 10
  starwars: 5,      // skill to rating 1 costs 5
  l5r: 2,           // skill rank 1 costs 2
  theonering: 1,    // skill rank 1 costs 1
  blades: 8,        // action advance threshold is 8
  ironsworn: 1,     // asset upgrade costs 1
  dnd5e: 300,       // level 2 threshold
}

// ── Chronicle Night Tracker ───────────────────────────────

interface ChronicleNightTrackerProps {
  campaign: import('./types').Campaign
}

function ChronicleNightTracker({ campaign }: ChronicleNightTrackerProps) {
  const night = campaign.chronicle_night ?? 1
  const startDOW = campaign.chronicle_night_start_dow ?? -1
  const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']

  // Show the day label only once the GM has narrated Night 1 and we've detected the in-story day.
  const dayLabel = startDOW >= 0
    ? ' — ' + days[(startDOW + night - 1) % 7]
    : ''

  return (
    <div className="chronicle-night-tracker">
      <span className="chronicle-label" title={`Chronicle night ${night}`}>
        Night {night}<span className="chronicle-day">{dayLabel}</span>
      </span>
    </div>
  )
}

// ── App ────────────────────────────────────────────────────

export default function App() {
  const [ctx, setCtx] = useState<GameContext | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [error, setError] = useState<string | null>(null)
  const [aiEnabled, setAiEnabled] = useState(false)
  const [mapOpen, setMapOpen] = useState(false)
  const [rightTab, setRightTab] = useState<string>('notes')
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)
  const [gmResponding, setGmResponding] = useState(false)
  const [streamingText, setStreamingText] = useState('')
  const [generatingMap, setGeneratingMap] = useState(false)
  const [whisperMode, setWhisperMode] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [showPlayerHistory, setShowPlayerHistory] = useState(false)
  const [theme, setTheme] = useState(() => localStorage.getItem('theme') ?? 'worn-grimoire')
  const [activeMapId, setActiveMapId] = useState<number | null>(null)
  const [activeMapImagePath, setActiveMapImagePath] = useState<string | null>(null)
  const [manageOpen, setManageOpen] = useState(false)
  const [gmScreenOpen, setGmScreenOpen] = useState(false)
  const [manageTab, setManageTab] = useState<'campaigns' | 'characters' | 'sessions' | 'rulebooks' | 'automation'>('campaigns')
  const [xpSuggestionsEvent, setXPSuggestionsEvent] = useState<XPSpendSuggestionsEvent | null>(null)
  const [xpPanelDismissed, setXpPanelDismissed] = useState(false)
  const [suggestingXP, setSuggestingXP] = useState(false)
  const [showTalentsPanel, setShowTalentsPanel] = useState(false)
  const [aiTalentDescs, setAiTalentDescs] = useState<Record<string, string>>({})
  const [rulesetName, setRulesetName] = useState<string | null>(null)
  const [typingNames, setTypingNames] = useState<string[]>([])
  const typingTimeouts = useRef<Record<string, ReturnType<typeof setTimeout>>>({})
  const [selectedCharacterId, setSelectedCharacterId] = useState<number | null>(() => {
    const stored = localStorage.getItem('active_player_character_id')
    return stored ? Number(stored) : null
  })
  const [charactersList, setCharactersList] = useState<{ id: number; name: string }[]>([])
  const loadGenRef = useRef(0)

  // Derived: character's current XP (or Karma) balance, parsed from data_json.
  const charXPBalance = (() => {
    if (!ctx?.character) return 0
    try {
      const cd = JSON.parse(ctx.character.data_json || '{}')
      return Number(cd.xp ?? cd.karma ?? 0) || 0
    } catch { return 0 }
  })()

  useEffect(() => {
    if (selectedCharacterId) {
      localStorage.setItem('active_player_character_id', String(selectedCharacterId))
    } else {
      localStorage.removeItem('active_player_character_id')
    }
  }, [selectedCharacterId])

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('theme', theme)
  }, [theme])

  useEffect(() => {
    const rulesetId = ctx?.campaign?.ruleset_id
    if (rulesetId == null) {
      setRulesetName(null)
      return
    }
    fetchRuleset(rulesetId)
      .then((rs) => setRulesetName(rs.name.toLowerCase()))
      .catch(() => setRulesetName(null))
  }, [ctx?.campaign?.ruleset_id])

  useEffect(() => {
    if (rulesetName === 'vtm') {
      // VtM uses a single fixed ambient track regardless of scene tags.
      setAmbientTrack('vtm/ambient')
      return
    }
    const tags = ctx?.session?.scene_tags ?? ''
    const firstTag = tags.split(',').filter(Boolean)[0] ?? null
    setAmbientTrack(firstTag)
  }, [ctx?.session?.scene_tags, rulesetName])

  useEffect(() => {
    fetch('/api/health')
      .then((r) => r.json())
      .then((data: { ai_enabled: boolean }) => setAiEnabled(data.ai_enabled))
      .catch(() => setAiEnabled(false))
  }, [])

  useEffect(() => {
    if (!ctx?.campaign?.id) { setCharactersList([]); return }
    fetch(`/api/campaigns/${ctx.campaign.id}/characters`)
      .then(r => r.json())
      .then(chars => {
        if (Array.isArray(chars)) {
          setCharactersList(chars.map((c: { id: number; name: string }) => ({ id: c.id, name: c.name })))
        }
      })
      .catch(() => setCharactersList([]))
  }, [ctx?.campaign?.id])

  const loadContext = useCallback(() => {
    const gen = ++loadGenRef.current
    fetchContext()
      .then((data) => {
        if (loadGenRef.current !== gen) return // stale — a newer fetch already resolved
        setCtx(data)
        setMessages(data.recent_messages ?? [])
      })
      .catch(() => {
        if (loadGenRef.current !== gen) return
        setError('Could not load game state')
      })
  }, [])

  useEffect(() => {
    loadContext()
  }, [loadContext])

  const handleEvent = useCallback((data: unknown) => {
    loadContext()
    const event = data as { type?: string }
    if (!getAudioMuted()) {
      if (event?.type === 'dice_rolled') playDiceRoll()
      else if (event?.type === 'message_created') playNotification()
      else if (event?.type === 'combat_started') playCombatStart()
    }
    if (event?.type === 'xp_spend_suggestions') {
      setXPSuggestionsEvent((data as { payload: XPSpendSuggestionsEvent }).payload)
      setXpPanelDismissed(false)
    }
    if (event?.type === 'campaign_updated') {
      const p = (data as { payload?: { chronicle_night?: number; chronicle_night_start_dow?: number } }).payload
      if (p?.chronicle_night !== undefined || p?.chronicle_night_start_dow !== undefined) {
        setCtx(prev => prev && prev.campaign
          ? { ...prev, campaign: { ...prev.campaign, ...p } }
          : prev
        )
      }
    }
    if (event?.type === 'typing') {
      const p = (data as { payload?: { character_name?: string; status?: string; character_id?: number } }).payload
      if (!p?.character_name) return
      const name = p.character_name
      if (p.status === 'thinking') {
        setTypingNames(prev => prev.includes(name) ? prev : [...prev, name])
        // Auto-clear after 30s in case Nyx never sends "done"
        if (typingTimeouts.current[name]) clearTimeout(typingTimeouts.current[name])
        typingTimeouts.current[name] = setTimeout(() => {
          setTypingNames(prev => prev.filter(n => n !== name))
          delete typingTimeouts.current[name]
        }, 30000)
      } else {
        setTypingNames(prev => prev.filter(n => n !== name))
        if (typingTimeouts.current[name]) {
          clearTimeout(typingTimeouts.current[name])
          delete typingTimeouts.current[name]
        }
      }
    }
  }, [loadContext])
  const { lastEvent } = useWebSocket(WS_URL, handleEvent)

  const handleSend = useCallback(async () => {
    const text = input.trim()
    if (!text || !ctx?.session || sending) return
    const isWhisper = whisperMode
    setSending(true)
    setInput('')
    setWhisperMode(false)
    try {
      await sendMessage(ctx.session.id, text, isWhisper, selectedCharacterId)
      loadContext()
      if (!isWhisper) {
        setGmResponding(true)
        setStreamingText('')
        await gmRespondStream(ctx.session.id, (chunk) => {
          setStreamingText((prev) => prev + chunk)
        })
        setStreamingText('')
        loadContext()
      }
    } catch {
      setInput(text)
    } finally {
      setSending(false)
      setGmResponding(false)
    }
  }, [input, ctx, sending, loadContext, whisperMode, selectedCharacterId])

  const handleGenerateMap = useCallback(async () => {
    if (!ctx?.campaign || !aiEnabled || generatingMap) return
    setGeneratingMap(true)
    setMapOpen(true)
    const recentText = messages.slice(-6).map(m => `[${m.role}]: ${m.content}`).join('\n')
    const context = `Campaign: ${ctx.campaign.name}\n\n${recentText}`
    const mapName = ctx.session?.title ?? ctx.campaign.name
    try {
      await generateMap(ctx.campaign.id, mapName, context)
    } finally {
      setGeneratingMap(false)
    }
  }, [ctx, aiEnabled, generatingMap, messages])

  const handleSpendXP = useCallback(async (characterId: number, field: string, newValue: number) => {
    const res = await fetch(`/api/characters/${characterId}/advance`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ field, new_value: newValue }),
    })
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || 'Advance failed')
    }
    loadContext()
  }, [loadContext])

  function handleExport() {
    if (!ctx) return
    const charNameMap: Record<number, string> = {}
    for (const c of charactersList) {
      charNameMap[c.id] = c.name
    }
    const sessionDate = ctx.session?.date
      ? new Date(ctx.session.date).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })
      : ''
    const lines: string[] = []
    lines.push(`# ${ctx.session?.title ?? 'Session'}`)
    if (sessionDate) lines.push(`*${sessionDate}*`)
    lines.push('')
    messages.forEach(m => {
      if (m.whisper) return
      if (m.role === 'assistant') {
        lines.push(m.content)
      } else {
        const name = m.character_id != null && charNameMap[m.character_id]
          ? charNameMap[m.character_id]
          : (ctx?.character?.name ?? 'Player')
        lines.push(`> **${name}:** ${m.content}`)
      }
      lines.push('')
    })
    const blob = new Blob([lines.join('\n')], { type: 'text/markdown' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${(ctx.session?.title ?? 'session').replace(/\s+/g, '-').toLowerCase()}.md`
    a.click()
    URL.revokeObjectURL(url)
  }

  if (error) return <div className="error">{error}</div>
  if (!ctx) return <div className="loading">Loading…</div>

  const displayMessages = searchQuery
    ? messages.filter(m => m.content.toLowerCase().includes(searchQuery.toLowerCase()))
    : messages

  return (
    <div className="grimoire">
      <header className="grimoire-header">
        <span className="h-campaign">{ctx.campaign?.name ?? 'No campaign'}</span>
        <span className="h-sep">›</span>
        <span className="h-char">{ctx.character?.name ?? 'No character'}</span>
        <span className="h-sep">›</span>
        <span className="h-session">{ctx.session?.title ?? 'No session'}</span>
        <CharacterSelector
          characters={charactersList}
          selectedId={selectedCharacterId}
          onSelect={setSelectedCharacterId}
        />
        <button
          className="h-theme"
          onClick={() => setTheme(t => t === 'worn-grimoire' ? 'parchment' : 'worn-grimoire')}
          title="Toggle theme"
        >
          {theme === 'worn-grimoire' ? '☀' : '🌙'}
        </button>
        <button
          className={`h-actions-btn${showPlayerHistory ? ' active' : ''}`}
          onClick={() => setShowPlayerHistory((v) => !v)}
          title="Your actions"
        >
          ⚔ Actions
        </button>
        {ctx.character && (
          <button
            className={`h-actions-btn${showTalentsPanel ? ' active' : ''}`}
            onClick={() => setShowTalentsPanel((v) => !v)}
            title={rulesetName === 'vtm' ? 'Disciplines, Merits & Flaws' : 'Character talents & psychic powers'}
          >
            {rulesetName === 'vtm' ? '✦ Disciplines' : '✦ Talents'}
          </button>
        )}
        <button className="h-export" onClick={handleExport} title="Export session">
          ↓ Export
        </button>
        <button
          className="h-manage"
          onClick={() => setGmScreenOpen(true)}
          title="GM Screen — campaign config, notes, and tools"
        >
          🎭 GM Screen
        </button>
        <button
          className="h-manage"
          onClick={() => setManageOpen(true)}
          title="Manage campaigns, characters, sessions"
        >
          ⚙ Manage
        </button>
        {ctx?.character && aiEnabled && charXPBalance >= (MIN_XP_TO_ADVANCE[rulesetName ?? ''] ?? 1) && (
          <button
            className={`xp-available-badge${suggestingXP ? ' xp-loading' : ''}`}
            disabled={suggestingXP}
            onClick={async () => {
              if (xpSuggestionsEvent && xpPanelDismissed) {
                setXpPanelDismissed(false)
                return
              }
              setSuggestingXP(true)
              try {
                await suggestAdvances(ctx.character!.id, charXPBalance)
                setXpPanelDismissed(false)
              } catch {
                // silently ignore — panel will appear when WS event arrives
              } finally {
                setSuggestingXP(false)
              }
            }}
            title={xpSuggestionsEvent && xpPanelDismissed
              ? `Advancement available — ${xpSuggestionsEvent.current_xp} ${xpSuggestionsEvent.xp_label}`
              : 'Request advancement suggestions'}
          >
            {suggestingXP ? '...' : '⬆ Advance'}
          </button>
        )}
        {ctx?.campaign && rulesetName === 'vtm' && (
          <ChronicleNightTracker campaign={ctx.campaign} />
        )}
        <AudioControls />
      </header>

      {gmScreenOpen && (
        <GMScreenPanel
          campaignId={ctx?.campaign?.id ?? null}
          sessionId={ctx?.session?.id ?? null}
          aiEnabled={aiEnabled}
          onClose={() => setGmScreenOpen(false)}
        />
      )}
      {manageOpen && (
        <ManagePanel
          activeCampaignId={ctx?.campaign?.id ?? null}
          activeCharacterId={ctx?.character?.id ?? null}
          activeSessionId={ctx?.session?.id ?? null}
          initialTab={manageTab}
          onTabChange={setManageTab}
          onClose={() => setManageOpen(false)}
          onContextChanged={() => { loadContext(); setManageOpen(false); setXPSuggestionsEvent(null) }}
          onCampaignActivated={() => { setMessages([]); loadContext(); setManageOpen(false); setXPSuggestionsEvent(null) }}
        />
      )}

      <SessionView
        ctx={ctx}
        messages={messages}
        displayMessages={displayMessages}
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        aiEnabled={aiEnabled}
        gmResponding={gmResponding}
        streamingText={streamingText}
        generatingMap={generatingMap}
        mapOpen={mapOpen}
        setMapOpen={setMapOpen}
        activeMapId={activeMapId}
        activeMapImagePath={activeMapImagePath}
        setActiveMapId={setActiveMapId}
        setActiveMapImagePath={setActiveMapImagePath}
        rightTab={rightTab}
        setRightTab={setRightTab}
        showPlayerHistory={showPlayerHistory}
        setShowPlayerHistory={setShowPlayerHistory}
        showTalentsPanel={showTalentsPanel}
        setShowTalentsPanel={setShowTalentsPanel}
        input={input}
        setInput={setInput}
        sending={sending}
        whisperMode={whisperMode}
        setWhisperMode={setWhisperMode}
        rulesetName={rulesetName}
        xpSuggestionsEvent={xpSuggestionsEvent}
        xpPanelDismissed={xpPanelDismissed}
        setXpPanelDismissed={setXpPanelDismissed}
        setXPSuggestionsEvent={setXPSuggestionsEvent}
        aiTalentDescs={aiTalentDescs}
        setAiTalentDescs={setAiTalentDescs}
        handleSend={handleSend}
        handleGenerateMap={handleGenerateMap}
        handleSpendXP={handleSpendXP}
        lastEvent={lastEvent}
        setCtx={setCtx}
        typingNames={typingNames}
        charactersList={charactersList}
        selectedCharacterId={selectedCharacterId}
        onCharacterSelect={setSelectedCharacterId}
      />
    </div>
  )
}
