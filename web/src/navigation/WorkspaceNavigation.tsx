import { useEffect, useState } from 'react'
import { PANEL_DEFINITIONS, type MobileDestination, type PanelGroup, type PanelID } from './panelRegistry'

const GROUPS: readonly PanelGroup[] = ['Play', 'World', 'GM']
const DESTINATIONS: ReadonlyArray<{ id: MobileDestination; label: string }> = [
  { id: 'story', label: 'Story' },
  { id: 'character', label: 'Character' },
  { id: 'world', label: 'World' },
  { id: 'gm', label: 'GM' },
]

interface WorkspaceNavigationProps {
  activePanel: PanelID
  onPanelChange: (panel: PanelID) => void
  mobileDestination: MobileDestination
  onMobileDestinationChange: (destination: MobileDestination) => void
  mobile?: boolean
}

function useMobileWorkspace(override?: boolean): boolean {
  const query = '(max-width: 599px)'
  const [mobile, setMobile] = useState(() => override ?? (typeof window.matchMedia === 'function' && window.matchMedia(query).matches))
  useEffect(() => {
    if (override !== undefined || typeof window.matchMedia !== 'function') return
    const media = window.matchMedia(query)
    const change = () => setMobile(media.matches)
    media.addEventListener('change', change)
    return () => media.removeEventListener('change', change)
  }, [override])
  return override ?? mobile
}

export function WorkspaceNavigation({
  activePanel,
  onPanelChange,
  mobileDestination,
  onMobileDestinationChange,
  mobile: mobileOverride,
}: WorkspaceNavigationProps) {
  const mobile = useMobileWorkspace(mobileOverride)

  if (mobile) {
    const visibleGroups: readonly PanelGroup[] = mobileDestination === 'gm'
      ? ['GM']
      : mobileDestination === 'world' ? ['Play', 'World'] : []
    return (
      <>
        {visibleGroups.length > 0 && (
          <div className="workspace-mobile-secondary" aria-label={`${mobileDestination} panels`}>
            {visibleGroups.map((group) => {
              const entries = PANEL_DEFINITIONS.filter((entry) => entry.group === group)
              return (
                <div className="workspace-nav-group" key={group}>
                  <h3>{group}</h3>
                  <div aria-label={`${group} panels`} className="workspace-nav-buttons">
                    {entries.map((entry) => (
                      <button
                        type="button"
                        key={entry.id}
                        id={`workspace-panel-control-mobile-${entry.id}`}
                        aria-current={activePanel === entry.id ? 'page' : undefined}
                        aria-controls="workspace-active-panel"
                        className="workspace-nav-button"
                        onClick={() => onPanelChange(entry.id)}
                      >
                        {entry.label}
                      </button>
                    ))}
                  </div>
                </div>
              )
            })}
          </div>
        )}
        <nav className="workspace-bottom-nav" aria-label="Workspace destinations">
          {DESTINATIONS.map(({ id, label }) => (
            <button
              type="button"
              key={id}
              aria-current={mobileDestination === id ? 'page' : undefined}
              onClick={() => onMobileDestinationChange(id)}
            >
              {label}
            </button>
          ))}
        </nav>
      </>
    )
  }

  return (
    <nav className="workspace-desktop-nav" aria-label="Workspace panels">
      {GROUPS.map((group) => {
        const entries = PANEL_DEFINITIONS.filter((entry) => entry.group === group)
        return (
          <section className="workspace-nav-group" key={group}>
            <h3>{group}</h3>
            <div aria-label={`${group} panels`} className="workspace-nav-buttons">
              {entries.map((entry) => (
                <button
                  type="button"
                  key={entry.id}
                  id={`workspace-panel-control-${entry.id}`}
                  aria-current={activePanel === entry.id ? 'page' : undefined}
                  aria-controls="workspace-active-panel"
                  className="workspace-nav-button"
                  onClick={() => onPanelChange(entry.id)}
                >
                  {entry.label}
                </button>
              ))}
            </div>
          </section>
        )
      })}
    </nav>
  )
}
