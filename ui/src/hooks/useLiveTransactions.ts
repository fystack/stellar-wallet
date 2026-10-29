import { useCallback, useEffect, useRef, useState } from 'react'
import { api, ApiError, BASE } from '../api.ts'
import type { PaymentRequest, Transaction, Wallet } from '../types.ts'

type LiveEvent =
  | { type: 'tx'; data: Transaction }
  | { type: 'wallet'; data: Wallet }
  | { type: 'payment_request'; data: PaymentRequest }

// Subscribes to the SSE stream and keeps the latest state of each transaction, keyed by id.
export function useLiveTransactions(
  onWallet: (wallet: Wallet) => void,
  onPaymentRequest: (request: PaymentRequest) => void,
  onUnauthorized: () => void,
) {
  const [liveTxns, setLiveTxns] = useState<Record<string, Transaction>>({})

  const track = useCallback((transaction: Transaction) => {
    setLiveTxns((current) => ({ ...current, [transaction.id]: transaction }))
  }, [])

  // Latest handlers in a ref, so switching workspace doesn't reopen the stream.
  const handlers = useRef({ onWallet, onPaymentRequest, onUnauthorized })
  handlers.current = { onWallet, onPaymentRequest, onUnauthorized }

  useEffect(() => {
    const events = new EventSource(`${BASE}/api/v1/events`, { withCredentials: true })
    events.onerror = () => {
      void api.me().catch((error: unknown) => {
        if (error instanceof ApiError && error.status === 401) handlers.current.onUnauthorized()
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
      else if (event.type === 'wallet') handlers.current.onWallet(event.data)
      else if (event.type === 'payment_request') handlers.current.onPaymentRequest(event.data)
    }
    return () => events.close()
  }, [track])

  return { liveTxns, track }
}
