import { acknowledgeSubmission, updateDraft, useSessionDraft, type Submission } from './session/sessionDraft'
import { SubmissionError } from './api/sessions'
import { useState, useEffect, useCallback, useRef } from 'react'
import { useWebSocket, webSocketURL } from './useWebSocket'
import { fetchContext, fetchMessages, sendMessage, gmRespondStream, generateMap, fetchRuleset, fetchAdvancementConfig, suggestAdvances, type AdvancementConfig } from './api'
import type { GameContext, Message, XPSuggestion, XPSpendSuggestionsEvent } from './types'
import type { RealtimeEvent } from './realtime.gen'
import { ManagePanel } from './ManagePanel'
import { GMScreenPanel } from './GMScreenPanel'
import AudioControls, { getAudioMuted } from './AudioControls'
import { playDiceRoll, playNotification, playCombatStart } from './audio/sounds'
import { setAmbientTrack } from './audio/ambient'
import { joinScore, leaveScore, revealScoredTurn, scoringTurn } from './audio/liveScore'
import { SessionView } from './SessionView'
import { useMobileWorkspace } from './layout/useMobileWorkspace'
import { CharacterSelector } from './CharacterSelector'
import { LoginScreen } from './LoginScreen'
import type { PanelID } from './navigation/panelRegistry'
import { ToastProvider, useToast } from './ui/ToastProvider'
import { fetchSessionInfo, request, setCSRFToken, type SessionInfo } from './transport'
import '@fontsource-variable/cormorant-garamond/index.css'
import '@fontsource-variable/noto-sans/index.css'
import '@fontsource-variable/noto-sans-mono/index.css'
import './App.css'

const appOwnedContextEvents = new Set([
  'campaign_updated', 'campaign_config_updated', 'context_updated',
  'session_started', 'session_ended', 'session_deleted', 'session_updated',
  'campaign_created', 'campaign_closed', 'campaign_deleted', 'campaign_reopened',
  'character_created', 'character_updated',
])

const locallyOwnedEvents = new Set([
  'message_created', 'typing', 'dice_rolled',
  'world_note_created', 'world_note_updated', 'world_note_revealed', 'map_pin_added', 'map_created',
  'npc_updated', 'objective_updated', 'item_updated', 'xp_added', 'xp_spend_suggestions',
  'relationship_updated', 'faction_updated', 'adventure_updated', 'npc_stat_updated',
  'secrets_updated', 'secret_revealed', 'calendar_updated', 'card_drawn',
  'token_placed', 'token_moved', 'token_removed', 'zone_revealed', 'map_fx',
  'resync_required',
])

function xpSuggestion(value: unknown): XPSuggestion | null {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return null
  const field = Reflect.get(value, 'field')
  const displayName = Reflect.get(value, 'display_name')
  const currentValue = Reflect.get(value, 'current_value')
  const newValue = Reflect.get(value, 'new_value')
  const xpCost = Reflect.get(value, 'xp_cost')
  const reasoning = Reflect.get(value, 'reasoning')
  if (typeof field !== 'string' || typeof displayName !== 'string' || typeof currentValue !== 'number' ||
      typeof newValue !== 'number' || typeof xpCost !== 'number' || typeof reasoning !== 'string') return null
  return { field, display_name: displayName, current_value: currentValue, new_value: newValue, xp_cost: xpCost, reasoning }
}

function parseXPSuggestionsEvent(event: Extract<RealtimeEvent, { type: 'xp_spend_suggestions' }>): XPSpendSuggestionsEvent | null {
  const { payload } = event
  if (typeof payload.character_id !== 'number' || typeof payload.character_name !== 'string' ||
      typeof payload.current_xp !== 'number' || typeof payload.xp_label !== 'string' || !Array.isArray(payload.suggestions)) return null
  const suggestions = payload.suggestions.map(xpSuggestion)
  if (suggestions.some((suggestion) => suggestion === null)) return null
  return {
    character_id: payload.character_id,
    character_name: payload.character_name,
    current_xp: payload.current_xp,
    xp_label: payload.xp_label,
    suggestions: suggestions.filter((suggestion): suggestion is XPSuggestion => suggestion !== null),
  }
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
  const [session, setSession] = useState<SessionInfo | null>(null)

  useEffect(() => {
    let active = true
    fetchSessionInfo()
      .then((info) => {
        if (!active) return
        setCSRFToken(info.csrf_token ?? null)
        setSession(info)
      })
      .catch(() => {
        if (!active) return
        setCSRFToken(null)
        setSession({ authenticated: false })
      })
    return () => { active = false }
  }, [])

  if (session === null) {
    return <main className="login-screen" aria-label="Checking session" />
  }
  if (!session.authenticated) {
    return <LoginScreen onAuthenticated={setSession} />
  }
  return <ToastProvider><GameApp /></ToastProvider>
}

