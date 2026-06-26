import { useEffect, useState, useRef, useCallback } from 'react'
import type { CampaignMap, MapPin, MapToken } from './api'
import { fetchMaps, fetchMapPins, fetchMapTokens, placeToken, moveToken, removeToken } from './api'
import type { SessionNPC, Character } from './types'

function isMapPinAddedEvent(e: unknown): e is { type: string; payload: { map_id: number } } {
  return (
    typeof e === 'object' &&
    e !== null &&
    (e as Record<string, unknown>)['type'] === 'map_pin_added' &&
    typeof (e as Record<string, unknown>)['payload'] === 'object' &&
    (e as Record<string, { map_id: unknown }>)['payload']['map_id'] !== undefined
  )
}

function isMapCreatedEvent(e: unknown): boolean {
  return typeof e === 'object' && e !== null && (e as Record<string, unknown>)['type'] === 'map_created'
}

interface MapPanelProps {
  campaignId: number | null
  lastEvent: unknown
  onActiveMapChange?: (mapId: number | null, imagePath: string | null) => void
  characters?: Character[]
  sessionNpcs?: SessionNPC[]
}

export function MapPanel({ campaignId, lastEvent, onActiveMapChange, characters, sessionNpcs }: MapPanelProps) {
  const [maps, setMaps] = useState<CampaignMap[]>([])
  const [activeMapIdx, setActiveMapIdx] = useState(0)
  const [pins, setPins] = useState<MapPin[]>([])
  const [selectedPin, setSelectedPin] = useState<MapPin | null>(null)
  const [tokens, setTokens] = useState<MapToken[]>([])
  const [dragging, setDragging] = useState<{ tokenId: number; offsetX: number; offsetY: number } | null>(null)
  const [hoveredToken, setHoveredToken] = useState<number | null>(null)
  const [showPalette, setShowPalette] = useState(false)
  const mapImgRef = useRef<HTMLImageElement>(null)

  function loadMaps(goToLast = false) {
    if (campaignId === null) return
    fetchMaps(campaignId).then((m) => {
      setMaps(m)
      if (goToLast && m.length > 0) {
        setActiveMapIdx(m.length - 1)
      }
    }).catch(console.error)
  }

  const loadTokens = useCallback((mapId: number) => {
    fetchMapTokens(mapId).then(setTokens).catch(console.error)
  }, [])

  useEffect(() => {
    setSelectedPin(null)
    loadMaps(true)
  }, [campaignId]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (isMapCreatedEvent(lastEvent)) {
      loadMaps(true)
    }
  }, [lastEvent]) // eslint-disable-line react-hooks/exhaustive-deps

  const activeMap = maps[activeMapIdx] ?? null

  useEffect(() => {
    if (!activeMap) {
      setPins([])
      setTokens([])
      onActiveMapChange?.(null, null)
      return
    }
    fetchMapPins(activeMap.id).then(setPins).catch(console.error)
    loadTokens(activeMap.id)
    onActiveMapChange?.(activeMap.id, activeMap.image_path)
  }, [activeMap?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (isMapPinAddedEvent(lastEvent) && activeMap && (lastEvent as { payload: { map_id: number } }).payload.map_id === activeMap.id) {
      fetchMapPins(activeMap.id).then(setPins).catch(console.error)
    }
  }, [lastEvent, activeMap?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!activeMap) return
    const e = lastEvent as { type?: string; payload?: Record<string, unknown> } | null
    if (!e) return
    if (e.type === 'token_placed' || e.type === 'token_moved' || e.type === 'token_removed') {
      const payload = e.payload
      if (payload && (payload['map_id'] as number) === activeMap.id) {
        loadTokens(activeMap.id)
      }
    }
  }, [lastEvent, activeMap?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  function handleMapMouseMove(e: React.MouseEvent<HTMLDivElement>) {
    if (!dragging || !mapImgRef.current) return
    const rect = mapImgRef.current.getBoundingClientRect()
    const x = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width))
    const y = Math.min(1, Math.max(0, (e.clientY - rect.top) / rect.height))
    setTokens(prev => prev.map(t => t.id === dragging.tokenId ? { ...t, x, y } : t))
  }

  async function handleMapMouseUp(e: React.MouseEvent<HTMLDivElement>) {
    if (!dragging || !mapImgRef.current) return
    const rect = mapImgRef.current.getBoundingClientRect()
    const x = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width))
    const y = Math.min(1, Math.max(0, (e.clientY - rect.top) / rect.height))
    const id = dragging.tokenId
    setDragging(null)
    try {
      await moveToken(id, x, y)
    } catch (err) {
      console.error(err)
      if (activeMap) loadTokens(activeMap.id)
    }
  }

  async function handleMapDrop(e: React.DragEvent<HTMLImageElement>) {
    e.preventDefault()
    if (!activeMap || !mapImgRef.current) return
    const rect = mapImgRef.current.getBoundingClientRect()
    const x = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width))
    const y = Math.min(1, Math.max(0, (e.clientY - rect.top) / rect.height))
    const data = e.dataTransfer.getData('text/plain')
    if (!data) return
    const { entityType, entityId } = JSON.parse(data) as { entityType: string; entityId: number }
    try {
      await placeToken(activeMap.id, entityType, entityId, x, y)
      loadTokens(activeMap.id)
    } catch (err) {
      console.error(err)
    }
  }

  if (campaignId === null) return null

  if (maps.length === 0) {
    return <p>No map uploaded.</p>
  }

  return (
    <div style={{ position: 'relative', height: '100%', display: 'flex', flexDirection: 'column' }}>
      {maps.length > 1 && (
        <div className="map-tab-bar">
          {maps.map((m, i) => (
            <button
              key={m.id}
              className={`map-tab-btn${i === activeMapIdx ? ' active' : ''}`}
              onClick={() => { setActiveMapIdx(i); setSelectedPin(null) }}
            >
              {m.name}
            </button>
          ))}
        </div>
      )}
      {activeMap && (
        <>
          <div
            className="map-scroll"
            style={{ position: 'relative', flex: 1, userSelect: dragging ? 'none' : 'auto' }}
            onMouseMove={handleMapMouseMove}
            onMouseUp={handleMapMouseUp}
          >
            <img
              ref={mapImgRef}
              src={`/api/files/${activeMap.image_path}`}
              alt={activeMap.name}
              style={{ width: '100%', display: 'block', minWidth: '400px' }}
              onDragOver={(e) => e.preventDefault()}
              onDrop={handleMapDrop}
            />
            {pins.map((pin) => (
              <button
                key={pin.id}
                className="map-pin-btn"
                style={{
                  position: 'absolute',
                  left: `${pin.x * 100}%`,
                  top: `${pin.y * 100}%`,
                  transform: 'translate(-50%,-50%)',
                  background: pin.color || 'var(--gold)',
                  color: '#000',
                  border: 'none',
                  borderRadius: '50%',
                  width: '18px',
                  height: '18px',
                  fontSize: '9px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontWeight: 700,
                }}
                title={pin.note || pin.label}
                onClick={() => setSelectedPin(selectedPin?.id === pin.id ? null : pin)}
              >
                ✦
              </button>
            ))}
            {tokens.map((token) => (
              <div
                key={token.id}
                style={{
                  position: 'absolute',
                  left: `${token.x * 100}%`,
                  top: `${token.y * 100}%`,
                  transform: 'translate(-50%,-50%)',
                  width: 32,
                  height: 32,
                  borderRadius: '50%',
                  background: '#1a1710',
                  border: `2px solid ${token.entity_type === 'character' ? '#c9a84c' : '#c0392b'}`,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  color: '#fff',
                  fontSize: 13,
                  fontWeight: 700,
                  cursor: 'grab',
                  zIndex: 10,
                }}
                title={token.name}
                onMouseDown={(e) => {
                  e.preventDefault()
                  setDragging({ tokenId: token.id, offsetX: 0, offsetY: 0 })
                }}
                onMouseEnter={() => setHoveredToken(token.id)}
                onMouseLeave={() => setHoveredToken(null)}
              >
                {token.name ? token.name[0].toUpperCase() : '?'}
                {hoveredToken === token.id && (
                  <button
                    style={{
                      position: 'absolute',
                      top: -6,
                      right: -6,
                      width: 16,
                      height: 16,
                      borderRadius: '50%',
                      background: '#c0392b',
                      border: 'none',
                      color: '#fff',
                      fontSize: 10,
                      cursor: 'pointer',
                      lineHeight: 1,
                      padding: 0,
                    }}
                    onMouseDown={(e) => e.stopPropagation()}
                    onClick={async (e) => {
                      e.stopPropagation()
                      await removeToken(token.id)
                      if (activeMap) loadTokens(activeMap.id)
                    }}
                  >
                    ×
                  </button>
                )}
              </div>
            ))}
            {selectedPin && (
              <div className="map-pin-tooltip">
                <strong>{selectedPin.label}</strong>
                {selectedPin.note && <p>{selectedPin.note}</p>}
                <button className="map-pin-tooltip-close" onClick={() => setSelectedPin(null)}>×</button>
              </div>
            )}
          </div>
          <div className="token-palette-section">
            <button className="token-palette-toggle" onClick={() => setShowPalette(!showPalette)}>
              🎭 Tokens {showPalette ? '▲' : '▼'}
            </button>
            {showPalette && (
              <div className="token-palette">
                {(characters ?? []).map((c) => {
                  const placed = tokens.some(t => t.entity_type === 'character' && t.entity_id === c.id)
                  return (
                    <div
                      key={`char-${c.id}`}
                      className={`token-chip${placed ? ' token-chip-placed' : ''}`}
                      draggable={!placed}
                      onDragStart={(e) => {
                        e.dataTransfer.setData('text/plain', JSON.stringify({ entityType: 'character', entityId: c.id }))
                      }}
                    >
                      {c.name} <span className="token-type-badge">PC</span>
                    </div>
                  )
                })}
                {(sessionNpcs ?? []).map((n) => {
                  const placed = tokens.some(t => t.entity_type === 'npc' && t.entity_id === n.id)
                  return (
                    <div
                      key={`npc-${n.id}`}
                      className={`token-chip${placed ? ' token-chip-placed' : ''}`}
                      draggable={!placed}
                      onDragStart={(e) => {
                        e.dataTransfer.setData('text/plain', JSON.stringify({ entityType: 'npc', entityId: n.id }))
                      }}
                    >
                      {n.name} <span className="token-type-badge">NPC</span>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}
