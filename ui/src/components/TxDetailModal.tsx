import { useEffect, useState } from 'react'
import Modal from './Modal.tsx'
import TxStatusBadge from './TxStatusBadge.tsx'
import TxTimeline from './TxTimeline.tsx'
import CopyAddress from './CopyAddress.tsx'
import { TokenLogo } from '../logos.tsx'
import { api, explorerTx } from '../api.ts'
import { formatAmount, formatUsd } from '../format.ts'
import { friendlyTxError } from '../txError.ts'
import type { Transaction } from '../types.ts'

const NETWORK: Record<string, string> = {
  stellar: 'Stellar · testnet',
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
    <div className="flex flex-col items-start gap-1 py-2 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
      <span className="shrink-0 text-muted">{label}</span>
      <span className="min-w-0 max-w-full text-left sm:text-right">{children}</span>
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
  const isSwap = tx.type === 'swap'
  const kind = isSwap
    ? 'Swap'
    : isTrustline
      ? 'Trustline'
      : incoming
        ? 'Received'
        : 'Sent'

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
      <div className="mb-5 flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <span
            className={
              'grid h-11 w-11 shrink-0 place-items-center text-xl font-bold ' +
              (isSwap
                ? 'bg-brand-soft text-brand'
                : incoming
                  ? 'bg-success-soft text-success'
                  : 'bg-hover text-ink-soft')
            }
          >
            {isSwap ? '⇅' : incoming ? '↓' : '↑'}
          </span>
          <div>
            <div className="text-sm text-muted">{kind}</div>
            {isSwap ? (
              <div className="flex flex-col gap-0.5 text-lg font-extrabold leading-tight sm:text-xl">
                <span className="flex items-center gap-1.5 text-success">
                  <TokenLogo symbol={tx.recvSymbol ?? ''} size={20} />+
                  {formatAmount(tx.recvAmount ?? '0')} {tx.recvSymbol}
                </span>
                <span className="flex items-center gap-1.5 text-ink-soft">
                  <TokenLogo symbol={tx.symbol} size={20} />−
                  {formatAmount(tx.amount)} {tx.symbol}
                </span>
              </div>
            ) : (
              <div className="flex flex-wrap items-center gap-1.5 text-xl font-extrabold leading-tight sm:text-2xl">
                <TokenLogo symbol={tx.symbol} size={22} />
                {isTrustline ? '' : incoming ? '+' : '−'}
                {formatAmount(tx.amount)}{' '}
                <span className="text-muted">{tx.symbol}</span>
              </div>
            )}
            {usdValue && !isTrustline && !isSwap && (
              <div className="mt-0.5 text-xs text-muted">≈ {usdValue}</div>
            )}
          </div>
        </div>
        <TxStatusBadge status={tx.status} />
      </div>

      {/* Details */}
      <div className="mb-6 divide-y divide-line bg-card px-4 text-sm">
        {isSwap ? (
          <Row label="Venue">
            <span className="text-ink">Stellar DEX (path payment)</span>
          </Row>
        ) : (
          <Row label={incoming ? 'From' : isTrustline ? 'Issuer' : 'To'}>
            <CopyAddress
              address={tx.counterparty}
              truncate
              className="max-w-full text-ink sm:max-w-[220px]"
            />
          </Row>
        )}
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

      {tx.status === 'failed' &&
        (() => {
          const friendly = friendlyTxError(tx.error)
          return (
            <div className="mb-6 border border-danger-line bg-[#fff5f5] px-3.5 py-3">
              <div className="mb-1 text-sm font-semibold text-danger">
                Transaction failed
              </div>
              {friendly ? (
                <>
                  <div className="text-sm text-ink">{friendly.title}</div>
                  {friendly.hint && (
                    <div className="mt-1 text-xs text-ink-soft">
                      {friendly.hint}
                    </div>
                  )}
                  <div className="mt-2 font-mono text-[11px] text-muted">
                    {tx.error}
                  </div>
                </>
              ) : (
                <div className="font-mono text-xs text-ink-soft">
                  {tx.error || 'Unknown error'}
                </div>
              )}
            </div>
          )
        })()}

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

      {tx.type !== 'in' && tx.status !== 'failed' && (
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
