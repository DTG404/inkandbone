/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const ACTION_FILES = [
  'ObjectivesPanel.tsx', 'CharacterSheetPanel.tsx', 'InventoryPanel.tsx', 'CombatPanel.tsx',
  'CalendarPanel.tsx', 'FactionsPanel.tsx', 'SecretsPanel.tsx', 'NPCRosterPanel.tsx',
  'NPCStatBlockPanel.tsx', 'MapPanel.tsx', 'AdventuresPanel.tsx', 'WorldNotesPanel.tsx',
  'DiceRoller.tsx', 'MacroBar.tsx', 'XPLogPanel.tsx', 'DecksPanel.tsx', 'CompendiumPanel.tsx',
  'JournalPanel.tsx', 'SessionView.tsx', 'ManagePanel.tsx', 'HandoutsPanel.tsx',
]

const source = (name: string) => readFileSync(join(process.cwd(), 'src', name), 'utf8')

describe('user action failure feedback source audit', () => {
  it('does not leave direct-action promise failures as catch(console.error)', () => {
    const violations = ACTION_FILES.filter((name) => /\.catch\(console\.error\)/.test(source(name)))
    expect(violations, `console-only promise catches remain in: ${violations.join(', ')}`).toEqual([])
  })

  it('does not leave catch blocks whose only observable action is console.error', () => {
    const violations = ACTION_FILES.filter((name) => (
      /catch\s*(?:\([^)]*\))?\s*\{\s*console\.error\([^\n]*\)\s*\}/s.test(source(name))
    ))
    expect(violations, `console-only catch blocks remain in: ${violations.join(', ')}`).toEqual([])
  })
})
