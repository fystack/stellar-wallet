import { useCallback, useEffect, useState } from 'react'
import Wallets from '../pages/Wallets.tsx'
import Send from '../pages/Send.tsx'
import Swap from '../pages/Swap.tsx'
import Settings from '../pages/Settings.tsx'
import Approvals from '../pages/Approvals.tsx'
import WalletDetail from '../pages/WalletDetail.tsx'
import CreateWalletModal from './CreateWalletModal.tsx'
import Sidebar from './Sidebar.tsx'
import WorkspaceSwitcher from './workspace/WorkspaceSwitcher.tsx'
import InviteAccept from './workspace/InviteAccept.tsx'
import WorkspacePicker from './workspace/WorkspacePicker.tsx'
import { toast } from '../toast.tsx'
import { api } from '../api.ts'
import { useRoute } from '../hooks/useRoute.ts'
import type { PageKey } from '../hooks/useRoute.ts'
import { useWallets } from '../hooks/useWallets.ts'
import { useWorkspaces } from '../hooks/useWorkspaces.ts'
import { useLiveTransactions } from '../hooks/useLiveTransactions.ts'
import { useApprovals } from '../hooks/useApprovals.ts'
import { takePendingApproval } from '../lib/pendingApproval.ts'
import type { InvitePreview, JoinResult, SendInput, SwapInput, Transaction } from '../types.ts'

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
  const [choosingWorkspace, setChoosingWorkspace] = useState(false)
  // Loaded here, not in the picker tab, so switching tabs doesn't refetch and flash the card.
  const [invitePreview, setInvitePreview] = useState<{ token: string; preview: InvitePreview | null; error: string } | null>(null)
  useEffect(() => {
    if (!inviteToken) return
    let current = true
    api
      .previewInvite(inviteToken)
      .then((preview) => current && setInvitePreview({ token: inviteToken, preview, error: '' }))
      .catch((failure: unknown) => {
        if (current) setInvitePreview({ token: inviteToken, preview: null, error: failure instanceof Error ? failure.message : 'This invite link is invalid.' })
      })
    return () => {
      current = false
    }
  }, [inviteToken])
  const linkPreview = invitePreview?.token === inviteToken ? invitePreview : null
  const linkInviteId = linkPreview?.preview?.inviteId ?? ''
  const [sendFromId, setSendFromId] = useState<string | undefined>(undefined)
  // A just-submitted transaction to auto-open so the signing animation is visible.
  const [focusTxId, setFocusTxId] = useState<string | null>(null)

  const { goTo, reset } = route
  const endSession = useCallback(() => {
    reset()
    onSessionExpired()
  }, [reset, onSessionExpired])

  const workspaces = useWorkspaces(endSession)
  // The opened link's invite may also be in the in-app list; then the list row is enough.
  const linkInviteListed = !!inviteToken && workspaces.invitations.some((invitation) => invitation.id === linkInviteId)
  const wallets = useWallets(endSession, workspaces.needsPick ? '' : workspaces.activeId)
  const canManageWallets = workspaces.active?.role === 'owner'
  const canSend = canManageWallets || workspaces.active?.role === 'operator'
  const approvals = useApprovals(workspaces.needsPick ? '' : workspaces.activeId, endSession)
  const live = useLiveTransactions(wallets.applyUpdate, approvals.applyUpdate, endSession)

  // An approval link may point at another workspace than the active one, so switch to it first.
  const { loaded: workspacesLoaded, activeId, needsPick, switchTo } = workspaces
  const { openRequest } = route
  const { applyUpdate: applyApprovalUpdate } = approvals
  useEffect(() => {
    if (!workspacesLoaded) return
    const requestId = takePendingApproval()
    if (!requestId) return
    void (async () => {
      try {
        const request = await api.paymentRequest(requestId)
        const switched = needsPick || request.workspaceId !== activeId ? await switchTo(request.workspaceId) : true
        if (!switched) return
        applyApprovalUpdate(request)
        openRequest(requestId)
      } catch (error) {
        toast.error(error instanceof Error ? error.message : 'Unable to open the payment request.')
      }
    })()
  }, [workspacesLoaded, activeId, needsPick, switchTo, openRequest, applyApprovalUpdate])

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
      const result = await api.createTxn(
        input.walletId,
        input.to,
        input.amount,
        input.memo,
        input.asset,
        input.memoType,
      )
      if (result.request) {
        approvals.applyUpdate(result.request)
        route.openRequest(result.request.id)
        toast.success(`Sent for approval. It needs ${result.request.requiredApprovals} approval${result.request.requiredApprovals === 1 ? '' : 's'}.`)
      } else if (result.transaction) {
        showSubmitted(result.transaction, input.walletId, 'Transaction submitted for signing')
      }
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
    if (!(await workspaces.switchTo(id))) return
    setChoosingWorkspace(false)
    route.reset()
  }

  function announceJoin(result: JoinResult, joinedMessage: string) {
    toast.success(joinedMessage)
    if (result.wallet) toast.success(`Creating ${result.wallet.name}…`)
    if (result.walletError) toast.error(result.walletError)
  }

  async function createWorkspace(name: string) {
    const result = await workspaces.create(name)
    setChoosingWorkspace(false)
    route.reset()
    announceJoin(result, `${result.workspace.name} created`)
  }

  async function acceptInvitation(id: string) {
    try {
      const result = await workspaces.acceptInvitation(id)
      if (id === linkInviteId) onInviteHandled()
      setChoosingWorkspace(false)
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
      if (id === linkInviteId) onInviteHandled()
      toast.success('Invitation declined')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Unable to decline the invitation.')
      await workspaces.reload()
    }
  }

  async function handleJoined(result: JoinResult) {
    onInviteHandled()
    setChoosingWorkspace(false)
    await workspaces.reload()
    route.reset()
    announceJoin(result, `You joined ${result.workspace.name}`)
  }

  async function signOutEverywhere() {
    try {
      await api.logoutEverywhere()
      toast.success('Signed out of all devices')
      endSession()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Unable to sign out of all devices.')
    }
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
            <button className="btn btn-primary btn-lg mt-4" onClick={() => void workspaces.reload()}>
              Try again
            </button>
          </div>
        ) : (
          <p role="status">Loading your workspaces…</p>
        )}
      </div>
    )
  }

  if (workspaces.needsPick || choosingWorkspace || inviteToken) {
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
        onBack={workspaces.needsPick || inviteToken ? undefined : () => setChoosingWorkspace(false)}
        linkInvite={
          inviteToken && !linkInviteListed ? (
            <InviteAccept
              token={inviteToken}
              userEmail={user}
              onJoined={(result) => void handleJoined(result)}
              onDismiss={onInviteHandled}
              onSignOut={() => void handleSignOut()}
              preview={linkPreview?.preview ?? null}
              loadError={linkPreview?.error ?? ''}
            />
          ) : undefined
        }
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
        pendingApprovals={approvals.pending.length}
        workspaceSwitcher={
          <WorkspaceSwitcher
            active={workspaces.active}
            invitationCount={workspaces.invitations.length}
            onOpen={() => setChoosingWorkspace(true)}
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
            onSend={canSend ? () => {
              setSendFromId(openWallet.id)
              goTo('send')
            } : undefined}
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
                readOnly={!canSend}
                approvalPolicy={approvals.policy?.policy ?? null}
                approvalPolicyKnown={approvals.loaded}
                approvalPolicyFailed={approvals.failed}
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
            {route.page === 'approvals' && (
              <Approvals
                workspace={workspaces.active}
                userEmail={user}
                pending={approvals.pending}
                loaded={approvals.loaded}
                failed={approvals.failed}
                onRetry={() => void approvals.reload()}
                onChanged={approvals.applyUpdate}
                focusRequestId={route.requestId}
                canApprove={canSend}
                updates={approvals.updates}
                history={approvals.history}
                historyLoaded={approvals.historyLoaded}
                historyFailed={approvals.historyFailed}
                onLoadHistory={approvals.loadHistory}
                hasMoreHistory={approvals.hasMoreHistory}
                loadingMoreHistory={approvals.loadingMoreHistory}
                onLoadMoreHistory={approvals.loadMoreHistory}
                isOwner={canManageWallets}
                onFocusHandled={route.clearRequest}
              />
            )}
            {route.page === 'settings' && (
              <Settings
                workspace={workspaces.active}
                userEmail={user}
                onCreateWorkspace={createWorkspace}
                isOwner={canManageWallets}
                approvalPolicy={approvals.policy}
                approvalPolicyFailed={approvals.failed}
                onApprovalPolicyChange={approvals.setPolicy}
                onMembersChange={() => void approvals.reloadPolicy()}
                onSignOutEverywhere={signOutEverywhere}
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
