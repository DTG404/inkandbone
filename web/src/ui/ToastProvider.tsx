import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'

type ToastKind = 'info' | 'success' | 'error'

export interface ToastAction {
  label: string
  onClick: () => void
}

interface ToastEntry {
  id: number
  kind: ToastKind
  message: string
  action?: ToastAction
}

interface ToastOptions {
  label: string
  onClick: () => void
}

interface ToastAPI {
  info: (message: string, action?: ToastOptions) => void
  success: (message: string, action?: ToastOptions) => void
  error: (message: string, action?: ToastOptions) => void
}

const ToastContext = createContext<ToastAPI | null>(null)

function ToastItem({ toast, dismiss }: { toast: ToastEntry; dismiss: (id: number) => void }) {
  const [paused, setPaused] = useState(false)
  const remaining = useRef(5000)
  const startedAt = useRef(0)

  useEffect(() => {
    if (toast.kind === 'error' || paused) return
    startedAt.current = Date.now()
    const timer = window.setTimeout(() => dismiss(toast.id), remaining.current)
    return () => {
      window.clearTimeout(timer)
      remaining.current = Math.max(0, remaining.current - (Date.now() - startedAt.current))
    }
  }, [dismiss, paused, toast.id, toast.kind])

  return (
    <div
      className={`ui-toast ui-toast-${toast.kind}`}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onFocusCapture={() => setPaused(true)}
      onBlurCapture={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) setPaused(false)
      }}
    >
      <span>{toast.message}</span>
      {toast.action && (
        <button type="button" onClick={() => { dismiss(toast.id); toast.action?.onClick() }}>
          {toast.action.label}
        </button>
      )}
      <button type="button" aria-label={`Dismiss ${toast.message}`} onClick={() => dismiss(toast.id)}>×</button>
    </div>
  )
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<ToastEntry[]>([])
  const nextId = useRef(1)
  const dismiss = useCallback((id: number) => setToasts((current) => current.filter((toast) => toast.id !== id)), [])
  const add = useCallback((kind: ToastKind, message: string, action?: ToastOptions) => {
    const entry: ToastEntry = { id: nextId.current++, kind, message, action }
    setToasts((current) => [...current, entry].slice(-5))
  }, [])
  const api = useMemo<ToastAPI>(() => ({
    info: (message, action) => add('info', message, action),
    success: (message, action) => add('success', message, action),
    error: (message, action) => add('error', message, action),
  }), [add])

  const polite = toasts.filter((toast) => toast.kind !== 'error')
  const assertive = toasts.filter((toast) => toast.kind === 'error')
  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="ui-toast-viewport">
        <div role="status" aria-live="polite" aria-atomic="false">
          {polite.map((toast) => <ToastItem key={toast.id} toast={toast} dismiss={dismiss} />)}
        </div>
        <div role="alert" aria-live="assertive" aria-atomic="false">
          {assertive.map((toast) => <ToastItem key={toast.id} toast={toast} dismiss={dismiss} />)}
        </div>
      </div>
    </ToastContext.Provider>
  )
}

export function useToast(): ToastAPI {
  const context = useContext(ToastContext)
  if (!context) throw new Error('useToast must be used inside ToastProvider')
  return context
}
