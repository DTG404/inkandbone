import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

test.describe.configure({ mode: 'serial' })

const VIEWPORTS = [
  { name: 'desktop-wide', width: 1440, height: 900 },
  { name: 'desktop-compact', width: 1024, height: 768 },
  { name: 'tablet', width: 768, height: 1024 },
  { name: 'mobile', width: 390, height: 844 },
  { name: 'mobile-narrow', width: 320, height: 568 },
] as const

const fixture = { campaignId: 0, characterId: 0, sessionId: 0 }

async function activateFixture(request: APIRequestContext) {
  if (!fixture.campaignId) {
    const rulesetResponse = await request.get('/api/rulesets')
    expect(rulesetResponse.ok()).toBe(true)
    const rulesets = await rulesetResponse.json() as Array<{ id: number }>
    expect(rulesets.length).toBeGreaterThan(0)
    const campaign = await request.post('/api/campaigns', {
      data: { name: 'Phase 3 Responsive Chronicle', description: 'Responsive gate', ruleset_id: rulesets[0].id },
    })
    expect(campaign.status()).toBe(201)
    fixture.campaignId = (await campaign.json() as { id: number }).id
    const character = await request.post(`/api/campaigns/${fixture.campaignId}/characters`, { data: { name: 'Viewport Hero' } })
    expect(character.status()).toBe(201)
    fixture.characterId = (await character.json() as { id: number }).id
    const deterministicCharacter = await request.patch(`/api/characters/${fixture.characterId}`, {
      data: { data_json: '{}' },
    })
    expect(deterministicCharacter.ok()).toBe(true)
    const session = await request.post(`/api/campaigns/${fixture.campaignId}/sessions`, {
      data: { title: 'Viewport Session', date: '2026-07-18' },
    })
    expect(session.status()).toBe(201)
    fixture.sessionId = (await session.json() as { id: number }).id
  }
  const settings = await request.patch('/api/settings', {
    data: { campaign_id: fixture.campaignId, character_id: fixture.characterId, session_id: fixture.sessionId },
  })
  expect(settings.ok()).toBe(true)
}

async function settleVisualState(page: Page, theme: 'worn-grimoire' | 'parchment') {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await page.goto('/')
  await page.evaluate((value) => {
    localStorage.setItem('theme', value)
    localStorage.removeItem('inkandbone.workspace.v1')
  }, theme)
  await page.reload()
  await expect(page.locator('.grimoire')).toBeVisible()
  await expect(page.getByRole('status', { name: 'Connection status' })).toHaveCount(0, { timeout: 10_000 })
  await page.addStyleTag({
    content: '*, *::before, *::after { animation: none !important; transition: none !important; caret-color: transparent !important; }',
  })
  await page.evaluate(() => document.fonts.ready)
}

test('tablet keeps Story unobstructed and opens one accessible workspace drawer at a time', async ({ page, request }) => {
  await activateFixture(request)
  await page.setViewportSize({ width: 768, height: 1024 })
  await page.goto('/')
  await expect(page.locator('.grimoire')).toBeVisible()

  const story = page.locator('.workspace-story')
  const body = page.locator('.workspace-body')
  await expect(story).toBeVisible()
  await expect(page.locator('.workspace-left')).toBeHidden()
  await expect(page.locator('.workspace-right')).toBeHidden()
  const storyBox = await story.boundingBox()
  const bodyBox = await body.boundingBox()
  expect(storyBox).not.toBeNull()
  expect(bodyBox).not.toBeNull()
  expect(storyBox!.width).toBe(bodyBox!.width)

  const characterOpener = page.getByRole('button', { name: 'Open character drawer' })
  const toolsOpener = page.getByRole('button', { name: 'Open tools drawer' })
  await expect(characterOpener).toBeVisible()
  await expect(toolsOpener).toBeVisible()

  await characterOpener.focus()
  await page.keyboard.press('Enter')
  const characterDrawer = page.getByRole('dialog', { name: 'Character drawer' })
  await expect(characterDrawer).toBeVisible()
  await expect(characterDrawer.locator(':focus')).toHaveCount(1)
  await expect(page.getByRole('dialog', { name: 'Tools drawer' })).toHaveCount(0)
  await expect(page.getByRole('dialog')).toHaveCount(1)
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog', { name: 'Character drawer' })).toHaveCount(0)
  await expect(characterOpener).toBeFocused()

  await toolsOpener.focus()
  await page.keyboard.press('Enter')
  const toolsDrawer = page.getByRole('dialog', { name: 'Tools drawer' })
  await expect(toolsDrawer).toBeVisible()
  await expect(toolsDrawer.locator(':focus')).toHaveCount(1)
  await expect(toolsDrawer.getByRole('region', { name: 'Notes', exact: true })).toBeVisible()
  await expect(page.getByRole('dialog', { name: 'Character drawer' })).toHaveCount(0)
  await expect(page.getByRole('dialog')).toHaveCount(1)

  for (const control of await toolsDrawer.locator('button[aria-controls="workspace-active-panel"]').all()) {
    const targetId = await control.getAttribute('aria-controls')
    expect(targetId).toBeTruthy()
    await expect(page.locator(`#${targetId}`)).toHaveCount(1)
  }
  await page.keyboard.press('Escape')
  await expect(toolsDrawer).toHaveCount(0)
  await expect(toolsOpener).toBeFocused()
})

