import type { KeyboardEvent, ReactNode } from 'react'

export interface TabListProps {
  label: string
  children: ReactNode
  className?: string
}

export function TabList({ label, children, className = '' }: TabListProps) {
  return <div role="tablist" aria-label={label} className={className}>{children}</div>
}

export interface TabProps {
  id: string
  selected: boolean
  onSelect: (id: string) => void
  children: ReactNode
  className?: string
  tabStop?: boolean
  domIdPrefix?: string
  controlsId?: string | null
}

export function Tab({ id, selected, onSelect, children, className = '', tabStop, domIdPrefix = 'tab', controlsId }: TabProps) {
  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
    const tabList = event.currentTarget.closest('[role="tablist"]')
    const tabs = tabList ? Array.from(tabList.querySelectorAll<HTMLButtonElement>('[role="tab"]')) : []
    const currentIndex = tabs.indexOf(event.currentTarget)
    if (currentIndex < 0 || tabs.length === 0) return

    event.preventDefault()
    let nextIndex = currentIndex
    if (event.key === 'Home') nextIndex = 0
    if (event.key === 'End') nextIndex = tabs.length - 1
    if (event.key === 'ArrowRight') nextIndex = (currentIndex + 1) % tabs.length
    if (event.key === 'ArrowLeft') nextIndex = (currentIndex - 1 + tabs.length) % tabs.length
    tabs[nextIndex].focus()
    tabs[nextIndex].click()
  }

  return (
    <button
      type="button"
      id={`${domIdPrefix}-${id}`}
      role="tab"
      aria-selected={selected}
      aria-controls={controlsId === null ? undefined : (controlsId ?? `panel-${id}`)}
      tabIndex={(tabStop ?? selected) ? 0 : -1}
      className={`ui-tab ${className}`.trim()}
      onClick={() => onSelect(id)}
      onKeyDown={handleKeyDown}
    >
      {children}
    </button>
  )
}

export interface TabPanelProps {
  tabId: string
  active: boolean
  children: ReactNode
  className?: string
}

export function TabPanel({ tabId, active, children, className = '' }: TabPanelProps) {
  return (
    <div
      id={`panel-${tabId}`}
      role="tabpanel"
      aria-labelledby={`tab-${tabId}`}
      hidden={!active}
      className={className}
    >
      {children}
    </div>
  )
}
