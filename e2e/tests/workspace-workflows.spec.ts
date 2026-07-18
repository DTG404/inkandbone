import { readFile } from 'node:fs/promises'
import { expect, test } from '@playwright/test'

test.describe.configure({ mode: 'serial' })

const BASE = process.env.INKANDBONE_E2E_BASE_URL
if (!BASE) throw new Error('workspace workflow tests require INKANDBONE_E2E_BASE_URL')

let campaignId: number
let characterId: number
let sessionId: number
let duneCampaignId: number
let duneCharacterId: number
let duneSessionId: number

test.beforeAll(async ({ playwright }) => {
  const api = await playwright.request.newContext({ baseURL: BASE })
  const rulesets = await (await api.get('/api/rulesets')).json() as Array<{ id: number; name: string }>
  const dnd5e = rulesets.find((ruleset) => ruleset.name === 'dnd5e')
  const dune = rulesets.find((ruleset) => ruleset.name === 'dune')
  if (!dnd5e || !dune) throw new Error('required dnd5e or dune ruleset missing')

  campaignId = (await (await api.post('/api/campaigns', {
    data: { name: 'Maintained Workflow Campaign', ruleset_id: dnd5e.id },
  })).json()).id
  characterId = (await (await api.post(`/api/campaigns/${campaignId}/characters`, {
    data: { name: 'Workflow Hero' },
  })).json()).id
  sessionId = (await (await api.post(`/api/campaigns/${campaignId}/sessions`, {
    data: { title: 'Maintained Workflow Session', date: '2026-07-18' },
  })).json()).id
  await api.post(`/api/sessions/${sessionId}/messages`, {
    data: { role: 'user', content: 'The amber lantern marks the hidden passage.', character_id: characterId },
  })
  await api.post(`/api/sessions/${sessionId}/messages`, {
    data: { role: 'user', content: 'A decoy message should disappear from filtered results.', character_id: characterId },
  })

  duneCampaignId = (await (await api.post('/api/campaigns', {
    data: { name: 'Maintained Dune Skill Campaign', ruleset_id: dune.id },
  })).json()).id
  duneCharacterId = (await (await api.post(`/api/campaigns/${duneCampaignId}/characters`, {
    data: { name: 'Liet Swordmaster' },
  })).json()).id
  duneSessionId = (await (await api.post(`/api/campaigns/${duneCampaignId}/sessions`, {
    data: { title: 'Maintained Dune Skill Session', date: '2026-07-18' },
  })).json()).id
  await api.patch(`/api/characters/${duneCharacterId}`, {
    data: { data_json: JSON.stringify({ battle: 4, communicate: 2, discipline: 3, move: 3, understand: 2 }) },
  })

  await api.patch('/api/settings', {
    data: { campaign_id: campaignId, character_id: characterId, session_id: sessionId },
  })
  await api.dispose()
})

test('XP log supports semantically named add and delete controls', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'XP Log' })).toBeVisible()

  await page.getByPlaceholder('Note…').fill('Protected the caravan')
  await page.getByPlaceholder('XP').fill('25')
  await page.getByRole('button', { name: 'Add XP entry' }).click()
  const entry = page.locator('.xp-log-entry').filter({ hasText: 'Protected the caravan' })
  await expect(entry).toContainText('+25')

  await entry.getByRole('button', { name: 'Delete XP entry: Protected the caravan' }).click()
  await expect(entry).toHaveCount(0)
})

test('story search, whisper privacy, and export form one maintained workflow', async ({ page, request }) => {
  await page.goto('/')
  await page.getByPlaceholder('Search story…').fill('amber lantern')
  await expect(page.getByText('The amber lantern marks the hidden passage.')).toBeVisible()
  await expect(page.getByText('A decoy message should disappear from filtered results.')).toHaveCount(0)
  await page.getByRole('button', { name: '×' }).click()

  await page.getByRole('button', { name: 'Enable whisper mode' }).click()
  await page.getByPlaceholder('Whisper (private, no GM response)…').fill('WHISPER_EXPORT_SENTINEL')
  await page.locator('.player-input-field').press('Enter')
  await expect(page.getByText('WHISPER_EXPORT_SENTINEL')).toBeVisible()

  const messages = await (await request.get(`${BASE}/api/sessions/${sessionId}/messages`)).json() as Array<{ content: string; whisper?: boolean }>
  expect(messages.find((message) => message.content === 'WHISPER_EXPORT_SENTINEL')?.whisper).toBe(true)

  const downloadPromise = page.waitForEvent('download')
  await page.getByRole('button', { name: 'Export session' }).click()
  const download = await downloadPromise
  const path = await download.path()
  if (!path) throw new Error('export download path missing')
  const markdown = await readFile(path, 'utf8')
  expect(markdown).toContain('The amber lantern marks the hidden passage.')
  expect(markdown).not.toContain('WHISPER_EXPORT_SENTINEL')
})

