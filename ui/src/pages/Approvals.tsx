import { approvalCount, hasVoted, isInitiator } from '../lib/paymentRequest.ts'
import { useEffect, useState } from 'react'
import { api } from '../api.ts'
import { CheckShieldIcon } from '../icons.tsx'
import { formatAmount, shortAddress } from '../lib/format.ts'
import { TokenLogo } from '../logos.tsx'
import { toast } from '../toast.tsx'
import EmptyState from '../components/EmptyState.tsx'
import PaymentRequestModal from '../components/approvals/PaymentRequestModal.tsx'
import RequestStatus from '../components/approvals/RequestStatus.tsx'
import type { PaymentRequest, Workspace } from '../types.ts'

type Tab = 'pending' | 'history'

function formatShortDate(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function ApprovalProgress({ approved, required }: { approved: number; required: number }) {
  return (
    <span className="flex items-center gap-2 text-[13px] text-muted">
      <span aria-hidden className="flex gap-0.5">
        {Array.from({ length: required }, (_, index) => (
          <span key={index} className={'h-1.5 w-4 ' + (index < approved ? 'bg-success' : 'bg-line-strong')} />
        ))}
      </span>
      {approved}/{required} approved
    </span>
  )
}

function RequestList({
  requests,
  userEmail,
  canApprove,
  onOpen,
}: {
  requests: PaymentRequest[]
  userEmail: string
  canApprove: boolean
  onOpen: (request: PaymentRequest) => void
}) {
  return (
    <ul className="surface !p-0">
      {requests.map((request) => {
        const approvals = approvalCount(request)
        const yourTurn =
          request.status === 'pending' &&
          canApprove &&
          !isInitiator(request, userEmail) &&
          !hasVoted(request, userEmail)
        return (
          <li key={request.id} className="border-b border-line last:border-none">
            <button
              onClick={() => onOpen(request)}
              aria-label={`Review ${request.amount} ${request.symbol} from ${request.walletName}`}
              className="grid min-h-[76px] w-full grid-cols-[36px_minmax(0,1fr)] items-center gap-x-3 gap-y-2 px-4 py-3 text-left transition-colors hover:bg-hover sm:flex sm:px-5"
            >
              <TokenLogo symbol={request.symbol} size={36} />
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-baseline gap-x-2">
                  <span className="font-semibold">
                    {formatAmount(request.amount)} {request.symbol}
                  </span>
                  <span className="whitespace-nowrap text-[13px] text-muted">
                    to <span className="font-mono">{shortAddress(request.destination)}</span>
                  </span>
                </div>
                <div className="truncate text-[13px] text-muted">
                  {request.walletName} · {request.initiatorEmail} · {formatShortDate(request.createdAt)}
                </div>
              </div>
              <div className="col-start-2 flex shrink-0 flex-wrap items-center gap-x-3 gap-y-1.5 sm:flex-col sm:items-end">
                {yourTurn ? (
                  <span className="badge badge-brand">Needs your approval</span>
                ) : (
                  <RequestStatus status={request.status} />
                )}
                <ApprovalProgress approved={approvals} required={request.requiredApprovals} />
              </div>
            </button>
          </li>
        )
      })}
    </ul>
  )
}

function LoadError({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div role="alert" className="surface text-center">
      <p className="mb-4">{message}</p>
      <button className="btn btn-primary" onClick={onRetry}>Try again</button>
    </div>
  )
}

export default function Approvals({
  workspace,
  userEmail,
  pending,
  loaded,
  failed,
  onRetry,
  onChanged,
  focusRequestId,
  onFocusHandled,
  canApprove,
  isOwner,
  updates,
  history,
  historyLoaded,
  historyFailed,
  onLoadHistory,
  hasMoreHistory,
  loadingMoreHistory,
  onLoadMoreHistory,
}: {
  updates: Record<string, PaymentRequest>
  history: PaymentRequest[]
  historyLoaded: boolean
  historyFailed: boolean
  onLoadHistory: () => Promise<void>
  hasMoreHistory: boolean
  loadingMoreHistory: boolean
  onLoadMoreHistory: () => Promise<void>
  canApprove: boolean
  isOwner: boolean
  workspace: Workspace | null
  userEmail: string
  pending: PaymentRequest[]
  loaded: boolean
  failed: boolean
  onRetry: () => void
  onChanged: (request: PaymentRequest) => void
  focusRequestId: string | null
  onFocusHandled: () => void
}) {
  const [tab, setTab] = useState<Tab>('pending')
  const [selected, setSelected] = useState<PaymentRequest | null>(null)

  useEffect(() => {
    if (tab === 'history' && !historyLoaded && !historyFailed) void onLoadHistory()
  }, [tab, historyLoaded, historyFailed, onLoadHistory])

  useEffect(() => {
    if (!focusRequestId) return
    const known = pending.find((item) => item.id === focusRequestId)
    if (known) {
      setSelected(known)
      onFocusHandled()
      return
    }
    api
      .paymentRequest(focusRequestId)
      .then(setSelected)
      .catch((error: unknown) => toast.error(error instanceof Error ? error.message : 'Unable to open the payment request.'))
      .finally(onFocusHandled)
  }, [focusRequestId, onFocusHandled, pending])

  useEffect(() => {
    setSelected((current) => (current && updates[current.id]) ?? current)
  }, [updates])

  function changed(request: PaymentRequest) {
    onChanged(request)
    setSelected(request)
  }

  const tabs: { key: Tab; label: string; count?: number }[] = [
    { key: 'pending', label: 'Pending', count: loaded ? pending.length : 0 },
    { key: 'history', label: 'History' },
  ]

  let body
  if (tab === 'pending') {
    if (failed) {
      body = <LoadError message="Unable to load payment requests." onRetry={onRetry} />
    } else if (!loaded) {
      body = <p role="status" className="text-muted">Loading payment requests…</p>
    } else if (pending.length === 0) {
      body = (
        <EmptyState
          icon={<CheckShieldIcon size={32} />}
          title="Nothing to approve"
          body="Payments that need a second sign-off appear here. Owners set the approval threshold in Settings."
        />
      )
    } else {
      body = <RequestList requests={pending} userEmail={userEmail} canApprove={canApprove} onOpen={setSelected} />
    }
  } else if (historyFailed) {
    body = <LoadError message="Unable to load past requests." onRetry={() => void onLoadHistory()} />
  } else if (!historyLoaded) {
    body = <p role="status" className="text-muted">Loading past requests…</p>
  } else if (history.length === 0) {
    body = <p className="surface text-center text-muted">No approved, rejected or cancelled payments yet.</p>
  } else {
    body = (
      <>
        <RequestList requests={history} userEmail={userEmail} canApprove={canApprove} onOpen={setSelected} />
        {hasMoreHistory && (
          <div className="pt-4">
            <button className="btn btn-secondary btn-sm" disabled={loadingMoreHistory} onClick={() => void onLoadMoreHistory()}>
              {loadingMoreHistory ? 'Loading…' : 'Load older requests'}
            </button>
          </div>
        )}
      </>
    )
  }

  return (
    <div className="mx-auto mt-2.5 max-w-[860px]">
      <header className="mb-6">
        <h1 className="mb-2 text-2xl font-bold sm:text-3xl">Approvals</h1>
        <p className="text-base leading-relaxed text-muted">
          Payments in {workspace?.name ?? 'this workspace'} that need sign-off from another owner or operator before the MPC nodes sign them.
        </p>
      </header>

      <div role="tablist" aria-label="Payment requests" className="mb-4 flex gap-1 border-b border-line">
        {tabs.map((item) => (
          <button
            key={item.key}
            role="tab"
            id={`approvals-tab-${item.key}`}
            aria-selected={tab === item.key}
            aria-controls="approvals-panel"
            onClick={() => setTab(item.key)}
            className={'-mb-px flex items-center gap-2 border-b-2 px-4 py-2.5 text-sm font-semibold transition-colors ' + (tab === item.key ? 'border-brand text-brand' : 'border-transparent text-ink-soft hover:text-ink')}
          >
            {item.label}
            {!!item.count && <span className="count">{item.count}</span>}
          </button>
        ))}
      </div>

      <div role="tabpanel" id="approvals-panel" aria-labelledby={`approvals-tab-${tab}`}>
        {body}
      </div>

      {selected && (
        <PaymentRequestModal
          request={selected}
          userEmail={userEmail}
          canApprove={canApprove}
          isOwner={isOwner}
          onChanged={changed}
          onClose={() => setSelected(null)}
        />
      )}
    </div>
  )
}
