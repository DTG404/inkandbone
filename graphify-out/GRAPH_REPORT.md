# Graph Report - inkandbone  (2026-04-25)

## Corpus Check
- 134 files · ~285,496 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1018 nodes · 2821 edges · 25 communities detected
- Extraction: 42% EXTRACTED · 58% INFERRED · 0% AMBIGUOUS · INFERRED: 1643 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 32|Community 32]]

## God Nodes (most connected - your core abstractions)
1. `newTestServer()` - 121 edges
2. `Server` - 108 edges
3. `DB` - 91 edges
4. `createCampaign()` - 56 edges
5. `parsePathID()` - 54 edges
6. `newTestMCP()` - 48 edges
7. `newTestDB()` - 44 edges
8. `seedCampaign()` - 40 edges
9. `createSession()` - 38 edges
10. `vtmCampaignAndChar()` - 33 edges

## Surprising Connections (you probably didn't know these)
- `main()` --calls--> `NewDeepSeekClient()`  [INFERRED]
  cmd/ttrpg/main.go → internal/ai/deepseek.go
- `main()` --calls--> `NewOpenRouterClient()`  [INFERRED]
  cmd/ttrpg/main.go → internal/ai/openrouter.go
- `main()` --calls--> `NewServer()`  [INFERRED]
  cmd/ttrpg/main.go → internal/api/server.go
- `main()` --calls--> `New()`  [INFERRED]
  cmd/ttrpg/main.go → internal/mcp/server.go
- `TestHandlePreSessionBrief_ok()` --calls--> `createObjective()`  [INFERRED]
  internal/api/routes_phase_c_test.go → web/src/api.ts

## Communities

### Community 0 - "Community 0"
Cohesion: 0.02
Nodes (74): fadeIn(), fadeOut(), setAmbientTrack(), createMapPin(), createNPC(), createObjective(), createRelationship(), createXP() (+66 more)

### Community 1 - "Community 1"
Cohesion: 0.05
Nodes (16): contextCombatSnapshot, contextResponse, createItem(), deleteCampaign(), deleteItem(), rollCheckResult, Server, bloodPotencyBonusDice() (+8 more)

### Community 2 - "Community 2"
Cohesion: 0.04
Nodes (45): advanceTurn(), getAudioMuted(), getAudioVolume(), setupActiveSession(), TestEndCombat(), TestStartCombat(), TestUpdateCombatant(), TestUpdateCombatant_missingHP() (+37 more)

### Community 3 - "Community 3"
Cohesion: 0.06
Nodes (85): stubCompleter, stubCompleterStreamer, seedCampaign(), TestFindWorldNoteByTitle(), TestUpdateWorldNotePersonality(), TestHandleAdvanceCharacter_notFound(), TestGetRuleset(), TestGetRuleset_notFound() (+77 more)

### Community 4 - "Community 4"
Cohesion: 0.05
Nodes (57): fetchCampaigns(), setupCampaign(), TestEndSession(), TestSetActive(), TestSetActive_campaignNotFound(), TestSetActive_reopensClosed(), TestStartSession(), TestStartSession_noCampaign() (+49 more)

### Community 5 - "Community 5"
Cohesion: 0.04
Nodes (38): ChatMessage, Client, Completer, DeepSeekClient, OllamaClient, OpenRouterClient, Responder, Streamer (+30 more)

### Community 6 - "Community 6"
Cohesion: 0.04
Nodes (72): CanAffordAny(), CostRulesDescription(), FieldHints(), TestCanAffordAny(), TestValidFields_wrathGlory(), TestVtMInClanDisciplinesSpacedName(), TestWGRecalcDerived(), TestXPCostFor_blades() (+64 more)

### Community 7 - "Community 7"
Cohesion: 0.11
Nodes (48): createCampaign(), createCharacter(), createSession(), TestGetContext_withActiveCombat(), TestGetContext_withActiveState(), newTestDB(), TestCampaigns(), TestCharacterCurrency() (+40 more)

### Community 8 - "Community 8"
Cohesion: 0.07
Nodes (23): DualDeepSeekClient, DualOllamaClient, DualOpenRouterClient, HybridClient, NewClient(), newDeepSeekAutoClient(), NewDeepSeekClient(), newDeepSeekClientWithModel() (+15 more)

### Community 9 - "Community 9"
Cohesion: 0.14
Nodes (31): captureCompleter, advanceRequest(), getCharStats(), TestVtM_AutoUpdateCharacterStats_StringXPNormalization(), TestVtM_AutoUpdateCharacterStats_XPGuard(), TestVtM_CharacterCRUD_VtMFields(), TestVtM_DetectAndApplyStains_BelowRemorseThreshold_StainsAccumulate(), TestVtM_DetectAndApplyStains_CapsAt10() (+23 more)

