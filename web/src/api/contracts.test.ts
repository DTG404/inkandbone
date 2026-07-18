/// <reference types="node" />
import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const domainGoldens: Record<string, string> = {
  'automation.ts': 'd8cf06f4d0b7be3f49b308662df97a00b95f2ef2a09458161f87be296b3ec1af',
  'characters.ts': 'ec9b5c58075599c663ff847d055d3631da36bdcf015bd736262a45dd65e72061',
  'combat.ts': '1c5806f8277892a57b511fb71ea6af5759733e94bc445dce3e2f673b4ad999d6',
  'decks.ts': '5012ab850d609cf4c6ce8c7eaf449538074c3be39619960c7400b4e9f811fe61',
  'macros.ts': 'abe4056b659a993a8eddfcd43afdbaf5f6d18d7c87867715abe92db691503db1',
  'management.ts': '6fb9af9257cd401279cfe87dde1783f71d0aad33c22b27b0590939b5b1a9649f',
  'rules.ts': '1422b1e80675bd66418b3ae50b1af7cd35ebbcddb4d0e489696af17a8be054ff',
  'sessions.ts': '087c4c65b919686e8d754d27f77ff6b46815ceefd46858f9a2a7dc6bf311c32e',
  'transport.ts': 'e526bc66b5db1b062b906cad31b4bbb53b07bd829cf9a608db40e43c561f9053',
  'world.ts': '2bc868a6744d6c8a1725b4782e1ae98e4ba5bb60b10f6a719db67592accc0879',
  'worldCatalog.ts': '8b576bed5fb18c044839317c990ec20798a42b2375424ea54b5091ed66a38686',
}

const apiSource = (file: string) => readFileSync(join(process.cwd(), 'src', 'api', file), 'utf8')

describe('domain API transport characterization', () => {
  it('freezes every domain export method, URL, headers, body, decode, and error branch byte-for-byte', () => {
    for (const [file, golden] of Object.entries(domainGoldens)) {
      const digest = createHash('sha256').update(apiSource(file)).digest('hex')
      expect(digest, file).toBe(golden)
    }
  })

  it('keeps every HTTP domain function on the shared transport with an explicit non-2xx branch', () => {
    let functionCount = 0
    for (const file of Object.keys(domainGoldens).filter((name) => name !== 'transport.ts')) {
      const source = apiSource(file)
      const starts = [...source.matchAll(/^export async function\s+(\w+)/gm)]
      for (const [index, match] of starts.entries()) {
        functionCount++
        const start = match.index
        const end = starts[index + 1]?.index ?? source.length
        const body = source.slice(start, end)
        expect(body, `${file}:${match[1]} shared request`).toContain('request(')
        expect(body, `${file}:${match[1]} non-2xx`).toMatch(/if \(!\w+\.ok\)/)
      }
    }
    expect(functionCount).toBe(125)
  })

  it('keeps the compatibility entrypoint as re-exports only', () => {
    const source = readFileSync(join(process.cwd(), 'src', 'api.ts'), 'utf8')
    expect(source).not.toContain('fetch(')
    expect(source).not.toContain('request(')
    expect(source.match(/^export \* from /gm)).toHaveLength(10)
  })
})
