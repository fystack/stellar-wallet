import StatusLabel from '../ui/StatusLabel.tsx'
import type { TxStatus } from '../../types.ts'

const cfg: Record<TxStatus, { label: string; dot: string }> = {
  policy_check: { label: 'Policy check', dot: 'bg-warning' },
  signing: { label: 'Signing', dot: 'bg-brand' },
  broadcast: { label: 'Broadcast', dot: 'bg-brand' },
  confirmed: { label: 'Confirmed', dot: 'bg-success' },
  failed: { label: 'Failed', dot: 'bg-danger' },
}

export default function TxStatusBadge({ status }: { status: TxStatus }) {
  return <StatusLabel {...cfg[status]} />
}
