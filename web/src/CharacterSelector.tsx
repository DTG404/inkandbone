interface CharacterSelectorProps {
  characters: { id: number; name: string }[]
  selectedId: number | null
  onSelect: (id: number) => void
}

export function CharacterSelector({ characters, selectedId, onSelect }: CharacterSelectorProps) {
  if (characters.length <= 1) return null

  return (
    <span className="character-selector">
      Playing as:
      <select
        value={selectedId ?? ''}
        onChange={(e) => onSelect(Number(e.target.value))}
      >
        {characters.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name}
          </option>
        ))}
      </select>
    </span>
  )
}
