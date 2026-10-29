import { useCallback, useEffect, useRef, useState } from 'react'
import { api } from '../api.ts'

export type AssetBalance = { symbol: string; balance: string; issuer?: string }

export function assetKey(asset: { symbol: string; issuer?: string }): string {
  return asset.issuer ? `${asset.symbol}:${asset.issuer}` : asset.symbol
}

// Live per-asset balances for a wallet; null while loading or after a failed load.
export function useBalances(walletId: string) {
  const [assets, setAssets] = useState<AssetBalance[] | null>(null)
  const latestRequest = useRef(0)

  const reload = useCallback(async () => {
    if (!walletId) return null
    const request = ++latestRequest.current
    let result: AssetBalance[] | null
    try {
      result = (await api.balance(walletId)).assets
    } catch {
      result = null
    }
    if (request === latestRequest.current) setAssets(result)
    return result
  }, [walletId])

  useEffect(() => {
    setAssets(null)
    void reload()
  }, [reload])

  return { assets, reload }
}
