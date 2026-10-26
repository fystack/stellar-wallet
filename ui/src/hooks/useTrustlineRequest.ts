import { useCallback, useEffect, useRef, useState } from 'react'
import { api } from '../api.ts'
import { toast } from '../toast.tsx'

const POLL_INTERVAL_MS = 2000
const MAX_POLLS = 15

export function trustlineKey(code: string, issuer: string): string {
  return `${code}:${issuer}`
}

// Requests stay pending through signing and confirmation so a second click can't submit a duplicate.
export function useTrustlineRequest(walletId: string, onReady: () => void) {
  const [pending, setPending] = useState<string[]>([])
  const polls = useRef(new Map<string, ReturnType<typeof setInterval> | null>())
  const currentWallet = useRef(walletId)
  const onReadyRef = useRef(onReady)

  useEffect(() => {
    onReadyRef.current = onReady
  }, [onReady])

  useEffect(() => {
    currentWallet.current = walletId
    const active = polls.current
    return () => {
      active.forEach((poll) => poll && clearInterval(poll))
      active.clear()
      setPending([])
    }
  }, [walletId])

  const finish = useCallback((key: string) => {
    const poll = polls.current.get(key)
    if (poll) clearInterval(poll)
    polls.current.delete(key)
    setPending((keys) => keys.filter((k) => k !== key))
  }, [])

  const add = useCallback(
    async (code: string, issuer: string) => {
      const key = trustlineKey(code, issuer)
      if (polls.current.has(key)) return
      polls.current.set(key, null)
      setPending((keys) => [...keys, key])
      try {
        await api.addTrustline(walletId, code, issuer)
      } catch (e) {
        toast.error(e instanceof Error ? e.message : 'Failed to add trustline')
        finish(key)
        return
      }
      if (currentWallet.current !== walletId || !polls.current.has(key)) return
      toast.success(`Adding ${code} trustline — signing…`)
      let count = 0
      polls.current.set(
        key,
        setInterval(async () => {
          count++
          const balance = await api.balance(walletId).catch(() => null)
          if (!polls.current.has(key)) return
          if (balance?.assets.some((a) => a.symbol === code && a.issuer === issuer)) {
            finish(key)
            onReadyRef.current()
            toast.success(`${code} trustline ready`)
          } else if (count >= MAX_POLLS) {
            finish(key)
            toast.error(`${code} trustline is taking longer than expected — try again shortly`)
          }
        }, POLL_INTERVAL_MS),
      )
    },
    [walletId, finish],
  )

  return { pending, add }
}
