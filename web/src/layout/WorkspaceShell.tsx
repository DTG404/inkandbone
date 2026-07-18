import { useCallback, useEffect, useState, type CSSProperties, type KeyboardEvent, type PointerEvent as ReactPointerEvent, type ReactNode } from 'react'
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

function useTabletWorkspace(): boolean {
  const query = '(min-width: 600px) and (max-width: 899px)'
  const [tablet, setTablet] = useState(() => (
    typeof window.matchMedia === 'function' && window.matchMedia(query).matches
  ))

  useEffect(() => {
    if (typeof window.matchMedia !== 'function') return
    const media = window.matchMedia(query)
    const update = () => setTablet(media.matches)
    media.addEventListener('change', update)
    return () => media.removeEventListener('change', update)
  }, [])

  return tablet
}

export function WorkspaceShell({ header, left, story, right, mobileNav, mobileDestination = 'story' }: WorkspaceShellProps) {
  const tablet = useTabletWorkspace()
  const [tabletDrawer, setTabletDrawer] = useState<TabletDrawer>(null)
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

  const startPointerResize = useCallback((side: ResizeSide, event: ReactPointerEvent<HTMLDivElement>) => {
    event.preventDefault()
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
    }
    document.addEventListener('pointermove', move)
    document.addEventListener('pointerup', stop)
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

  const style = {
    '--left-panel-width': `${layout.leftWidth}px`,
    '--right-panel-width': `${layout.rightWidth}px`,
  } as CSSProperties

  return (
    <div className="workspace-shell" style={style} data-mobile-destination={mobileDestination}>
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
        {!tablet && (layout.leftCollapsed ? (
          <button type="button" className="workspace-restore workspace-restore-left" onClick={() => setLeftCollapsed(false)}>
            Show character panel
          </button>
        ) : (
          <div className="workspace-panel workspace-left">
            <button type="button" className="workspace-collapse" onClick={() => setLeftCollapsed(true)} aria-label="Collapse character panel">‹</button>
            {left}
          </div>
        ))}
        {!tablet && !layout.leftCollapsed && (
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
        {!tablet && !layout.rightCollapsed && (
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
        {!tablet && (layout.rightCollapsed ? (
          <button type="button" className="workspace-restore workspace-restore-right" onClick={() => setRightCollapsed(false)}>
            Show tools panel
          </button>
        ) : (
          <div className="workspace-panel workspace-right">
            <button type="button" className="workspace-collapse" onClick={() => setRightCollapsed(true)} aria-label="Collapse tools panel">›</button>
            {right}
          </div>
        ))}
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
