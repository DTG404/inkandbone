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
    const tokens = readFileSync(join(process.cwd(), 'src/styles/tokens.css'), 'utf8')
    const styles = ['base.css', 'narrative.css', 'panels.css']
      .map((file) => readFileSync(join(process.cwd(), 'src/styles', file), 'utf8'))
      .join('\n')

    for (const dependency of [
      '@fontsource-variable/cormorant-garamond',
      '@fontsource-variable/noto-sans',
      '@fontsource-variable/noto-sans-mono',
    ]) {
      expect(packageJSON.dependencies?.[dependency], dependency).toMatch(/^\d+\.\d+\.\d+$/)
      expect(appEntry).toContain(`import '${dependency}/index.css'`)
    }
    expect(appEntry).toMatch(
      /import ['"]@fontsource-variable\/cormorant-garamond\/index\.css['"]/,
    )
    expect(tokens).toMatch(/--serif:\s*'Cormorant Garamond Variable'/)
    expect(tokens).toMatch(/--sans:\s*'Noto Sans Variable'/)
    expect(tokens).toMatch(/--mono:\s*'Noto Sans Mono Variable'/)
    expect(styles).not.toContain('system-ui')
    expect(styles).not.toMatch(/font-family:\s*monospace/)
    expect(styles).toMatch(/button,\s*input,\s*select,\s*textarea\s*{[^}]*font:\s*inherit/s)
  })
})
