import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { GridIcon, SendIcon, SettingsIcon, SwapIcon } from './icons.tsx'
import Wallets from './pages/Wallets.tsx'
import Send from './pages/Send.tsx'
import Swap from './pages/Swap.tsx'
import Settings from './pages/Settings.tsx'
import WalletDetail from './pages/WalletDetail.tsx'
import Login from './pages/Login.tsx'
import CreateWalletModal from './components/CreateWalletModal.tsx'
import { AppLogo } from './logos.tsx'
import { Toaster, toast } from './toast.tsx'
import { api, ApiError, clearLegacySession, BASE } from './api.ts'
import { consumeAuthError } from './authErrors.ts'
import type { Chain, Transaction, Wallet } from './types.ts'

type PageKey = 'wallets' | 'send' | 'swap' | 'settings'

type NavItem = {
  key: PageKey
  label: string
  icon: ReactNode
}

const nav: NavItem[] = [
  { key: 'wallets', label: 'Wallets', icon: <GridIcon /> },
  { key: 'send', label: 'Send', icon: <SendIcon /> },
  { key: 'swap', label: 'Swap', icon: <SwapIcon /> },
  { key: 'settings', label: 'Settings', icon: <SettingsIcon /> },
]

// Parse the current URL hash into a route (used for initial state + back/forward).
function initialRoute(): { page: PageKey; openId: string | null } {
  const h = window.location.hash.replace(/^#\/?/, '')
  if (h.startsWith('wallet/')) {
    return { page: 'wallets', openId: h.slice('wallet/'.length) }
  }
  if (h === 'send' || h === 'swap' || h === 'settings' || h === 'wallets') {
    return { page: h, openId: null }
  }
  return { page: 'wallets', openId: null }
}

export default function App() {
  const [user, setUser] = useState<string | null>(null)
  const [authLoading, setAuthLoading] = useState(true)
  const [authError, setAuthError] = useState('')
  const [authNotice, setAuthNotice] = useState('')
  const [signingOut, setSigningOut] = useState(false)
  const [page, setPage] = useState<PageKey>(() => initialRoute().page)
  const [wallets, setWallets] = useState<Wallet[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [openWalletId, setOpenWalletId] = useState<string | null>(
    () => initialRoute().openId,
  )
  const [sendFromId, setSendFromId] = useState<string | undefined>(undefined)
  // Live transaction updates pushed over SSE, keyed by tx id.
  const [liveTxns, setLiveTxns] = useState<Record<string, Transaction>>({})
  // A just-sent tx to auto-open (so the signing animation is visible).
  const [focusTxId, setFocusTxId] = useState<string | null>(null)

  async function loadWallets() {
    try {
      setWallets(await api.listWallets())
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) clearUserState()
      else toast.error('Unable to load wallets. Please try again.')
    }
  }

  async function restoreSession() {
    setAuthLoading(true)
    setAuthError('')
    try {
      const account = await api.me()
      setUser(account.email)
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) setUser(null)
      else setAuthError('Unable to reach the wallet service. Please try again.')
    } finally { setAuthLoading(false) }
  }

  useEffect(() => {
    const notice = consumeAuthError()
    if (notice) setAuthNotice(notice)
    clearLegacySession()
    void restoreSession()
  }, [])

  useEffect(() => {
    if (user) loadWallets()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user])

  // Subscribe to the SSE stream for live transaction status updates.
  useEffect(() => {
    if (!user) return
    const es = new EventSource(`${BASE}/api/v1/events`, { withCredentials: true })
    es.onerror = () => {
      void api.me().catch((error: unknown) => {
        if (error instanceof ApiError && error.status === 401) clearUserState()
      })
    }
    es.onmessage = (e) => {
      try {
        const env = JSON.parse(e.data) as { type: string; data: unknown }
        if (env.type === 'tx') {
          const tx = env.data as Transaction
          setLiveTxns((prev) => ({ ...prev, [tx.id]: tx }))
        } else if (env.type === 'wallet') {
          const w = env.data as Wallet
          setWallets((prev) =>
            prev.map((x) => (x.id === w.id ? w : x)),
          )
        }
      } catch {
        /* ignore */
      }
    }
    return () => es.close()
  }, [user])

  // --- Hash routing: keep the URL in sync so reload/deep-link works ---
  // Initial route is read synchronously in useState above; here we only react
  // to later hash changes (browser back/forward).
  useEffect(() => {
    const apply = () => {
      const r = initialRoute()
      setOpenWalletId(r.openId)
      setPage(r.page)
    }
    window.addEventListener('hashchange', apply)
    return () => window.removeEventListener('hashchange', apply)
  }, [])

  useEffect(() => {
    if (!user) return
    const desired = openWalletId ? `#/wallet/${openWalletId}` : `#/${page}`
    if (window.location.hash !== desired) window.location.hash = desired
  }, [user, page, openWalletId])

  async function handleCreate(data: { name: string; chain: Chain }) {
    const w = await api.createWallet(data.name, data.chain)
    await loadWallets()
    return w.id // keygen runs async; modal watches for readiness
  }

  async function handleSend(data: {
    walletId: string
    to: string
    amount: string
    memo: string
    memoType?: string
    asset?: { code: string; issuer: string }
  }) {
    try {
      const tx = await api.createTxn(
        data.walletId,
        data.to,
        data.amount,
        data.memo,
        data.asset,
        data.memoType,
      )
      setLiveTxns((prev) => ({ ...prev, [tx.id]: tx }))
      setFocusTxId(tx.id) // auto-open its detail so the signing animation shows
      setOpenWalletId(data.walletId)
      setPage('wallets')
      toast.success('Transaction submitted for signing')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to send')
    }
  }

  async function handleSwap(data: {
    walletId: string
    from: string
    to: string
    amount: string
    slippageBps: number
  }) {
    try {
      const { transaction } = await api.createSwap(
        data.walletId,
        data.from,
        data.to,
        data.amount,
        data.slippageBps,
      )
      setLiveTxns((prev) => ({ ...prev, [transaction.id]: transaction }))
      setFocusTxId(transaction.id)
      setOpenWalletId(data.walletId)
      setPage('wallets')
      toast.success('Swap submitted for signing')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to swap')
    }
  }

  async function handleDeleteWallet(id: string) {
    await api.deleteWallet(id)
    setOpenWalletId(null)
    await loadWallets()
    toast.success('Wallet deleted')
  }

  async function handleSignOut() {
    setSigningOut(true)
    try {
      await api.logout()
      clearUserState()
    } catch {
      toast.error('Unable to sign out. Please try again.')
    } finally { setSigningOut(false) }
  }

  function clearUserState() {
    setLiveTxns({})
    setFocusTxId(null)
    setShowCreate(false)
    setAuthNotice('')
    setUser(null)
    setWallets([])
    setOpenWalletId(null)
    setSendFromId(undefined)
    setPage('wallets')
  }

  if (authLoading) return <div className="grid min-h-screen place-items-center" role="status">Checking your session…</div>
  if (authError) return <div className="grid min-h-screen place-items-center p-6"><div role="alert"><p>{authError}</p><button className="cta mt-4" onClick={() => void restoreSession()}>Try again</button></div></div>
  if (!user) {
    return <Login initialError={authNotice} onLogin={(email) => { setAuthNotice(''); setUser(email) }} />
  }

  const openWallet = wallets.find((w) => w.id === openWalletId) ?? null

  return (
    <div className="flex min-h-full min-w-0 flex-col md:flex-row">
      <aside className="sticky top-0 z-40 flex h-16 w-full flex-shrink-0 items-center border-b border-line bg-white px-4 md:top-0 md:h-screen md:w-[260px] md:flex-col md:items-stretch md:overflow-y-auto md:border-b-0 md:border-r md:p-4">
        <button
          type="button"
          onClick={() => {
            setOpenWalletId(null)
            setSendFromId(undefined)
            setPage('wallets')
          }}
          aria-label="Go to wallets"
          className="flex items-center gap-2.5 text-left transition-opacity hover:opacity-80 md:gap-3 md:px-2 md:pb-6 md:pt-1.5"
        >
          <AppLogo size={40} />
          <div>
            <div className="font-bold md:text-lg">MPC Stellar Wallet</div>
            <div className="hidden text-[13px] text-muted sm:block">
              Keys never assembled
            </div>
          </div>
        </button>

        <nav className="fixed inset-x-0 bottom-0 z-40 grid grid-cols-4 border-t border-line bg-white px-2 pb-[max(0.5rem,env(safe-area-inset-bottom))] pt-2 md:static md:flex md:flex-col md:gap-1 md:border-0 md:p-0">
          {nav.map((item) => {
            const active = page === item.key && !openWallet
            return (
              <button
                key={item.key}
                onClick={() => {
                  setOpenWalletId(null)
                  if (item.key === 'send') setSendFromId(undefined)
                  setPage(item.key)
                }}
                className={
                  'flex min-w-0 flex-col items-center gap-1 px-1 py-2 text-xs font-semibold transition md:flex-row md:gap-3 md:px-3.5 md:py-3 md:text-base ' +
                  (active
                    ? 'bg-brand-soft text-brand'
                    : 'text-ink-soft hover:bg-hover')
                }
              >
                <span className="grid place-items-center">{item.icon}</span>
                {item.label}
              </button>
            )
          })}
        </nav>

        <button
          onClick={() => void handleSignOut()}
          disabled={signingOut}
          aria-label={`Sign out ${user}`}
          className="ml-auto flex items-center gap-3 text-left text-[15px] text-ink-soft transition-colors hover:text-ink md:ml-0 md:mt-auto md:px-2 md:py-3"
        >
          <span className="grid h-9 w-9 shrink-0 place-items-center bg-[#1b1c1f] text-sm font-semibold uppercase text-white md:h-10 md:w-10">
            {user.charAt(0)}
          </span>
          <span className="hidden min-w-0 md:block">
            <span className="block truncate text-sm font-semibold text-ink">
              {user}
            </span>
            <span className="block text-[13px] text-muted">Sign out</span>
          </span>
        </button>
      </aside>

      <main className="min-w-0 flex-1 overflow-y-auto px-4 pb-24 pt-5 sm:px-6 md:px-10 md:pb-16 md:pt-6">
        {authNotice && <div role="alert" className="mb-5 border border-line bg-white p-4 text-sm">{authNotice}<button className="ml-3 font-semibold underline" onClick={() => setAuthNotice('')}>Dismiss</button></div>}
        {openWallet ? (
          <WalletDetail
            wallet={openWallet}
            liveTxns={liveTxns}
            focusTxId={focusTxId}
            onFocusHandled={() => setFocusTxId(null)}
            onBack={() => setOpenWalletId(null)}
            onDelete={handleDeleteWallet}
            onSend={() => {
              setSendFromId(openWallet.id)
              setOpenWalletId(null)
              setPage('send')
            }}
          />
        ) : (
          <>
            {page === 'wallets' && (
              <Wallets
                wallets={wallets}
                onCreate={() => setShowCreate(true)}
                onOpen={(id) => setOpenWalletId(id)}
              />
            )}
            {page === 'send' && (
              <Send
                wallets={wallets}
                preselectId={sendFromId}
                onGoCreate={() => setShowCreate(true)}
                onSubmit={handleSend}
              />
            )}
            {page === 'swap' && (
              <Swap
                wallets={wallets}
                preselectId={sendFromId}
                onGoCreate={() => setShowCreate(true)}
                onSubmit={handleSwap}
              />
            )}
            {page === 'settings' && <Settings />}
          </>
        )}
      </main>

      <Toaster />

      {showCreate && (
        <CreateWalletModal
          wallets={wallets}
          onClose={() => setShowCreate(false)}
          onCreate={handleCreate}
          onOpen={(id) => {
            setShowCreate(false)
            setOpenWalletId(id)
          }}
        />
      )}
    </div>
  )
}
