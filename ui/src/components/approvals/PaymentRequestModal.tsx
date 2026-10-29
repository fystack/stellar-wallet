import { approvalCount, hasVoted, isInitiator } from '../../lib/paymentRequest.ts'
import { useState } from 'react'
import Modal from '../Modal.tsx'
import RequestStatus from './RequestStatus.tsx'
import StatusLabel from '../ui/StatusLabel.tsx'
import { TokenLogo } from '../../logos.tsx'
import { formatAmount, shortAddress } from '../../lib/format.ts'
import { api, explorerTx } from '../../api.ts'
import { toast } from '../../toast.tsx'
import type { PaymentRequest } from '../../types.ts'

type Action = 'approve' | 'reject' | 'cancel'

const run: Record<Action, (id: string) => Promise<PaymentRequest>> = {
  approve: api.approvePayment,
  reject: api.rejectPayment,
  cancel: api.cancelPayment,
}

const done: Record<Action, string> = {
  approve: 'Payment approved',
  reject: 'Payment rejected',
  cancel: 'Payment request cancelled',
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 py-3 sm:flex-row sm:gap-4">
      <dt className="w-28 shrink-0 text-sm text-muted">{label}</dt>
      <dd className="min-w-0 break-all text-sm font-medium text-ink">{children}</dd>
    </div>
  )
}

export default function PaymentRequestModal({
  request,
  userEmail,
  canApprove,
  isOwner,
  onChanged,
  onClose,
}: {
  request: PaymentRequest
  userEmail: string
  canApprove: boolean
  isOwner: boolean
  onChanged: (request: PaymentRequest) => void
  onClose: () => void
}) {
  const [busy, setBusy] = useState<Action | null>(null)
  const approvals = approvalCount(request)
  const mine = isInitiator(request, userEmail)
  const voted = hasVoted(request, userEmail)
  const open = request.status === 'pending'
  const canVote = open && canApprove && !mine && !voted
  const canCancel = open && (mine || isOwner)

  async function act(action: Action) {
    setBusy(action)
    try {
      const updated = await run[action](request.id)
      onChanged(updated)
      if (updated.status === 'failed') toast.error(updated.reason ?? 'The payment could not be sent.')
      else toast.success(done[action])
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Something went wrong.')
    } finally {
      setBusy(null)
    }
  }

  const when = (iso: string) =>
    new Date(iso).toLocaleString('en-US', { dateStyle: 'medium', timeStyle: 'short' })

  return (
    <Modal title="Payment request" onClose={onClose} maxWidth={560}>
      <div className="mb-4 flex items-center gap-3 bg-card px-4 py-4">
        <TokenLogo symbol={request.symbol} size={36} />
        <div className="min-w-0 flex-1">
          <div className="text-2xl font-extrabold leading-tight">
            {formatAmount(request.amount)} <span className="text-base font-semibold text-muted">{request.symbol}</span>
          </div>
          <div className="truncate text-xs text-muted">from {request.walletName}</div>
        </div>
        <RequestStatus status={request.status} />
      </div>

      <dl className="mb-5 divide-y divide-line">
        <Row label="To">
          <span className="font-mono text-[13px]">{request.destination}</span>
        </Row>
        {request.memo && <Row label="Memo">{request.memo}</Row>}
        <Row label="Started by">
          <span className="block truncate">
            {request.initiatorEmail}
            {mine && <span className="font-normal text-muted"> (you)</span>}
          </span>
          <span className="block text-xs font-normal text-muted">{when(request.createdAt)}</span>
        </Row>
        <Row label="Approvals">
          {approvals} of {request.requiredApprovals}
        </Row>
        {open && <Row label="Expires">{when(new Date(request.expiresAt * 1000).toISOString())}</Row>}
        {request.reason && <Row label="Reason">{request.reason}</Row>}
        {request.txHash && (
          <Row label="Transaction">
            <a
              className="btn btn-link font-mono text-[13px]"
              href={explorerTx('stellar', request.txHash)}
              target="_blank"
              rel="noreferrer"
            >
              {shortAddress(request.txHash, 10, 10)}
            </a>
          </Row>
        )}
        {!request.txHash && request.transactionStatus && <Row label="Transaction">{request.transactionStatus}</Row>}
      </dl>

      {request.votes.length > 0 && (
        <div className="mb-5">
          <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted">Responses</div>
          <ul className="divide-y divide-line">
            {request.votes.map((vote) => (
              <li key={vote.userId} className="flex items-center gap-3 py-2.5 text-sm">
                <span className="min-w-0 flex-1 truncate font-medium">{vote.email}</span>
                <StatusLabel
                  label={vote.decision === 'approve' ? 'Approved' : 'Rejected'}
                  dot={vote.decision === 'approve' ? 'bg-success' : 'bg-danger'}
                />
                <span className="hidden shrink-0 text-xs text-muted sm:inline">{when(vote.createdAt)}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {open && mine && <p className="mb-4 text-sm text-muted">Another owner or operator needs to approve your payment.</p>}

      {(canVote || canCancel) && (
        <div className="flex flex-col gap-2 sm:flex-row">
          {canVote && (
            <>
              <button className="btn btn-primary btn-lg flex-1" disabled={busy !== null} onClick={() => void act('approve')}>
                {busy === 'approve' ? 'Approving…' : 'Approve payment'}
              </button>
              <button className="btn btn-secondary flex-1" disabled={busy !== null} onClick={() => void act('reject')}>
                {busy === 'reject' ? 'Rejecting…' : 'Reject'}
              </button>
            </>
          )}
          {canCancel && (
            <button className="btn btn-secondary flex-1" disabled={busy !== null} onClick={() => void act('cancel')}>
              {busy === 'cancel' ? 'Cancelling…' : 'Cancel request'}
            </button>
          )}
        </div>
      )}
    </Modal>
  )
}
