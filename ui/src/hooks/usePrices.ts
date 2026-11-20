import { useEffect, useState } from 'react'
import { api } from '../api.ts'

const REFRESH_MS = 30_000

// USD prices keyed by asset code. Empty while loading or when prices are unavailable.
export function usePrices() {
  const [prices, setPrices] = useState<Record<string, number>>({})
  const [failed, setFailed] = useState(false)
  useEffect(() => {
    let current = true
    const load = () =>
      api
        .prices()
        .then((next) => {
          if (!current) return
          setPrices(next)
          setFailed(false)
        })
        .catch(() => current && setFailed(true))
    void load()
    const timer = setInterval(() => void load(), REFRESH_MS)
    return () => {
      current = false
      clearInterval(timer)
    }
  }, [])
  return { prices, failed }
}

// Dollar value of an amount, or null when the asset has no price.
export function usdValue(prices: Record<string, number>, symbol: string, amount: string): number | null {
  const price = prices[symbol]
  const units = Number(amount)
  if (!price || !Number.isFinite(units)) return null
  return units * price
}