function FirstRunEmptyState({ onOpenManage, aiEnabled }: { onOpenManage: () => void; aiEnabled: boolean }) {
  return (
    <main className="first-run-empty-state">
      <h1>Begin your chronicle</h1>
      <ol>
        <li>Create a campaign.</li>
        <li>Create or select a character.</li>
        <li>Create or select a session.</li>
      </ol>
      <p>{aiEnabled ? 'An AI backend is available for configured game features.' : 'No AI backend is currently available; non-AI campaign tools still work.'}</p>
      <button type="button" onClick={onOpenManage}>Open campaign management</button>
    </main>
  )
}

function GameApp() {
  const toast = useToast()
  const [ctx, setCtx] = useState<GameContext | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [error, setError] = useState<string | null>(null)
  const [aiEnabled, setAiEnabled] = useState(false)
  const [mapEnabled, setMapEnabled] = useState(false)
  const [mapOpen, setMapOpen] = useState(false)
  const [rightTab, setRightTab] = useState<PanelID>('notes')
  const sessionID = ctx?.session?.id ?? null
  const draft = useSessionDraft(sessionID)
  const input = draft.text
  const whisperMode = draft.whisper
  const setInput = (value: string) => { if (sessionID !== null) updateDraft(sessionID, d => ({...d, text: value})) }
  const setWhisperMode = (value: boolean | ((previous: boolean) => boolean)) => {
    if (sessionID !== null) updateDraft(sessionID, d => ({...d, whisper: typeof value === 'function' ? value(d.whisper) : value}))
  }
  const submitting = useRef(false)
  const [streamSession, setStreamSession] = useState<number | null>(null)
  const [sending, setSending] = useState(false)
  const [gmResponding, setGmResponding] = useState(false)
  const [streamingText, setStreamingText] = useState('')
  const [generatingMap, setGeneratingMap] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [showPlayerHistory, setShowPlayerHistory] = useState(false)
  const [theme, setTheme] = useState(() => localStorage.getItem('theme') ?? 'worn-grimoire')
  const [activeMapId, setActiveMapId] = useState<number | null>(null)
  const [activeMapImagePath, setActiveMapImagePath] = useState<string | null>(null)
  const generateMapRetryRef = useRef<() => void>(() => undefined)
  const phone = useMobileWorkspace()
  const [manageOpen, setManageOpen] = useState(false)
  const [gmScreenOpen, setGmScreenOpen] = useState(false)
  const [manageTab, setManageTab] = useState<'campaigns' | 'characters' | 'sessions' | 'rulebooks' | 'automation'>('campaigns')
  const [xpSuggestionsEvent, setXPSuggestionsEvent] = useState<XPSpendSuggestionsEvent | null>(null)
  const [xpPanelDismissed, setXpPanelDismissed] = useState(false)
  const [suggestingXP, setSuggestingXP] = useState(false)
  const [showTalentsPanel, setShowTalentsPanel] = useState(false)
  const [aiTalentDescs, setAiTalentDescs] = useState<Record<string, string>>({})
  const [rulesetName, setRulesetName] = useState<string | null>(null)
  const [advancementConfig, setAdvancementConfig] = useState<AdvancementConfig | null>(null)
  const [typingNames, setTypingNames] = useState<string[]>([])
  const typingTimeouts = useRef<Record<string, ReturnType<typeof setTimeout>>>({})
  const [selectedCharacterId, setSelectedCharacterId] = useState<number | null>(() => {
    const stored = localStorage.getItem('active_player_character_id')
    return stored ? Number(stored) : null
  })
  const [charactersList, setCharactersList] = useState<{ id: number; name: string }[]>([])
  const contextGenRef = useRef(0)
  const transcriptGenRef = useRef(0)
  const transcriptRefreshPendingRef = useRef(false)
  const activeSessionRef = useRef<number | null>(null)

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
      setAdvancementConfig(null)
      return
    }
    setAdvancementConfig(null)
    fetchRuleset(rulesetId)
      .then((rs) => setRulesetName(rs.name.toLowerCase()))
      .catch(() => setRulesetName(null))
    fetchAdvancementConfig(rulesetId)
      .then(setAdvancementConfig)
      .catch(() => setAdvancementConfig(null))
  }, [ctx?.campaign?.ruleset_id])

  const scoreSession = ctx?.live_score ? ctx.session?.id : undefined
  const [scoredTurn, setScoredTurn] = useState<{session: number; after: number} | null>(null)
  useEffect(() => {
    if (!scoreSession) return
    void joinScore(scoreSession).catch(() => console.warn('Could not join live score'))
    return leaveScore
  }, [scoreSession])

  useEffect(() => {
    if (scoreSession || !ctx?.session) { setAmbientTrack(null); return }
    if (rulesetName === 'vtm') {
      // VtM uses a single fixed ambient track regardless of scene tags.
      setAmbientTrack('vtm/ambient')
      return
    }
    const tags = ctx?.session?.scene_tags ?? ''
    const firstTag = tags.split(',').filter(Boolean)[0] ?? null
    setAmbientTrack(firstTag)
  }, [ctx?.session, rulesetName, scoreSession])

  useEffect(() => {
    fetch('/api/health')
      .then((r) => r.json())
      .then((data: { ai_enabled: boolean; map_generation_enabled?: boolean }) => {
        setAiEnabled(data.ai_enabled)
        setMapEnabled(data.map_generation_enabled ?? data.ai_enabled)
      })
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

  const loadContext = useCallback((refreshTranscript = false) => {
    if (refreshTranscript) transcriptRefreshPendingRef.current = true
    const contextGen = ++contextGenRef.current
    return (async (): Promise<boolean> => {
      let data: GameContext
      try {
        data = await fetchContext()
      } catch {
        if (contextGenRef.current !== contextGen) return false
        setError('Could not load game state')
        return false
      }
      if (contextGenRef.current !== contextGen) return false // stale — a newer context fetch already resolved
      setCtx(data)
      setError(null)
      const sessionId = data.session?.id ?? null
      if (sessionId === null) {
        activeSessionRef.current = null
        transcriptRefreshPendingRef.current = false
        transcriptGenRef.current++
        setMessages([])
        return true
      }
      const sessionChanged = activeSessionRef.current !== sessionId
      if (sessionChanged) {
        activeSessionRef.current = sessionId
        transcriptGenRef.current++
        setMessages([])
      }
      if (!sessionChanged && !transcriptRefreshPendingRef.current) return true
      transcriptRefreshPendingRef.current = false
      const transcriptGen = ++transcriptGenRef.current
      try {
        const sessionMessages = await fetchMessages(sessionId)
        if (transcriptGenRef.current !== transcriptGen || activeSessionRef.current !== sessionId) return false
        setMessages(sessionMessages)
      } catch (err) {
        if (transcriptGenRef.current !== transcriptGen || activeSessionRef.current !== sessionId) return false
        console.error(err)
        return false
      }
      return true
    })()
  }, [])

  const refreshTranscript = useCallback(async () => {
    const sessionId = activeSessionRef.current
    if (sessionId === null) return
    const transcriptGen = ++transcriptGenRef.current
    try {
      const sessionMessages = await fetchMessages(sessionId)
      if (transcriptGenRef.current !== transcriptGen || activeSessionRef.current !== sessionId) return
      setMessages(sessionMessages)
    } catch (err) {
      if (transcriptGenRef.current === transcriptGen && activeSessionRef.current === sessionId) console.error(err)
    }
  }, [])

  useEffect(() => {
    loadContext(true)
  }, [loadContext])

  const handleEvent = useCallback((event: RealtimeEvent) => {
    if (event.type === 'message_created') {
      void refreshTranscript()
    } else if (event.type && appOwnedContextEvents.has(event.type)) {
      void loadContext(false)
    } else if (event.type && !locallyOwnedEvents.has(event.type)) {
      void loadContext(false)
    }
    if (!getAudioMuted()) {
      if (event?.type === 'dice_rolled') playDiceRoll()
      else if (event?.type === 'message_created') playNotification()
      else if (event?.type === 'combat_started') playCombatStart()
    }
    if (event.type === 'xp_spend_suggestions') {
      const parsed = parseXPSuggestionsEvent(event)
      if (parsed) {
        setXPSuggestionsEvent(parsed)
        setXpPanelDismissed(false)
      }
    }
    if (event.type === 'campaign_updated') {
      const p = event.payload
      if (p?.chronicle_night !== undefined || p?.chronicle_night_start_dow !== undefined) {
        setCtx(prev => prev && prev.campaign
          ? { ...prev, campaign: { ...prev.campaign, ...p } }
          : prev
        )
      }
    }
    if (event.type === 'typing') {
      const p = event.payload
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
  }, [loadContext, refreshTranscript])
  const { lastEvent, status: webSocketStatus, needsReconcile, reconcileGeneration, acknowledgeReconcile } = useWebSocket(webSocketURL(window.location), handleEvent)

  useEffect(() => {
    if (!needsReconcile || webSocketStatus !== 'open') return
    const generation = reconcileGeneration
    let cancelled = false
    let retryTimer: ReturnType<typeof setTimeout> | null = null
    let retryDelay = 250
    const reconcile = async () => {
      const loaded = await loadContext(true)
      if (cancelled) return
      if (loaded) {
        acknowledgeReconcile(generation)
        return
      }
      retryTimer = setTimeout(() => { void reconcile() }, retryDelay)
      retryDelay = Math.min(retryDelay * 2, 30000)
    }
    void reconcile()
    return () => {
      cancelled = true
      if (retryTimer !== null) clearTimeout(retryTimer)
    }
  }, [needsReconcile, reconcileGeneration, webSocketStatus, loadContext, acknowledgeReconcile])

  const waitingAction = aiEnabled && messages.findLast(message => message.session_id === sessionID && message.pending_gm)
  const pendingKey = draft.submission?.key
  useEffect(() => {
    if (sessionID === null || !pendingKey) return
    const saved = messages.find(message => message.session_id === sessionID && message.client_action_key === pendingKey)
    if (saved) {
      acknowledgeSubmission(sessionID, pendingKey)
    }
  }, [sessionID, pendingKey, messages])
  const submitTurn = async (item?: Submission, resumeID?: number): Promise<boolean> => {
    if (sessionID === null || submitting.current) return false
    const sid = sessionID
    submitting.current = true
    setSending(true)
    setStreamSession(sid)
    let accepted = resumeID !== undefined
    try {
      let actionID = resumeID
      if (item) {
        updateDraft(sid, d => ({...d, submission: item}))
        actionID = await sendMessage(sid, item.text, item.whisper, item.character, item.key)
        if (!Number.isSafeInteger(actionID) || actionID <= 0) throw new Error('Missing action acknowledgement')
        accepted = true
        acknowledgeSubmission(sid, item.key)
        void refreshTranscript()
        if (item.whisper || !aiEnabled) return true
      }
      setGmResponding(true)
      setStreamingText('')
      if (scoreSession) {
        scoringTurn(scoreSession)
        setScoredTurn({session: scoreSession, after: actionID ?? 0})
      }
      const narration = await gmRespondStream(sid, chunk => {
        if (!scoreSession && activeSessionRef.current === sid) setStreamingText(prev => prev + chunk)
      }, !!scoreSession, actionID)
      if (scoreSession && activeSessionRef.current === sid) {
        await revealScoredTurn(scoreSession, narration, setStreamingText)
      }
      return true
    } catch (err) {
      console.error(err)
      if (!accepted && item && err instanceof SubmissionError && err.status >= 400 && err.status < 500) {
        updateDraft(sid, d => d.submission?.key === item.key ? {...d, submission: undefined} : d)
      }
      toast.error(accepted ? 'GM response interrupted. Resume the saved action from the journal.' : 'Action not confirmed. Your draft is saved.')
      return accepted
    } finally {
      submitting.current = false
      scoringTurn(null)
      setScoredTurn(null)
      setSending(false)
      setGmResponding(false)
      setStreamingText('')
      void loadContext(true)
    }
  }
  const newSubmission = (text: string, fromComposer: boolean): Submission => ({
    key: Array.from(crypto.getRandomValues(new Uint8Array(16)), n => n.toString(16).padStart(2, '0')).join(''), text, whisper: fromComposer && whisperMode, character: selectedCharacterId, fromComposer,
  })
  const handleSendText = async (text: string) => {
    if (!text.trim() || draft.submission || waitingAction) return false
    return submitTurn(newSubmission(text.trim(), false))
  }
  const handleSend = async () => {
    if (input.trim() && !draft.submission && (!waitingAction || whisperMode)) await submitTurn(newSubmission(input.trim(), true))
  }
  const recoverTurn = draft.submission
    ? () => { void submitTurn(draft.submission) }
    : waitingAction ? () => { void submitTurn(undefined, waitingAction.id) } : undefined

  const handleGenerateMap = useCallback(async () => {
    if (!ctx?.campaign || !mapEnabled || generatingMap) return
    setGeneratingMap(true)
    setMapOpen(true)
    const recentText = messages.filter(m => !m.whisper).slice(-6).map(m => `[${m.role}]: ${m.content}`).join('\n')
    const context = `Campaign: ${ctx.campaign.name.slice(0, 200)}\n\n${recentText.slice(-8000)}`
    const mapName = ctx.session?.title ?? ctx.campaign.name
    try {
      await generateMap(ctx.campaign.id, mapName, context)
      toast.success('Map ready.')
    } catch (err) {
      console.error(err)
      toast.error('The map could not be generated.', { label: 'Retry', onClick: () => generateMapRetryRef.current() })
    } finally {
      setGeneratingMap(false)
    }
  }, [ctx, mapEnabled, generatingMap, messages, toast])

  useEffect(() => {
    generateMapRetryRef.current = () => { void handleGenerateMap() }
  }, [handleGenerateMap])

  const handleSpendXP = useCallback(async (characterId: number, field: string, newValue: number) => {
    const res = await request(`/api/characters/${characterId}/advance`, {
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

  const visibleMessages = scoredTurn?.session === ctx.session?.id && scoredTurn
    ? messages.filter(m => m.role !== 'assistant' || m.id <= scoredTurn.after) : messages
  const displayMessages = searchQuery
    ? visibleMessages.filter(m => m.content.toLowerCase().includes(searchQuery.toLowerCase()))
    : visibleMessages

  const dismissMenu = phone ? { popoverTarget: 'table-menu', popoverTargetAction: 'hide' as const } : {}
  const toolbar = (
    <>
      <CharacterSelector
        characters={charactersList}
        selectedId={selectedCharacterId}
        onSelect={setSelectedCharacterId}
      />
      <button
        className="h-theme"
        onClick={() => setTheme(t => t === 'worn-grimoire' ? 'parchment' : 'worn-grimoire')}
        aria-label="Toggle theme"
        title="Toggle theme"
      >
        {theme === 'worn-grimoire' ? '☀' : '🌙'}{phone && ' Theme'}
      </button>
      <button
        {...dismissMenu}
        className={`h-actions-btn${showPlayerHistory ? ' active' : ''}`}
        onClick={() => setShowPlayerHistory((v) => !v)}
        aria-label="Your actions"
        title="Your actions"
      >
        ⚔ Actions
      </button>
      {ctx.character && (
        <button
          {...dismissMenu}
          className={`h-actions-btn${showTalentsPanel ? ' active' : ''}`}
          onClick={() => setShowTalentsPanel((v) => !v)}
          aria-label={rulesetName === 'vtm' ? 'Disciplines, Merits & Flaws' : 'Character talents & psychic powers'}
          title={rulesetName === 'vtm' ? 'Disciplines, Merits & Flaws' : 'Character talents & psychic powers'}
        >
          {rulesetName === 'vtm' ? '✦ Disciplines' : '✦ Talents'}
        </button>
      )}
      <button {...dismissMenu} className="h-export" onClick={handleExport} aria-label="Export session" title="Export session">
        ↓ Export
      </button>
      <button
        {...dismissMenu}
        className="h-manage"
        onClick={() => setGmScreenOpen(true)}
        aria-label="GM Screen"
        title="GM Screen — campaign config, notes, and tools"
      >
        {phone ? 'GM Screen' : '🎭 GM Screen'}
      </button>
      <button
        {...dismissMenu}
        className="h-manage"
        onClick={() => setManageOpen(true)}
        aria-label="Manage"
        title="Manage campaigns, characters, sessions"
      >
        ⚙ Manage
      </button>
      {ctx?.character && aiEnabled && advancementConfig?.supported === true && charXPBalance >= advancementConfig.minimum_xp && (
        <button
          {...dismissMenu}
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
            } catch (cause) {
              console.error(cause)
              toast.error('Advancement suggestions could not be requested. Try again.')
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
      <AudioControls showLabels={phone} />
    </>
  )

  return (
    <div className="grimoire">
      <header className="grimoire-header">
        {phone ? (
          <div className="phone-context">
            <strong>{ctx.campaign?.name ?? 'Ink & Bone'}</strong>
            <span>{[ctx.character?.name, ctx.session?.title].filter(Boolean).join(' · ')}</span>
          </div>
        ) : (
          <>
            <span className="h-campaign">{ctx.campaign?.name ?? 'No campaign'}</span>
            <span className="h-sep">›</span>
            <span className="h-char">{ctx.character?.name ?? 'No character'}</span>
            <span className="h-sep">›</span>
            <span className="h-session">{ctx.session?.title ?? 'No session'}</span>
          </>
        )}
        {phone ? (
          <>
            <button type="button" className="phone-menu-trigger" popoverTarget="table-menu">Menu</button>
            <section id="table-menu" className="phone-menu" aria-label="Table menu" popover="auto">
              <div className="phone-menu-title">
                <strong>Table menu</strong>
                <button type="button" popoverTarget="table-menu" popoverTargetAction="hide">Close</button>
              </div>
              {toolbar}
            </section>
          </>
        ) : toolbar}
      </header>

      {webSocketStatus !== 'open' && (
        <div className={`connection-banner connection-${webSocketStatus}`} role="status" aria-label="Connection status" aria-live="polite">
          {webSocketStatus === 'offline'
            ? 'Offline. Changes from other clients will refresh when the connection returns.'
            : webSocketStatus === 'reconnecting'
              ? 'Reconnecting to live updates…'
              : 'Connecting to live updates…'}
        </div>
      )}

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
          onContextChanged={() => { loadContext(true); setManageOpen(false); setXPSuggestionsEvent(null) }}
          onCampaignActivated={() => { setMessages([]); loadContext(true); setManageOpen(false); setXPSuggestionsEvent(null) }}
        />
      )}

      {!ctx.campaign ? (
        <FirstRunEmptyState
          aiEnabled={aiEnabled}
          onOpenManage={() => { setManageTab('campaigns'); setManageOpen(true) }}
        />
      ) : <SessionView
        ctx={ctx}
        messages={messages}
        displayMessages={displayMessages}
        searchQuery={searchQuery}
        setSearchQuery={setSearchQuery}
        aiEnabled={aiEnabled}
        mapEnabled={mapEnabled}
        gmResponding={gmResponding && streamSession === sessionID}
        streamingText={streamSession === sessionID ? streamingText : ''}
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
        recoverTurn={recoverTurn}
        retrySubmission={!!draft.submission}
        handleSend={handleSend}
        onSendText={handleSendText}
        handleGenerateMap={handleGenerateMap}
        handleSpendXP={handleSpendXP}
        lastEvent={lastEvent}
        setCtx={setCtx}
        typingNames={typingNames}
        charactersList={charactersList}
        selectedCharacterId={selectedCharacterId}
        onCharacterSelect={setSelectedCharacterId}
      />}
    </div>
  )
}
