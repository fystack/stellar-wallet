import { useCallback, useState } from 'react'
import Wallets from '../pages/Wallets.tsx'
import Send from '../pages/Send.tsx'
import Swap from '../pages/Swap.tsx'
import Settings from '../pages/Settings.tsx'
import WalletDetail from '../pages/WalletDetail.tsx'
import CreateWalletModal from './CreateWalletModal.tsx'
import Sidebar from './Sidebar.tsx'
import { toast } from '../toast.tsx'
import { api } from '../api.ts'
import { useRoute } from '../hooks/useRoute.ts'
import type { PageKey } from '../hooks/useRoute.ts'
import { useWallets } from '../hooks/useWallets.ts'
import { useLiveTransactions } from '../hooks/useLiveTransactions.ts'
import type { SendInput, SwapInput, Transaction } from '../types.ts'

export default function AppShell({
  user,
  notice,
  onDismissNotice,
  signingOut,
  onSignOut,
  onSessionExpired,
}: {
  user: string
  notice: string
  onDismissNotice: () => void
  signingOut: boolean
  onSignOut: () => Promise<boolean>
  onSessionExpired: () => void
}) {
  const route = useRoute()
  const [showCreate, setShowCreate] = useState(false)
  const [sendFromId, setSendFromId] = useState<string | undefined>(undefined)
  // A just-submitted transaction to auto-open so the signing animation is visible.
  const [focusTxId, setFocusTxId] = useState<string | null>(null)

  const { goTo, reset } = route
  const endSession = useCallback(() => {
    reset()
    onSessionExpired()
  }, [reset, onSessionExpired])

  const wallets = useWallets(endSession)
  const live = useLiveTransactions(wallets.applyUpdate, endSession)

  function navigate(page: PageKey) {
    setSendFromId(undefined)
    goTo(page)
  }

  function showSubmitted(transaction: Transaction, walletId: string, message: string) {
    live.track(transaction)
    setFocusTxId(transaction.id)
    route.openWallet(walletId)
    toast.success(message)
  }

  async function handleSend(input: SendInput) {
    try {
      const transaction = await api.createTxn(
        input.walletId,
        input.to,
        input.amount,
        input.memo,
        input.asset,
        input.memoType,
      )
      showSubmitted(transaction, input.walletId, 'Transaction submitted for signing')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to send')
    }
  }

  async function handleSwap(input: SwapInput) {
    try {
      const { transaction } = await api.createSwap(
        input.walletId,
        input.from,
        input.to,
        input.amount,
        input.slippageBps,
      )
      showSubmitted(transaction, input.walletId, 'Swap submitted for signing')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to swap')
    }
  }

  async function handleDeleteWallet(id: string) {
    await wallets.remove(id)
    route.closeWallet()
    toast.success('Wallet deleted')
  }

  async function handleSignOut() {
    if (await onSignOut()) endSession()
  }

  const openWallet = wallets.wallets.find((w) => w.id === route.walletId) ?? null

  return (
    <div className="flex min-h-full min-w-0 flex-col md:flex-row">
      <Sidebar
        user={user}
        activePage={openWallet ? null : route.page}
        signingOut={signingOut}
        onNavigate={navigate}
        onSignOut={() => void handleSignOut()}
      />

      <main className="min-w-0 flex-1 overflow-y-auto px-4 pb-24 pt-5 sm:px-6 md:px-10 md:pb-16 md:pt-6">
        {notice && (
          <div role="alert" className="mb-5 border border-line bg-white p-4 text-sm">
            {notice}
            <button className="ml-3 font-semibold underline" onClick={onDismissNotice}>
              Dismiss
            </button>
          </div>
        )}
        {openWallet ? (
          <WalletDetail
            key={openWallet.id}
            wallet={openWallet}
            liveTxns={live.liveTxns}
            focusTxId={focusTxId}
            onFocusHandled={() => setFocusTxId(null)}
            onBack={route.closeWallet}
            onDelete={handleDeleteWallet}
            onSend={() => {
              setSendFromId(openWallet.id)
              goTo('send')
            }}
          />
        ) : (
          <>
            {route.page === 'wallets' && (
              <Wallets
                wallets={wallets.wallets}
                onCreate={() => setShowCreate(true)}
                onOpen={route.openWallet}
              />
            )}
            {route.page === 'send' && (
              <Send
                wallets={wallets.wallets}
                preselectId={sendFromId}
                onGoCreate={() => setShowCreate(true)}
                onSubmit={handleSend}
              />
            )}
            {route.page === 'swap' && (
              <Swap
                wallets={wallets.wallets}
                preselectId={sendFromId}
                onGoCreate={() => setShowCreate(true)}
                onSubmit={handleSwap}
              />
            )}
            {route.page === 'settings' && <Settings />}
          </>
        )}
      </main>

      {showCreate && (
        <CreateWalletModal
          wallets={wallets.wallets}
          onClose={() => setShowCreate(false)}
          onCreate={(data) => wallets.create(data.name, data.chain)}
          onOpen={(id) => {
            setShowCreate(false)
            route.openWallet(id)
          }}
        />
      )}
    </div>
  )
}
