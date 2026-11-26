// Opt-in acceptance on a private, disposable campaign copy with the real GM.
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { chromium } from '@playwright/test'

const { origin, campaign, session, actions } = JSON.parse(await readFile(process.argv[2], 'utf8'))
assert.ok(origin && campaign && session && process.env.TTRPG_AUTH_SECRET)
assert.ok(actions.length === 30 && actions.every(a => typeof a === 'string' && a.trim()))
assert.equal(new Set(actions).size, actions.length, 'Each fixture action must identify one turn')
const browser = await chromium.launch()
try {
  const page = await browser.newPage({ ignoreHTTPSErrors: true })
  await page.goto(origin)
  await page.getByLabel('Master secret').fill(process.env.TTRPG_AUTH_SECRET)
  await page.getByRole('button', { name: 'Unlock' }).click()
  const input = page.getByPlaceholder('What do you do?')
  await input.waitFor()
  const context = await page.evaluate(() => fetch('/api/context').then(r => r.json()))
  assert.equal(context.campaign.id, campaign)
  assert.equal(context.session.id, session)
  async function messages() {
    const response = await page.request.get(new URL('/api/sessions/' + session + '/messages', origin).href)
    assert.ok(response.ok())
    return (await response.json()).filter(m => !m.whisper)
  }
  for (const [index, action] of actions.entries()) {
    const before = await messages()
    const saved = before.findIndex(m => m.role === 'user' && m.content === action)
    if (saved >= 0) {
      assert.equal(before[saved + 1]?.role, 'assistant', 'Previously interrupted turn ' + (index + 1))
      continue
    }
    const reply = page.waitForResponse(r => new URL(r.url()).pathname === '/api/sessions/' + session + '/gm-respond-stream')
    await input.fill(action)
    await input.press('Enter')
    assert.ok((await reply).ok())
    const deadline = Date.now() + 600000
    let player, gm
    while (Date.now() < deadline) {
      const history = await messages()
      const sent = history.findIndex(m => m.role === 'user' && m.content === action)
      if (sent >= 0 && history[sent + 1]?.role === 'assistant') {
        player = history[sent]
        gm = history[sent + 1]
        break
      }
      await new Promise(resolve => setTimeout(resolve, 1000))
    }
    assert.ok(gm?.content.trim().length > 30, 'Missing persisted GM reply on turn ' + (index + 1))
    const excerpt = gm.content.split('\n').find(l => l.trim()).replace(/[*_`#]/g, '').trim().slice(0, 30)
    await page.getByText(excerpt, { exact: false }).last().waitFor({ timeout: 180000 })
    console.log('Turn %d saved and rendered (messages %d and %d)', index + 1, player.id, gm.id)
  }
  await page.reload()
  await input.waitFor()
  const history = await messages()
  let previous = -1
  for (const action of actions) {
    const i = history.findIndex(m => m.role === 'user' && m.content === action)
    assert.ok(i > previous && history[i + 1]?.role === 'assistant' && history[i + 1].content.trim())
    const excerpt = history[i + 1].content.trim().replace(/[*_`#]/g, '').slice(0, 30)
    await page.getByText(excerpt, { exact: false }).last().waitFor()
    previous = i
  }
  console.log('Verified all 30 player and GM pairs after reload')
} finally {
  await browser.close()
}