test('persisted desktop collapse reclaims Story width without suppressing 390px destinations', async ({ page, request }) => {
  await activateFixture(request)
  await page.addInitScript(() => {
    localStorage.setItem('inkandbone.workspace.v1', JSON.stringify({
      leftWidth: 420, rightWidth: 320, leftCollapsed: true, rightCollapsed: true,
    }))
  })
  await page.setViewportSize({ width: 1440, height: 900 })
  await page.goto('/')
  const desktopBody = await page.locator('.workspace-body').boundingBox()
  const desktopStory = await page.locator('.workspace-story').boundingBox()
  expect(desktopBody).not.toBeNull()
  expect(desktopStory).not.toBeNull()
  expect(desktopStory!.width).toBeGreaterThanOrEqual(desktopBody!.width - 2)

  await page.setViewportSize({ width: 390, height: 844 })
  await page.reload()
  const destinations = page.getByRole('navigation', { name: 'Workspace destinations' })
  await destinations.getByRole('button', { name: 'Character' }).click()
  await expect(page.locator('.sidebar-left')).toBeVisible()
  await destinations.getByRole('button', { name: 'World' }).click()
  await expect(page.locator('.sidebar-right')).toBeVisible()
  await destinations.getByRole('button', { name: 'GM' }).click()
  await expect(page.locator('.sidebar-right')).toBeVisible()
})

test.describe('coarse pointer sizing', () => {
  test.use({ hasTouch: true, viewport: { width: 1024, height: 768 } })

  test('legacy attribute, faction, and calendar controls meet label and 44px target floors', async ({ page, request }) => {
    await activateFixture(request)
    const factions = await request.get(`/api/campaigns/${fixture.campaignId}/factions`)
    expect(factions.ok()).toBe(true)
    if ((await factions.json() as unknown[]).length === 0) {
      const created = await request.post(`/api/campaigns/${fixture.campaignId}/factions`, {
        data: { name: 'Sizing Guild', faction_type: 'guild', description: '', influence: 5, color: '#996633' },
      })
      expect(created.ok()).toBe(true)
    }
    await page.goto('/')

    async function expectTarget(selector: string) {
      const control = page.locator(selector).first()
      await expect(control).toBeVisible()
      const box = await control.boundingBox()
      expect(box).not.toBeNull()
      expect(box!.width).toBeGreaterThanOrEqual(44)
      expect(box!.height).toBeGreaterThanOrEqual(44)
    }
    async function expectLabel(selector: string) {
      const label = page.locator(selector).first()
      await expect(label).toBeVisible()
      expect(await label.evaluate((element) => Number.parseFloat(getComputedStyle(element).fontSize))).toBeGreaterThanOrEqual(12)
    }

    const characterStat = page.getByLabel('Hunt')
    await expect(characterStat).toBeVisible()
    const characterStatBox = await characterStat.boundingBox()
    expect(characterStatBox).not.toBeNull()
    expect(characterStatBox!.width).toBeGreaterThanOrEqual(44)
    expect(characterStatBox!.height).toBeGreaterThanOrEqual(44)
    expect(await characterStat.evaluate((element) => Number.parseFloat(getComputedStyle(element.parentElement!).fontSize))).toBeGreaterThanOrEqual(12)
    await page.locator('.workspace-desktop-nav button').filter({ hasText: /^Factions$/ }).click()
    await expectTarget('.faction-header')
    await expectLabel('.faction-type-badge')
    await page.locator('.workspace-desktop-nav button').filter({ hasText: /^Calendar$/ }).click()
    await expectTarget('.calendar-advance-btn')
    await expectLabel('.calendar-date-label')
  })
})

for (const viewport of VIEWPORTS) {
  test(`${viewport.name} keeps story controls and destinations inside the viewport`, async ({ page, request }) => {
    await activateFixture(request)
    await page.setViewportSize(viewport)
    await page.goto('/')
    await expect(page.locator('.grimoire')).toBeVisible()

    const input = page.locator('.player-input-field')
    await expect(input).toBeVisible()
    const inputBox = await input.boundingBox()
    expect(inputBox).not.toBeNull()
    expect(inputBox!.x).toBeGreaterThanOrEqual(0)
    expect(inputBox!.x + inputBox!.width).toBeLessThanOrEqual(viewport.width)
    expect(inputBox!.y + inputBox!.height).toBeLessThanOrEqual(viewport.height)

    if (viewport.width < 600) {
      const storyBox = await page.locator('.workspace-story').boundingBox()
      expect(storyBox).not.toBeNull()
      expect(storyBox!.width).toBeGreaterThanOrEqual(viewport.width - 1)
      const destinations = page.getByRole('navigation', { name: 'Workspace destinations' })
      for (const label of ['Story', 'Character', 'World', 'GM']) {
        await expect(destinations.getByRole('button', { name: label })).toBeVisible()
      }
    } else {
      await expect(page.getByRole('navigation', { name: 'Workspace panels' })).toBeVisible()
    }

    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(
      await page.evaluate(() => document.documentElement.clientWidth),
    )
  })
}

for (const visual of [
  { name: 'dark-desktop', theme: 'worn-grimoire' as const, width: 1440, height: 900 },
  { name: 'light-desktop', theme: 'parchment' as const, width: 1440, height: 900 },
  { name: 'dark-mobile', theme: 'worn-grimoire' as const, width: 390, height: 844 },
  { name: 'light-mobile', theme: 'parchment' as const, width: 390, height: 844 },
]) {
  test(`${visual.name} visual baseline`, async ({ page, request }) => {
    await activateFixture(request)
    await page.setViewportSize({ width: visual.width, height: visual.height })
    await settleVisualState(page, visual.theme)
    await expect(page).toHaveScreenshot(`${visual.name}.png`, {
      animations: 'disabled',
      caret: 'hide',
      maxDiffPixelRatio: 0.0001,
      scale: 'css',
    })
  })
}
