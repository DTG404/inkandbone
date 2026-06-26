import { useState, useEffect, useRef, useCallback } from 'react'
import { searchRulebook } from './api'
import type { RulebookResult } from './api'

interface Props {
  rulesetId: number
}

export function CompendiumPanel({ rulesetId }: Props) {
  const [query, setQuery] = useState(() => sessionStorage.getItem('compendium-last-query') ?? '')
  const [results, setResults] = useState<RulebookResult[]>([])
  const [mode, setMode] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [searched, setSearched] = useState(false)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const doSearch = useCallback(async (q: string) => {
    if (!q.trim()) {
      setResults([])
      setMode(null)
      setSearched(false)
      return
    }
    setLoading(true)
    try {
      const data = await searchRulebook(rulesetId, q)
      setResults(data.results)
      setMode(data.mode)
      setSearched(true)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }, [rulesetId])

  useEffect(() => {
    sessionStorage.setItem('compendium-last-query', query)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => doSearch(query), 400)
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current) }
  }, [query, doSearch])

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      doSearch(query)
    }
  }

  return (
    <div className="compendium-panel">
      <input
        className="compendium-search"
        placeholder="Search rulebook…"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={handleKeyDown}
      />
      {mode === 'keyword' && (
        <p className="compendium-fallback-banner">Semantic search unavailable — showing keyword results</p>
      )}
      {loading && <p className="panel-loading">Searching…</p>}
      {!loading && searched && results.length === 0 && (
        <p className="panel-empty">No results found.</p>
      )}
      {!loading && results.map((r, i) => (
        <div key={i} className="compendium-result">
          {r.heading && <strong className="compendium-heading">{r.heading}</strong>}
          <p className="compendium-content">{r.content}</p>
          {r.source && <span className="compendium-source">{r.source}</span>}
        </div>
      ))}
    </div>
  )
}
