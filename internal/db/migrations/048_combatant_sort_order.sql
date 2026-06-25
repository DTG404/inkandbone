ALTER TABLE combatants ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0;

UPDATE combatants
SET sort_order = (
  SELECT COUNT(*)
  FROM combatants c2
  WHERE c2.encounter_id = combatants.encounter_id
    AND (c2.initiative > combatants.initiative
         OR (c2.initiative = combatants.initiative AND c2.id < combatants.id))
);
