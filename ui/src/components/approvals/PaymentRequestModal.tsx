import { approvalCount } from '../../lib/paymentRequest.ts'
import { useState } from 'react'
import Modal from '../Modal.tsx'
import RequestStatus from './RequestStatus.tsx'
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
    <div className="flex flex-col gap-0.5 border-b border-line py-2.5 last:border-none sm:flex-row sm:gap-4">
      <dt className="w-32 shrink-0 text-sm text-muted">{label}</dt>
      <dd className="min-w-0 break-all text-sm font-medium">{children}</dd>
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
  const mine = request.initiatorEmail === userEmail
  const voted = request.votes.some((vote) => vote.email === userEmail)
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

  return (
    <Modal title="Payment request" onClose={onClose} maxWidth={560}>
      <div className="mb-4 flex flex-wrap items-baseline justify-between gap-2">
        <div className="text-2xl font-bold">
          {request.amount} {request.symbol}
        </div>
        <RequestStatus status={request.status} />
      </div>

      <dl className="mb-5">
        <Row label="From wallet">{request.walletName}</Row>
        <Row label="To">{request.destination}</Row>
        {request.memo && <Row label="Memo">{request.memo}</Row>}
        <Row label="Started by">
          {request.initiatorEmail}
          {mine && <span className="font-normal text-muted"> (you)</span>} · {new Date(request.createdAt).toLocaleString()}
        </Row>
        <Row label="Approvals">
          {approvals} of {request.requiredApprovals}
        </Row>
        {open && <Row label="Expires">{new Date(request.expiresAt * 1000).toLocaleString()}</Row>}
        {request.reason && <Row label="Reason">{request.reason}</Row>}
        {request.txHash && (
          <Row label="Transaction">
            <a className="text-brand underline" href={explorerTx('stellar', request.txHash)} target="_blank" rel="noreferrer">
              {request.txHash}
            </a>
          </Row>
        )}
        {!request.txHash && request.transactionStatus && <Row label="Transaction">{request.transactionStatus}</Row>}
      </dl>

      {request.votes.length > 0 && (
        <div className="mb-5">
          <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted">Responses</div>
          <ul>
            {request.votes.map((vote) => (
              <li key={vote.userId} className="flex items-center gap-3 border-b border-line py-2 text-sm last:border-none">
                <span className="min-w-0 flex-1 truncate">{vote.email}</span>
                <span className={vote.decision === 'approve' ? 'font-semibold text-success' : 'font-semibold text-danger'}>
                  {vote.decision === 'approve' ? 'Approved' : 'Rejected'}
                </span>
                <span className="shrink-0 text-muted">{new Date(vote.createdAt).toLocaleString()}</span>
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
