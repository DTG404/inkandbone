import { useCallback, useEffect, useRef, useState } from 'react'

export interface WorkspaceLayout {
  leftWidth: number
  rightWidth: number
  leftCollapsed: boolean
  rightCollapsed: boolean
}

export const LAYOUT_STORAGE_KEY = 'inkandbone.workspace.v1'
export const DEFAULT_LAYOUT: WorkspaceLayout = {
  leftWidth: 420,
  rightWidth: 320,
  leftCollapsed: false,
  rightCollapsed: false,
}
export const LAYOUT_LIMITS = { left: [280, 620], right: [260, 520] } as const

function clamp(value: number, [minimum, maximum]: readonly [number, number]): number {
  return Math.min(maximum, Math.max(minimum, value))
}

function parseStoredLayout(): WorkspaceLayout {
  const stored = localStorage.getItem(LAYOUT_STORAGE_KEY)
  if (!stored) return DEFAULT_LAYOUT
  try {
    const value = JSON.parse(stored) as Partial<WorkspaceLayout>
    if (
      typeof value.leftWidth !== 'number' || !Number.isFinite(value.leftWidth) ||
      typeof value.rightWidth !== 'number' || !Number.isFinite(value.rightWidth) ||
      typeof value.leftCollapsed !== 'boolean' || typeof value.rightCollapsed !== 'boolean'
    ) throw new Error('invalid layout')
    return {
      leftWidth: clamp(value.leftWidth, LAYOUT_LIMITS.left),
      rightWidth: clamp(value.rightWidth, LAYOUT_LIMITS.right),
      leftCollapsed: value.leftCollapsed,
      rightCollapsed: value.rightCollapsed,
    }
  } catch {
    localStorage.removeItem(LAYOUT_STORAGE_KEY)
    return DEFAULT_LAYOUT
  }
}

export function useWorkspaceLayout() {
  const [layout, setLayout] = useState<WorkspaceLayout>(parseStoredLayout)
  const skipNextPersistence = useRef(false)

  useEffect(() => {
    if (skipNextPersistence.current) {
      skipNextPersistence.current = false
      return
    }
    localStorage.setItem(LAYOUT_STORAGE_KEY, JSON.stringify(layout))
  }, [layout])

  const setLeftWidth = useCallback((width: number) => {
    setLayout((current) => ({ ...current, leftWidth: clamp(width, LAYOUT_LIMITS.left) }))
  }, [])
  const setRightWidth = useCallback((width: number) => {
    setLayout((current) => ({ ...current, rightWidth: clamp(width, LAYOUT_LIMITS.right) }))
  }, [])
  const setLeftCollapsed = useCallback((collapsed: boolean) => {
    setLayout((current) => ({ ...current, leftCollapsed: collapsed }))
  }, [])
  const setRightCollapsed = useCallback((collapsed: boolean) => {
    setLayout((current) => ({ ...current, rightCollapsed: collapsed }))
  }, [])
  const resetLayout = useCallback(() => {
    skipNextPersistence.current = true
    localStorage.removeItem(LAYOUT_STORAGE_KEY)
    setLayout({ ...DEFAULT_LAYOUT })
  }, [])

  return {
    layout,
    setLeftWidth,
    setRightWidth,
    setLeftCollapsed,
    setRightCollapsed,
    resetLayout,
  }
}
