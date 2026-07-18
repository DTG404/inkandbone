import { useEffect, useId, useRef, type ReactNode } from 'react'

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

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

  useEffect(() => {
    onCloseRef.current = onClose
  }, [onClose])

  useEffect(() => {
    if (!open) return

    previousActiveRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
    const previousOverflow = document.body.style.overflow
    if (modal) document.body.style.overflow = 'hidden'
    const dialog = dialogRef.current
    const firstFocusable = dialog ? focusableElements(dialog)[0] : undefined
    ;(firstFocusable ?? dialog)?.focus()

    const handleKeyDown = (event: KeyboardEvent) => {
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
      if (modal) document.body.style.overflow = previousOverflow
      previousActiveRef.current?.focus()
    }
  }, [modal, open])

  if (!open) return null

  return (
    <div
      className="ui-dialog-backdrop"
      data-testid="dialog-backdrop"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose()
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
