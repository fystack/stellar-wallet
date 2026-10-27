import { useCallback, useEffect, useState } from 'react'

export type PageKey = 'wallets' | 'send' | 'swap' | 'approvals' | 'settings'

type Route = { page: PageKey; walletId: string | null; requestId: string | null }

const pages: PageKey[] = ['wallets', 'send', 'swap', 'approvals', 'settings']

function parseHash(): Route {
  const hash = window.location.hash.replace(/^#\/?/, '')
  if (hash.startsWith('wallet/')) {
    return { page: 'wallets', walletId: hash.slice('wallet/'.length), requestId: null }
  }
  if (hash.startsWith('approvals/')) {
    return { page: 'approvals', walletId: null, requestId: hash.slice('approvals/'.length) }
  }
  const page = pages.find((key) => key === hash)
  return { page: page ?? 'wallets', walletId: null, requestId: null }
}

function hashFor(route: Route) {
  if (route.walletId) return `#/wallet/${route.walletId}`
  if (route.requestId) return `#/approvals/${route.requestId}`
  return `#/${route.page}`
}

// Hash routing keeps reloads and deep links working.
export function useRoute() {
  const [route, setRoute] = useState<Route>(parseHash)

  useEffect(() => {
    const apply = () => setRoute(parseHash())
    window.addEventListener('hashchange', apply)
    return () => window.removeEventListener('hashchange', apply)
  }, [])

  useEffect(() => {
    const desired = hashFor(route)
    if (window.location.hash !== desired) window.location.hash = desired
  }, [route])

  const goTo = useCallback((page: PageKey) => setRoute({ page, walletId: null, requestId: null }), [])
  const openWallet = useCallback(
    (walletId: string) => setRoute({ page: 'wallets', walletId, requestId: null }),
    [],
  )
  const openRequest = useCallback(
    (requestId: string) => setRoute({ page: 'approvals', walletId: null, requestId }),
    [],
  )
  const clearRequest = useCallback(
    () => setRoute((current) => ({ ...current, requestId: null })),
    [],
  )
  const closeWallet = useCallback(
    () => setRoute((current) => ({ ...current, walletId: null })),
    [],
  )

  // Writes the URL immediately, for callers that unmount before the sync effect would run.
  const reset = useCallback(() => {
    const home: Route = { page: 'wallets', walletId: null, requestId: null }
    window.location.hash = hashFor(home)
    setRoute(home)
  }, [])

  return { page: route.page, walletId: route.walletId, requestId: route.requestId, goTo, openWallet, closeWallet, openRequest, clearRequest, reset }
}
