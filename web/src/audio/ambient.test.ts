import { afterEach, describe, expect, it } from 'vitest'
import { setAmbientTrack, stopAmbient } from './ambient'

afterEach(() => {
  stopAmbient()
})

describe('ambient audio embed', () => {
  it('uses the YouTube origin allowed by the server content security policy', async () => {
    await setAmbientTrack('tavern')

    const iframe = document.getElementById('yt-ambient') as HTMLIFrameElement | null
    expect(iframe).not.toBeNull()
    expect(new URL(iframe!.src).origin).toBe('https://www.youtube.com')
    expect(iframe!.allow).toBe('autoplay')
  })
})
