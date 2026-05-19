import { useState, useEffect, useCallback } from 'react'
import type { CampaignConfig } from './api'
import { fetchCampaignConfig, patchCampaignConfig, postImprovise, postPreSessionBrief, postDetectThreads, postCampaignAsk } from './api'
import ReactMarkdown from 'react-markdown'

interface GMScreenPanelProps {
  campaignId: number | null
  sessionId: number | null
  aiEnabled: boolean
  onClose: () => void
}

export function GMScreenPanel({ campaignId, sessionId, aiEnabled, onClose }: GMScreenPanelProps) {
  const [config, setConfig] = useState<CampaignConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [savingDesc, setSavingDesc] = useState(false)
  const [savingNotes, setSavingNotes] = useState(false)
  const [savingPrompt, setSavingPrompt] = useState(false)

  // GM Tools state
  const [toolsLoading, setToolsLoading] = useState(false)
  const [toolsResult, setToolsResult] = useState('')
  const [askQuestion, setAskQuestion] = useState('')
  const [activeTool, setActiveTool] = useState<string>('')

  useEffect(() => {
    if (!campaignId) {
      setLoading(false)
      return
    }
    setLoading(true)
    fetchCampaignConfig(campaignId)
      .then(setConfig)
      .catch(() => setError('Failed to load campaign config'))
      .finally(() => setLoading(false))
  }, [campaignId])

  const saveField = useCallback(async (field: 'description' | 'gm_notes' | 'system_prompt_override', value: string) => {
    if (!campaignId) return
    const setSaving = field === 'description' ? setSavingDesc : field === 'gm_notes' ? setSavingNotes : setSavingPrompt
    setSaving(true)
    try {
      await patchCampaignConfig(campaignId, { [field]: value })
      setConfig(prev => prev ? { ...prev, [field]: value } : prev)
    } catch {
      setError(`Failed to save ${field}`)
    } finally {
      setSaving(false)
    }
  }, [campaignId])

  async function handleTool(tool: string) {
    if (!aiEnabled || !campaignId) return
    setActiveTool(tool)
    setToolsLoading(true)
    setToolsResult('')
    try {
      let text = ''
      switch (tool) {
        case 'improvise':
          text = sessionId ? await postImprovise(sessionId) : 'No active session.'
          break
        case 'brief':
          text = await postPreSessionBrief(campaignId)
          break
        case 'threads':
          text = sessionId ? await postDetectThreads(sessionId) : 'No active session.'
          break
        case 'ask':
          if (!askQuestion.trim()) { setToolsLoading(false); return }
          text = await postCampaignAsk(campaignId, askQuestion)
          break
      }
      setToolsResult(text)
    } catch (e) {
      setToolsResult('Error: ' + (e instanceof Error ? e.message : 'Unknown error'))
    } finally {
      setToolsLoading(false)
    }
  }

  if (!campaignId) {
    return (
      <div className="manage-backdrop" onClick={(e) => { if (e.target === e.currentTarget) onClose() }}>
        <div className="manage-panel">
          <div className="manage-header">
            <span className="manage-title">GM Screen</span>
            <button className="manage-close" onClick={onClose}>×</button>
          </div>
          <div className="manage-content">
            <p className="gm-tool-disabled">No campaign selected.</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="manage-backdrop" onClick={(e) => { if (e.target === e.currentTarget) onClose() }}>
      <div className="manage-panel gm-screen-panel">
        <div className="manage-header">
          <span className="manage-title">GM Screen</span>
          <button className="manage-close" onClick={onClose}>×</button>
        </div>

        {error && <div className="manage-error">{error}</div>}

        <div className="manage-content">
          {loading ? (
            <p className="gm-tool-thinking">Loading campaign config…</p>
          ) : config ? (
            <>
              {/* Campaign Overview */}
              <div className="manage-section">
                <div className="manage-section-title">Campaign Overview</div>
                <div className="gm-screen-stats">
                  <span className="gm-screen-stat">Ruleset: <strong>{config.ruleset_name}</strong></span>
                  <span className="gm-screen-stat">Sessions: <strong>{config.session_count}</strong></span>
                  <span className="gm-screen-stat">Characters: <strong>{config.character_count}</strong></span>
                </div>
              </div>

              {/* Campaign Description */}
              <div className="manage-section">
                <div className="manage-section-title">Description</div>
                <textarea
                  className="gm-screen-textarea"
                  defaultValue={config.description}
                  rows={3}
                  disabled={savingDesc}
                  onBlur={(e) => {
                    const val = e.target.value.trim()
                    if (val !== config.description) saveField('description', val)
                  }}
                />
                {savingDesc && <span className="gm-screen-saving">Saving…</span>}
              </div>

              {/* GM Notes */}
              <div className="manage-section">
                <div className="manage-section-title">GM Notes</div>
                <p className="gm-tool-desc">Private notes visible only to you. Not shared with the AI.</p>
                <textarea
                  className="gm-screen-textarea gm-screen-textarea--tall"
                  defaultValue={config.gm_notes}
                  rows={8}
                  disabled={savingNotes}
                  onBlur={(e) => {
                    const val = e.target.value.trim()
                    if (val !== config.gm_notes) saveField('gm_notes', val)
                  }}
                />
                {savingNotes && <span className="gm-screen-saving">Saving…</span>}
              </div>

              {/* System Prompt Override */}
              <div className="manage-section">
                <div className="manage-section-title">System Prompt Override</div>
                <p className="gm-tool-desc">
                  Custom instructions injected into the AI's system prompt. Use this to set tone, house rules, or specific narrative directions.
                </p>
                <textarea
                  className="gm-screen-textarea gm-screen-textarea--tall"
                  defaultValue={config.system_prompt_override}
                  rows={6}
                  disabled={savingPrompt}
                  onBlur={(e) => {
                    const val = e.target.value.trim()
                    if (val !== config.system_prompt_override) saveField('system_prompt_override', val)
                  }}
                />
                {savingPrompt && <span className="gm-screen-saving">Saving…</span>}
              </div>

              {/* GM Tools */}
              <div className="manage-section">
                <div className="manage-section-title">GM Tools</div>
                <div className="gm-screen-tools">
                  <button
                    className={`gm-tool-tab${activeTool === 'improvise' ? ' active' : ''}`}
                    onClick={() => handleTool('improvise')}
                    disabled={toolsLoading || !sessionId}
                  >
                    Improvise
                  </button>
                  <button
                    className={`gm-tool-tab${activeTool === 'brief' ? ' active' : ''}`}
                    onClick={() => handleTool('brief')}
                    disabled={toolsLoading}
                  >
                    Brief
                  </button>
                  <button
                    className={`gm-tool-tab${activeTool === 'threads' ? ' active' : ''}`}
                    onClick={() => handleTool('threads')}
                    disabled={toolsLoading || !sessionId}
                  >
                    Threads
                  </button>
                  <button
                    className={`gm-tool-tab${activeTool === 'ask' ? ' active' : ''}`}
                    onClick={() => setActiveTool('ask')}
                  >
                    Ask
                  </button>
                </div>

                {activeTool === 'ask' ? (
                  <div className="gm-tool-ask-form" style={{ marginTop: '0.5rem' }}>
                    <textarea
                      className="gm-tool-input"
                      placeholder="Ask something about your campaign…"
                      value={askQuestion}
                      onChange={e => setAskQuestion(e.target.value)}
                      rows={3}
                    />
                    <button
                      className="gm-tool-btn"
                      onClick={() => handleTool('ask')}
                      disabled={toolsLoading || !askQuestion.trim()}
                      style={{ marginTop: '0.4rem' }}
                    >
                      {toolsLoading ? 'Thinking…' : 'Ask'}
                    </button>
                  </div>
                ) : toolsLoading ? (
                  <p className="gm-tool-thinking" style={{ marginTop: '0.5rem' }}>▸ Generating…</p>
                ) : null}

                {toolsResult && (
                  <div className="gm-tool-result" style={{ marginTop: '0.5rem' }}>
                    <ReactMarkdown>{toolsResult}</ReactMarkdown>
                  </div>
                )}
              </div>
            </>
          ) : (
            <p className="gm-tool-disabled">Could not load campaign data.</p>
          )}
        </div>
      </div>
    </div>
  )
}
