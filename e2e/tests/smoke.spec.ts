/**
 * End-to-end smoke test for inkandbone.
 *
 * Covers Group C features (shared dice feed, handout modal, particle FX API,
 * combat round counter) as well as the core player loop.
 *
 * Tests run serially against a single ephemeral server instance.
 * beforeAll bootstraps campaign → character → session → activates all via API.
 */

import { test, expect } from '@playwright/test'

test.describe.configure({ mode: 'serial' })

const BASE = 'http://localhost:7432'
const CHAR_NAME = 'Theron Brightwald'

// Shared IDs populated by beforeAll
let campaignId: number
let characterId: number
let sessionId: number

// The 10 RPG actions a player takes during the smoke run
const RPG_ACTIONS = [
  'I look around the dimly lit tavern, scanning for unusual faces.',
  'I approach the bartender and lean in close to speak quietly.',
  'I ask him about the merchant who went missing three nights ago.',
  'I slip a gold coin across the bar and wait for his answer.',
  'I examine the corner booth where the merchant was last seen.',
  'I crouch down to check beneath the table for dropped items.',
  'I signal to the barmaid and ask if she noticed anyone following him.',
  'I step outside to inspect the narrow alley behind the tavern.',
  'I press myself against the wall and listen for footsteps in the dark.',
  'I slip back inside and order an ale to blend in while I think.',
]

test.beforeAll(async ({ playwright }) => {
  const api = await playwright.request.newContext({ baseURL: BASE })

  // 1. Discover the VtM ruleset ID (VtM sheet renders DiceHistoryPanel via afterTracks)
  const rsResp = await api.get('/api/rulesets')
  expect(rsResp.ok()).toBe(true)
  const rulesets = (await rsResp.json()) as Array<{ id: number; name: string }>
  const vtm = rulesets.find((r) => r.name === 'vtm')
  if (!vtm) throw new Error('vtm ruleset not found in fresh DB')

  // 2. Create campaign
  const campResp = await api.post('/api/campaigns', {
    data: { name: 'E2E Smoke Campaign', description: 'Automated smoke test', ruleset_id: vtm.id },
  })
  if (campResp.status() !== 201) {
    const body = await campResp.text()
    throw new Error(`createCampaign failed ${campResp.status()}: ${body}`)
  }
  campaignId = (await campResp.json()).id

  // 3. Create character
  const charResp = await api.post(`/api/campaigns/${campaignId}/characters`, {
    data: { name: CHAR_NAME },
  })
  expect(charResp.status()).toBe(201)
  characterId = (await charResp.json()).id

  // 4. Create session
  const sessResp = await api.post(`/api/campaigns/${campaignId}/sessions`, {
    data: { title: 'Session Zero — The Missing Merchant', date: '2026-06-26' },
  })
  expect(sessResp.status()).toBe(201)
  sessionId = (await sessResp.json()).id

  // 5. Activate campaign, character, session
  const settingsResp = await api.patch('/api/settings', {
    data: {
      campaign_id: campaignId,
      character_id: characterId,
      session_id: sessionId,
    },
  })
  expect(settingsResp.ok()).toBe(true)

  await api.dispose()
})

// ── Test 1: App loads ─────────────────────────────────────────────────────────

test('app loads and shows session view', async ({ page }) => {
  await page.goto('/')

  // The app renders .grimoire immediately once context resolves
  const grimoire = page.locator('.grimoire')
  await expect(grimoire).toBeVisible({ timeout: 10_000 })

  // Header should show the campaign, character, and session
  await expect(page.locator('.h-campaign')).toContainText('E2E Smoke Campaign')
  await expect(page.locator('.h-char')).toContainText(CHAR_NAME)
  await expect(page.locator('.h-session')).toContainText('Session Zero')

  // Main body (left + right panels) should be visible
  await expect(page.locator('.workspace-body')).toBeVisible()
})

// ── Test 2: 10 RPG actions ────────────────────────────────────────────────────

test('player sends 10 RPG actions via chat', async ({ page }) => {
  await page.goto('/')
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })

  for (const action of RPG_ACTIONS) {
    const field = page.locator('.player-input-field')
    await field.fill(action)
    // Ctrl+Enter or Enter both submit — use Enter
    await field.press('Enter')
    // Wait for this message to appear before typing the next
    await expect(
      page.locator('.prose-player-text').last()
    ).toContainText(action.slice(0, 25), { timeout: 8_000 })
  }

  // All 10 messages should be in the journal
  await expect(page.locator('.prose-player-text')).toHaveCount(10)
})

// ── Test 3: Dice roll with character name (Group C feature) ───────────────────

test('dice roll via API appears in live feed', async ({ page, request }) => {
  // Pre-roll before navigation so fetchDiceRolls populates the panel on mount
  // (avoids WS race: the panel returns null until liveRolls is non-empty)
  const roll = await request.post(`${BASE}/api/sessions/${sessionId}/dice-rolls`, {
    data: { expression: '1d20' },
  })
  expect(roll.ok()).toBe(true)

  await page.goto('/')
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })

  // The dice compact panel populates from DB fetch on mount
  await expect(page.locator('.dice-compact')).toBeVisible({ timeout: 5_000 })
  await expect(page.locator('.dice-compact-result').first()).not.toBeEmpty()
})

