import { useCallback, useEffect, useRef, useState, type CSSProperties, type KeyboardEvent, type PointerEvent as ReactPointerEvent, type ReactNode } from 'react'
import { LAYOUT_LIMITS, useWorkspaceLayout } from './useWorkspaceLayout'
import type { MobileDestination } from '../navigation/panelRegistry'
import { Drawer } from '../ui/Drawer'

interface WorkspaceShellProps {
  header?: ReactNode
  left: ReactNode
  story: ReactNode
  right: ReactNode
  mobileNav?: ReactNode
  mobileDestination?: MobileDestination
}

type ResizeSide = 'left' | 'right'
type TabletDrawer = 'character' | 'tools' | null

function useWorkspaceMedia(query: string): boolean {
  const [matches, setMatches] = useState(() => (
    typeof window.matchMedia === 'function' && window.matchMedia(query).matches
  ))

  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return
    const media = window.matchMedia(query)
    const update = () => setMatches(media.matches)
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [query])

  return matches
}

export function WorkspaceShell({ header, left, story, right, mobileNav, mobileDestination = 'story' }: WorkspaceShellProps) {
  const tablet = useWorkspaceMedia('(min-width: 600px) and (max-width: 899px)')
  const mobile = useWorkspaceMedia('(max-width: 599px)')
  const [tabletDrawer, setTabletDrawer] = useState<TabletDrawer>(null)
  const activeResizeCleanup = useRef<(() => void) | null>(null)
  const {
    layout,
    setLeftWidth,
    setRightWidth,
    setLeftCollapsed,
    setRightCollapsed,
    resetLayout,
  } = useWorkspaceLayout()

  useEffect(() => {
    if (!tablet) setTabletDrawer(null)
  }, [tablet])

  useEffect(() => () => activeResizeCleanup.current?.(), [])

  const startPointerResize = useCallback((side: ResizeSide, event: ReactPointerEvent<HTMLDivElement>) => {
    event.preventDefault()
    activeResizeCleanup.current?.()
    const startX = event.clientX
    const startWidth = side === 'left' ? layout.leftWidth : layout.rightWidth
    const move = (moveEvent: PointerEvent) => {
      const delta = moveEvent.clientX - startX
      if (side === 'left') setLeftWidth(startWidth + delta)
      else setRightWidth(startWidth - delta)
    }
    const stop = () => {
      document.removeEventListener('pointermove', move)
      document.removeEventListener('pointerup', stop)
      document.removeEventListener('pointercancel', stop)
      if (activeResizeCleanup.current === stop) activeResizeCleanup.current = null
    }
    activeResizeCleanup.current = stop
    document.addEventListener('pointermove', move)
    document.addEventListener('pointerup', stop)
    document.addEventListener('pointercancel', stop)
  }, [layout.leftWidth, layout.rightWidth, setLeftWidth, setRightWidth])

  const resizeByKeyboard = (side: ResizeSide, event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return
    event.preventDefault()
    const amount = event.shiftKey ? 48 : 16
    if (side === 'left') {
      setLeftWidth(layout.leftWidth + (event.key === 'ArrowRight' ? amount : -amount))
    } else {
      setRightWidth(layout.rightWidth + (event.key === 'ArrowLeft' ? amount : -amount))
    }
  }

  const desktop = !mobile && !tablet
  const style = {
    '--left-panel-width': `${desktop && layout.leftCollapsed ? 0 : layout.leftWidth}px`,
    '--right-panel-width': `${desktop && layout.rightCollapsed ? 0 : layout.rightWidth}px`,
    '--left-resizer-width': desktop && !layout.leftCollapsed ? '0.5rem' : '0px',
    '--right-resizer-width': desktop && !layout.rightCollapsed ? '0.5rem' : '0px',
  } as CSSProperties

  return (
    <div
      className="workspace-shell"
      style={style}
      data-mobile-destination={mobileDestination}
      data-left-collapsed={desktop ? String(layout.leftCollapsed) : undefined}
      data-right-collapsed={desktop ? String(layout.rightCollapsed) : undefined}
    >
      {header && <div className="workspace-header">{header}</div>}
      <div className="workspace-command-bar">
        <button type="button" onClick={resetLayout}>Reset workspace layout</button>
      </div>
      <nav className="workspace-tablet-controls" aria-label="Tablet workspace panels">
        <button
          type="button"
          aria-haspopup="dialog"
          aria-expanded={tabletDrawer === 'character'}
          onClick={() => setTabletDrawer('character')}
        >
          Open character drawer
        </button>
        <button
          type="button"
          aria-haspopup="dialog"
          aria-expanded={tabletDrawer === 'tools'}
          onClick={() => setTabletDrawer('tools')}
        >
          Open tools drawer
        </button>
      </nav>
      <div className="workspace-body">
        {desktop && layout.leftCollapsed && (
          <button type="button" className="workspace-restore workspace-restore-left" onClick={() => setLeftCollapsed(false)}>
            Show character panel
          </button>
        )}
        {(mobile || (desktop && !layout.leftCollapsed)) && (
          <div className="workspace-panel workspace-left">
            {desktop && <button type="button" className="workspace-collapse" onClick={() => setLeftCollapsed(true)} aria-label="Collapse character panel">‹</button>}
            {left}
          </div>
        )}
        {desktop && !layout.leftCollapsed && (
          <div
            className="workspace-resizer workspace-resizer-left"
            role="separator"
            aria-label="Resize character panel"
            aria-orientation="vertical"
            aria-valuemin={LAYOUT_LIMITS.left[0]}
            aria-valuemax={LAYOUT_LIMITS.left[1]}
            aria-valuenow={layout.leftWidth}
            tabIndex={0}
            onPointerDown={(event) => startPointerResize('left', event)}
            onKeyDown={(event) => resizeByKeyboard('left', event)}
          />
        )}
        <div className="workspace-story">{story}</div>
        {desktop && !layout.rightCollapsed && (
          <div
            className="workspace-resizer workspace-resizer-right"
            role="separator"
            aria-label="Resize tools panel"
            aria-orientation="vertical"
            aria-valuemin={LAYOUT_LIMITS.right[0]}
            aria-valuemax={LAYOUT_LIMITS.right[1]}
            aria-valuenow={layout.rightWidth}
            tabIndex={0}
            onPointerDown={(event) => startPointerResize('right', event)}
            onKeyDown={(event) => resizeByKeyboard('right', event)}
          />
        )}
        {desktop && layout.rightCollapsed && (
          <button type="button" className="workspace-restore workspace-restore-right" onClick={() => setRightCollapsed(false)}>
            Show tools panel
          </button>
        )}
        {(mobile || (desktop && !layout.rightCollapsed)) && (
          <div className="workspace-panel workspace-right">
            {desktop && <button type="button" className="workspace-collapse" onClick={() => setRightCollapsed(true)} aria-label="Collapse tools panel">›</button>}
            {right}
          </div>
        )}
      </div>
      {tablet && (
        <>
          <Drawer open={tabletDrawer === 'character'} title="Character drawer" side="left" onClose={() => setTabletDrawer(null)}>
            <div className="workspace-tablet-drawer-content">{left}</div>
          </Drawer>
          <Drawer open={tabletDrawer === 'tools'} title="Tools drawer" side="right" onClose={() => setTabletDrawer(null)}>
            <div className="workspace-tablet-drawer-content">{right}</div>
          </Drawer>
        </>
      )}
      {mobileNav && <div className="workspace-mobile-nav">{mobileNav}</div>}
    </div>
  )
}
