import { useState, useEffect } from 'react'
import { fetchAutomationSettings, patchAutomationSetting } from './api'
import type { AutomationSetting } from './api'
import { StatusRegion } from './ui/StatusRegion'
import { useToast } from './ui/ToastProvider'

export function AutomationSettingsPanel() {
  const toast = useToast()
  const [settings, setSettings] = useState<AutomationSetting[]>([])
  const [loading, setLoading] = useState(true)
  const [toggling, setToggling] = useState<string | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    fetchAutomationSettings()
      .then(setSettings)
      .catch((cause) => {
        console.error(cause)
        setError('Automation settings could not be loaded.')
      })
      .finally(() => setLoading(false))
  }, [])

  async function handleToggle(key: string, current: boolean) {
    setToggling(key)
    try {
      await patchAutomationSetting(key, !current)
      setSettings(prev => prev.map(s => s.key === key ? { ...s, enabled: !current } : s))
    } catch (e) {
      console.error(e)
      toast.error('The automation setting could not be changed. Try again.')
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
      <StatusRegion message={error} priority="assertive" />
      <div className="automation-list">
        {settings.map(s => (
          <div key={s.key} className="manage-row">
            <div className="manage-row-info">
              <span className="manage-row-name">{s.label}</span>
              <span className="manage-row-meta" style={{ fontSize: '0.7rem', fontFamily: 'var(--mono)', opacity: 0.5 }}>{s.key}</span>
              <span className="automation-health">
                {s.cooling_down
                  ? 'Cooling down; a probe will run automatically after the cooldown.'
                  : s.running > 0 ? `${s.running} running` : 'Ready'}
                {s.queued > 0 ? ` · ${s.queued} queued` : ''}
                {s.last_error ? ' · Recent automation error' : ''}
              </span>
            </div>
            <div className="manage-row-actions">
              <label className="automation-toggle">
                <input
                  type="checkbox"
                  aria-label={`Enable ${s.label}`}
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
