import { useEffect, useRef, useState } from 'react'
import { ApiError } from '../api.ts'

// A null fetcher clears the result; responses from superseded runs are dropped.
export function useDebouncedQuery<T>(
  fetcher: (() => Promise<T>) | null,
  deps: unknown[],
  delay: number,
  fallbackError: string,
) {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const latestRun = useRef(0)

  useEffect(() => {
    const run = ++latestRun.current
    setData(null)
    setError('')
    if (!fetcher) {
      setLoading(false)
      return
    }
    setLoading(true)
    const timer = setTimeout(async () => {
      try {
        const result = await fetcher()
        if (run === latestRun.current) setData(result)
      } catch (e) {
        if (run === latestRun.current) setError(e instanceof ApiError ? e.message : fallbackError)
      } finally {
        if (run === latestRun.current) setLoading(false)
      }
    }, delay)
    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  return { data, error, loading }
}
