import { useEffect, useState } from 'react'

export function useMobileWorkspace(override?: boolean): boolean {
  const query = '(max-width: 599px)'
  const [mobile, setMobile] = useState(() => override ?? (typeof window.matchMedia === 'function' && window.matchMedia(query).matches))
  useEffect(() => {
    if (override !== undefined || typeof window.matchMedia !== 'function') return
    const media = window.matchMedia(query)
    const change = () => setMobile(media.matches)
    media.addEventListener('change', change)
    return () => media.removeEventListener('change', change)
  }, [override])
  return override ?? mobile
}
