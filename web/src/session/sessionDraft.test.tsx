import { act, renderHook } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import { acknowledgeSubmission, updateDraft, useSessionDraft } from './sessionDraft'
afterEach(() => localStorage.clear())

it('keeps private drafts and unresolved submissions in their originating session after reopening', () => {
  const item = {key:'stable-key',text:'Whisper softly',fromComposer:true,whisper:true,character:9}
  const hook = renderHook(({id}) => useSessionDraft(id), {initialProps:{id:1}})
  act(() => updateDraft(1, () => ({text:item.text, whisper:true,submission:item})))
  hook.rerender({id:2})
  expect(hook.result.current).toEqual({text:'',whisper:false})
  act(() => updateDraft(2, () => ({text:'A different action',whisper:false})))
  hook.unmount()
  const reopened=renderHook(() => useSessionDraft(1))
  expect(reopened.result.current.submission?.key).toBe(item.key)
  act(() => acknowledgeSubmission(1,item.key))
  expect(reopened.result.current.text).toBe('')
  expect(renderHook(() => useSessionDraft(2)).result.current.text).toBe('A different action')
})

it('preserves edits made while a prior submission was being acknowledged', () => {
  const item={key:'old',text:'first draft',whisper:false,fromComposer:true,character:null}
  updateDraft(1, () => ({text:'new draft',whisper:false,submission:item}))
  acknowledgeSubmission(1,item.key)
  expect(renderHook(() => useSessionDraft(1)).result.current).toEqual({text:'new draft',whisper:false})
})
