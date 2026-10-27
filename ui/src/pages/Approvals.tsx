import { approvalCount } from '../lib/paymentRequest.ts'
import { useEffect, useState } from 'react'
import { api } from '../api.ts'
import { CheckShieldIcon } from '../icons.tsx'
import { shortAddress } from '../lib/format.ts'
import { toast } from '../toast.tsx'
import EmptyState from '../components/EmptyState.tsx'
import PaymentRequestModal from '../components/approvals/PaymentRequestModal.tsx'
import RequestStatus from '../components/approvals/RequestStatus.tsx'
import type { PaymentRequest, Workspace } from '../types.ts'

type Tab = 'pending' | 'history'

function RequestList({ requests, onOpen }: { requests: PaymentRequest[]; onOpen: (request: PaymentRequest) => void }) {
  return (
    <ul className="surface !p-0">
      {requests.map((request) => {
        const approvals = approvalCount(request)
        return (
          <li key={request.id} className="border-b border-line last:border-none">
            <button
              onClick={() => onOpen(request)}
              aria-label={`Review ${request.amount} ${request.symbol} from ${request.walletName}`}
              className="flex w-full flex-col gap-1 px-4 py-3.5 text-left hover:bg-hover sm:flex-row sm:items-center sm:gap-4 sm:px-5"
            >
              <div className="min-w-0 flex-1">
                <div className="font-semibold">
                  {request.amount} {request.symbol} <span className="font-normal text-muted">to {shortAddress(request.destination)}</span>
                </div>
                <div className="truncate text-[13px] text-muted">
                  {request.walletName} · by {request.initiatorEmail} · {new Date(request.createdAt).toLocaleDateString()}
                </div>
              </div>
              <span className="shrink-0 text-[13px] text-ink-soft">
                {approvals}/{request.requiredApprovals} approved
              </span>
              <RequestStatus status={request.status} />
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
}: {
  updates: Record<string, PaymentRequest>
  history: PaymentRequest[]
  historyLoaded: boolean
  historyFailed: boolean
  onLoadHistory: () => Promise<void>
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

  const tabs: { key: Tab; label: string }[] = [
    { key: 'pending', label: `Pending${loaded && pending.length ? ` (${pending.length})` : ''}` },
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
      body = <RequestList requests={pending} onOpen={setSelected} />
    }
  } else if (historyFailed) {
    body = <LoadError message="Unable to load past requests." onRetry={() => void onLoadHistory()} />
  } else if (!historyLoaded) {
    body = <p role="status" className="text-muted">Loading past requests…</p>
  } else if (history.length === 0) {
    body = <p className="text-muted">No approved, rejected or cancelled payments yet.</p>
  } else {
    body = <RequestList requests={history} onOpen={setSelected} />
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
            className={'-mb-px border-b-2 px-4 py-2.5 text-sm font-semibold ' + (tab === item.key ? 'border-brand text-brand' : 'border-transparent text-ink-soft hover:text-ink')}
          >
            {item.label}
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
