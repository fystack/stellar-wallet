import { useCallback, useEffect, useRef, useState } from 'react'
import { api } from '../api.ts'
import type { AssetOption } from '../components/AssetSelect.tsx'

// Testnet assets with live DEX liquidity from XLM, so the receive list is useful on a fresh wallet.
const CIRCLE_USDC = 'USDC:GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5'
const TESTNET_MM = 'GAHPYWLK6YRN7CVYZOO4H3VDRZ7PVF5UJGLZCSPAEIKJE2XSWF5LAGER'
const SUGGESTED_TESTNET_ASSETS: AssetOption[] = [
  { label: 'USDC', value: CIRCLE_USDC },
  { label: 'USDT', value: `USDT:${TESTNET_MM}` },
  { label: 'BTC', value: `BTC:${TESTNET_MM}` },
  { label: 'ETH', value: `ETH:${TESTNET_MM}` },
]

// Held assets fund the pay side; held, registered and suggested assets are valid receive targets.
export function useSwapAssets(walletId: string, workspaceId: string) {
  const [held, setHeld] = useState<AssetOption[]>([])
  const [receivable, setReceivable] = useState<AssetOption[]>([])
  const latestRequest = useRef(0)

  const reload = useCallback(async () => {
    if (!walletId) return
    const request = ++latestRequest.current
    const [balance, registry] = await Promise.all([
      api.balance(walletId),
      workspaceId ? api.listAssets(workspaceId).catch(() => null) : null,
    ])
    if (request !== latestRequest.current) return
    const heldOptions = balance.assets.map((a) => ({
      label: a.symbol,
      value: a.issuer ? `${a.symbol}:${a.issuer}` : 'XLM',
      balance: a.balance,
    }))
    const targets = new Map<string, AssetOption>([['XLM', { label: 'XLM', value: 'XLM' }]])
    heldOptions.forEach((option) => targets.set(option.value, option))
    registry?.assets.forEach((a) =>
      targets.set(`${a.code}:${a.issuer}`, { label: a.code, value: `${a.code}:${a.issuer}` }),
    )
    SUGGESTED_TESTNET_ASSETS.forEach((option) => {
      if (!targets.has(option.value)) targets.set(option.value, option)
    })
    setHeld(heldOptions)
    setReceivable([...targets.values()])
  }, [walletId, workspaceId])

  useEffect(() => {
    reload().catch(() => {})
  }, [reload])

  return { held, receivable, reload }
}
