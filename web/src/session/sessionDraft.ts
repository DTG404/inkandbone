import { useSyncExternalStore } from 'react'

export interface Submission {
  key: string
  text: string
  character: number | null
  whisper: boolean
  fromComposer: boolean
}
export interface SessionDraft { text: string; whisper: boolean; submission?: Submission }
const empty: SessionDraft = { text: '', whisper: false }
const listeners = new Set<() => void>()
const fallback = new Map<number, string>()
const storageKey = (id: number) => `inkandbone:session-draft:${id}`

function read(id: number | null): string {
  if (id === null) return ''
  if (fallback.has(id)) return fallback.get(id) ?? ''
  try { return localStorage.getItem(storageKey(id)) ?? '' }
  catch { return fallback.get(id) ?? '' }
}

function parse(raw: string): SessionDraft {
  try {
    const draft = JSON.parse(raw) as SessionDraft
    if (typeof draft.text === 'string' && typeof draft.whisper === 'boolean') return draft
  } catch { /* A missing or obsolete draft starts empty. */ }
  return empty
}

export function updateDraft(id: number, update: (draft: SessionDraft) => SessionDraft) {
  const next = update(parse(read(id)))
  const raw = JSON.stringify(next)
  try {
    if (!next.text && !next.whisper && !next.submission) localStorage.removeItem(storageKey(id))
    else localStorage.setItem(storageKey(id), raw)
    fallback.delete(id)
  } catch { fallback.set(id, raw) }
  listeners.forEach(listener => listener())
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  window.addEventListener('storage', listener)
  return () => { listeners.delete(listener); window.removeEventListener('storage', listener) }
}

export function useSessionDraft(id: number | null) {
  const raw = useSyncExternalStore(subscribe, () => read(id), () => '')
  return parse(raw)
}

export function acknowledgeSubmission(id: number, key: string) {
  updateDraft(id, draft => {
    const item = draft.submission
    if (!item || item.key !== key) return draft
    if (item.fromComposer && draft.text.trim() === item.text && draft.whisper === item.whisper) return empty
    return { ...draft, submission: undefined }
  })
}