### Community 10 - "Community 10"
Cohesion: 0.13
Nodes (20): bladesStats(), ironswornStats(), randPick(), roll4d6DropLowest(), rollNd(), RollStats(), rollVtMV5Stats(), rollWrathGloryStats() (+12 more)

### Community 11 - "Community 11"
Cohesion: 0.26
Nodes (7): getTension(), TestGetMasqueradeIntegrity_Default(), TestUpdateMasqueradeIntegrity_ClampMax(), TestUpdateMasqueradeIntegrity_ClampMin(), seedSession(), TestGetTension_Default(), TestUpdateTension_Clamp()

### Community 12 - "Community 12"
Cohesion: 0.39
Nodes (6): CharacterOptions(), TestVtMOptions_Generation(), TestVtMOptions_PredatorType(), TestVtMOptions_Sect(), TestVtMOptions_V5Clans(), TestVtMPlayerGuideClans()

### Community 13 - "Community 13"
Cohesion: 0.4
Nodes (4): DiceRoll, Map, MapPin, WorldNote

### Community 14 - "Community 14"
Cohesion: 0.4
Nodes (4): Campaign, CampaignStats, Character, Ruleset

### Community 16 - "Community 16"
Cohesion: 0.67
Nodes (2): RulebookChunk, RulebookSource

### Community 17 - "Community 17"
Cohesion: 0.67
Nodes (2): combatSnapshot, contextSnapshot

### Community 18 - "Community 18"
Cohesion: 1.0
Nodes (1): TimelineEntry

### Community 19 - "Community 19"
Cohesion: 1.0
Nodes (1): SessionNPC

### Community 20 - "Community 20"
Cohesion: 1.0
Nodes (1): Item

### Community 21 - "Community 21"
Cohesion: 1.0
Nodes (1): XPEntry

### Community 22 - "Community 22"
Cohesion: 1.0
Nodes (1): Relationship

### Community 23 - "Community 23"
Cohesion: 1.0
Nodes (1): Objective

### Community 24 - "Community 24"
Cohesion: 1.0
Nodes (1): combatantInput

### Community 32 - "Community 32"
Cohesion: 1.0
Nodes (1): MockWebSocket

## Knowledge Gaps
- **35 isolated node(s):** `EventType`, `Event`, `contextCombatSnapshot`, `contextResponse`, `rollCheckResult` (+30 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 16`** (3 nodes): `RulebookChunk`, `RulebookSource`, `queries_rulebook.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 17`** (3 nodes): `context.go`, `combatSnapshot`, `contextSnapshot`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 18`** (2 nodes): `TimelineEntry`, `queries_timeline.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 19`** (2 nodes): `SessionNPC`, `queries_npcs.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 20`** (2 nodes): `Item`, `queries_items.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 21`** (2 nodes): `XPEntry`, `queries_xp.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 22`** (2 nodes): `Relationship`, `queries_relationships.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 23`** (2 nodes): `Objective`, `queries_objectives.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 24`** (2 nodes): `combat.go`, `combatantInput`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 32`** (2 nodes): `MockWebSocket`, `App.test.tsx`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Server` connect `Community 1` to `Community 0`, `Community 2`, `Community 3`, `Community 5`, `Community 6`, `Community 7`, `Community 8`, `Community 9`, `Community 11`?**
  _High betweenness centrality (0.136) - this node is a cross-community bridge._
- **Why does `DB` connect `Community 2` to `Community 1`, `Community 3`, `Community 4`, `Community 5`, `Community 7`, `Community 9`, `Community 11`?**
  _High betweenness centrality (0.095) - this node is a cross-community bridge._
- **Why does `newTestServer()` connect `Community 3` to `Community 9`, `Community 2`, `Community 5`, `Community 7`?**
  _High betweenness centrality (0.082) - this node is a cross-community bridge._
- **Are the 120 inferred relationships involving `newTestServer()` (e.g. with `TestIngestRulebook_textPlain()` and `TestIngestRulebook_noHeadings()`) actually correct?**
  _`newTestServer()` has 120 INFERRED edges - model-reasoned connections that need verification._
- **Are the 55 inferred relationships involving `createCampaign()` (e.g. with `TestAutoSuggestXPSpend_noopForCoC()` and `TestAutoSuggestXPSpend_sessionCap()`) actually correct?**
  _`createCampaign()` has 55 INFERRED edges - model-reasoned connections that need verification._
- **Are the 18 inferred relationships involving `parsePathID()` (e.g. with `.handleNextTurn()` and `.handleListXP()`) actually correct?**
  _`parsePathID()` has 18 INFERRED edges - model-reasoned connections that need verification._
- **What connects `EventType`, `Event`, `contextCombatSnapshot` to the rest of the system?**
  _35 weakly-connected nodes found - possible documentation gaps or missing edges._