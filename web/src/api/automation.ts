import { request } from './transport'

export interface AutomationSetting {
  key: string
  label: string
  enabled: boolean
  status: 'closed' | 'open' | 'half-open'
  failure_count: number
  cooling_down: boolean
  queued: number
  running: number
  last_success: string | null
  last_error: string
}
export async function fetchAutomationSettings(): Promise<AutomationSetting[]> {
  const res = await request('/api/settings/automations')
  if (!res.ok) throw new Error('fetchAutomationSettings failed')
  return res.json()
}

export async function patchAutomationSetting(key: string, enabled: boolean): Promise<void> {
  const res = await request('/api/settings/automations', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ key, enabled }),
  })
  if (!res.ok) throw new Error('patchAutomationSetting failed')
}
