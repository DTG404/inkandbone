import { useState, useEffect, useRef, type ChangeEvent } from 'react'
import { fetchRuleset, patchCharacter, uploadPortrait } from './api'
import type { Ruleset } from './api'
import type { Character } from './types'

interface SchemaField {
  key: string
  label: string
  type: 'text' | 'number' | 'textarea'
}

interface CharacterSheetPanelProps {
  character: Character | null
  rulesetId: number | null
  lastEvent: unknown
  afterTracks?: React.ReactNode
}

interface CharacterUpdatedPayload {
  id: number
  data_json?: string
  portrait_path?: string
}

interface CharacterUpdatedEvent {
  type: 'character_updated'
  payload: CharacterUpdatedPayload
}

function isCharacterUpdatedEvent(ev: unknown): ev is CharacterUpdatedEvent {
  if (typeof ev !== 'object' || ev === null) return false
  const e = ev as Record<string, unknown>
  if (e['type'] !== 'character_updated') return false
  const p = e['payload']
  if (typeof p !== 'object' || p === null) return false
  return typeof (p as Record<string, unknown>)['id'] === 'number'
}

const ATTRIBUTE_KEYS = new Set(['edge', 'heart', 'iron', 'shadow', 'wits'])
const TRACK_KEYS     = new Set(['health', 'spirit', 'supply', 'momentum'])

function AttributePips({ value }: { value: number }) {
  return (
    <div className="attr-pips">
      {[1, 2, 3].map((i) => (
        <div key={i} className={`pip ${i <= value ? 'filled' : 'empty'}`} />
      ))}
    </div>
  )
}

function TrackBar({ fieldKey, value }: { fieldKey: string; value: number }) {
  const isMomentum = fieldKey === 'momentum'
  const max = isMomentum ? 10 : 5
  const filled = Math.max(0, Math.min(max, value))
  const colorClass = isMomentum ? 'filled-momentum' : 'filled-health'
  const displayValue = isMomentum ? value : `${value}/${max}`

  return (
    <div className="track-row">
      <div className="track-header">
        <span className="track-label">{fieldKey}</span>
        <span className="track-value">{displayValue}</span>
      </div>
      <div className="track-segments">
        {Array.from({ length: max }, (_, i) => (
          <div
            key={i}
            className={`track-seg ${i < filled ? colorClass : 'empty-seg'}`}
          />
        ))}
      </div>
    </div>
  )
}

// VtM constants
const VTM_PHYSICAL_ATTRS = ['strength', 'dexterity', 'stamina']
const VTM_SOCIAL_ATTRS = ['charisma', 'manipulation', 'composure']
const VTM_MENTAL_ATTRS = ['intelligence', 'wits', 'resolve']
const VTM_PHYSICAL_SKILLS = ['athletics', 'brawl', 'craft', 'drive', 'firearms', 'larceny', 'melee', 'stealth', 'survival']
const VTM_SOCIAL_SKILLS = ['animal_ken', 'etiquette', 'insight', 'intimidation', 'leadership', 'performance', 'persuasion', 'streetwise', 'subterfuge']
const VTM_MENTAL_SKILLS = ['academics', 'awareness', 'finance', 'investigation', 'medicine', 'occult', 'politics', 'technology']
const VTM_DISCIPLINES = ['animalism', 'auspex', 'blood_sorcery', 'celerity', 'dominate', 'fortitude', 'obfuscate', 'oblivion', 'potence', 'presence', 'protean']

function PipRow({ label, value, max, onChange, color = 'gold' }: {
  label: string; value: number; max: number; onChange?: (v: number) => void; color?: string
}) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '0.4rem', marginBottom: '3px' }}>
      <span style={{ fontSize: '10px', textTransform: 'uppercase', letterSpacing: '1px', color: 'var(--gold-dim)', width: '72px', flexShrink: 0 }}>{label}</span>
      <div style={{ display: 'flex', gap: '3px' }}>
        {Array.from({ length: max }, (_, i) => (
          <div
            key={i}
            onClick={() => onChange?.(i < value ? i : i + 1)}
            style={{
              width: 12, height: 12, borderRadius: '50%',
              background: i < value ? (color === 'red' ? '#c0392b' : 'var(--gold)') : 'transparent',
              border: `1px solid ${color === 'red' ? '#c0392b' : 'var(--gold-dim)'}`,
              cursor: onChange ? 'pointer' : 'default',
            }}
          />
        ))}
      </div>
    </div>
  )
}

