-- 033_vtm_pg_compulsions.sql: Add VtM Player's Guide clan Compulsion oracle tables.
-- 7 new clans: Banu Haqim, Hecata, Lasombra, Ministry, Ravnos, Salubri, Tzimisce.
-- 10 entries each (rolls 1-10); entries 7-10 duplicate 1-6 for d10 compatibility.

-- Banu Haqim Compulsion: Judgment (must condemn a witnessed wrong)
INSERT INTO oracle_tables (ruleset_id, table_name, roll_min, roll_max, result)
SELECT id, 'compulsion_banu_haqim', 1, 1, 'You must expose a crime or wrongdoing you have witnessed tonight — silence is complicity.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_banu_haqim', 2, 2, 'You refuse to act until someone has answered for an injustice you know of.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_banu_haqim', 3, 3, 'You impose a sentence on someone present who has wronged another, however severe.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_banu_haqim', 4, 4, 'You are convinced someone nearby is guilty of a serious crime and begin investigating immediately.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_banu_haqim', 5, 5, 'You cannot assist someone you believe to be unjust until they have been held accountable.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_banu_haqim', 6, 6, 'You declare that swift, decisive action must be taken against a known wrongdoer — now, not later.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_banu_haqim', 7, 7, 'You must expose a crime or wrongdoing you have witnessed tonight — silence is complicity.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_banu_haqim', 8, 8, 'You refuse to act until someone has answered for an injustice you know of.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_banu_haqim', 9, 9, 'You are convinced someone nearby is guilty of a serious crime and begin investigating immediately.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_banu_haqim', 10, 10, 'You declare that swift, decisive action must be taken against a known wrongdoer — now, not later.' FROM rulesets WHERE name = 'vtm';

-- Hecata Compulsion: Morbidity (fixated on death and dying)
INSERT INTO oracle_tables (ruleset_id, table_name, roll_min, roll_max, result)
SELECT id, 'compulsion_hecata', 1, 1, 'You must spend time examining or meditating on something dead or dying in the scene before acting.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_hecata', 2, 2, 'You become fascinated by a corpse, wound, or illness and cannot act until you have studied it thoroughly.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_hecata', 3, 3, 'You must speak at length about death, decay, or what inevitably awaits all mortals present.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_hecata', 4, 4, 'You feel compelled to prepare for your own Final Death — writing a will, settling debts, saying goodbyes.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_hecata', 5, 5, 'You are overcome by grief for a specific dead individual and cannot function until you express it openly.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_hecata', 6, 6, 'You must determine the exact manner of death of the most recently deceased individual in the area.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_hecata', 7, 7, 'You must spend time examining or meditating on something dead or dying in the scene before acting.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_hecata', 8, 8, 'You become fascinated by a corpse, wound, or illness and cannot act until you have studied it thoroughly.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_hecata', 9, 9, 'You feel compelled to prepare for your own Final Death — writing a will, settling debts, saying goodbyes.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_hecata', 10, 10, 'You must determine the exact manner of death of the most recently deceased individual in the area.' FROM rulesets WHERE name = 'vtm';

-- Lasombra Compulsion: Ruthlessness (must dominate or destroy)
INSERT INTO oracle_tables (ruleset_id, table_name, roll_min, roll_max, result)
SELECT id, 'compulsion_lasombra', 1, 1, 'You must demonstrate dominance over someone weaker before taking any other action.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_lasombra', 2, 2, 'You cannot accept compromise — only total victory on the matter at hand will satisfy you.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_lasombra', 3, 3, 'You cut loose an ally who appears to be failing — they are a liability now.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_lasombra', 4, 4, 'You sacrifice something of value to prove you will pay any price for victory.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_lasombra', 5, 5, 'You refuse to retreat under any circumstances, even when tactically suicidal to remain.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_lasombra', 6, 6, 'You demand unconditional surrender or total destruction from your opponent — no quarter given.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_lasombra', 7, 7, 'You must demonstrate dominance over someone weaker before taking any other action.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_lasombra', 8, 8, 'You cannot accept compromise — only total victory on the matter at hand will satisfy you.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_lasombra', 9, 9, 'You refuse to retreat under any circumstances, even when tactically suicidal to remain.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_lasombra', 10, 10, 'You demand unconditional surrender or total destruction from your opponent — no quarter given.' FROM rulesets WHERE name = 'vtm';

