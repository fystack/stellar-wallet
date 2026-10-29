import TxStatusBadge from '../transaction/TxStatusBadge.tsx'
import { TokenLogo } from '../../logos.tsx'
import { formatAmount, shortAddress } from '../../lib/format.ts'
import { TRUSTLINE_MEMO } from '../../lib/stellar.ts'
import type { Transaction } from '../../types.ts'

function formatShortDate(iso: string): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function kindLabel(transaction: Transaction): string {
  if (transaction.memo === TRUSTLINE_MEMO) return `Trustline · ${transaction.symbol}`
  if (transaction.type === 'swap') return 'Swap'
  return transaction.type === 'in' ? 'Received' : 'Sent'
}

export default function TransactionRow({
  transaction,
  onOpen,
}: {
  transaction: Transaction
  onOpen: () => void
}) {
  const trustline = transaction.memo === TRUSTLINE_MEMO
  const swap = transaction.type === 'swap'
  const incoming = transaction.type === 'in'

  return (
    <button
      onClick={onOpen}
      className="flex min-h-[76px] items-center gap-2.5 border-b border-line py-3 text-left transition-colors last:border-none hover:bg-hover sm:gap-3"
    >
      <div className="relative shrink-0">
        <TokenLogo symbol={transaction.symbol} size={36} />
        <span
          className={
            'absolute -bottom-1 -right-1 grid h-[18px] w-[18px] place-items-center rounded-full border-2 border-white text-[9px] font-bold text-white ' +
            (swap ? 'bg-brand' : incoming ? 'bg-success' : 'bg-ink')
          }
        >
          {swap ? '⇅' : incoming ? '↓' : '↑'}
        </span>
      </div>
      <div className="min-w-0 flex-1">
        <div className="font-semibold">{kindLabel(transaction)}</div>
        <div className="truncate text-[13px] text-muted">
          {swap ? (
            <span>Stellar DEX</span>
          ) : (
            <span className="font-mono">{shortAddress(transaction.counterparty, 6, 6)}</span>
          )}
          {' · '}
          {formatShortDate(transaction.createdAt)}
        </div>
      </div>
      <div className="max-w-[140px] shrink-0 text-right text-sm sm:max-w-none sm:text-base">
        {swap ? (
          <div className="font-semibold leading-tight">
            <div className="text-success">
              +{formatAmount(transaction.recvAmount ?? '0')} {transaction.recvSymbol}
            </div>
            <div className="text-muted">
              −{formatAmount(transaction.amount)} {transaction.symbol}
            </div>
          </div>
        ) : (
          <div
            className={
              'font-semibold ' + (trustline ? 'text-muted' : incoming ? 'text-success' : 'text-ink')
            }
          >
            {trustline
              ? '—'
              : `${incoming ? '+' : '−'}${formatAmount(transaction.amount)} ${transaction.symbol}`}
          </div>
        )}
        <div className="mt-0.5">
          <TxStatusBadge status={transaction.status} />
        </div>
      </div>
    </button>
  )
}