test('journal, player history, talents, GM screen, and automation remain reachable', async ({ page }) => {
  await page.goto('/')

  await page.locator('.workspace-desktop-nav').getByRole('button', { name: 'Journal', exact: true }).click()
  await page.getByRole('button', { name: 'Timeline', exact: true }).click()
  await expect(page.getByText('Session Timeline')).toBeVisible()

  await page.getByRole('button', { name: 'Your actions' }).click()
  await expect(page.getByRole('dialog', { name: 'Your Actions' })).toBeVisible()
  await page.keyboard.press('Escape')

  await page.getByRole('button', { name: 'Character talents & psychic powers' }).click()
  await expect(page.getByRole('dialog', { name: 'Talents & Powers' })).toBeVisible()
  await page.keyboard.press('Escape')

  await page.getByRole('button', { name: 'GM Screen' }).click()
  await expect(page.getByRole('dialog', { name: 'GM Screen' })).toBeVisible()
  await page.keyboard.press('Escape')

  await page.getByRole('button', { name: 'Manage' }).click()
  await page.getByRole('button', { name: 'Automation', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Automation' })).toBeVisible()
  await expect(page.getByText('Extract NPCs', { exact: true })).toBeVisible()
  await expect(page.getByRole('checkbox', { name: 'Enable Extract NPCs' })).toBeAttached()
})

test('character attribute click-to-roll records the maintained action shape', async ({ page, request }) => {
  await page.goto('/')
  const attribute = page.locator('button.attr-row[title^="Click to roll"]').first()
  await expect(attribute).toBeVisible()
  await attribute.click()

  await expect.poll(async () => {
    const messages = await (await request.get(`${BASE}/api/sessions/${sessionId}/messages`)).json() as Array<{ role: string; content: string }>
    return messages.filter((message) => message.role === 'user').at(-1)?.content
  }).toMatch(/attempts a .+ check\./)
  await expect.poll(() => page.evaluate(() => localStorage.getItem('inkandbone_roll_hint_shown'))).toBe('1')
})

test('Dune skill click-to-roll records the maintained action shape', async ({ page, request }) => {
  await request.patch(`${BASE}/api/settings`, {
    data: { campaign_id: duneCampaignId, character_id: duneCharacterId, session_id: duneSessionId },
  })
  await page.goto('/')

  const skill = page.locator('label[title^="Click to roll"]').first()
  await expect(skill).toBeVisible()
  await skill.click()

  await expect.poll(async () => {
    const messages = await (await request.get(`${BASE}/api/sessions/${duneSessionId}/messages`)).json() as Array<{ role: string; content: string }>
    return messages.filter((message) => message.role === 'user').at(-1)?.content
  }).toMatch(/attempts a .+ check\./)

  await request.patch(`${BASE}/api/settings`, {
    data: { campaign_id: campaignId, character_id: characterId, session_id: sessionId },
  })
})

test('macro workflow fires, reorders, deletes, and enforces the ten-item cap', async ({ page, request }) => {
  for (const macro of [
    { label: 'Guard', action_text: 'I raise my shield.', color: '#2980b9' },
    { label: 'Slash', action_text: 'I slash at the nearest enemy.', color: '#c0392b' },
  ]) {
    expect((await request.post(`${BASE}/api/characters/${characterId}/macros`, { data: macro })).status()).toBe(201)
  }

  await page.goto('/')
  await page.getByRole('button', { name: 'Slash', exact: true }).click()
  await expect.poll(async () => {
    const messages = await (await request.get(`${BASE}/api/sessions/${sessionId}/messages`)).json() as Array<{ role: string; content: string }>
    return messages.filter((message) => message.role === 'user').at(-1)?.content
  }).toBe('I slash at the nearest enemy.')

  await page.locator('button[title="Edit macros"]').click()
  const slashSlot = page.locator('.macro-slot').filter({ hasText: 'Slash' })
  await slashSlot.locator('button[title="Move left"]').click()
  await expect.poll(async () => {
    const macros = await (await request.get(`${BASE}/api/characters/${characterId}/macros`)).json() as Array<{ label: string }>
    return macros.map((macro) => macro.label)
  }).toEqual(['Slash', 'Guard'])

  await page.locator('.macro-slot').filter({ hasText: 'Guard' }).locator('button[title="Delete"]').click()
  await expect(page.getByRole('button', { name: 'Guard', exact: true })).toHaveCount(0)

  let macros = await (await request.get(`${BASE}/api/characters/${characterId}/macros`)).json() as Array<{ id: number }>
  while (macros.length < 10) {
    const suffix = macros.length + 1
    expect((await request.post(`${BASE}/api/characters/${characterId}/macros`, {
      data: { label: `Macro ${suffix}`, action_text: `Action ${suffix}`, color: 'var(--gold)' },
    })).status()).toBe(201)
    macros = await (await request.get(`${BASE}/api/characters/${characterId}/macros`)).json() as Array<{ id: number }>
  }
  await page.reload()
  await expect(page.locator('button[title="Maximum 10 macros"]')).toBeDisabled()
})
