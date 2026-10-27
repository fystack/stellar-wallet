import { useCallback, useState } from 'react'
import Wallets from '../pages/Wallets.tsx'
import Send from '../pages/Send.tsx'
import Swap from '../pages/Swap.tsx'
import Settings from '../pages/Settings.tsx'
import WalletDetail from '../pages/WalletDetail.tsx'
import CreateWalletModal from './CreateWalletModal.tsx'
import Sidebar from './Sidebar.tsx'
import WorkspaceSwitcher from './WorkspaceSwitcher.tsx'
import InviteAccept from './InviteAccept.tsx'
import WorkspacePicker from './WorkspacePicker.tsx'
import { toast } from '../toast.tsx'
import { api } from '../api.ts'
import { useRoute } from '../hooks/useRoute.ts'
import type { PageKey } from '../hooks/useRoute.ts'
import { useWallets } from '../hooks/useWallets.ts'
import { useWorkspaces } from '../hooks/useWorkspaces.ts'
import { useLiveTransactions } from '../hooks/useLiveTransactions.ts'
import type { JoinResult, SendInput, SwapInput, Transaction } from '../types.ts'

export default function AppShell({
  user,
  notice,
  onDismissNotice,
  signingOut,
  onSignOut,
  onSessionExpired,
  inviteToken,
  onInviteHandled,
}: {
  user: string
  notice: string
  onDismissNotice: () => void
  signingOut: boolean
  onSignOut: () => Promise<boolean>
  onSessionExpired: () => void
  inviteToken: string
  onInviteHandled: () => void
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

  const workspaces = useWorkspaces(endSession)
  const wallets = useWallets(endSession, workspaces.needsPick ? '' : workspaces.activeId)
  const canManageWallets = workspaces.active?.role === 'owner'
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

  async function switchWorkspace(id: string) {
    await workspaces.switchTo(id)
    route.reset()
  }

  function announceJoin(result: JoinResult, joinedMessage: string) {
    toast.success(joinedMessage)
    if (result.wallet) toast.success(`Creating ${result.wallet.name}…`)
    if (result.walletError) toast.error(result.walletError)
  }

  async function createWorkspace(name: string) {
    const result = await workspaces.create(name)
    route.reset()
    announceJoin(result, `${result.workspace.name} created`)
  }

  async function acceptInvitation(id: string) {
    try {
      const result = await workspaces.acceptInvitation(id)
      route.reset()
      announceJoin(result, `You joined ${result.workspace.name}`)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Unable to accept the invitation.')
      await workspaces.reload()
    }
  }

  async function declineInvitation(id: string) {
    try {
      await workspaces.declineInvitation(id)
      toast.success('Invitation declined')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Unable to decline the invitation.')
      await workspaces.reload()
    }
  }

  async function handleJoined(result: JoinResult) {
    onInviteHandled()
    await workspaces.reload()
    route.reset()
    announceJoin(result, `You joined ${result.workspace.name}`)
  }

  async function handleSignOut() {
    if (await onSignOut()) endSession()
  }

  const openWallet = wallets.wallets.find((w) => w.id === route.walletId) ?? null

  if (!workspaces.loaded) {
    return (
      <div className="grid min-h-screen place-items-center p-6">
        {workspaces.error ? (
          <div role="alert" className="text-center">
            <p>{workspaces.error}</p>
            <button className="cta mt-4" onClick={() => void workspaces.reload()}>
              Try again
            </button>
          </div>
        ) : (
          <p role="status">Loading your workspaces…</p>
        )}
      </div>
    )
  }

  if (workspaces.needsPick && !inviteToken) {
    return (
      <WorkspacePicker
        userEmail={user}
        workspaces={workspaces.workspaces}
        onPick={switchWorkspace}
        onCreate={createWorkspace}
        signingOut={signingOut}
        onSignOut={() => void handleSignOut()}
        invitations={workspaces.invitations}
        invitationsVerified={workspaces.invitationsVerified}
        onAcceptInvitation={acceptInvitation}
        onDeclineInvitation={declineInvitation}
      />
    )
  }

  return (
    <div className="flex min-h-full min-w-0 flex-col md:flex-row">
      <Sidebar
        user={user}
        activePage={openWallet ? null : route.page}
        signingOut={signingOut}
        onNavigate={navigate}
        onSignOut={() => void handleSignOut()}
        workspaceSwitcher={
          <WorkspaceSwitcher
            workspaces={workspaces.workspaces}
            active={workspaces.active}
            onSwitch={switchWorkspace}
            onCreate={createWorkspace}
            invitations={workspaces.invitations}
            invitationsVerified={workspaces.invitationsVerified}
            onAcceptInvitation={acceptInvitation}
            onDeclineInvitation={declineInvitation}
          />
        }
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
        {inviteToken && (
          <InviteAccept
            token={inviteToken}
            userEmail={user}
            onJoined={(result) => void handleJoined(result)}
            onDismiss={onInviteHandled}
            onSignOut={() => void handleSignOut()}
          />
        )}
        {openWallet ? (
          <WalletDetail
            key={openWallet.id}
            wallet={openWallet}
            liveTxns={live.liveTxns}
            focusTxId={focusTxId}
            onFocusHandled={() => setFocusTxId(null)}
            onBack={route.closeWallet}
            onDelete={canManageWallets ? handleDeleteWallet : undefined}
            readOnly={!canManageWallets}
            onSend={() => {
              setSendFromId(openWallet.id)
              goTo('send')
            }}
          />
        ) : (
          <>
            {route.page === 'wallets' && (
              <Wallets
                loading={!wallets.loaded}
                failed={wallets.failed}
                onRetry={() => void wallets.reload()}
                wallets={wallets.wallets}
                workspaceName={workspaces.active?.name ?? ''}
                onCreate={canManageWallets ? () => setShowCreate(true) : undefined}
                onOpen={route.openWallet}
              />
            )}
            {route.page === 'send' && (
              <Send
                readOnly={!canManageWallets}
                wallets={wallets.wallets}
                preselectId={sendFromId}
                onGoCreate={() => setShowCreate(true)}
                onSubmit={handleSend}
              />
            )}
            {route.page === 'swap' && (
              <Swap
                readOnly={!canManageWallets}
                wallets={wallets.wallets}
                preselectId={sendFromId}
                onGoCreate={() => setShowCreate(true)}
                onSubmit={handleSwap}
              />
            )}
            {route.page === 'settings' && (
              <Settings
                workspace={workspaces.active}
                userEmail={user}
                onCreateWorkspace={createWorkspace}
              />
            )}
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