function DamageTrack({ label, max, superficial, aggravated, onClickBox }: {
  label: string; max: number; superficial: number; aggravated: number
  onClickBox?: (index: number) => void
}) {
  return (
    <div style={{ marginBottom: '6px' }}>
      <span style={{ fontSize: '10px', textTransform: 'uppercase', letterSpacing: '1px', color: 'var(--gold-dim)' }}>{label}</span>
      <div style={{ display: 'flex', gap: '3px', marginTop: '3px' }}>
        {Array.from({ length: max }, (_, i) => {
          const fromRight = max - 1 - i
          const isAgg = fromRight < aggravated
          const isSup = !isAgg && fromRight < aggravated + superficial
          return (
            <div
              key={i}
              onClick={() => onClickBox?.(i)}
              style={{
                width: 14, height: 14, border: '1px solid var(--gold-dim)',
                background: isAgg ? '#8b0000' : isSup ? '#555' : 'transparent',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: '9px', color: isAgg ? '#fff' : isSup ? '#ccc' : 'transparent',
                cursor: onClickBox ? 'pointer' : 'default',
              }}
            >
              {isAgg ? 'X' : isSup ? '/' : ''}
            </div>
          )
        })}
      </div>
    </div>
  )
}


interface VtMSheetProps {
  character: Character
  fields: Record<string, string>
  onChange: (key: string, value: string) => void
  afterTracks?: React.ReactNode
}

