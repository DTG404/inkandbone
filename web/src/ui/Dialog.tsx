import { useEffect, useId, useRef, type ReactNode } from 'react'

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

const dialogStack: symbol[] = []
let bodyScrollLocks = 0
let bodyOverflowBeforeLock = ''

function registerDialog(token: symbol): () => void {
  dialogStack.push(token)
  return () => {
    const index = dialogStack.lastIndexOf(token)
    if (index >= 0) dialogStack.splice(index, 1)
  }
}

function isTopmostDialog(token: symbol): boolean {
  return dialogStack.at(-1) === token
}

function acquireBodyScrollLock(): () => void {
  if (bodyScrollLocks === 0) bodyOverflowBeforeLock = document.body.style.overflow
  bodyScrollLocks += 1
  document.body.style.overflow = 'hidden'
  let released = false
  return () => {
    if (released) return
    released = true
    bodyScrollLocks = Math.max(0, bodyScrollLocks - 1)
    if (bodyScrollLocks === 0) document.body.style.overflow = bodyOverflowBeforeLock
  }
}

function focusableElements(container: HTMLElement): HTMLElement[] {
  return Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR))
    .filter((element) => {
      for (let current: HTMLElement | null = element; current && current !== container; current = current.parentElement) {
        const style = window.getComputedStyle(current)
        if (
          current.hasAttribute('hidden') || current.getAttribute('aria-hidden') === 'true' ||
          style.display === 'none' || style.visibility === 'hidden'
        ) return false
      }
      return true
    })
}

export interface DialogProps {
  open: boolean
  title: string
  onClose: () => void
  children: ReactNode
  className?: string
  modal?: boolean
}

export function Dialog({ open, title, onClose, children, className = '', modal = true }: DialogProps) {
  const titleId = useId()
  const dialogRef = useRef<HTMLDivElement>(null)
  const previousActiveRef = useRef<HTMLElement | null>(null)
  const onCloseRef = useRef(onClose)
  const stackTokenRef = useRef(Symbol('dialog'))

  useEffect(() => {
    onCloseRef.current = onClose
  }, [onClose])

  useEffect(() => {
    if (!open) return

    previousActiveRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
    const token = stackTokenRef.current
    const unregister = registerDialog(token)
    const releaseScrollLock = modal ? acquireBodyScrollLock() : () => undefined
    const dialog = dialogRef.current
    const firstFocusable = dialog ? focusableElements(dialog)[0] : undefined
    ;(firstFocusable ?? dialog)?.focus()

    const handleKeyDown = (event: KeyboardEvent) => {
      if (!isTopmostDialog(token)) return
      if (event.key === 'Escape') {
        event.preventDefault()
        onCloseRef.current()
        return
      }
      if (event.key !== 'Tab' || !dialog) return

      const focusable = focusableElements(dialog)
      if (focusable.length === 0) {
        event.preventDefault()
        dialog.focus()
        return
      }
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (event.shiftKey && (document.activeElement === first || document.activeElement === dialog)) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      unregister()
      releaseScrollLock()
      previousActiveRef.current?.focus()
    }
  }, [modal, open])

  if (!open) return null

  return (
    <div
      className="ui-dialog-backdrop"
      data-testid="dialog-backdrop"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && isTopmostDialog(stackTokenRef.current)) onCloseRef.current()
      }}
    >
      <div
        ref={dialogRef}
        className={`ui-dialog ${className}`.trim()}
        role="dialog"
        aria-modal={modal ? 'true' : undefined}
        aria-labelledby={titleId}
        tabIndex={-1}
      >
        <h2 id={titleId} className="ui-dialog-title">{title}</h2>
        {children}
      </div>
    </div>
  )
}