test('dice roll with character name shows name in live feed', async ({ page, request }) => {
  // Pre-roll with character name before navigation so it loads from DB on mount.
  // The DB-backed fromDbRoll sets characterName:''; only WS events carry the name.
  // So we navigate first, wait for WS to connect, then roll.
  await page.goto('/')
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })
  // Give WS connection time to fully open
  await page.waitForTimeout(800)

  // Roll with character_name — this WS event reaches the already-connected page
  const roll = await request.post(`${BASE}/api/sessions/${sessionId}/dice-rolls`, {
    data: { expression: '1d20', character_name: CHAR_NAME },
  })
  expect(roll.ok()).toBe(true)

  // Live feed should show the character name from the WS event
  await expect(page.locator('.dice-compact')).toBeVisible({ timeout: 5_000 })
  await expect(page.locator('.dice-compact-expr span').first()).toContainText(CHAR_NAME, { timeout: 5_000 })
})

test('hidden dice roll shows [GM] in live feed', async ({ page, request }) => {
  await page.goto('/')
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })
  await page.waitForTimeout(800)

  const roll = await request.post(`${BASE}/api/sessions/${sessionId}/dice-rolls`, {
    data: { expression: '1d12', hidden: true },
  })
  expect(roll.ok()).toBe(true)

  // Hidden rolls show [GM] with result as ?
  await expect(page.locator('.dice-compact-expr').first()).toContainText('[GM]', { timeout: 5_000 })
  await expect(page.locator('.dice-compact-result').first()).toContainText('?')
})

// ── Test 4: Secret creation and handout modal (Group C feature) ───────────────

test('creating and revealing a secret triggers the handout modal', async ({ page }) => {
  await page.goto('/')
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })

  // Navigate to the Secrets right-panel destination
  await page.locator('.workspace-desktop-nav').getByRole('button', { name: 'Secrets', exact: true }).click()
  await expect(page.locator('.secrets-panel')).toBeVisible({ timeout: 5_000 })

  // Create a secret via the in-panel form
  await page.locator('button', { hasText: '+ Add Secret' }).click()
  await page.locator('.secret-form input[placeholder="Title"]').fill('The Hidden Map Fragment')
  await page.locator('.secret-form select').selectOption('handout')
  await page.locator('.secret-form textarea[placeholder="Content"]').fill(
    'A torn piece of parchment reveals a hidden passage beneath the tavern cellar.'
  )
  await page.locator('.secret-form button', { hasText: 'Save' }).click()

  // Secret should appear in the Hidden section
  await expect(page.locator('.secret-item--hidden')).toBeVisible({ timeout: 5_000 })
  await expect(page.locator('.secret-item--hidden .secret-title')).toContainText('The Hidden Map Fragment')

  // Expand the secret card
  await page.locator('.secret-item--hidden .secret-header').click()

  // Reveal it — this triggers the WS secret_revealed event → handout modal
  await page.locator('.secret-item--hidden button', { hasText: 'Reveal' }).click()

  // Handout modal should appear
  await expect(page.getByRole('dialog', { name: 'The Hidden Map Fragment' })).toBeVisible({ timeout: 5_000 })
  await expect(page.locator('.handout-modal-content')).toContainText('hidden passage')

  // Dismiss via the × button
  await page.locator('.handout-modal-close').click()
  await expect(page.getByRole('dialog', { name: 'The Hidden Map Fragment' })).not.toBeVisible({ timeout: 3_000 })
})

test('Push button re-triggers handout modal for already-revealed secret', async ({ page }) => {
  await page.goto('/')
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })

  // Navigate to Secrets
  await page.locator('.workspace-desktop-nav').getByRole('button', { name: 'Secrets', exact: true }).click()
  await expect(page.locator('.secrets-panel')).toBeVisible({ timeout: 5_000 })

  // Switch to "Revealed" filter so we see the secret from the prior test
  await page.locator('.secrets-filter-btn', { hasText: /Revealed/ }).click()
  await expect(page.locator('.secret-item--revealed')).toBeVisible({ timeout: 5_000 })

  // Expand the revealed secret card so the Push button appears
  await page.locator('.secret-item--revealed .secret-header').first().click()
  await expect(page.locator('.secret-item--revealed .secret-body')).toBeVisible({ timeout: 3_000 })

  // The Push button re-fires the WS event
  await page.locator('.secret-item--revealed button', { hasText: 'Push' }).click()

  await expect(page.getByRole('dialog', { name: 'The Hidden Map Fragment' })).toBeVisible({ timeout: 5_000 })

  // Dismiss via the × close button
  await page.locator('.handout-modal-close').click()
  await expect(page.getByRole('dialog', { name: 'The Hidden Map Fragment' })).not.toBeVisible({ timeout: 3_000 })
})

