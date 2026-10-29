import StatusLabel from '../ui/StatusLabel.tsx'
import type { PaymentRequest } from '../../types.ts'

const labels: Record<PaymentRequest['status'], { text: string; dot: string }> = {
  pending: { text: 'Waiting for approval', dot: 'bg-warning' },
  executing: { text: 'Sending', dot: 'bg-brand' },
  executed: { text: 'Sent', dot: 'bg-success' },
  failed: { text: 'Failed', dot: 'bg-danger' },
  rejected: { text: 'Rejected', dot: 'bg-danger' },
  cancelled: { text: 'Cancelled', dot: 'bg-line-strong' },
  expired: { text: 'Expired', dot: 'bg-line-strong' },
}

export default function RequestStatus({ status }: { status: PaymentRequest['status'] }) {
  return <StatusLabel label={labels[status].text} dot={labels[status].dot} />
}
