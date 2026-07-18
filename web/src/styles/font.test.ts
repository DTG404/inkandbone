/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('theme typography', () => {
  it('bundles Cormorant Garamond instead of relying on the host fallback', () => {
    const packageJSON = JSON.parse(readFileSync(join(process.cwd(), 'package.json'), 'utf8')) as {
      dependencies?: Record<string, string>
    }
    const appEntry = readFileSync(join(process.cwd(), 'src/App.tsx'), 'utf8')
    const version = packageJSON.dependencies?.['@fontsource-variable/cormorant-garamond']

    expect(version).toMatch(/^\d+\.\d+\.\d+$/)
    expect(appEntry).toMatch(
      /import ['"]@fontsource-variable\/cormorant-garamond\/index\.css['"]/,
    )
  })
})