// ── Test 5: Map FX API endpoint (Group C feature) ────────────────────────────

test('map FX endpoint rejects invalid effect', async ({ request }) => {
  const res = await request.post(`${BASE}/api/maps/1/fx`, {
    data: { effect: '', x: 0.5, y: 0.5 },
  })
  expect(res.status()).toBe(400)
})

test('map FX endpoint rejects out-of-range coordinates', async ({ request }) => {
  const res = await request.post(`${BASE}/api/maps/1/fx`, {
    data: { effect: 'fire', x: 1.5, y: 0.5 },
  })
  expect(res.status()).toBe(400)
})

// ── Test 6: Right-panel destination navigation ───────────────────────────────

test('navigates all right-panel destinations without crashing', async ({ page }) => {
  await page.goto('/')
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })

  const destinations = ['Notes', 'NPCs', 'Objectives', 'Oracle', 'Secrets', 'Handouts', 'Journal']
  for (const destination of destinations) {
    const btn = page.locator('.workspace-desktop-nav').getByRole('button', { name: destination, exact: true })
    // Some destinations may not exist in all rulesets — skip if absent
    if ((await btn.count()) === 0) continue
    await btn.click()
    // Just verify no error overlay appears
    await expect(page.locator('.error')).toHaveCount(0)
    await page.waitForTimeout(200)
  }

  // Return to Notes — world notes search input should be visible
  await page.locator('.workspace-desktop-nav').getByRole('button', { name: 'Notes', exact: true }).click()
  await expect(page.locator('.notes-search')).toBeVisible({ timeout: 3_000 })
})

// ── Test 7: NPC creation via the NPCs panel ───────────────────────────────────

test('creates an NPC via the right-panel', async ({ page }) => {
  await page.goto('/')
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })

  await page.locator('.workspace-desktop-nav').getByRole('button', { name: 'NPCs', exact: true }).click()
  await expect(page.locator('.npcs-panel, .npc-list, [class*="npc"]').first()).toBeVisible({ timeout: 5_000 })

  // Add a new NPC via the API directly (avoids UI differences across rulesets)
  const ok = await page.evaluate(async ({ sid }) => {
    const res = await fetch(`/api/sessions/${sid}/npcs`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'Gareth the Barkeep', note: 'A gruff but fair innkeeper.' }),
    })
    return res.ok
  }, { sid: sessionId })
  expect(ok).toBe(true)

  // Reload the page — NPC should appear after the WS/reload cycle
  await page.reload()
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })
  await page.locator('.workspace-desktop-nav').getByRole('button', { name: 'NPCs', exact: true }).click()
  await expect(page.locator('body')).toContainText('Gareth the Barkeep', { timeout: 5_000 })
})

// ── Test 8: Theme toggle ──────────────────────────────────────────────────────

test('theme toggle switches between worn-grimoire and parchment', async ({ page }) => {
  await page.goto('/')
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })

  const html = page.locator('html')
  const initial = await html.getAttribute('data-theme')

  await page.locator('.h-theme').click()
  const toggled = await html.getAttribute('data-theme')
  expect(toggled).not.toBe(initial)

  // Toggle back
  await page.locator('.h-theme').click()
  expect(await html.getAttribute('data-theme')).toBe(initial)
})

// ── Test 9: Roll all standard dice expressions via API ────────────────────────

test('all standard dice expressions roll successfully', async ({ page }) => {
  await page.goto('/')
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })

  // Roll each standard die and verify a result comes back via the live feed
  for (const expr of ['1d4', '1d6', '1d8', '1d10', '1d12', '1d20']) {
    const ok = await page.evaluate(async ({ sid, expression }) => {
      const res = await fetch(`/api/sessions/${sid}/dice-rolls`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ expression }),
      })
      return res.ok
    }, { sid: sessionId, expression: expr })
    expect(ok, `${expr} roll API call failed`).toBe(true)
    await page.waitForTimeout(100)
  }

  // After 6 rolls the dice feed should be visible with results
  await expect(page.locator('.dice-compact')).toBeVisible({ timeout: 3_000 })
})

// ── Test 10: Manage panel opens and closes ────────────────────────────────────

test('manage panel opens, shows tabs, and closes', async ({ page }) => {
  await page.goto('/')
  await page.locator('.workspace-body').waitFor({ timeout: 10_000 })

  // The "⚙ Manage" button (not "🎭 GM Screen") opens the campaign manage panel
  await page.locator('button.h-manage[title*="Manage campaigns"]').click()
  await expect(page.getByRole('dialog', { name: 'Manage Campaign' })).toBeVisible({ timeout: 5_000 })

  // The manage panel should show the existing campaign
  await expect(page.locator('.manage-panel')).toContainText('E2E Smoke Campaign')

  // Close via the × button inside the panel
  await page.locator('.manage-close').click()
  await expect(page.getByRole('dialog', { name: 'Manage Campaign' })).not.toBeVisible({ timeout: 3_000 })
})