-- Ministry Compulsion: Transgression (must tempt and corrupt)
INSERT INTO oracle_tables (ruleset_id, table_name, roll_min, roll_max, result)
SELECT id, 'compulsion_ministry', 1, 1, 'You must tempt someone present into breaking a personal conviction or code before doing anything else.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ministry', 2, 2, 'You cannot proceed until you have offered someone a means to indulge a forbidden desire.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ministry', 3, 3, 'You expose a secret vice of someone in the room, helpfully and without judgment.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ministry', 4, 4, 'You offer information that could destroy something someone else holds sacred.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ministry', 5, 5, 'You find the most pious or principled person present and begin corrupting their certainty.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ministry', 6, 6, 'You provide an irresistible opportunity for sin to whoever seems most resistant to it.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ministry', 7, 7, 'You must tempt someone present into breaking a personal conviction or code before doing anything else.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ministry', 8, 8, 'You cannot proceed until you have offered someone a means to indulge a forbidden desire.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ministry', 9, 9, 'You find the most pious or principled person present and begin corrupting their certainty.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ministry', 10, 10, 'You provide an irresistible opportunity for sin to whoever seems most resistant to it.' FROM rulesets WHERE name = 'vtm';

-- Ravnos Compulsion: Tempted (aimless thrill-seeking)
INSERT INTO oracle_tables (ruleset_id, table_name, roll_min, roll_max, result)
SELECT id, 'compulsion_ravnos', 1, 1, 'You steal something — money, secrets, attention — purely for the thrill of it.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ravnos', 2, 2, 'You create an elaborate misdirection that serves no purpose but your own amusement.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ravnos', 3, 3, 'You wander away from the group, following whatever catches your interest for the rest of the scene.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ravnos', 4, 4, 'You pick a fight — not out of rage or principle, but out of bone-deep boredom.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ravnos', 5, 5, 'You put yourself in unnecessary danger just to see if you can escape it.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ravnos', 6, 6, 'You spin a lie so convincing that for a moment even you forget the truth.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ravnos', 7, 7, 'You steal something — money, secrets, attention — purely for the thrill of it.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ravnos', 8, 8, 'You create an elaborate misdirection that serves no purpose but your own amusement.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ravnos', 9, 9, 'You put yourself in unnecessary danger just to see if you can escape it.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_ravnos', 10, 10, 'You spin a lie so convincing that for a moment even you forget the truth.' FROM rulesets WHERE name = 'vtm';

-- Salubri Compulsion: Affective (compelled to heal and protect)
INSERT INTO oracle_tables (ruleset_id, table_name, roll_min, roll_max, result)
SELECT id, 'compulsion_salubri', 1, 1, 'You must heal or comfort whoever appears most wounded or suffering in the scene — everything else waits.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_salubri', 2, 2, 'You cannot take a violent action until you have first offered your target a genuine chance to yield.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_salubri', 3, 3, 'You are overwhelmed by the suffering of those around you and must address it before your own needs.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_salubri', 4, 4, 'You confess a personal sin or failing before asking anything of anyone present.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_salubri', 5, 5, 'You intervene in a conflict on behalf of the weaker party, regardless of whose side you were on.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_salubri', 6, 6, 'You attempt to mend a broken relationship or reconcile two enemies before any other goal.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_salubri', 7, 7, 'You must heal or comfort whoever appears most wounded or suffering in the scene — everything else waits.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_salubri', 8, 8, 'You cannot take a violent action until you have first offered your target a genuine chance to yield.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_salubri', 9, 9, 'You intervene in a conflict on behalf of the weaker party, regardless of whose side you were on.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_salubri', 10, 10, 'You attempt to mend a broken relationship or reconcile two enemies before any other goal.' FROM rulesets WHERE name = 'vtm';

-- Tzimisce Compulsion: Covetous (must own and control territory)
INSERT INTO oracle_tables (ruleset_id, table_name, roll_min, roll_max, result)
SELECT id, 'compulsion_tzimisce', 1, 1, 'You must claim ownership of something in the scene — a place, an object, a person — and mark it as yours.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_tzimisce', 2, 2, 'You cannot tolerate an intruder in a space you have mentally designated as your own.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_tzimisce', 3, 3, 'You must alter or visibly improve something you control, right now, however inconvenient.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_tzimisce', 4, 4, 'You demand explicit fealty from someone present — acknowledgment that they belong to you.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_tzimisce', 5, 5, 'You catalogue everything of value in the room and plan how to acquire or control all of it.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_tzimisce', 6, 6, 'You refuse to allow anyone to leave a space you consider yours without your explicit permission.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_tzimisce', 7, 7, 'You must claim ownership of something in the scene — a place, an object, a person — and mark it as yours.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_tzimisce', 8, 8, 'You cannot tolerate an intruder in a space you have mentally designated as your own.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_tzimisce', 9, 9, 'You catalogue everything of value in the room and plan how to acquire or control all of it.' FROM rulesets WHERE name = 'vtm'
UNION ALL SELECT id, 'compulsion_tzimisce', 10, 10, 'You refuse to allow anyone to leave a space you consider yours without your explicit permission.' FROM rulesets WHERE name = 'vtm';
