import AxeBuilder from '@axe-core/playwright'
import { expect, test, type APIRequestContext, type Page } from '@playwright/test'

test.describe.configure({ mode: 'serial' })

const VIEWPORTS = [
  { name: 'desktop-wide', width: 1440, height: 900 },
  { name: 'desktop-compact', width: 1024, height: 768 },
  { name: 'tablet', width: 768, height: 1024 },
  { name: 'mobile', width: 390, height: 844 },
  { name: 'mobile-narrow', width: 320, height: 568 },
] as const

const fixture = {
  campaignId: 0,
  characterId: 0,
  sessionId: 0,
  campaignName: 'Phase 3 Accessible Chronicle',
  sessionName: 'Keyboard Session',
}

async function clearActiveContext(request: APIRequestContext) {
  const response = await request.patch('/api/settings', {
    data: { campaign_id: 0, character_id: 0, session_id: 0 },
  })
  expect(response.ok()).toBe(true)
}

async function provisionFixture(request: APIRequestContext) {
  if (fixture.campaignId) return
  const rulesetResponse = await request.get('/api/rulesets')
  expect(rulesetResponse.ok()).toBe(true)
  const rulesets = await rulesetResponse.json() as Array<{ id: number }>
  expect(rulesets.length).toBeGreaterThan(0)

  const campaignResponse = await request.post('/api/campaigns', {
    data: { name: fixture.campaignName, description: 'Phase 3 browser gate', ruleset_id: rulesets[0].id },
  })
  expect(campaignResponse.status()).toBe(201)
  fixture.campaignId = (await campaignResponse.json() as { id: number }).id

  const characterResponse = await request.post(`/api/campaigns/${fixture.campaignId}/characters`, {
    data: { name: 'Keyboard Hero' },
  })
  expect(characterResponse.status()).toBe(201)
  fixture.characterId = (await characterResponse.json() as { id: number }).id

  const sessionResponse = await request.post(`/api/campaigns/${fixture.campaignId}/sessions`, {
    data: { title: fixture.sessionName, date: '2026-07-18' },
  })
  expect(sessionResponse.status()).toBe(201)
  fixture.sessionId = (await sessionResponse.json() as { id: number }).id
}

async function activateFixture(request: APIRequestContext) {
  await provisionFixture(request)
  const response = await request.patch('/api/settings', {
    data: {
      campaign_id: fixture.campaignId,
      character_id: fixture.characterId,
      session_id: fixture.sessionId,
    },
  })
  expect(response.ok()).toBe(true)
}

async function disableAnimation(page: Page) {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await page.addStyleTag({
    content: '*, *::before, *::after { animation: none !important; transition: none !important; caret-color: transparent !important; }',
  })
}

test('first-run onboarding is complete and keyboard reachable', async ({ page, request }) => {
  await clearActiveContext(request)
  await page.setViewportSize({ width: 1440, height: 900 })
  await page.goto('/')

  await expect(page.getByRole('heading', { name: 'Begin your chronicle' })).toBeVisible()
  await expect(page.getByText('Create a campaign.')).toBeVisible()
  await expect(page.getByText('Create or select a character.')).toBeVisible()
  await expect(page.getByText('Create or select a session.')).toBeVisible()

  const openManagement = page.getByRole('button', { name: 'Open campaign management' })
  await openManagement.focus()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('dialog', { name: 'Manage Campaign' })).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(openManagement).toBeFocused()

  await provisionFixture(request)
})

test('campaign and session selection work without a pointer', async ({ page, request }) => {
  await clearActiveContext(request)
  await page.goto('/')
  const manage = page.getByRole('button', { name: /Manage$/ })
  await manage.focus()
  await page.keyboard.press('Enter')

  const campaignRow = page.getByRole('button', { name: `Select campaign ${fixture.campaignName}` })
  await campaignRow.focus()
  await page.keyboard.press('Enter')
  const setCampaign = page.getByRole('button', { name: 'Set Active' }).first()
  await setCampaign.focus()
  await page.keyboard.press('Enter')
  await expect(page.locator('.h-campaign')).toHaveText(fixture.campaignName)

  const manageAgain = page.getByRole('button', { name: /Manage$/ })
  await manageAgain.focus()
  await page.keyboard.press('Enter')
  const sessionsTab = page.getByRole('button', { name: 'Sessions' })
  await sessionsTab.focus()
  await page.keyboard.press('Enter')
  const setSession = page.getByRole('button', { name: 'Set Active' }).first()
  await setSession.focus()
  await page.keyboard.press('Enter')
  await expect(page.locator('.h-session')).toHaveText(fixture.sessionName)

  await activateFixture(request)
})

test('GM dialog traps keyboard focus, closes on Escape, and restores its opener', async ({ page, request }) => {
  await activateFixture(request)
  await page.goto('/')
  const opener = page.getByRole('button', { name: /GM Screen/ })
  await opener.focus()
  await page.keyboard.press('Enter')
  const dialog = page.getByRole('dialog', { name: 'GM Screen' })
  await expect(dialog).toBeVisible()
  await page.keyboard.press('Tab')
  await expect(dialog.locator(':focus')).toHaveCount(1)
  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()
  await expect(opener).toBeFocused()
})

for (const viewport of VIEWPORTS) {
  test(`${viewport.name} exposes primary navigation with no serious accessibility violations`, async ({ page, request }) => {
    await activateFixture(request)
    await page.setViewportSize(viewport)
    await page.goto('/')
    await disableAnimation(page)

    await expect(page.locator('.player-input-field')).toBeVisible()
    if (viewport.width < 600) {
      for (const destination of ['Story', 'Character', 'World', 'GM']) {
        await expect(page.getByRole('navigation', { name: 'Workspace destinations' }).getByRole('button', { name: destination })).toBeVisible()
      }
    } else {
      await expect(page.getByRole('navigation', { name: 'Workspace panels' })).toBeVisible()
    }
    const hasOverflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth)
    expect(hasOverflow).toBe(false)

    await page.getByRole('button', { name: /Manage$/ }).click()
    const dialog = page.getByRole('dialog', { name: 'Manage Campaign' })
    await expect(dialog).toBeVisible()
    const box = await dialog.boundingBox()
    expect(box).not.toBeNull()
    expect(box!.x).toBeGreaterThanOrEqual(0)
    expect(box!.y).toBeGreaterThanOrEqual(0)
    expect(box!.x + box!.width).toBeLessThanOrEqual(viewport.width)
    expect(box!.y + box!.height).toBeLessThanOrEqual(viewport.height)
    await page.keyboard.press('Escape')

    const results = await new AxeBuilder({ page }).analyze()
    const seriousOrCritical = results.violations.filter((violation) => (
      violation.impact === 'serious' || violation.impact === 'critical'
    ))
    expect(seriousOrCritical, JSON.stringify(seriousOrCritical, null, 2)).toEqual([])
  })
}
