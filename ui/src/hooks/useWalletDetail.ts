import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { api } from '../api.ts'
import { toast } from '../toast.tsx'
import { formatUsd } from '../lib/format.ts'
import { useBalances } from './useBalances.ts'
import { useTrustlineRequest } from './useTrustlineRequest.ts'
import type { CustomAsset, Transaction, Wallet } from '../types.ts'

// Horizon indexes a broadcast shortly after it confirms, so balances are re-read once more after this delay.
const BALANCE_RETRY_MS = 2500

export function useWalletDetail(wallet: Wallet, liveTxns: Record<string, Transaction>) {
  const [funding, setFunding] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [prices, setPrices] = useState<Record<string, number>>({})
  const [registry, setRegistry] = useState<CustomAsset[]>([])
  const [history, setHistory] = useState<Transaction[]>([])
  const [historyLoaded, setHistoryLoaded] = useState(false)

  useEffect(() => {
    api.prices().then(setPrices).catch(() => {})
  }, [])

  useEffect(() => {
    if (wallet.chain !== 'stellar') return
    api
      .getConfig()
      .then((config) => setRegistry(config.assets))
      .catch(() => {})
  }, [wallet.chain])

  const reloadHistory = useCallback(
    () =>
      api
        .walletTxns(wallet.id)
        .then(setHistory)
        .catch(() => {})
        .finally(() => setHistoryLoaded(true)),
    [wallet.id],
  )
  useEffect(() => {
    void reloadHistory()
  }, [reloadHistory])

  // Fetched history merged with live SSE updates, newest first.
  const transactions = useMemo(() => {
    const byId = new Map<string, Transaction>()
    history.forEach((t) => byId.set(t.id, t))
    Object.values(liveTxns)
      .filter((t) => t.walletId === wallet.id)
      .forEach((t) => byId.set(t.id, t))
    return [...byId.values()].sort((a, b) => (a.createdAt < b.createdAt ? 1 : -1))
  }, [history, liveTxns, wallet.id])

  function usd(symbol: string, amount: string) {
    const price = prices[symbol]
    return price ? formatUsd(Number(amount) * price) : ''
  }

  const balances = useBalances(wallet.id)
  const { reload: reloadBalances } = balances
  const trustline = useTrustlineRequest(wallet.id, reloadBalances)

  // Re-read balances when a transaction confirms or the cached balance changes over SSE.
  const confirmedCount = transactions.filter((t) => t.status === 'confirmed').length
  const lastBalanceTrigger = useRef<string | null>(null)
  useEffect(() => {
    if (!historyLoaded) return
    const trigger = `${confirmedCount}:${wallet.balance}`
    const previous = lastBalanceTrigger.current
    lastBalanceTrigger.current = trigger
    if (previous === null || previous === trigger) return
    void reloadBalances()
    const retry = setTimeout(() => void reloadBalances(), BALANCE_RETRY_MS)
    return () => clearTimeout(retry)
  }, [historyLoaded, confirmedCount, wallet.balance, reloadBalances])

  // Pulls incoming on-chain payments, then reloads history and balances.
  async function refresh() {
    if (refreshing) return
    setRefreshing(true)
    try {
      await api.sync(wallet.id).catch(() => {})
      await reloadHistory()
      void reloadBalances()
    } finally {
      setRefreshing(false)
    }
  }

  async function fund() {
    setFunding(true)
    try {
      await api.fund(wallet.id)
      void reloadBalances()
      toast.success('Wallet funded with testnet XLM')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Funding failed')
    } finally {
      setFunding(false)
    }
  }

  return {
    usd,
    registry,
    assets: balances.assets,
    transactions,
    pendingTrustlines: trustline.pending,
    addTrustline: trustline.add,
    refreshing,
    refresh,
    funding,
    fund,
  }
}