function VtMCharacterSheet({ character, fields, onChange, afterTracks }: VtMSheetProps) {
  const labelStyle: React.CSSProperties = {
    fontSize: '9px', textTransform: 'uppercase', letterSpacing: '1.5px',
    color: 'var(--gold-dim)', fontFamily: 'var(--serif)'
  }
  const inputStyle: React.CSSProperties = {
    background: 'var(--surface)', border: '1px solid var(--border)',
    color: 'var(--text)', fontSize: '12px', padding: '0.15rem 0.3rem', width: '100%'
  }
  const sectionHead: React.CSSProperties = {
    fontSize: '11px', textTransform: 'uppercase', letterSpacing: '2px',
    color: 'var(--gold)', borderBottom: '1px solid var(--border)',
    paddingBottom: '3px', marginTop: '12px', marginBottom: '6px'
  }
  const n = (key: string) => parseInt(fields[key] ?? '0') || 0

  const charType = fields['character_type'] || 'Vampire'
  const isThinBlood = charType === 'Thin-Blooded'
  const isGhoul = charType === 'Ghoul'
  const isMortal = charType === 'Mortal'
  const isVampire = !isThinBlood && !isGhoul && !isMortal

  // Suppress unused variable warning — character is available for future use
  void character

  return (
    <>
      {/* Character Type */}
      <div style={{ marginBottom: '10px' }}>
        <label style={labelStyle}>
          Character Type
          <select
            value={fields['character_type'] ?? 'Vampire'}
            onChange={(e) => onChange('character_type', e.target.value)}
            style={{ ...inputStyle, marginTop: '2px' }}
          >
            {['Vampire', 'Thin-Blooded', 'Ghoul', 'Mortal'].map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </select>
        </label>
      </div>

      {/* Hunger — hidden for Ghoul and Mortal; capped at 4 for Thin-Blooded */}
      {!isGhoul && !isMortal && (
        <div style={{ marginBottom: '10px' }}>
          <div style={{ fontSize: '10px', textTransform: 'uppercase', letterSpacing: '1px', color: '#c0392b', marginBottom: '4px' }}>
            Hunger{isThinBlood && ' (max 4)'}
          </div>
          <div style={{ display: 'flex', gap: '4px' }}>
            {[1, 2, 3, 4, ...(isVampire ? [5] : [])].map((i) => (
              <div
                key={i}
                onClick={() => onChange('hunger', String(i === n('hunger') ? i - 1 : i))}
                style={{
                  width: 20, height: 20,
                  background: i <= n('hunger') ? '#c0392b' : 'transparent',
                  border: '1px solid #c0392b',
                  cursor: 'pointer',
                  animation: n('hunger') >= (isVampire ? 5 : 4) && i <= (isVampire ? 5 : 4) ? 'pulse 1s infinite' : undefined,
                }}
              />
            ))}
          </div>
        </div>
      )}

      {/* Core stats row */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '4px', marginBottom: '8px' }}>
        <label style={labelStyle}>
          humanity
          <input type="number" value={fields['humanity'] ?? ''} onChange={(e) => onChange('humanity', e.target.value)} style={{ ...inputStyle, marginTop: '2px' }} />
        </label>
        {isVampire && (
          <label style={labelStyle}>
            blood potency
            <input type="number" value={fields['blood_potency'] ?? ''} onChange={(e) => onChange('blood_potency', e.target.value)} style={{ ...inputStyle, marginTop: '2px' }} />
          </label>
        )}
        {isVampire && (
          <label style={labelStyle}>
            bane severity
            <input type="number" value={fields['bane_severity'] ?? ''} onChange={(e) => onChange('bane_severity', e.target.value)} style={{ ...inputStyle, marginTop: '2px' }} />
          </label>
        )}
        <label style={labelStyle}>
          stains
          <input type="number" value={fields['stains'] ?? ''} onChange={(e) => onChange('stains', e.target.value)} style={{ ...inputStyle, marginTop: '2px' }} />
        </label>
      </div>

      {/* XP */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '10px' }}>
        <label style={{ ...labelStyle, display: 'flex', alignItems: 'center', gap: '6px' }}>
          XP
          <input
            type="number"
            min={0}
            value={fields['xp'] ?? '0'}
            onChange={(e) => onChange('xp', e.target.value)}
            style={{ ...inputStyle, width: '60px', marginTop: 0 }}
          />
        </label>
      </div>

      {/* Damage tracks — Health and Willpower side by side */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 16px', marginBottom: '4px' }}>
        <DamageTrack
          label="Health"
          max={n('health_max') || 4}
          superficial={n('health_superficial')}
          aggravated={n('health_aggravated')}
          onClickBox={(i) => {
            const max = n('health_max') || 4
            const fromRight = max - 1 - i
            const curAgg = n('health_aggravated')
            const curSup = n('health_superficial')
            if (fromRight < curAgg) {
              onChange('health_aggravated', String(Math.max(0, curAgg - 1)))
            } else if (fromRight < curAgg + curSup) {
              onChange('health_superficial', String(Math.max(0, curSup - 1)))
            } else {
              onChange('health_superficial', String(curSup + 1))
            }
          }}
        />
        <DamageTrack
          label="Willpower"
          max={n('willpower_max') || 3}
          superficial={n('willpower_superficial')}
          aggravated={n('willpower_aggravated')}
          onClickBox={(i) => {
            const max = n('willpower_max') || 3
            const fromRight = max - 1 - i
            const curAgg = n('willpower_aggravated')
            const curSup = n('willpower_superficial')
            if (fromRight < curAgg) {
              onChange('willpower_aggravated', String(Math.max(0, curAgg - 1)))
            } else if (fromRight < curAgg + curSup) {
              onChange('willpower_superficial', String(Math.max(0, curSup - 1)))
            } else {
              onChange('willpower_superficial', String(curSup + 1))
            }
          }}
        />
      </div>

      {afterTracks}

      {/* Attributes */}
      <div style={sectionHead}>Attributes</div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '0 12px' }}>
        {[
          { label: 'Physical', attrs: VTM_PHYSICAL_ATTRS },
          { label: 'Social', attrs: VTM_SOCIAL_ATTRS },
          { label: 'Mental', attrs: VTM_MENTAL_ATTRS },
        ].map(({ label, attrs }) => (
          <div key={label}>
            <div style={{ fontSize: '9px', color: 'var(--gold)', marginBottom: '4px' }}>{label}</div>
            {attrs.map((key) => (
              <PipRow
                key={key}
                label={key}
                value={n(key)}
                max={5}
                onChange={(v) => onChange(key, String(v))}
              />
            ))}
          </div>
        ))}
      </div>

      {/* Skills */}
      <div style={sectionHead}>Skills</div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '0 12px' }}>
        {[
          { label: 'Physical', skills: VTM_PHYSICAL_SKILLS },
          { label: 'Social', skills: VTM_SOCIAL_SKILLS },
          { label: 'Mental', skills: VTM_MENTAL_SKILLS },
        ].map(({ label, skills }) => (
          <div key={label}>
            <div style={{ fontSize: '9px', color: 'var(--gold)', marginBottom: '4px' }}>{label}</div>
            {skills.map((key) => (
              <PipRow
                key={key}
                label={key.replace(/_/g, ' ')}
                value={n(key)}
                max={5}
                onChange={(v) => onChange(key, String(v))}
              />
            ))}
          </div>
        ))}
      </div>

      {/* Disciplines — hidden for Mortals; Thin-Blooded get Alchemy instead */}
      {!isMortal && (
        <>
          {isThinBlood ? (
            <>
              <div style={sectionHead}>Thin-Blood Alchemy</div>
              <PipRow
                label="alchemy"
                value={n('thin_blood_alchemy')}
                max={5}
                onChange={(v) => onChange('thin_blood_alchemy', String(v))}
              />
              <label style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px', marginTop: '6px' }}>
                known formulae
                <textarea
                  value={fields['known_formulae'] ?? ''}
                  onChange={(e) => onChange('known_formulae', e.target.value)}
                  style={{ ...inputStyle, fontFamily: 'inherit', resize: 'vertical', minHeight: '3rem' }}
                />
              </label>
            </>
          ) : (
            <>
              <div style={sectionHead}>Disciplines</div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0 12px' }}>
                {VTM_DISCIPLINES.map((key) => (
                  <PipRow
                    key={key}
                    label={key.replace(/_/g, ' ')}
                    value={n(key)}
                    max={5}
                    onChange={(v) => onChange(key, String(v))}
                  />
                ))}
              </div>
              {/* Blood Sorcery Rituals — shown when blood_sorcery >= 1 */}
              {n('blood_sorcery') >= 1 && (
                <label style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px', marginTop: '6px' }}>
                  rituals
                  <textarea
                    value={fields['rituals'] ?? ''}
                    onChange={(e) => onChange('rituals', e.target.value)}
                    style={{ ...inputStyle, fontFamily: 'inherit', resize: 'vertical', minHeight: '3rem' }}
                  />
                </label>
              )}
              {/* Oblivion Ceremonies — shown when oblivion >= 1 */}
              {n('oblivion') >= 1 && (
                <label style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px', marginTop: '4px' }}>
                  ceremonies
                  <textarea
                    value={fields['ceremonies'] ?? ''}
                    onChange={(e) => onChange('ceremonies', e.target.value)}
                    style={{ ...inputStyle, fontFamily: 'inherit', resize: 'vertical', minHeight: '3rem' }}
                  />
                </label>
              )}
            </>
          )}
        </>
      )}

      {/* Identity */}
      <div style={sectionHead}>Identity</div>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '4px 8px' }}>
        {isVampire && (['clan', 'predator_type', 'sect', 'generation'] as string[]).map((key) => (
          <label key={key} style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px' }}>
            {key.replace(/_/g, ' ')}
            <input type="text" value={fields[key] ?? ''} onChange={(e) => onChange(key, e.target.value)} style={inputStyle} />
          </label>
        ))}
        {isThinBlood && (['clan', 'predator_type'] as string[]).map((key) => (
          <label key={key} style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px' }}>
            {key.replace(/_/g, ' ')}
            <input type="text" value={fields[key] ?? ''} onChange={(e) => onChange(key, e.target.value)} style={inputStyle} />
          </label>
        ))}
        {isGhoul && (
          <>
            <label style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px' }}>
              domitor
              <input type="text" value={fields['domitor'] ?? ''} onChange={(e) => onChange('domitor', e.target.value)} style={inputStyle} />
            </label>
            <label style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px' }}>
              predator type
              <input type="text" value={fields['predator_type'] ?? ''} onChange={(e) => onChange('predator_type', e.target.value)} style={inputStyle} />
            </label>
          </>
        )}
        {(['ambition', 'desire'] as string[]).map((key) => (
          <label key={key} style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px' }}>
            {key}
            <input type="text" value={fields[key] ?? ''} onChange={(e) => onChange(key, e.target.value)} style={inputStyle} />
          </label>
        ))}
      </div>

      {/* Bond Strength for Ghouls */}
      {isGhoul && (
        <div style={{ marginTop: '6px' }}>
          <PipRow
            label="bond strength"
            value={n('bond_strength')}
            max={3}
            onChange={(v) => onChange('bond_strength', String(v))}
          />
        </div>
      )}

      {(['convictions', 'touchstones', 'skill_specialties', 'merits_flaws'] as string[]).map((key) => (
        <label key={key} style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px', marginTop: '4px' }}>
          {key.replace(/_/g, ' ')}
          <textarea
            value={fields[key] ?? ''}
            onChange={(e) => onChange(key, e.target.value)}
            style={{ ...inputStyle, fontFamily: 'inherit', resize: 'vertical', minHeight: '3rem' }}
          />
        </label>
      ))}
      {isVampire && (
        <label style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px', marginTop: '4px' }}>
          loresheets
          <textarea
            value={fields['loresheets'] ?? ''}
            onChange={(e) => onChange('loresheets', e.target.value)}
            style={{ ...inputStyle, fontFamily: 'inherit', resize: 'vertical', minHeight: '3rem' }}
          />
        </label>
      )}
      <label style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px', marginTop: '4px' }}>
        notes
        <textarea
          value={fields['notes'] ?? ''}
          onChange={(e) => onChange('notes', e.target.value)}
          style={{ ...inputStyle, fontFamily: 'inherit', resize: 'vertical', minHeight: '3rem' }}
        />
      </label>
    </>
  )
}

