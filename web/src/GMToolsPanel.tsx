import { useState } from 'react'
import ReactMarkdown from 'react-markdown'
import { postImprovise, postPreSessionBrief, postDetectThreads, postCampaignAsk } from './api'

interface GMToolsPanelProps {
  sessionId: number | null
  campaignId: number | null
  aiEnabled: boolean
}

type ToolTab = 'improvise' | 'brief' | 'threads' | 'ask'

export function GMToolsPanel({ sessionId, campaignId, aiEnabled }: GMToolsPanelProps) {
  const [activeTool, setActiveTool] = useState<ToolTab>('improvise')
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState('')
  const [askQuestion, setAskQuestion] = useState('')

  const hasSession = sessionId !== null
  const hasCampaign = campaignId !== null

  async function handleTool(tool: ToolTab) {
    setActiveTool(tool)
    if (!aiEnabled) return
    setLoading(true)
    setResult('')
    try {
      let text = ''
      switch (tool) {
        case 'improvise':
          text = await postImprovise(sessionId!)
          break
        case 'brief':
          text = await postPreSessionBrief(campaignId!)
          break
        case 'threads':
          text = await postDetectThreads(sessionId!)
          break
        case 'ask':
          if (!askQuestion.trim()) { setLoading(false); return }
          text = await postCampaignAsk(campaignId!, askQuestion)
          break
      }
      setResult(text)
    } catch (e) {
      setResult('Error: ' + (e instanceof Error ? e.message : 'Unknown error'))
    } finally {
      setLoading(false)
    }
  }

  const tools: { key: ToolTab; label: string; desc: string; needs: 'session' | 'campaign' }[] = [
    { key: 'improvise', label: 'Improvise', desc: 'Generate an improvised complication or plot twist from recent events.', needs: 'session' },
    { key: 'brief', label: 'Brief', desc: 'Generate a pre-session brief from world notes and objectives.', needs: 'campaign' },
    { key: 'threads', label: 'Threads', desc: 'Detect unresolved narrative threads and loose ends.', needs: 'session' },
    { key: 'ask', label: 'Ask', desc: 'Ask a freeform question about your campaign.', needs: 'campaign' },
  ]

  const needsCtx = (tool: ToolTab) => tool === 'improvise' || tool === 'threads'

  return (
    <div className="gm-tools-panel">
      <div className="gm-tools-tabs">
        {tools.map(t => (
          <button
            key={t.key}
            className={`gm-tool-tab${activeTool === t.key ? ' active' : ''}`}
            onClick={() => setActiveTool(t.key)}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="gm-tool-content">
        <p className="gm-tool-desc">{tools.find(t => t.key === activeTool)?.desc}</p>

        {!aiEnabled ? (
          <p className="gm-tool-disabled">AI not configured. Set DEEPSEEK_API_KEY to use GM tools.</p>
        ) : activeTool === 'ask' ? (
          <div className="gm-tool-ask-form">
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
              disabled={loading || !askQuestion.trim()}
            >
              {loading ? 'Thinking…' : 'Ask'}
            </button>
          </div>
        ) : (
          <button
            className="gm-tool-btn"
            onClick={() => handleTool(activeTool)}
            disabled={loading || (needsCtx(activeTool) ? !hasSession : !hasCampaign)}
          >
            {loading ? 'Thinking…' : `Generate ${activeTool === 'improvise' ? 'Complication' : activeTool === 'brief' ? 'Brief' : 'Threads'}`}
          </button>
        )}

        {loading && <p className="gm-tool-thinking">▸ Generating…</p>}
        {result && (
          <div className="gm-tool-result">
            <ReactMarkdown>{result}</ReactMarkdown>
          </div>
        )}
      </div>
    </div>
  )
}
