import { useEffect, useState, useRef, useCallback } from 'react'
import type { CampaignMap, MapPin, MapToken, MapZone } from './api'
import { fetchMaps, fetchMapPins, fetchMapTokens, placeToken, moveToken, removeToken, fetchMapZones, createMapZone, patchMapZone, deleteMapZone, mapAssetURL } from './api'
import type { SessionNPC, Character } from './types'
import { isScopedEvent } from './wsEvents'
import { useToast } from './ui/ToastProvider'

function isMapPinAddedEvent(e: unknown): e is { type: string; payload: { map_id: number } } {
  return (
    typeof e === 'object' &&
    e !== null &&
    (e as Record<string, unknown>)['type'] === 'map_pin_added' &&
    typeof (e as Record<string, unknown>)['payload'] === 'object' &&
    (e as Record<string, { map_id: unknown }>)['payload']['map_id'] !== undefined
  )
}

interface MapPanelProps {
  campaignId: number | null
  lastEvent: unknown
  onActiveMapChange?: (mapId: number | null, imagePath: string | null) => void
  characters?: Character[]
  sessionNpcs?: SessionNPC[]
}

export function MapPanel({ campaignId, lastEvent, onActiveMapChange, characters, sessionNpcs }: MapPanelProps) {
  const toast = useToast()
  const [maps, setMaps] = useState<CampaignMap[]>([])
  const [activeMapIdx, setActiveMapIdx] = useState(0)
  const [pins, setPins] = useState<MapPin[]>([])
  const [selectedPin, setSelectedPin] = useState<MapPin | null>(null)
  const [tokens, setTokens] = useState<MapToken[]>([])
  const [dragging, setDragging] = useState<{ tokenId: number } | null>(null)
  const [hoveredToken, setHoveredToken] = useState<number | null>(null)
  const [showPalette, setShowPalette] = useState(false)
  const [zones, setZones] = useState<MapZone[]>([])
  const [zoneEditMode, setZoneEditMode] = useState(false)
  const [zoneDrawing, setZoneDrawing] = useState<{ startX: number; startY: number; endX: number; endY: number } | null>(null)
  const [pendingZoneName, setPendingZoneName] = useState<{ x: number; y: number; w: number; h: number } | null>(null)
  const [newZoneName, setNewZoneName] = useState('')
  const mapImgRef = useRef<HTMLImageElement>(null)
  const fxCanvasRef = useRef<HTMLCanvasElement>(null)

  function loadMaps(goToLast = false) {
    if (campaignId === null) return
    fetchMaps(campaignId).then((m) => {
      setMaps(m)
      if (goToLast && m.length > 0) {
        setActiveMapIdx(m.length - 1)
      }
    }).catch(() => setMaps([])) // Background load retries on campaign or map events.
  }

  const loadTokens = useCallback((mapId: number) => {
    fetchMapTokens(mapId).then(setTokens).catch(() => {}) // Background token refresh is event-driven best effort.
  }, [])

  useEffect(() => {
    setSelectedPin(null)
    loadMaps(true)
  }, [campaignId]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (isScopedEvent(lastEvent, 'map_created', 'campaign_id', campaignId)) {
      loadMaps(true)
    }
  }, [lastEvent, campaignId]) // eslint-disable-line react-hooks/exhaustive-deps

  const activeMap = maps[activeMapIdx] ?? null

  useEffect(() => {
    setZoneEditMode(false)
    setZoneDrawing(null)
    setPendingZoneName(null)
    if (!activeMap) {
      setPins([])
      setTokens([])
      setZones([])
      onActiveMapChange?.(null, null)
      return
    }
    fetchMapPins(activeMap.id).then(setPins).catch(() => setPins([]))
    loadTokens(activeMap.id)
    fetchMapZones(activeMap.id).then(setZones).catch(() => setZones([]))
    onActiveMapChange?.(activeMap.id, activeMap.image_path)
  }, [activeMap?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (isMapPinAddedEvent(lastEvent) && activeMap && (lastEvent as { payload: { map_id: number } }).payload.map_id === activeMap.id) {
      fetchMapPins(activeMap.id).then(setPins).catch(() => {})
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
        fetchMapZones(activeMap.id).then(setZones).catch(() => {})
      }
    }
  }, [lastEvent, activeMap?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const canvas = fxCanvasRef.current
    const img = mapImgRef.current
    if (!canvas || !img) return

    const ro = new ResizeObserver(() => {
      canvas.width = img.offsetWidth
      canvas.height = img.offsetHeight
    })
    ro.observe(img)
    canvas.width = img.offsetWidth
    canvas.height = img.offsetHeight
    return () => ro.disconnect()
  }, [maps[activeMapIdx]?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const ev = lastEvent as { type?: string; payload?: Record<string, unknown> } | null
    if (ev?.type !== 'map_fx' || !ev.payload) return
    const { map_id, effect, x, y, duration_ms } = ev.payload as {
      map_id: number; effect: string; x: number; y: number; duration_ms: number
    }
    const activeMap2 = maps[activeMapIdx] ?? null
    if (!activeMap2 || activeMap2.id !== map_id) return

    const canvas = fxCanvasRef.current
    if (!canvas) return
    const ctx2d: CanvasRenderingContext2D = canvas.getContext('2d')!
    if (!ctx2d) return

    type Particle = {
      x: number; y: number; vx: number; vy: number
      size: number; color: string; born: number; lifespan: number
    }

    const presets: Record<string, { colors: string[]; count: number; vxRange: [number,number]; vyRange: [number,number]; sizeRange: [number,number]; lifespan: number }> = {
      fire:      { colors: ['#e75f00','#ff8800','#ffdd00'], count: 40, vxRange: [-0.3,0.3], vyRange: [-0.8,0],   sizeRange: [4,8],  lifespan: 700  },
      frost:     { colors: ['#80c8ff','#c8e8ff','#ffffff'], count: 30, vxRange: [-0.4,0.4], vyRange: [-0.6,0.1], sizeRange: [3,6],  lifespan: 900  },
      lightning: { colors: ['#d0e8ff'],                     count: 20, vxRange: [-1.0,1.0], vyRange: [-1.2,1.2], sizeRange: [2,4],  lifespan: 300  },
      smoke:     { colors: ['#555555','#777777','#888888'], count: 25, vxRange: [-0.2,0.2], vyRange: [-0.3,0],   sizeRange: [6,12], lifespan: 1200 },
      blood:     { colors: ['#8b0000','#aa0000','#cc0000'], count: 35, vxRange: [-0.5,0.5], vyRange: [0,0.8],    sizeRange: [3,7],  lifespan: 800  },
      magic:     { colors: ['#8000ff','#cc00aa','#ff00cc'], count: 45, vxRange: [-0.5,0.5], vyRange: [-0.7,0.3], sizeRange: [3,7],  lifespan: 1000 },
    }

    const preset = presets[effect] ?? presets['magic']
    const originX = x * canvas.width
    const originY = y * canvas.height
    const rng = (lo: number, hi: number) => lo + Math.random() * (hi - lo)

    const particles: Particle[] = Array.from({ length: preset.count }, () => ({
      x: originX,
      y: originY,
      vx: rng(...preset.vxRange) * canvas.width * 0.004,
      vy: rng(...preset.vyRange) * canvas.height * 0.004,
      size: rng(...preset.sizeRange),
      color: preset.colors[Math.floor(Math.random() * preset.colors.length)],
      born: performance.now(),
      lifespan: preset.lifespan * rng(0.7, 1.3),
    }))

    let rafId: number
    function draw() {
      ctx2d.clearRect(0, 0, canvas!.width, canvas!.height)
      const now = performance.now()
      let alive = false
      for (const p of particles) {
        const age = now - p.born
        if (age >= p.lifespan) continue
        alive = true
        const alpha = 1 - age / p.lifespan
        p.x += p.vx
        p.y += p.vy
        ctx2d.globalAlpha = alpha
        ctx2d.fillStyle = p.color
        ctx2d.beginPath()
        ctx2d.arc(p.x, p.y, p.size * alpha, 0, Math.PI * 2)
        ctx2d.fill()
      }
      ctx2d.globalAlpha = 1
      if (alive) rafId = requestAnimationFrame(draw)
    }
    rafId = requestAnimationFrame(draw)

    const clearId = setTimeout(() => {
      cancelAnimationFrame(rafId)
      ctx2d.clearRect(0, 0, canvas.width, canvas.height)
    }, duration_ms + 200)

    return () => {
      cancelAnimationFrame(rafId)
      clearTimeout(clearId)
    }
  }, [lastEvent, maps, activeMapIdx])

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
    try {
      await createMapZone(activeMap.id, newZoneName.trim(), x, y, w, h)
      setPendingZoneName(null)
      setNewZoneName('')
      fetchMapZones(activeMap.id).then(setZones).catch(() => {})
    } catch (cause) {
      console.error(cause)
      toast.error('Could not create map zone.')
    }
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
      toast.error('Could not move map token.')
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
      toast.error('Could not place map token.')
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
            onMouseLeave={() => { setDragging(null); setZoneDrawing(null) }}
          >
            <img
              ref={mapImgRef}
              src={mapAssetURL(activeMap.id)}
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
                  fontSize: '12px',
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
                  setDragging({ tokenId: token.id })
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
                      try {
                        await removeToken(token.id)
                        if (activeMap) loadTokens(activeMap.id)
                      } catch (cause) {
                        console.error(cause)
                        toast.error('Could not remove map token.')
                      }
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
            <canvas
              ref={fxCanvasRef}
              className="map-fx-canvas"
              style={{ position: 'absolute', top: 0, left: 0, pointerEvents: 'none', zIndex: 20 }}
            />
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
                      try {
                        await patchMapZone(zone.id, { is_revealed: !zone.is_revealed })
                        if (activeMap) fetchMapZones(activeMap.id).then(setZones).catch(() => {})
                      } catch (cause) {
                        console.error(cause)
                        toast.error('Could not change zone visibility.')
                      }
                    }}
                  >
                    {zone.is_revealed ? 'Hide' : 'Reveal'}
                  </button>
                  <button onClick={async () => {
                    try {
                      await deleteMapZone(zone.id)
                      if (activeMap) fetchMapZones(activeMap.id).then(setZones).catch(() => {})
                    } catch (cause) {
                      console.error(cause)
                      toast.error('Could not delete map zone.')
                    }
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
