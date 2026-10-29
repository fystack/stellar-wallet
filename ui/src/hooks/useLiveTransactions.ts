import { useCallback, useEffect, useState } from 'react'
import { api, ApiError, BASE } from '../api.ts'
import type { Transaction, Wallet } from '../types.ts'

type LiveEvent = { type: 'tx'; data: Transaction } | { type: 'wallet'; data: Wallet }

// Subscribes to the SSE stream and keeps the latest state of each transaction, keyed by id.
export function useLiveTransactions(
  onWallet: (wallet: Wallet) => void,
  onUnauthorized: () => void,
) {
  const [liveTxns, setLiveTxns] = useState<Record<string, Transaction>>({})

  const track = useCallback((transaction: Transaction) => {
    setLiveTxns((current) => ({ ...current, [transaction.id]: transaction }))
  }, [])

  useEffect(() => {
    const events = new EventSource(`${BASE}/api/v1/events`, { withCredentials: true })
    events.onerror = () => {
      void api.me().catch((error: unknown) => {
        if (error instanceof ApiError && error.status === 401) onUnauthorized()
      })
    }
    events.onmessage = (message) => {
      let event: LiveEvent
      try {
        event = JSON.parse(message.data) as LiveEvent
      } catch {
        return
      }
      if (event.type === 'tx') track(event.data)
      else if (event.type === 'wallet') onWallet(event.data)
    }
    return () => events.close()
  }, [onWallet, onUnauthorized, track])

  return { liveTxns, track }
}
