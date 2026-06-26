import { useEffect, useState, useRef, useCallback } from 'react'
import type { CampaignMap, MapPin, MapToken, MapZone } from './api'
import { fetchMaps, fetchMapPins, fetchMapTokens, placeToken, moveToken, removeToken, fetchMapZones, createMapZone, patchMapZone, deleteMapZone } from './api'
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
  const [zones, setZones] = useState<MapZone[]>([])
  const [zoneEditMode, setZoneEditMode] = useState(false)
  const [zoneDrawing, setZoneDrawing] = useState<{ startX: number; startY: number; endX: number; endY: number } | null>(null)
  const [pendingZoneName, setPendingZoneName] = useState<{ x: number; y: number; w: number; h: number } | null>(null)
  const [newZoneName, setNewZoneName] = useState('')
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
      setZones([])
      onActiveMapChange?.(null, null)
      return
    }
    fetchMapPins(activeMap.id).then(setPins).catch(console.error)
    loadTokens(activeMap.id)
    fetchMapZones(activeMap.id).then(setZones).catch(console.error)
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
    if (e.type === 'zone_revealed') {
      const p = e.payload as { map_id: number }
      if (p && p.map_id === activeMap.id) {
        fetchMapZones(activeMap.id).then(setZones).catch(console.error)
      }
    }
  }, [lastEvent, activeMap?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  function handleZoneMouseDown(e: React.MouseEvent<HTMLImageElement>) {
    if (!zoneEditMode || !mapImgRef.current) return
    const rect = mapImgRef.current.getBoundingClientRect()
    const sx = (e.clientX - rect.left) / rect.width
    const sy = (e.clientY - rect.top) / rect.height
    setZoneDrawing({ startX: sx, startY: sy, endX: sx, endY: sy })
  }

  function handleZoneMouseMove(e: React.MouseEvent<HTMLDivElement>) {
    if (!zoneEditMode || !zoneDrawing || !mapImgRef.current) return
    const rect = mapImgRef.current.getBoundingClientRect()
    const ex = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width))
    const ey = Math.min(1, Math.max(0, (e.clientY - rect.top) / rect.height))
    setZoneDrawing(prev => prev ? { ...prev, endX: ex, endY: ey } : null)
  }

  function handleZoneMouseUp() {
    if (!zoneEditMode || !zoneDrawing) return
    const x = Math.min(zoneDrawing.startX, zoneDrawing.endX)
    const y = Math.min(zoneDrawing.startY, zoneDrawing.endY)
    const w = Math.abs(zoneDrawing.endX - zoneDrawing.startX)
    const h = Math.abs(zoneDrawing.endY - zoneDrawing.startY)
    if (w < 0.02 || h < 0.02) { setZoneDrawing(null); return }
    setZoneDrawing(null)
    setPendingZoneName({ x, y, w, h })
    setNewZoneName('')
  }

  async function handleCreateZone() {
    if (!pendingZoneName || !newZoneName.trim() || !activeMap) return
    const { x, y, w, h } = pendingZoneName
    await createMapZone(activeMap.id, newZoneName.trim(), x, y, w, h)
    setPendingZoneName(null)
    setNewZoneName('')
    fetchMapZones(activeMap.id).then(setZones).catch(console.error)
  }

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
        <button
          className={`zone-mode-btn${zoneEditMode ? ' active' : ''}`}
          onClick={() => setZoneEditMode(!zoneEditMode)}
          title="Toggle zone editor"
        >
          ⬜ Zones
        </button>
      )}
      {activeMap && (
        <>
          <div
            className="map-scroll"
            style={{ position: 'relative', flex: 1, userSelect: dragging ? 'none' : 'auto' }}
            onMouseMove={(e) => { handleMapMouseMove(e); handleZoneMouseMove(e) }}
            onMouseUp={(e) => { handleMapMouseUp(e); handleZoneMouseUp() }}
          >
            <img
              ref={mapImgRef}
              src={`/api/files/${activeMap.image_path}`}
              alt={activeMap.name}
              style={{ width: '100%', display: 'block', minWidth: '400px' }}
              onDragOver={(e) => e.preventDefault()}
              onDrop={handleMapDrop}
              onMouseDown={(e) => { if (zoneEditMode) handleZoneMouseDown(e) }}
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
            {/* Fog layer — unrevealed zones */}
            {zones.map((zone) => (
              <div
                key={zone.id}
                style={{
                  position: 'absolute',
                  left: `${zone.x * 100}%`,
                  top: `${zone.y * 100}%`,
                  width: `${zone.width * 100}%`,
                  height: `${zone.height * 100}%`,
                  background: zoneEditMode
                    ? 'rgba(201,168,76,0.2)'
                    : zone.is_revealed ? 'transparent' : 'rgba(0,0,0,0.75)',
                  border: zoneEditMode ? '2px dashed #c9a84c' : 'none',
                  transition: 'background 0.6s ease-out',
                  pointerEvents: zoneEditMode ? 'auto' : 'none',
                  zIndex: zone.is_revealed ? 0 : 5,
                  boxSizing: 'border-box',
                }}
                title={zoneEditMode ? zone.name : undefined}
              />
            ))}
            {/* Zone draw preview */}
            {zoneEditMode && zoneDrawing && (
              <div
                style={{
                  position: 'absolute',
                  left: `${Math.min(zoneDrawing.startX, zoneDrawing.endX) * 100}%`,
                  top: `${Math.min(zoneDrawing.startY, zoneDrawing.endY) * 100}%`,
                  width: `${Math.abs(zoneDrawing.endX - zoneDrawing.startX) * 100}%`,
                  height: `${Math.abs(zoneDrawing.endY - zoneDrawing.startY) * 100}%`,
                  border: '2px dashed #fff',
                  background: 'rgba(255,255,255,0.1)',
                  pointerEvents: 'none',
                  zIndex: 20,
                }}
              />
            )}
            {/* Pending zone name prompt */}
            {pendingZoneName && (
              <div className="zone-name-prompt">
                <input
                  autoFocus
                  placeholder="Zone name"
                  value={newZoneName}
                  onChange={e => setNewZoneName(e.target.value)}
                  onKeyDown={e => { if (e.key === 'Enter') handleCreateZone() }}
                />
                <button onClick={handleCreateZone}>Add</button>
                <button onClick={() => setPendingZoneName(null)}>Cancel</button>
              </div>
            )}
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
          {zoneEditMode && zones.length > 0 && (
            <div className="zone-list-panel">
              <strong>Zones</strong>
              {zones.map((zone) => (
                <div key={zone.id} className="zone-list-row">
                  <span>{zone.name}</span>
                  <button
                    onClick={async () => {
                      await patchMapZone(zone.id, { is_revealed: !zone.is_revealed })
                      if (activeMap) fetchMapZones(activeMap.id).then(setZones).catch(console.error)
                    }}
                  >
                    {zone.is_revealed ? 'Hide' : 'Reveal'}
                  </button>
                  <button onClick={async () => {
                    await deleteMapZone(zone.id)
                    if (activeMap) fetchMapZones(activeMap.id).then(setZones).catch(console.error)
                  }}>
                    ×
                  </button>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  )
}