export function CharacterSheetPanel({ character, rulesetId, lastEvent, afterTracks }: CharacterSheetPanelProps) {
  const [ruleset, setRuleset] = useState<Ruleset | null>(null)
  const [fields, setFields] = useState<Record<string, string>>({})
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (rulesetId === null) return
    fetchRuleset(rulesetId)
      .then(setRuleset)
      .catch(console.error)
  }, [rulesetId])

  useEffect(() => {
    if (!character) return
    try {
      const data = JSON.parse(character.data_json || '{}') as Record<string, unknown>
      setFields(Object.fromEntries(Object.entries(data).map(([k, v]) => [k, String(v ?? '')])))
    } catch {
      setFields({})
    }
  }, [character?.id])

  useEffect(() => {
    if (!isCharacterUpdatedEvent(lastEvent)) return
    if (lastEvent.payload.id !== character?.id) return
    if (lastEvent.payload.data_json) {
      try {
        const data = JSON.parse(lastEvent.payload.data_json) as Record<string, unknown>
        setFields(Object.fromEntries(Object.entries(data).map(([k, v]) => [k, String(v ?? '')])))
      } catch { /* ignore */ }
    }
  }, [lastEvent, character?.id])

  if (!character) return null

  const schema: SchemaField[] = (() => {
    try {
      const parsed = JSON.parse(ruleset?.schema_json ?? '[]') as unknown
      if (!Array.isArray(parsed) && typeof parsed === 'object' && parsed !== null) {
        const legacy = parsed as Record<string, unknown>
        if (Array.isArray(legacy['fields'])) {
          return (legacy['fields'] as string[]).map((key) => ({
            key,
            label: key.charAt(0).toUpperCase() + key.slice(1).replace(/_/g, ' '),
            type: 'text' as const,
          }))
        }
      }
      if (!Array.isArray(parsed)) return []
      return parsed as SchemaField[]
    } catch {
      return []
    }
  })()

  function handleChange(key: string, value: string) {
    const next = { ...fields, [key]: value }
    setFields(next)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      if (schema.length > 0) {
        const updates: Record<string, unknown> = {}
        schema.forEach((f) => {
          const v = next[f.key]
          updates[f.key] = f.type === 'number' ? (v === '' ? null : Number(v)) : v
        })
        patchCharacter(character!.id, updates).catch(console.error)
      } else {
        // VtM (schema-free): send ALL current fields so UpdateCharacterData (full-replace) doesn't lose stats.
        const numericKeys = new Set([
          'hunger','blood_potency','bane_severity','humanity','stains','xp',
          'strength','dexterity','stamina','charisma','manipulation','composure',
          'intelligence','wits','resolve','health_max','health_superficial','health_aggravated',
          'willpower_max','willpower_superficial','willpower_aggravated',
          'athletics','brawl','craft','drive','firearms','larceny','melee','stealth','survival',
          'animal_ken','etiquette','insight','intimidation','leadership','performance',
          'persuasion','streetwise','subterfuge','academics','awareness','finance',
          'investigation','medicine','occult','politics','technology',
          'animalism','auspex','blood_sorcery','celerity','dominate','fortitude',
          'obfuscate','oblivion','potence','presence','protean',
          'bond_strength','thin_blood_alchemy',
        ])
        const updates: Record<string, unknown> = {}
        for (const [k, v] of Object.entries(next)) {
          updates[k] = numericKeys.has(k) ? (v === '' ? null : Number(v)) : v
        }
        patchCharacter(character!.id, updates).catch(console.error)
      }
    }, 500)
  }

  function handlePortraitChange(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    uploadPortrait(character!.id, file).catch(console.error)
  }

  const isVtM = ruleset?.name?.toLowerCase() === 'vtm'
  if (isVtM) {
    return <VtMCharacterSheet character={character} fields={fields} onChange={handleChange} afterTracks={afterTracks} />
  }

  const attributeFields = schema.filter((f) => ATTRIBUTE_KEYS.has(f.key))
  const trackFields     = schema.filter((f) => TRACK_KEYS.has(f.key))
  const otherFields     = schema.filter((f) => !ATTRIBUTE_KEYS.has(f.key) && !TRACK_KEYS.has(f.key))

  return (
    <>
      {/* Portrait */}
      <div className="portrait-wrap">
        {character.portrait_path ? (
          <img
            className="portrait-circle"
            src={`/api/files/${character.portrait_path}`}
            alt={character.name}
          />
        ) : (
          <div className="portrait-placeholder-circle">{character.name[0] ?? '?'}</div>
        )}
        <label className="portrait-change">
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            onChange={handlePortraitChange}
            style={{ display: 'none' }}
          />
          Change portrait
        </label>
      </div>

      {/* Attributes — pip dots */}
      {attributeFields.length > 0 && (
        <div>
          {attributeFields.map((f) => (
            <div key={f.key} className="attr-row">
              <span className="attr-label">{f.key}</span>
              <AttributePips value={Number(fields[f.key] ?? 0)} />
            </div>
          ))}
        </div>
      )}

      {/* Tracks — segmented bars */}
      {trackFields.length > 0 && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
          {trackFields.map((f) => (
            <TrackBar
              key={f.key}
              fieldKey={f.key}
              value={Number(fields[f.key] ?? 0)}
            />
          ))}
        </div>
      )}

      {/* Other fields — number fields in 2-col grid, text/textarea full width */}
      {(() => {
        const numFields = otherFields.filter((f) => f.type === 'number')
        const wideFields = otherFields.filter((f) => f.type !== 'number')
        const labelStyle: React.CSSProperties = { fontSize: '9px', textTransform: 'uppercase', letterSpacing: '1.5px', color: 'var(--gold-dim)', fontFamily: 'var(--serif)', display: 'flex', flexDirection: 'column', gap: '2px' }
        const inputStyle: React.CSSProperties = { background: 'var(--surface)', border: '1px solid var(--border)', color: 'var(--text)', fontSize: '12px', padding: '0.15rem 0.3rem', width: '100%' }
        return (
          <>
            {numFields.length > 0 && (
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.35rem 0.5rem' }}>
                {numFields.map((field) => (
                  <label key={field.key} style={labelStyle}>
                    {field.label}
                    <input
                      type="number"
                      value={fields[field.key] ?? ''}
                      onChange={(e) => handleChange(field.key, e.target.value)}
                      style={inputStyle}
                    />
                  </label>
                ))}
              </div>
            )}
            {wideFields.map((field) => (
              <label key={field.key} style={{ ...labelStyle, display: 'flex', flexDirection: 'column', gap: '2px' }}>
                {field.label}
                {field.type === 'textarea' ? (
                  <textarea
                    value={fields[field.key] ?? ''}
                    onChange={(e) => handleChange(field.key, e.target.value)}
                    style={{ ...inputStyle, fontFamily: 'inherit', resize: 'vertical', minHeight: '3rem' }}
                  />
                ) : (
                  <input
                    type="text"
                    value={fields[field.key] ?? ''}
                    onChange={(e) => handleChange(field.key, e.target.value)}
                    style={inputStyle}
                  />
                )}
              </label>
            ))}
          </>
        )
      })()}
    </>
  )
}
