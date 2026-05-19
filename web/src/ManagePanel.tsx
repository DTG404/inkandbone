import { useState, useEffect, useCallback } from 'react'
import type { Campaign, Character, Session } from './types'
import type { Ruleset, RulebookSource } from './api'
import {
  fetchRulesets,
  fetchCampaigns,
  fetchCharacters,
  fetchSessions,
  createCampaign,
  deleteCampaign,
  createCharacter,
  deleteCharacter,
  createSession,
  deleteSession,
  patchSettings,
  fetchRulebookSources,
  uploadRulebook,
  fetchCharacterOptions,
} from './api'
import { AutomationSettingsPanel } from './AutomationSettingsPanel'

type Tab = 'campaigns' | 'characters' | 'sessions' | 'rulebooks' | 'automation'

interface ManagePanelProps {
  activeCampaignId: number | null
  activeCharacterId: number | null
  activeSessionId: number | null
  initialTab?: Tab
  onTabChange?: (tab: Tab) => void
  onClose: () => void
  onContextChanged: () => void
  onCampaignActivated?: () => void
}

export function ManagePanel({
  activeCampaignId,
  activeCharacterId,
  activeSessionId,
  initialTab = 'campaigns',
  onTabChange,
  onClose,
  onContextChanged,
  onCampaignActivated,
}: ManagePanelProps) {
  const [tab, setTab] = useState<Tab>(initialTab)

  // Shared data
  const [rulesets, setRulesets] = useState<Ruleset[]>([])
  const [campaigns, setCampaigns] = useState<Campaign[]>([])
  const [selectedCampaignId, setSelectedCampaignId] = useState<number | null>(activeCampaignId)

  // Campaign form
  const [newCampaignName, setNewCampaignName] = useState('')
  const [newCampaignDesc, setNewCampaignDesc] = useState('')
  const [newCampaignRuleset, setNewCampaignRuleset] = useState(0)
  const [creatingCampaign, setCreatingCampaign] = useState(false)

  // Characters
  const [characters, setCharacters] = useState<Character[]>([])
  const [newCharName, setNewCharName] = useState('')
  const [creatingChar, setCreatingChar] = useState(false)
  const [characterOptions, setCharacterOptions] = useState<Record<string, string[]>>({})
  const [charOverrides, setCharOverrides] = useState<Record<string, string>>({})

  // Sessions
  const [sessions, setSessions] = useState<Session[]>([])
  const [newSessionTitle, setNewSessionTitle] = useState('')
  const [newSessionDate, setNewSessionDate] = useState(() => new Date().toISOString().slice(0, 10))
  const [creatingSession, setCreatingSession] = useState(false)

  // Rulebooks
  const [rulebookSources, setRulebookSources] = useState<RulebookSource[]>([])
  const [rulebookFile, setRulebookFile] = useState<File | null>(null)
  const [rulebookSourceLabel, setRulebookSourceLabel] = useState('')
  const [uploadingRulebook, setUploadingRulebook] = useState(false)
  const [rulebookMsg, setRulebookMsg] = useState('')

  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  // Load initial data
  useEffect(() => {
    fetchRulesets().then(setRulesets).catch(console.error)
    fetchCampaigns().then(setCampaigns).catch(console.error)
  }, [])

  useEffect(() => {
    if (rulesets.length > 0 && newCampaignRuleset === 0) {
      setNewCampaignRuleset(rulesets[0].id)
    }
  }, [rulesets, newCampaignRuleset])

  const loadCharacters = useCallback((campaignId: number) => {
    fetchCharacters(campaignId).then(setCharacters).catch(console.error)
  }, [])

  const loadSessions = useCallback((campaignId: number) => {
    fetchSessions(campaignId).then(setSessions).catch(console.error)
  }, [])

  const loadRulebookSources = useCallback((campaignId: number) => {
    const campaign = campaigns.find(c => c.id === campaignId)
    if (!campaign) return
    fetchRulebookSources(campaign.ruleset_id).then(setRulebookSources).catch(console.error)
  }, [campaigns])

  useEffect(() => {
    if (selectedCampaignId) {
      loadCharacters(selectedCampaignId)
      loadSessions(selectedCampaignId)
      if (tab === 'rulebooks') loadRulebookSources(selectedCampaignId)
      // Fetch chooseable field options for this campaign's ruleset.
      const campaign = campaigns.find(c => c.id === selectedCampaignId)
      if (campaign) {
        fetchCharacterOptions(campaign.ruleset_id)
          .then(opts => { setCharacterOptions(opts); setCharOverrides({}) })
          .catch(console.error)
      }
    }
  }, [selectedCampaignId, tab, loadCharacters, loadSessions, loadRulebookSources, campaigns])

  // --- Campaign actions ---

  async function handleCreateCampaign() {
    if (!newCampaignName.trim() || !newCampaignRuleset) return
    setCreatingCampaign(true)
    setError('')
    try {
      await createCampaign(newCampaignName.trim(), newCampaignDesc.trim(), newCampaignRuleset)
      setNewCampaignName('')
      setNewCampaignDesc('')
      const updated = await fetchCampaigns()
      setCampaigns(updated)
    } catch (e) {
      setError(String(e))
    } finally {
      setCreatingCampaign(false)
    }
  }

  async function handleDeleteCampaign(id: number) {
    if (!confirm('Delete this campaign and all its data? This cannot be undone.')) return
    setBusy(true)
    setError('')
    try {
      await deleteCampaign(id)
      const updated = await fetchCampaigns()
      setCampaigns(updated)
      if (selectedCampaignId === id) setSelectedCampaignId(null)
      if (activeCampaignId === id) onContextChanged()
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  async function handleSetActiveCampaign(id: number) {
    setBusy(true)
    setError('')
    try {
      // Clear character and session so switching campaigns starts fresh.
      await patchSettings({ campaign_id: id, character_id: 0, session_id: 0 })
      ;(onCampaignActivated ?? onContextChanged)()
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  // --- Character actions ---

  async function handleCreateCharacter() {
    if (!newCharName.trim() || !selectedCampaignId) return
    setCreatingChar(true)
    setError('')
    try {
      await createCharacter(selectedCampaignId, newCharName.trim(), charOverrides)
      setNewCharName('')
      setCharOverrides({})
      loadCharacters(selectedCampaignId)
    } catch (e) {
      setError(String(e))
    } finally {
      setCreatingChar(false)
    }
  }

  async function handleDeleteCharacter(id: number) {
    if (!confirm('Delete this character and all their items?')) return
    setBusy(true)
    setError('')
    try {
      await deleteCharacter(id)
      if (selectedCampaignId) loadCharacters(selectedCampaignId)
      if (activeCharacterId === id) onContextChanged()
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  async function handleSetActiveCharacter(id: number) {
    setBusy(true)
    setError('')
    try {
      // Also activate the parent campaign so header never shows "No campaign"
      const patch: Parameters<typeof patchSettings>[0] = { character_id: id }
      if (selectedCampaignId) patch.campaign_id = selectedCampaignId
      await patchSettings(patch)
      onContextChanged()
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  // --- Session actions ---

  async function handleCreateSession() {
    if (!newSessionTitle.trim() || !selectedCampaignId) return
    setCreatingSession(true)
    setError('')
    try {
      await createSession(selectedCampaignId, newSessionTitle.trim(), newSessionDate)
      setNewSessionTitle('')
      loadSessions(selectedCampaignId)
    } catch (e) {
      setError(String(e))
    } finally {
      setCreatingSession(false)
    }
  }

  async function handleDeleteSession(id: number) {
    if (!confirm('Delete this session and all its messages? This cannot be undone.')) return
    setBusy(true)
    setError('')
    try {
      await deleteSession(id)
      if (selectedCampaignId) loadSessions(selectedCampaignId)
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  async function handleSetActiveSession(id: number) {
    setBusy(true)
    setError('')
    try {
      // Also activate the parent campaign so header never shows "No campaign"
      const patch: Parameters<typeof patchSettings>[0] = { session_id: id }
      if (selectedCampaignId) patch.campaign_id = selectedCampaignId
      await patchSettings(patch)
      onContextChanged()
    } catch (e) {
      setError(String(e))
    } finally {
      setBusy(false)
    }
  }

  // --- Rulebook actions ---

  async function handleUploadRulebook() {
    if (!rulebookFile || !selectedCampaignId) return
    const campaign = campaigns.find(c => c.id === selectedCampaignId)
    if (!campaign) return
    setUploadingRulebook(true)
    setRulebookMsg('')
    setError('')
    try {
      const result = await uploadRulebook(campaign.ruleset_id, rulebookFile, rulebookSourceLabel)
      setRulebookMsg(`Uploaded "${result.source}" — ${result.chunks_created} chunks indexed`)
      setRulebookFile(null)
      setRulebookSourceLabel('')
      loadRulebookSources(selectedCampaignId)
    } catch (e) {
      setError(String(e))
    } finally {
      setUploadingRulebook(false)
    }
  }

  // --- Derived ---
  const selectedCampaign = campaigns.find(c => c.id === selectedCampaignId)
  const selectedRuleset = rulesets.find(r => r.id === selectedCampaign?.ruleset_id)

  return (
    <div className="manage-backdrop" onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div className="manage-panel">
        <div className="manage-header">
          <span className="manage-title">⚙ Manage Campaign</span>
          <button className="manage-close" onClick={onClose}>×</button>
        </div>

        <div className="manage-tabs">
          {(['campaigns', 'characters', 'sessions', 'rulebooks', 'automation'] as Tab[]).map(t => (
            <button
              key={t}
              className={`manage-tab${tab === t ? ' active' : ''}`}
              onClick={() => { setTab(t); onTabChange?.(t) }}
            >
              {t.charAt(0).toUpperCase() + t.slice(1)}
            </button>
          ))}
        </div>

        {error && <div className="manage-error">{error}</div>}

        <div className="manage-content">

          {/* ─── Campaigns Tab ─── */}
          {tab === 'campaigns' && (
            <div className="manage-section">
              <h3 className="manage-section-title">Campaigns</h3>
              <div className="manage-list">
                {campaigns.length === 0 && (
                  <p className="manage-empty">No campaigns yet. Create one below.</p>
                )}
                {campaigns.map(c => {
                  const ruleset = rulesets.find(r => r.id === c.ruleset_id)
                  const isActive = c.id === activeCampaignId
                  return (
                    <div key={c.id} className={`manage-row${isActive ? ' manage-row--active' : ''}`}>
                      <div
                        className="manage-row-info"
                        onClick={() => setSelectedCampaignId(c.id)}
                        style={{ cursor: 'pointer' }}
                      >
                        <span className="manage-row-name">{c.name}</span>
                        <span className="manage-row-meta">{ruleset?.name ?? `ruleset #${c.ruleset_id}`}</span>
                        {c.description && <span className="manage-row-desc">{c.description}</span>}
                      </div>
                      <div className="manage-row-actions">
                        {!isActive && (
                          <button
                            className="manage-btn manage-btn--primary"
                            onClick={() => handleSetActiveCampaign(c.id)}
                            disabled={busy}
                          >
                            Set Active
                          </button>
                        )}
                        {isActive && <span className="manage-badge">Active</span>}
                        <button
                          className="manage-btn manage-btn--danger"
                          onClick={() => handleDeleteCampaign(c.id)}
                          disabled={busy}
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  )
                })}
              </div>

              <div className="manage-form">
                <h4 className="manage-form-title">New Campaign</h4>
                <input
                  className="manage-input"
                  placeholder="Campaign name"
                  value={newCampaignName}
                  onChange={e => setNewCampaignName(e.target.value)}
                />
                <input
                  className="manage-input"
                  placeholder="Description (optional)"
                  value={newCampaignDesc}
                  onChange={e => setNewCampaignDesc(e.target.value)}
                />
                <select
                  className="manage-select"
                  value={newCampaignRuleset}
                  onChange={e => setNewCampaignRuleset(Number(e.target.value))}
                >
                  {rulesets.map(r => (
                    <option key={r.id} value={r.id}>{r.name}</option>
                  ))}
                </select>
                <button
                  className="manage-btn manage-btn--primary"
                  onClick={handleCreateCampaign}
                  disabled={creatingCampaign || !newCampaignName.trim()}
                >
                  {creatingCampaign ? 'Creating…' : 'Create Campaign'}
                </button>
              </div>
            </div>
          )}

          {/* ─── Characters Tab ─── */}
          {tab === 'characters' && (
            <div className="manage-section">
              <h3 className="manage-section-title">Characters</h3>
              <div className="manage-campaign-picker">
                <label className="manage-label">Campaign:</label>
                <select
                  className="manage-select"
                  value={selectedCampaignId ?? ''}
                  onChange={e => setSelectedCampaignId(Number(e.target.value) || null)}
                >
                  <option value="">— select campaign —</option>
                  {campaigns.map(c => (
                    <option key={c.id} value={c.id}>{c.name}</option>
                  ))}
                </select>
              </div>

              {selectedCampaignId && (
                <>
                  <div className="manage-list">
                    {characters.length === 0 && (
                      <p className="manage-empty">No characters in this campaign.</p>
                    )}
                    {characters.map(c => {
                      const isActive = c.id === activeCharacterId
                      return (
                        <div key={c.id} className={`manage-row${isActive ? ' manage-row--active' : ''}`}>
                          <div className="manage-row-info">
                            <span className="manage-row-name">{c.name}</span>
                          </div>
                          <div className="manage-row-actions">
                            {!isActive && (
                              <button
                                className="manage-btn manage-btn--primary"
                                onClick={() => handleSetActiveCharacter(c.id)}
                                disabled={busy}
                              >
                                Set Active
                              </button>
                            )}
                            {isActive && <span className="manage-badge">Active</span>}
                            <button
                              className="manage-btn manage-btn--danger"
                              onClick={() => handleDeleteCharacter(c.id)}
                              disabled={busy}
                            >
                              Delete
                            </button>
                          </div>
                        </div>
                      )
                    })}
                  </div>

                  <div className="manage-form">
                    <h4 className="manage-form-title">New Character</h4>
                    <input
                      className="manage-input"
                      placeholder="Character name"
                      value={newCharName}
                      onChange={e => setNewCharName(e.target.value)}
                      onKeyDown={e => { if (e.key === 'Enter') handleCreateCharacter() }}
                    />
                    {Object.entries(characterOptions).map(([field, choices]) => (
                      <div key={field} className="manage-field-row">
                        <label className="manage-field-label">
                          {field.charAt(0).toUpperCase() + field.slice(1)}
                        </label>
                        <select
                          className="manage-select"
                          value={charOverrides[field] ?? ''}
                          onChange={e => setCharOverrides(prev => ({ ...prev, [field]: e.target.value }))}
                        >
                          <option value="">Random</option>
                          {choices.map(c => <option key={c} value={c}>{c}</option>)}
                        </select>
                      </div>
                    ))}
                    <button
                      className="manage-btn manage-btn--primary"
                      onClick={handleCreateCharacter}
                      disabled={creatingChar || !newCharName.trim()}
                    >
                      {creatingChar ? 'Creating…' : 'Create Character'}
                    </button>
                  </div>
                </>
              )}
            </div>
          )}

          {/* ─── Sessions Tab ─── */}
          {tab === 'sessions' && (
            <div className="manage-section">
              <h3 className="manage-section-title">Sessions</h3>
              <div className="manage-campaign-picker">
                <label className="manage-label">Campaign:</label>
                <select
                  className="manage-select"
                  value={selectedCampaignId ?? ''}
                  onChange={e => setSelectedCampaignId(Number(e.target.value) || null)}
                >
                  <option value="">— select campaign —</option>
                  {campaigns.map(c => (
                    <option key={c.id} value={c.id}>{c.name}</option>
                  ))}
                </select>
              </div>

              {selectedCampaignId && (
                <>
                  <div className="manage-list">
                    {sessions.length === 0 && (
                      <p className="manage-empty">No sessions in this campaign.</p>
                    )}
                    {sessions.map(s => {
                      const isActive = s.id === activeSessionId
                      return (
                        <div key={s.id} className={`manage-row${isActive ? ' manage-row--active' : ''}`}>
                          <div className="manage-row-info">
                            <span className="manage-row-name">{s.title}</span>
                            <span className="manage-row-meta">{s.date}</span>
                          </div>
                          <div className="manage-row-actions">
                            {!isActive && (
                              <button
                                className="manage-btn manage-btn--primary"
                                onClick={() => handleSetActiveSession(s.id)}
                                disabled={busy}
                              >
                                Set Active
                              </button>
                            )}
                            {isActive && <span className="manage-badge">Active</span>}
                            <button
                              className="manage-btn manage-btn--danger"
                              onClick={() => handleDeleteSession(s.id)}
                              disabled={busy}
                            >
                              Delete
                            </button>
                          </div>
                        </div>
                      )
                    })}
                  </div>

                  <div className="manage-form">
                    <h4 className="manage-form-title">New Session</h4>
                    <input
                      className="manage-input"
                      placeholder="Session title"
                      value={newSessionTitle}
                      onChange={e => setNewSessionTitle(e.target.value)}
                    />
                    <input
                      className="manage-input"
                      type="date"
                      value={newSessionDate}
                      onChange={e => setNewSessionDate(e.target.value)}
                    />
                    <button
                      className="manage-btn manage-btn--primary"
                      onClick={handleCreateSession}
                      disabled={creatingSession || !newSessionTitle.trim()}
                    >
                      {creatingSession ? 'Creating…' : 'Create Session'}
                    </button>
                  </div>
                </>
              )}
            </div>
          )}

          {/* ─── Rulebooks Tab ─── */}
          {tab === 'rulebooks' && (
            <div className="manage-section">
              <h3 className="manage-section-title">Rulebooks</h3>
              <div className="manage-campaign-picker">
                <label className="manage-label">Campaign:</label>
                <select
                  className="manage-select"
                  value={selectedCampaignId ?? ''}
                  onChange={e => setSelectedCampaignId(Number(e.target.value) || null)}
                >
                  <option value="">— select campaign —</option>
                  {campaigns.map(c => (
                    <option key={c.id} value={c.id}>{c.name}</option>
                  ))}
                </select>
              </div>

              {selectedCampaignId && (
                <>
                  {selectedRuleset && (
                    <div className="manage-ruleset-info">
                      Ruleset: <strong>{selectedRuleset.name}</strong>
                    </div>
                  )}

                  <div className="manage-list">
                    <h4 className="manage-form-title">Uploaded Books</h4>
                    {rulebookSources.length === 0 && (
                      <p className="manage-empty">No rulebooks uploaded yet.</p>
                    )}
                    {rulebookSources.map(src => (
                      <div key={src.source} className="manage-row">
                        <div className="manage-row-info">
                          <span className="manage-row-name">{src.source}</span>
                          <span className="manage-row-meta">{src.chunks} chunks indexed</span>
                        </div>
                      </div>
                    ))}
                  </div>

                  <div className="manage-form">
                    <h4 className="manage-form-title">Upload Rulebook</h4>
                    <p className="manage-form-hint">
                      Upload a PDF or text file. The GM will use it to answer rules questions.
                      Give each book a unique label (e.g. "Core Rulebook", "Monster Manual").
                    </p>
                    <input
                      className="manage-input"
                      placeholder="Book label (e.g. Core Rulebook)"
                      value={rulebookSourceLabel}
                      onChange={e => setRulebookSourceLabel(e.target.value)}
                    />
                    <input
                      type="file"
                      className="manage-file-input"
                      accept=".pdf,.txt,.md"
                      onChange={e => setRulebookFile(e.target.files?.[0] ?? null)}
                    />
                    {rulebookMsg && <p className="manage-success">{rulebookMsg}</p>}
                    <button
                      className="manage-btn manage-btn--primary"
                      onClick={handleUploadRulebook}
                      disabled={uploadingRulebook || !rulebookFile || !rulebookSourceLabel.trim()}
                    >
                      {uploadingRulebook ? 'Uploading…' : 'Upload Rulebook'}
                    </button>
                  </div>
                </>
              )}
            </div>
          )}

          {/* ─── Automation Tab ─── */}
          {tab === 'automation' && (
            <AutomationSettingsPanel />
          )}

        </div>
      </div>
    </div>
  )
}
