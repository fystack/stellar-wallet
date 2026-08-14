import { useEffect, useState } from 'react'
import Modal from './Modal.tsx'
import TxStatusBadge from './TxStatusBadge.tsx'
import TxTimeline from './TxTimeline.tsx'
import CopyAddress from './CopyAddress.tsx'
import { TokenLogo } from '../logos.tsx'
import { api, explorerTx } from '../api.ts'
import { formatAmount, formatUsd } from '../format.ts'
import type { Transaction } from '../types.ts'

const NETWORK: Record<string, string> = {
  stellar: 'Stellar · testnet',
  solana: 'Solana · devnet',
}

function fmtDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function Row({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-2">
      <span className="shrink-0 text-muted">{label}</span>
      <span className="min-w-0 text-right">{children}</span>
    </div>
  )
}

export default function TxDetailModal({
  tx,
  chain,
  onClose,
}: {
  tx: Transaction
  chain: string
  onClose: () => void
}) {
  const incoming = tx.type === 'in'
  const isTrustline = tx.memo === 'Add trustline'
  const kind = isTrustline ? 'Trustline' : incoming ? 'Received' : 'Sent'

  const [prices, setPrices] = useState<Record<string, number>>({})
  useEffect(() => {
    api
      .prices()
      .then(setPrices)
      .catch(() => {})
  }, [])
  const price = prices[tx.symbol]
  const usdValue = price ? formatUsd(Number(tx.amount) * price) : ''

  // Real on-chain metadata (fee charged, ledger) once broadcast.
  const [onChain, setOnChain] = useState<{
    fee: string
    ledger: number
    operations: number
  } | null>(null)
  useEffect(() => {
    if (!tx.txHash || chain !== 'stellar') return
    api
      .txChain(tx.txHash)
      .then(setOnChain)
      .catch(() => setOnChain(null))
  }, [tx.txHash, chain])

  return (
    <Modal title="Transaction" onClose={onClose} maxWidth={460}>
      {/* Amount header */}
      <div className="mb-5 flex items-start justify-between">
        <div className="flex items-center gap-3">
          <span
            className={
              'grid h-11 w-11 shrink-0 place-items-center text-xl font-bold ' +
              (incoming
                ? 'bg-green-50 text-green-600'
                : 'bg-[#f2f5f9] text-ink-soft')
            }
          >
            {incoming ? '↓' : '↑'}
          </span>
          <div>
            <div className="text-sm text-muted">{kind}</div>
            <div className="flex items-center gap-1.5 text-2xl font-extrabold leading-tight">
              <TokenLogo symbol={tx.symbol} size={22} />
              {isTrustline ? '' : incoming ? '+' : '−'}
              {formatAmount(tx.amount)}{' '}
              <span className="text-muted">{tx.symbol}</span>
            </div>
            {usdValue && !isTrustline && (
              <div className="mt-0.5 text-xs text-muted">≈ {usdValue}</div>
            )}
          </div>
        </div>
        <TxStatusBadge status={tx.status} />
      </div>

      {/* Details */}
      <div className="mb-6 divide-y divide-line bg-card px-4 text-sm">
        <Row label={incoming ? 'From' : isTrustline ? 'Issuer' : 'To'}>
          <CopyAddress
            address={tx.counterparty}
            truncate
            className="max-w-[220px] text-ink"
          />
        </Row>
        <Row label="Asset">
          <span className="flex items-center justify-end gap-1.5 text-ink">
            <TokenLogo symbol={tx.symbol} size={16} /> {tx.symbol}
          </span>
        </Row>
        <Row label="Network">
          <span className="text-ink">{NETWORK[chain] ?? chain}</span>
        </Row>
        {!incoming && (
          <Row label="Network fee">
            <span className="text-ink">{onChain?.fee ?? '0.00001'} XLM</span>
          </Row>
        )}
        {onChain && (
          <Row label="Ledger">
            <span className="font-mono text-ink">#{onChain.ledger}</span>
          </Row>
        )}
        <Row label="Date">
          <span className="text-ink">{fmtDate(tx.createdAt)}</span>
        </Row>
        {tx.memo && (
          <Row label="Memo">
            <span className="text-ink">{tx.memo}</span>
          </Row>
        )}
      </div>

      {tx.status === 'failed' && (
        <div className="mb-6 border border-[#f0c2c2] bg-[#fff5f5] px-3.5 py-3">
          <div className="mb-1 text-sm font-semibold text-[#d33a3a]">
            Transaction failed
          </div>
          <div className="font-mono text-xs text-ink-soft">
            {tx.error || 'Unknown error'}
          </div>
        </div>
      )}

      {tx.txHash && (
        <div className="mb-6">
          <div className="mb-1.5 text-xs font-medium uppercase tracking-wide text-muted">
            On-chain transaction
          </div>
          <div className="flex items-center justify-between gap-2 bg-card px-3 py-2.5">
            <span className="truncate font-mono text-xs text-ink-soft">
              {tx.txHash}
            </span>
            <a
              href={explorerTx(chain, tx.txHash)}
              target="_blank"
              rel="noreferrer"
              className="shrink-0 text-xs font-semibold text-brand hover:underline"
            >
              Explorer ↗
            </a>
          </div>
        </div>
      )}

      {tx.signature && (
        <div className="mb-6">
          <div className="mb-1.5 text-xs font-medium uppercase tracking-wide text-muted">
            MPC signature (2-of-3)
          </div>
          <div className="max-h-20 overflow-y-auto break-all bg-card px-3 py-2 font-mono text-xs text-ink-soft">
            {tx.signature}
          </div>
        </div>
      )}

      {tx.type === 'out' && tx.status !== 'failed' && (
        <div className="border-t border-line pt-5">
          <h3 className="mb-4 text-sm font-semibold text-ink-soft">
            Signing progress
          </h3>
          <TxTimeline status={tx.status} />
        </div>
      )}
    </Modal>
  )
}
