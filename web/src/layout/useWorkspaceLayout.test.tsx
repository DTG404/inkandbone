import { act, cleanup, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { DEFAULT_LAYOUT, LAYOUT_STORAGE_KEY, useWorkspaceLayout } from './useWorkspaceLayout'

describe('useWorkspaceLayout', () => {
  beforeEach(() => localStorage.clear())
  afterEach(cleanup)

  it('uses defaults and persists clamped widths and collapsed state', () => {
    const { result } = renderHook(() => useWorkspaceLayout())
    expect(result.current.layout).toEqual(DEFAULT_LAYOUT)

    act(() => {
      result.current.setLeftWidth(900)
      result.current.setRightWidth(100)
      result.current.setLeftCollapsed(true)
    })

    expect(result.current.layout).toEqual({
      leftWidth: 620,
      rightWidth: 260,
      leftCollapsed: true,
      rightCollapsed: false,
    })
    expect(JSON.parse(localStorage.getItem(LAYOUT_STORAGE_KEY)!)).toEqual(result.current.layout)
  })

  it('loads a clamped saved layout and recovers from corrupt storage', () => {
    localStorage.setItem(LAYOUT_STORAGE_KEY, JSON.stringify({
      leftWidth: 100,
      rightWidth: 900,
      leftCollapsed: true,
      rightCollapsed: true,
    }))
    const saved = renderHook(() => useWorkspaceLayout())
    expect(saved.result.current.layout).toEqual({ leftWidth: 280, rightWidth: 520, leftCollapsed: true, rightCollapsed: true })
    saved.unmount()

    localStorage.setItem(LAYOUT_STORAGE_KEY, '{broken')
    const corrupt = renderHook(() => useWorkspaceLayout())
    expect(corrupt.result.current.layout).toEqual(DEFAULT_LAYOUT)
  })

  it('resets defaults and removes the versioned storage entry', () => {
    const { result } = renderHook(() => useWorkspaceLayout())
    act(() => result.current.setRightWidth(480))
    expect(localStorage.getItem(LAYOUT_STORAGE_KEY)).not.toBeNull()

    act(() => result.current.resetLayout())

    expect(result.current.layout).toEqual(DEFAULT_LAYOUT)
    expect(localStorage.getItem(LAYOUT_STORAGE_KEY)).toBeNull()
  })

  it('persists the first resize after resetting an already-default layout', () => {
    const { result } = renderHook(() => useWorkspaceLayout())

    act(() => result.current.resetLayout())
    act(() => result.current.setLeftWidth(500))

    expect(JSON.parse(localStorage.getItem(LAYOUT_STORAGE_KEY)!)).toMatchObject({ leftWidth: 500 })
  })
})
