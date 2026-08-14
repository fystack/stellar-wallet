import { useEffect, useMemo, useState } from 'react'
import {
  RefreshIcon,
  SendIcon,
  ShieldIcon,
  SpinnerIcon,
  TrashIcon,
} from '../icons.tsx'
import CopyAddress from '../components/CopyAddress.tsx'
import TxStatusBadge from '../components/TxStatusBadge.tsx'
import ReceiveModal from '../components/ReceiveModal.tsx'
import TxDetailModal from '../components/TxDetailModal.tsx'
import { api, explorerAddress } from '../api.ts'
import { toast } from '../toast.tsx'
import { ChainLogo, TokenLogo } from '../logos.tsx'
import { formatAmount, formatUsd, shortAddress } from '../format.ts'

function fmtShort(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
import type { Transaction, Wallet } from '../types.ts'

type Props = {
  wallet: Wallet
  liveTxns: Record<string, Transaction>
  focusTxId?: string | null
  onFocusHandled?: () => void
  onBack: () => void
  onSend: () => void
  onDelete: (id: string) => Promise<void>
}

export default function WalletDetail({
  wallet,
  liveTxns,
  focusTxId,
  onFocusHandled,
  onBack,
  onSend,
  onDelete,
}: Props) {
  const [base, setBase] = useState<Transaction[]>([])
  const [showReceive, setShowReceive] = useState(false)
  const [openTxId, setOpenTxId] = useState<string | null>(null)
  const [assets, setAssets] = useState<
    { symbol: string; balance: string }[] | null
  >(null)
  const [funding, setFunding] = useState(false)
  const [prices, setPrices] = useState<Record<string, number>>({})
  const [sortBy, setSortBy] = useState<'amount' | 'name'>('amount')
  useEffect(() => {
    api
      .prices()
      .then(setPrices)
      .catch(() => {})
  }, [])
  const usd = (symbol: string, amount: string) => {
    const p = prices[symbol]
    if (!p) return ''
    return formatUsd(Number(amount) * p)
  }
  const [confirmDel, setConfirmDel] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [registry, setRegistry] = useState<{ code: string; issuer: string }[]>(
    [],
  )
  // Assets we've already requested a trustline for — hidden until they show up
  // in the balance, so a slow confirmation can't be double-submitted.
  const [pendingTrust, setPendingTrust] = useState<string[]>([])

  // Custom-asset registry (from Settings) — offer these as trustlines to add.
  useEffect(() => {
    if (wallet.chain !== 'stellar') return
    api
      .getConfig()
      .then((r) => setRegistry(r.assets))
      .catch(() => {})
  }, [wallet.chain])

  async function handleAddAsset(code: string, issuer: string) {
    setPendingTrust((p) => [...p, code])
    try {
      await api.addTrustline(wallet.id, code, issuer)
      toast.success(`Adding ${code} trustline…`)
    } catch (e) {
      setPendingTrust((p) => p.filter((c) => c !== code))
      toast.error(e instanceof Error ? e.message : 'Could not add asset')
    }
  }

  const [refreshing, setRefreshing] = useState(false)

  const loadTxns = () =>
    api
      .walletTxns(wallet.id)
      .then(setBase)
      .catch(() => {})

  // Initial history load; live updates then arrive via SSE (no polling).
  useEffect(() => {
    loadTxns()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [wallet.id])

  // Manual refresh: pull incoming on-chain payments (covers the case where the
  // user never opened the Receive QR, whose modal polls), then reload history
  // and balance.
  async function handleRefresh() {
    if (refreshing) return
    setRefreshing(true)
    try {
      await api.sync(wallet.id).catch(() => {})
      await loadTxns()
      loadBalance()
    } finally {
      setRefreshing(false)
    }
  }

  // Live on-chain balances (per asset).
  const loadBalance = () => {
    api
      .balance(wallet.id)
      .then((r) => setAssets(r.assets))
      .catch(() => setAssets(null))
  }
  useEffect(loadBalance, [wallet.id])

  async function handleFund() {
    setFunding(true)
    try {
      await api.fund(wallet.id)
      loadBalance()
      toast.success('Wallet funded with testnet XLM')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Funding failed')
    } finally {
      setFunding(false)
    }
  }

  // Merge the fetched history with live SSE updates for this wallet.
  const txns = useMemo(() => {
    const map = new Map<string, Transaction>()
    base.forEach((t) => map.set(t.id, t))
    Object.values(liveTxns)
      .filter((t) => t.walletId === wallet.id)
      .forEach((t) => map.set(t.id, t))
    return [...map.values()].sort((a, b) =>
      a.createdAt < b.createdAt ? 1 : -1,
    )
  }, [base, liveTxns, wallet.id])

  // Refresh balance when a tx confirms or the cached balance changes (via SSE).
  // A short retry covers Horizon's brief indexing lag after a broadcast.
  const confirmedCount = txns.filter(
    (t) => t.walletId === wallet.id && t.status === 'confirmed',
  ).length
  useEffect(() => {
    loadBalance()
    const t = setTimeout(loadBalance, 2500)
    return () => clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [confirmedCount, wallet.balance])

  // Auto-open the detail of a just-sent tx (to show the signing animation).
  useEffect(() => {
    if (focusTxId) {
      setOpenTxId(focusTxId)
      onFocusHandled?.()
    }
  }, [focusTxId, onFocusHandled])

  const openTx = txns.find((t) => t.id === openTxId) ?? null

  return (
    <div className="mx-auto max-w-[1040px]">
      <div className="mb-5 flex items-center justify-between">
        <button
          onClick={onBack}
          className="text-sm font-semibold text-ink-soft hover:text-ink"
        >
          ← Back to wallets
        </button>
        <button
          onClick={() => setConfirmDel(true)}
          className="flex items-center gap-1.5 text-sm font-semibold text-muted hover:text-[#d33a3a]"
        >
          <TrashIcon size={16} /> Delete
        </button>
      </div>

      {confirmDel && (
        <div className="mb-5 flex flex-col items-start gap-3 border border-[#f0c2c2] bg-[#fff7f7] px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
          <span className="text-sm text-ink-soft">
            Delete <strong>{wallet.name}</strong> and its history? This can't be
            undone.
          </span>
          <div className="flex w-full shrink-0 gap-2 sm:w-auto">
            <button
              onClick={() => setConfirmDel(false)}
              className="flex-1 border border-line px-3 py-1.5 text-sm font-semibold text-ink-soft hover:bg-[#f2f5f9] sm:flex-none"
            >
              Cancel
            </button>
            <button
              disabled={deleting}
              onClick={async () => {
                setDeleting(true)
                try {
                  await onDelete(wallet.id)
                } finally {
                  setDeleting(false)
                }
              }}
              className="flex-1 bg-[#d33a3a] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#bf2f2f] disabled:opacity-50 sm:flex-none"
            >
              {deleting ? 'Deleting…' : 'Delete'}
            </button>
          </div>
        </div>
      )}

      <section className="surface mb-5">
        <div className="mb-5 flex flex-col items-start gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex w-full min-w-0 items-center gap-3 sm:flex-1">
            <ChainLogo chain={wallet.chain} size={44} />
            <div className="min-w-0 flex-1">
              <div className="text-lg font-bold">{wallet.name}</div>
              <CopyAddress address={wallet.address} truncate />
            </div>
          </div>
          <a
            href={explorerAddress(wallet.chain, wallet.address)}
            target="_blank"
            rel="noreferrer"
            className="shrink-0 whitespace-nowrap bg-brand-soft px-2.5 py-1 text-xs font-semibold uppercase text-brand hover:underline"
          >
            {wallet.chain} · explorer ↗
          </a>
        </div>

        <div className="mb-1 text-sm text-muted">Balances</div>
        {assets === null ? (
          <div className="mb-5 text-4xl font-extrabold leading-none text-muted">
            …
          </div>
        ) : (
          <div className="mb-5">
            {/* native asset — headline */}
            <div className="mb-3">
              <div className="flex items-center gap-2.5">
                <TokenLogo
                  symbol={assets[0]?.symbol ?? wallet.symbol}
                  size={30}
                />
                <div className="min-w-0 text-3xl font-extrabold leading-none sm:text-4xl">
                  {formatAmount(assets[0]?.balance ?? '0')}{' '}
                  <span className="text-xl font-semibold text-muted">
                    {assets[0]?.symbol ?? wallet.symbol}
                  </span>
                </div>
              </div>
              {assets[0] && usd(assets[0].symbol, assets[0].balance) && (
                <div className="mt-1.5 pl-[40px] text-sm text-muted">
                  ≈ {usd(assets[0].symbol, assets[0].balance)}
                </div>
              )}
            </div>
            {/* additional held tokens */}
            {assets.length > 1 &&
              (() => {
                const extra = assets.slice(1)
                const sorted = [...extra].sort((a, b) =>
                  sortBy === 'amount'
                    ? Number(b.balance) - Number(a.balance)
                    : a.symbol.localeCompare(b.symbol),
                )
                return (
                  <div>
                    {extra.length > 1 && (
                      <div className="mb-1.5 flex items-center justify-between">
                        <span className="text-xs font-medium uppercase tracking-wide text-muted">
                          {extra.length} tokens
                        </span>
                        <select
                          value={sortBy}
                          onChange={(e) =>
                            setSortBy(e.target.value as 'amount' | 'name')
                          }
                          className="border border-line bg-white px-2 py-1 text-xs font-semibold text-ink-soft"
                        >
                          <option value="amount">Sort: Amount</option>
                          <option value="name">Sort: Name</option>
                        </select>
                      </div>
                    )}
                    <div className="flex flex-col divide-y divide-line border border-line">
                      {sorted.map((a) => {
                        const v = usd(a.symbol, a.balance)
                        return (
                          <div
                            key={a.symbol}
                            className="flex min-h-[52px] items-center justify-between gap-3 px-3"
                          >
                            <span className="flex items-center gap-2 font-semibold text-ink-soft">
                              <TokenLogo symbol={a.symbol} size={22} /> {a.symbol}
                            </span>
                            <span className="text-right">
                              <div className="font-mono text-sm font-semibold">
                                {formatAmount(a.balance)}
                              </div>
                              {v && <div className="text-xs text-muted">{v}</div>}
                            </span>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                )
              })()}
          </div>
        )}

        {/* Add-asset (trustline) — registry assets the wallet doesn't hold yet */}
        {wallet.chain === 'stellar' &&
          (() => {
            const held = new Set((assets ?? []).map((a) => a.symbol))
            const pending = new Set(pendingTrust)
            const available = registry.filter(
              (a) => !held.has(a.code) && !pending.has(a.code),
            )
            const adding = registry.filter(
              (a) => pending.has(a.code) && !held.has(a.code),
            )
            if (available.length === 0 && adding.length === 0) return null
            return (
              <div className="mb-5 flex flex-wrap items-center gap-2">
                <span className="text-xs font-medium uppercase tracking-wide text-muted">
                  Add asset
                </span>
                {available.map((a) => (
                  <button
                    key={a.code + a.issuer}
                    onClick={() => handleAddAsset(a.code, a.issuer)}
                    className="flex items-center gap-1.5 border border-line px-2.5 py-1 text-xs font-semibold text-ink-soft transition-colors hover:border-brand hover:text-brand"
                  >
                    <TokenLogo symbol={a.code} size={16} />+ {a.code}
                  </button>
                ))}
                {adding.map((a) => (
                  <span
                    key={a.code + a.issuer}
                    className="flex items-center gap-1.5 border border-line px-2.5 py-1 text-xs font-semibold text-muted opacity-60"
                  >
                    <TokenLogo symbol={a.code} size={16} /> Adding {a.code}…
                  </span>
                ))}
              </div>
            )
          })()}

        <div className="grid grid-cols-2 gap-2 sm:flex sm:gap-3">
          <button
            onClick={onSend}
            className="flex items-center justify-center gap-2 bg-brand px-4 py-3 font-semibold text-white transition-colors hover:bg-brand-deep sm:px-5"
          >
            <SendIcon size={18} /> Send
          </button>
          <button
            onClick={() => setShowReceive(true)}
            className="border border-line px-4 py-3 font-semibold text-ink-soft transition-colors hover:bg-[#f2f5f9] sm:px-5"
          >
            Receive
          </button>
          <button
            onClick={handleFund}
            disabled={funding}
            className="col-span-2 border border-line px-4 py-3 font-semibold text-ink-soft transition-colors hover:bg-[#f2f5f9] disabled:opacity-50 sm:col-auto sm:px-5"
          >
            {funding ? 'Funding…' : 'Fund (testnet)'}
          </button>
        </div>
      </section>

      <div className="mb-1 flex items-start gap-2 px-1 text-sm text-ink-soft sm:items-center">
        <span className="text-brand">
          <ShieldIcon size={16} />
        </span>
        Every outgoing transaction is signed by 2 of 3 nodes.
      </div>

      <section className="surface">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-[17px] font-bold">Transactions</h3>
          <button
            type="button"
            onClick={handleRefresh}
            disabled={refreshing}
            title="Refresh transactions"
            className="flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-ink-soft hover:bg-card disabled:opacity-60"
          >
            {refreshing ? <SpinnerIcon size={14} /> : <RefreshIcon size={14} />}
            {refreshing ? 'Refreshing…' : 'Refresh'}
          </button>
        </div>
        {txns.length === 0 ? (
          <div className="py-10 text-center text-sm text-muted">
            No transactions yet.
          </div>
        ) : (
          <div className="flex flex-col">
            {txns.map((t) => {
              const trust = t.memo === 'Add trustline'
              const kind = trust
                ? `Trustline · ${t.symbol}`
                : t.type === 'in'
                  ? 'Received'
                  : 'Sent'
              return (
                <button
                  key={t.id}
                  onClick={() => setOpenTxId(t.id)}
                  className="flex items-center gap-2.5 border-b border-line py-3.5 text-left transition-colors last:border-none hover:bg-[#fafbfc] sm:gap-3"
                >
                  <div className="relative shrink-0">
                    <TokenLogo symbol={t.symbol} size={36} />
                    <span
                      className={
                        'absolute -bottom-1 -right-1 grid h-[18px] w-[18px] place-items-center rounded-full border-2 border-white text-[9px] font-bold text-white ' +
                        (t.type === 'in' ? 'bg-green-600' : 'bg-[#334]')
                      }
                    >
                      {t.type === 'in' ? '↓' : '↑'}
                    </span>
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="font-semibold">{kind}</div>
                    <div className="truncate text-[13px] text-muted">
                      <span className="font-mono">
                        {shortAddress(t.counterparty, 6, 6)}
                      </span>
                      {' · '}
                      {fmtShort(t.createdAt)}
                    </div>
                  </div>
                  <div className="max-w-[120px] shrink-0 text-right text-sm sm:max-w-none sm:text-base">
                    <div
                      className={
                        'font-semibold ' +
                        (trust
                          ? 'text-muted'
                          : t.type === 'in'
                            ? 'text-green-600'
                            : 'text-ink')
                      }
                    >
                      {trust ? '—' : (t.type === 'in' ? '+' : '−') + formatAmount(t.amount)}
                      {!trust && ` ${t.symbol}`}
                    </div>
                    <div className="mt-0.5">
                      <TxStatusBadge status={t.status} />
                    </div>
                  </div>
                </button>
              )
            })}
          </div>
        )}
      </section>

      {showReceive && (
        <ReceiveModal wallet={wallet} onClose={() => setShowReceive(false)} />
      )}
      {openTx && (
        <TxDetailModal
          tx={openTx}
          chain={wallet.chain}
          onClose={() => setOpenTxId(null)}
        />
      )}
    </div>
  )
}
