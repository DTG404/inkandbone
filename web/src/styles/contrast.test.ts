/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

function channel(value: number): number {
  const normalized = value / 255
  return normalized <= 0.04045 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4
}

function luminance(hex: string): number {
  const value = hex.replace('#', '')
  const [red, green, blue] = [0, 2, 4].map((offset) => channel(Number.parseInt(value.slice(offset, offset + 2), 16)))
  return 0.2126 * red + 0.7152 * green + 0.0722 * blue
}

function contrast(foreground: string, background: string): number {
  const lighter = Math.max(luminance(foreground), luminance(background))
  const darker = Math.min(luminance(foreground), luminance(background))
  return (lighter + 0.05) / (darker + 0.05)
}

function tokens(block: string): Record<string, string> {
  return Object.fromEntries(Array.from(block.matchAll(/--([\w-]+):\s*(#[0-9a-f]{6})/gi), ([, name, value]) => [name, value]))
}

const source = readFileSync(join(process.cwd(), 'src/styles/tokens.css'), 'utf8')
const dark = tokens(source.match(/:root\s*{([^}]*)}/s)?.[1] ?? '')
const light = tokens(source.match(/\[data-theme=["']parchment["']\]\s*{([^}]*)}/s)?.[1] ?? '')

describe('theme contrast tokens', () => {
  const cases: Array<[string, Record<string, string>, string, string, number]> = [
    ['dark body text', dark, 'text', 'bg', 4.5],
    ['dark muted labels', dark, 'text-dim', 'bg', 4.5],
    ['dark active tabs', dark, 'gold', 'surface2', 4.5],
    ['dark meaningful borders', dark, 'border', 'surface2', 3],
    ['dark errors', dark, 'accent', 'bg', 4.5],
    ['dark health', dark, 'health', 'bg', 4.5],
    ['parchment body text', light, 'text', 'bg', 4.5],
    ['parchment muted labels', light, 'text-dim', 'bg', 4.5],
    ['parchment active tabs', light, 'gold', 'surface2', 4.5],
    ['parchment meaningful borders', light, 'border', 'surface2', 3],
    ['parchment errors', light, 'accent', 'bg', 4.5],
    ['parchment health', light, 'health', 'bg', 4.5],
  ]

  it.each(cases)('%s meets WCAG contrast', (_name, palette, foreground, background, minimum) => {
    expect(palette[foreground], `${foreground} token`).toMatch(/^#[0-9a-f]{6}$/i)
    expect(palette[background], `${background} token`).toMatch(/^#[0-9a-f]{6}$/i)
    expect(contrast(palette[foreground], palette[background])).toBeGreaterThanOrEqual(minimum)
  })
})
