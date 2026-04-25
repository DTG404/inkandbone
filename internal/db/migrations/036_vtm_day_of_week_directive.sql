-- 036_vtm_day_of_week_directive.sql: Instruct the VtM GM to name the day of the week
-- in the opening scene so the chronicle night tracker can anchor correctly.
UPDATE rulesets SET gm_context = gm_context || '

NIGHT TRACKER: When narrating the opening scene of a session or the start of a new night, always name the day of the week explicitly (e.g. "It''s a Thursday night", "Friday evening settles over the city", "The Sunday darkness wraps around you"). This is required — do not omit the day name.'
WHERE name = 'vtm';
