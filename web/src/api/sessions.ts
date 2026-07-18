import { request } from './transport'
import { parseSSE } from '../sse'
import type { GameContext, Message, DiceRoll, TimelineEntry, XPEntry } from '../types'

export async function fetchContext(): Promise<GameContext> {
  const res = await request('/api/context')
  if (!res.ok) throw new Error(`GET /api/context failed: ${res.status}`)
  return res.json()
}
export async function fetchMessages(sessionId: number): Promise<Message[]> {
  const url = `/api/sessions/${sessionId}/messages`
  const res = await request(url)
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`)
  return res.json()
}


export async function fetchDiceRolls(sessionId: number): Promise<DiceRoll[]> {
  const url = `/api/sessions/${sessionId}/dice-rolls`
  const res = await request(url)
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`)
  return res.json()
}

export async function fetchTimeline(sessionId: number): Promise<TimelineEntry[]> {
  const url = `/api/sessions/${sessionId}/timeline`
  const res = await request(url)
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`)
  return res.json()
}


export async function patchSession(sessionId: number, updates: { scene_tags?: string; summary?: string; notes?: string }): Promise<void> {
  const res = await request(`/api/sessions/${sessionId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  })
  if (!res.ok) throw new Error(`patchSession failed: ${res.status}`)
}

export async function patchSessionSummary(sessionId: number, summary: string): Promise<void> {
  const res = await request(`/api/sessions/${sessionId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ summary }),
  })
  if (!res.ok) throw new Error(`PATCH /api/sessions/${sessionId} failed: ${res.status}`)
}

export async function patchSessionNotes(sessionId: number, notes: string): Promise<void> {
  const res = await request(`/api/sessions/${sessionId}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ notes }),
  })
  if (!res.ok) throw new Error(`patchSessionNotes failed: ${res.status}`)
}

export async function generateRecap(sessionId: number): Promise<{ summary: string }> {
  const url = `/api/sessions/${sessionId}/recap`
  const res = await request(url, { method: 'POST' })
  if (!res.ok) throw new Error(`POST ${url} failed: ${res.status}`)
  return res.json()
}


export async function sendMessage(sessionId: number, content: string, whisper?: boolean, characterId?: number | null): Promise<void> {
  const body: Record<string, unknown> = { role: 'user', content }
  if (whisper) body['whisper'] = true
  if (characterId != null) body['character_id'] = characterId
  const res = await request(`/api/sessions/${sessionId}/messages`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(`sendMessage failed: ${res.status}`)
}


export async function gmRespondStream(
  sessionId: number,
  onChunk: (text: string) => void,
): Promise<string> {
  const res = await request(`/api/sessions/${sessionId}/gm-respond-stream`, { method: 'POST' })
  if (!res.ok) throw new Error(`gmRespondStream failed: ${res.status}`)
  let accumulated = ''
  let streamError: Error | undefined
  let protocolError: Error | undefined
  let terminalCount = 0
  await parseSSE(res, (event) => {
    if (terminalCount > 0) {
      protocolError = new Error('GM stream sent multiple terminal events')
      return
    }
    if (event.type === 'delta') {
      accumulated += event.delta
      onChunk(event.delta)
    } else if (event.type === 'error') {
      terminalCount++
      const request = event.request_id ? `; request ${event.request_id}` : ''
      streamError = new Error(`GM stream failed (${event.code}${request})`)
    } else if (event.type === 'done') {
      terminalCount++
    }
  })
  if (protocolError || terminalCount > 1) throw protocolError ?? new Error('GM stream sent multiple terminal events')
  if (terminalCount === 0) throw new Error('GM stream ended without completion')
  if (streamError) throw streamError
  return accumulated
}

export async function rollDice(
  sessionId: number,
  expression: string,
): Promise<{ expression: string; result: number; rolls: number[] }> {
  const res = await request(`/api/sessions/${sessionId}/dice-rolls`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ expression }),
  })
  if (!res.ok) throw new Error(`rollDice failed: ${res.status}`)
  return res.json()
}


export async function fetchXP(sessionId: number): Promise<XPEntry[]> {
  const res = await request(`/api/sessions/${sessionId}/xp`)
  if (!res.ok) throw new Error(`fetchXP failed: ${res.status}`)
  return res.json()
}

export async function createXP(sessionId: number, note: string, amount?: number): Promise<XPEntry> {
  const res = await request(`/api/sessions/${sessionId}/xp`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ note, amount: amount ?? null }),
  })
  if (!res.ok) throw new Error(`createXP failed: ${res.status}`)
  return res.json()
}

export async function deleteXP(id: number): Promise<void> {
  const res = await request(`/api/xp/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteXP failed: ${res.status}`)
}

export async function postImprovise(sessionId: number): Promise<string> {
  const res = await request(`/api/sessions/${sessionId}/improvise`, { method: 'POST' })
  if (!res.ok) throw new Error('Improvise failed')
  const data = await res.json()
  return data.result
}

export async function postPreSessionBrief(campaignId: number): Promise<string> {
  const res = await request(`/api/campaigns/${campaignId}/pre-session-brief`, { method: 'POST' })
  if (!res.ok) throw new Error('Pre-session brief failed')
  const data = await res.json()
  return data.result
}

export async function postDetectThreads(sessionId: number): Promise<string> {
  const res = await request(`/api/sessions/${sessionId}/detect-threads`, { method: 'POST' })
  if (!res.ok) throw new Error('Detect threads failed')
  const data = await res.json()
  return data.result
}

export async function postCampaignAsk(campaignId: number, question: string): Promise<string> {
  const res = await request(`/api/campaigns/${campaignId}/ask`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ question }),
  })
  if (!res.ok) throw new Error('Campaign ask failed')
  const data = await res.json()
  return data.result
}

// --- Management API ---


export async function fetchSessions(campaignId: number): Promise<import('../types').Session[]> {
  const res = await request(`/api/campaigns/${campaignId}/sessions`)
  if (!res.ok) throw new Error(`fetchSessions failed: ${res.status}`)
  return res.json()
}


export async function createSession(
  campaignId: number,
  title: string,
  date: string,
): Promise<import('../types').Session> {
  const res = await request(`/api/campaigns/${campaignId}/sessions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title, date }),
  })
  if (!res.ok) throw new Error(`createSession failed: ${res.status}`)
  return res.json()
}

export async function deleteSession(id: number): Promise<void> {
  const res = await request(`/api/sessions/${id}`, { method: 'DELETE' })
  if (!res.ok) throw new Error(`deleteSession failed: ${res.status}`)
}


export async function reanalyzeSession(sessionId: number): Promise<void> {
  const res = await request(`/api/sessions/${sessionId}/reanalyze`, { method: 'POST' })
  if (!res.ok) throw new Error('Reanalyze failed')
}
