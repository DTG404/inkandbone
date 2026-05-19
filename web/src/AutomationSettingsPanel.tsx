import { useState, useEffect } from 'react'
import { fetchAutomationSettings, patchAutomationSetting } from './api'
import type { AutomationSetting } from './api'

export function AutomationSettingsPanel() {
  const [settings, setSettings] = useState<AutomationSetting[]>([])
  const [loading, setLoading] = useState(true)
  const [toggling, setToggling] = useState<string | null>(null)

  useEffect(() => {
    fetchAutomationSettings()
      .then(setSettings)
      .catch(console.error)
      .finally(() => setLoading(false))
  }, [])

  async function handleToggle(key: string, current: boolean) {
    setToggling(key)
    try {
      await patchAutomationSetting(key, !current)
      setSettings(prev => prev.map(s => s.key === key ? { ...s, enabled: !current } : s))
    } catch (e) {
      console.error(e)
    } finally {
      setToggling(null)
    }
  }

  if (loading) return <div className="manage-section"><p className="manage-empty">Loading automation settings…</p></div>

  return (
    <div className="manage-section">
      <h3 className="manage-section-title">Automation</h3>
      <p className="manage-form-hint" style={{ marginBottom: 12 }}>
        Enable or disable background automation goroutines. These run after every GM response.
      </p>
      <div className="automation-list">
        {settings.map(s => (
          <div key={s.key} className="manage-row">
            <div className="manage-row-info">
              <span className="manage-row-name">{s.label}</span>
              <span className="manage-row-meta" style={{ fontSize: '0.7rem', fontFamily: 'monospace', opacity: 0.5 }}>{s.key}</span>
            </div>
            <div className="manage-row-actions">
              <label className="automation-toggle">
                <input
                  type="checkbox"
                  checked={s.enabled}
                  disabled={toggling === s.key}
                  onChange={() => handleToggle(s.key, s.enabled)}
                />
                <span className={`automation-toggle-slider${s.enabled ? ' on' : ''}`} />
              </label>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
