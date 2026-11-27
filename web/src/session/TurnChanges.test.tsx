import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { TurnChanges } from './TurnChanges'
import { refreshTurnChanges } from './turnChangeStore'

afterEach(() => { cleanup(); vi.unstubAllGlobals() })

it('loads one session once, shows each GM turn separately and undoes a single entry', async () => {
  const rows = [
    { id: 1, message_id: 4, kind: 'stat', field: 'hp', operation: 'set', label: 'hp', before: 10, after: 8, can_undo: true },
    { id: 2, message_id: 5, kind: 'item', operation: 'add', label: 'Brass key', after: { quantity: 1 }, can_undo: true },
  ]
  const repaired = { ...rows[0], corrected_at: 'today', can_undo: false }
  const fetcher = vi.fn()
    .mockResolvedValueOnce(Response.json(rows))
    .mockResolvedValueOnce(Response.json(repaired))
    .mockResolvedValue(Response.json([repaired, rows[1]]))
  vi.stubGlobal('fetch', fetcher)
  render(<><TurnChanges sessionId={3} messageId={4} /><TurnChanges sessionId={3} messageId={5} /></>)
  await screen.findByText('Brass key')
  expect(fetcher).toHaveBeenCalledTimes(1)
  fireEvent.click(screen.getByRole('button', { name: 'Undo hp' }))
  await screen.findByText('Undone')
  expect(fetcher).toHaveBeenCalledWith('/api/sessions/3/changes/1/undo', { method: 'POST' })
  expect(screen.getByRole('button', { name: 'Undo Brass key' })).toBeInTheDocument()
})

it('refreshes after automation and shows a safe conflict',async () => {
  const row={id:8,message_id:4,kind:'currency',operation:'set',label:'Gold',before:1,after:4,can_undo:true}
  const fetcher=vi.fn().mockResolvedValueOnce(Response.json([])).mockResolvedValueOnce(Response.json([row]))
    .mockResolvedValueOnce(new Response('Later edits prevent undoing this change.',{status:409}))
    .mockResolvedValueOnce(Response.json([{...row,can_undo:false,superseded:true}]))
  vi.stubGlobal('fetch',fetcher)
  render(<TurnChanges sessionId={7} messageId={4}/>)
  await waitFor(()=>expect(fetcher).toHaveBeenCalledTimes(1))
  act(()=>refreshTurnChanges())
  fireEvent.click(await screen.findByRole('button',{name:'Undo Gold'}))
  await screen.findByText('Changed later')
  expect(screen.queryByRole('button',{name:'Undo Gold'})).toBeNull()
})
