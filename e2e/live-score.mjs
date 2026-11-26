// Opt-in acceptance on a disposable deployment with real GM and Syrinscape services.
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { chromium } from '@playwright/test'

const origin = process.env.INKANDBONE_E2E_BASE_URL
const secret = process.env.TTRPG_AUTH_SECRET
const fixtures = JSON.parse(await readFile(process.argv[2], 'utf8'))
assert.ok(origin && secret && fixtures.length, 'Supply a disposable game URL, login and fixture list')

const browser = await chromium.launch({ args: ['--autoplay-policy=user-gesture-required'] })
try {
  const page = await browser.newPage({ ignoreHTTPSErrors: true })
  await page.addInitScript(() => {
    window.sceneAudioPeak = 0
    const connect = AudioNode.prototype.connect
    AudioNode.prototype.connect = function (destination, ...args) {
      if (destination instanceof AudioDestinationNode && this.context instanceof AudioContext) {
        const meter = this.context.createAnalyser()
        connect.call(this, meter)
        const data = new Float32Array(meter.fftSize)
        setInterval(() => {
          meter.getFloatTimeDomainData(data)
          for (const v of data) window.sceneAudioPeak = Math.max(window.sceneAudioPeak, Math.abs(v))
        }, 90)
      }
      return connect.call(this, destination, ...args)
    }
  })
  const operation = (name, session, timeout = 120000) => page.waitForResponse(r =>
    new URL(r.url()).pathname === '/sound-test/live/' + name &&
    r.request().postDataJSON()?.session === session && r.ok(), { timeout })

  await page.goto(origin)
  await page.getByLabel('Master secret').fill(secret)
  await page.getByRole('button', { name: 'Unlock' }).click()
  for (const table of fixtures) {
    const { cid, sid, rule, title, action } = table
    const context = () => page.evaluate(() => fetch('/api/context').then(r => r.json()))
    const current = await context()
    const resume = operation('resume', sid)
    if (current.session?.id === sid) {
      await page.reload()
      await page.getByPlaceholder('What do you do?').click()
    } else {
      await page.getByRole('button', { name: 'Manage', exact: true }).click()
      const dialog = page.getByRole('dialog')
      await dialog.getByRole('button', { name: 'Sessions', exact: true }).click()
      await dialog.locator('select').selectOption(String(cid))
      await dialog.locator('.manage-row').filter({ hasText: title })
        .getByRole('button', { name: 'Set Active', exact: true }).click()
    }
    await resume
    const active = await context()
    assert.equal(active.live_score, true)
    assert.equal(active.campaign.id, cid)
    assert.equal(active.session.id, sid)
    assert.equal(active.score_ruleset, rule)
    assert.equal(await page.locator('#yt-ambient').count(), 0)
    if (action) {
      const input = page.getByPlaceholder('What do you do?')
      const plan = operation('plan', sid, 180000)
      const heard = operation('heard', sid, 180000)
      await input.fill(action)
      await input.press('Enter')
      const [planned] = await Promise.all([plan, heard])
      assert.ok((await planned.json()).cues.length, 'Expected an audible transition')
      await page.waitForFunction(() => !document.querySelector('.player-input-field')?.disabled)
      const after = await context()
      assert.ok(after.recent_messages.some(m => m.role === 'user' && m.content === action))
      assert.equal(after.recent_messages.at(-1).role, 'assistant')
    }
    await page.evaluate(() => { window.sceneAudioPeak = 0 })
    await page.waitForFunction(() => window.sceneAudioPeak > 0.0005, null, { timeout: 45000 })
    console.log('%s: campaign %d, session %d, audible%s', rule, cid, sid, action ? ' after GM turn' : '')
  }
} finally {
  await browser.close()
}
