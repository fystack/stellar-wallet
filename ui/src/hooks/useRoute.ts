import { useCallback, useEffect, useState } from 'react'

export type PageKey = 'wallets' | 'send' | 'swap' | 'settings'

type Route = { page: PageKey; walletId: string | null }

const pages: PageKey[] = ['wallets', 'send', 'swap', 'settings']

function parseHash(): Route {
  const hash = window.location.hash.replace(/^#\/?/, '')
  if (hash.startsWith('wallet/')) {
    return { page: 'wallets', walletId: hash.slice('wallet/'.length) }
  }
  const page = pages.find((key) => key === hash)
  return { page: page ?? 'wallets', walletId: null }
}

function hashFor(route: Route) {
  return route.walletId ? `#/wallet/${route.walletId}` : `#/${route.page}`
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

  const goTo = useCallback((page: PageKey) => setRoute({ page, walletId: null }), [])
  const openWallet = useCallback(
    (walletId: string) => setRoute({ page: 'wallets', walletId }),
    [],
  )
  const closeWallet = useCallback(
    () => setRoute((current) => ({ ...current, walletId: null })),
    [],
  )

  // Writes the URL immediately, for callers that unmount before the sync effect would run.
  const reset = useCallback(() => {
    const home: Route = { page: 'wallets', walletId: null }
    window.location.hash = hashFor(home)
    setRoute(home)
  }, [])

  return { page: route.page, walletId: route.walletId, goTo, openWallet, closeWallet, reset }
}
