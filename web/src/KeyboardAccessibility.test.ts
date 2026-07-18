/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = (file: string) => readFileSync(join(process.cwd(), 'src', file), 'utf8')

describe('keyboard accessibility source gates', () => {
  it('contains no clickable div actions', () => {
    for (const file of ['ManagePanel.tsx', 'GMScreenPanel.tsx', 'SessionView.tsx', 'SecretsPanel.tsx', 'FactionsPanel.tsx', 'NPCStatBlockPanel.tsx', 'AdventuresPanel.tsx']) {
      expect(source(file).match(/<div\b[^>]*\bonClick=/gs) ?? [], file).toEqual([])
    }
  })

  it('uses the shared dialog primitive for every owned overlay', () => {
    expect(source('ManagePanel.tsx')).toContain('<Dialog')
    expect(source('GMScreenPanel.tsx')).toContain('<Dialog')
    expect(source('SessionView.tsx').match(/<Dialog/g)?.length ?? 0).toBeGreaterThanOrEqual(4)
  })

  it('does not remove focus outlines without replacement', () => {
    expect(source('App.css')).not.toMatch(/outline:\s*none/)
    expect(source('App.css')).toContain(':focus-visible')
  })
})
