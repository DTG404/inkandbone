import { useEffect, useState } from 'react'
import { Tab, TabList } from '../ui/Tabs'
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
              const hasSelected = entries.some((entry) => entry.id === activePanel)
              return (
                <div className="workspace-nav-group" key={group}>
                  <h3>{group}</h3>
                  <TabList label={`${group} panels`} className="workspace-nav-tabs">
                    {entries.map((entry, index) => (
                      <Tab
                        key={entry.id}
                        id={entry.id}
                        domIdPrefix="mobile-tab"
                        controlsId="workspace-active-panel"
                        selected={activePanel === entry.id}
                        tabStop={activePanel === entry.id || (!hasSelected && index === 0)}
                        onSelect={(id) => onPanelChange(id as PanelID)}
                      >
                        {entry.label}
                      </Tab>
                    ))}
                  </TabList>
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
        const hasSelected = entries.some((entry) => entry.id === activePanel)
        return (
          <section className="workspace-nav-group" key={group}>
            <h3>{group}</h3>
            <TabList label={`${group} panels`} className="workspace-nav-tabs">
              {entries.map((entry, index) => (
                <Tab
                  key={entry.id}
                  id={entry.id}
                  domIdPrefix="desktop-tab"
                  controlsId="workspace-active-panel"
                  selected={activePanel === entry.id}
                  tabStop={activePanel === entry.id || (!hasSelected && index === 0)}
                  onSelect={(id) => onPanelChange(id as PanelID)}
                >
                  {entry.label}
                </Tab>
              ))}
            </TabList>
          </section>
        )
      })}
    </nav>
  )
}
