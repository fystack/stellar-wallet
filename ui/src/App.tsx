import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import { GridIcon, SendIcon, SettingsIcon } from './icons.tsx'
import Wallets from './pages/Wallets.tsx'
import Send from './pages/Send.tsx'
import Settings from './pages/Settings.tsx'
import WalletDetail from './pages/WalletDetail.tsx'
import Login from './pages/Login.tsx'
import CreateWalletModal from './components/CreateWalletModal.tsx'
import { AppLogo } from './logos.tsx'
import { Toaster, toast } from './toast.tsx'
import { api, getEmail, getToken, clearSession, BASE } from './api.ts'
import type { Chain, Transaction, Wallet } from './types.ts'

type PageKey = 'wallets' | 'send' | 'settings'

type NavItem = {
  key: PageKey
  label: string
  icon: ReactNode
}

const nav: NavItem[] = [
  { key: 'wallets', label: 'Wallets', icon: <GridIcon /> },
  { key: 'send', label: 'Send', icon: <SendIcon /> },
  { key: 'settings', label: 'Settings', icon: <SettingsIcon /> },
]

export default function App() {
  const [user, setUser] = useState<string | null>(
    getToken() ? getEmail() : null,
  )
  const [page, setPage] = useState<PageKey>('wallets')
  const [wallets, setWallets] = useState<Wallet[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [openWalletId, setOpenWalletId] = useState<string | null>(null)
  const [sendFromId, setSendFromId] = useState<string | undefined>(undefined)
  // Live transaction updates pushed over SSE, keyed by tx id.
  const [liveTxns, setLiveTxns] = useState<Record<string, Transaction>>({})
  // A just-sent tx to auto-open (so the signing animation is visible).
  const [focusTxId, setFocusTxId] = useState<string | null>(null)

  async function loadWallets() {
    try {
      setWallets(await api.listWallets())
    } catch {
      // token likely expired
      handleSignOut()
    }
  }

  useEffect(() => {
    if (user) loadWallets()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user])

  // Subscribe to the SSE stream for live transaction status updates.
  useEffect(() => {
    const token = getToken()
    if (!user || !token) return
    const es = new EventSource(`${BASE}/api/v1/events?token=${token}`)
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
  useEffect(() => {
    const apply = () => {
      const h = window.location.hash.replace(/^#\/?/, '')
      if (h.startsWith('wallet/')) {
        setOpenWalletId(h.slice('wallet/'.length))
      } else if (h === 'send' || h === 'settings' || h === 'wallets') {
        setOpenWalletId(null)
        setPage(h)
      }
    }
    apply()
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
    asset?: { code: string; issuer: string }
  }) {
    try {
      const tx = await api.createTxn(
        data.walletId,
        data.to,
        data.amount,
        data.memo,
        data.asset,
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

  async function handleDeleteWallet(id: string) {
    await api.deleteWallet(id)
    setOpenWalletId(null)
    await loadWallets()
    toast.success('Wallet deleted')
  }

  function handleSignOut() {
    clearSession()
    setUser(null)
    setWallets([])
    setOpenWalletId(null)
    setSendFromId(undefined)
    setPage('wallets')
  }

  if (!user) {
    return <Login onLogin={setUser} />
  }

  const openWallet = wallets.find((w) => w.id === openWalletId) ?? null

  return (
    <div className="flex min-h-full">
      <aside className="flex w-[260px] flex-shrink-0 flex-col border-r border-line bg-white p-4">
        <div className="flex items-center gap-3 px-2 pb-6 pt-1.5">
          <AppLogo size={46} />
          <div>
            <div className="text-lg font-bold">Wallet</div>
            <div className="text-[13px] text-muted">MPC-secured</div>
          </div>
        </div>

        <nav className="flex flex-col gap-1">
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
                  'flex items-center gap-3 px-3.5 py-3 text-base font-semibold transition ' +
                  (active
                    ? 'bg-brand-soft text-brand'
                    : 'text-ink-soft hover:bg-[#f2f5f9]')
                }
              >
                <span className="grid place-items-center">{item.icon}</span>
                {item.label}
              </button>
            )
          })}
        </nav>

        <button
          onClick={handleSignOut}
          className="mt-auto flex items-center gap-3 px-2 py-3 text-left text-[15px] text-ink-soft transition-colors hover:text-ink"
        >
          <span className="grid h-10 w-10 shrink-0 place-items-center bg-[#1b1c1f] text-sm font-semibold uppercase text-white">
            {user.charAt(0)}
          </span>
          <span className="min-w-0">
            <span className="block truncate text-sm font-semibold text-ink">
              {user}
            </span>
            <span className="block text-[13px] text-muted">Sign out</span>
          </span>
        </button>
      </aside>

      <main className="flex-1 overflow-y-auto px-10 pb-16 pt-6">
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
