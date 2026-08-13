import type { TxStatus } from '../types.ts'

const cfg: Record<TxStatus, { label: string; bg: string; color: string }> = {
  policy_check: { label: 'Policy check', bg: '#fef3c7', color: '#d97706' },
  signing: { label: 'Signing', bg: '#e8efff', color: '#2757c6' },
  broadcast: { label: 'Broadcast', bg: '#ccddf9', color: '#0f5cc0' },
  confirmed: { label: 'Confirmed', bg: '#d2f8d6', color: '#1c8f74' },
  failed: { label: 'Failed', bg: '#f8d2d2', color: '#d0021b' },
}

export default function TxStatusBadge({ status }: { status: TxStatus }) {
  const c = cfg[status]
  return (
    <span
      className="inline-flex items-center px-2 py-0.5 text-xs font-semibold"
      style={{ background: c.bg, color: c.color }}
    >
      {c.label}
    </span>
  )
}
