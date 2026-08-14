import { useEffect, useMemo, useState } from 'react'
import { SendIcon, ShieldIcon, TrashIcon } from '../icons.tsx'
import CopyAddress from '../components/CopyAddress.tsx'
import TxStatusBadge from '../components/TxStatusBadge.tsx'
import ReceiveModal from '../components/ReceiveModal.tsx'
import TxDetailModal from '../components/TxDetailModal.tsx'
import { api, explorerAddress } from '../api.ts'
import { toast } from '../toast.tsx'
import { ChainLogo, TokenLogo } from '../logos.tsx'
import { formatAmount, formatUsd } from '../format.ts'
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

  // Initial history load; live updates then arrive via SSE (no polling).
  useEffect(() => {
    api
      .walletTxns(wallet.id)
      .then(setBase)
      .catch(() => {})
  }, [wallet.id])

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
        <div className="mb-5 flex items-center justify-between gap-4 border border-[#f0c2c2] bg-[#fff7f7] px-4 py-3">
          <span className="text-sm text-ink-soft">
            Delete <strong>{wallet.name}</strong> and its history? This can't be
            undone.
          </span>
          <div className="flex shrink-0 gap-2">
            <button
              onClick={() => setConfirmDel(false)}
              className="border border-line px-3 py-1.5 text-sm font-semibold text-ink-soft hover:bg-[#f2f5f9]"
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
              className="bg-[#d33a3a] px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#bf2f2f] disabled:opacity-50"
            >
              {deleting ? 'Deleting…' : 'Delete'}
            </button>
          </div>
        </div>
      )}

      <section className="surface mb-5">
        <div className="mb-5 flex items-center justify-between gap-3">
          <div className="flex min-w-0 items-center gap-3">
            <ChainLogo chain={wallet.chain} size={44} />
            <div className="min-w-0">
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
                <div className="text-4xl font-extrabold leading-none">
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

        <div className="flex gap-3">
          <button
            onClick={onSend}
            className="flex items-center gap-2 bg-brand px-5 py-3 font-semibold text-white transition-colors hover:bg-brand-deep"
          >
            <SendIcon size={18} /> Send
          </button>
          <button
            onClick={() => setShowReceive(true)}
            className="border border-line px-5 py-3 font-semibold text-ink-soft transition-colors hover:bg-[#f2f5f9]"
          >
            Receive
          </button>
          <button
            onClick={handleFund}
            disabled={funding}
            className="border border-line px-5 py-3 font-semibold text-ink-soft transition-colors hover:bg-[#f2f5f9] disabled:opacity-50"
          >
            {funding ? 'Funding…' : 'Fund (testnet)'}
          </button>
        </div>
      </section>

      <div className="mb-1 flex items-center gap-2 px-1 text-sm text-ink-soft">
        <span className="text-brand">
          <ShieldIcon size={16} />
        </span>
        Every outgoing transaction is signed by 2 of 3 nodes.
      </div>

      <section className="surface">
        <h3 className="mb-4 text-[17px] font-bold">Transactions</h3>
        {txns.length === 0 ? (
          <div className="py-10 text-center text-sm text-muted">
            No transactions yet.
          </div>
        ) : (
          <div className="flex flex-col">
            {txns.map((t) => (
              <button
                key={t.id}
                onClick={() => setOpenTxId(t.id)}
                className="flex items-center gap-3 border-b border-line py-3.5 text-left transition-colors last:border-none hover:bg-[#fafbfc]"
              >
                <span
                  className={
                    'grid h-9 w-9 shrink-0 place-items-center text-base font-bold ' +
                    (t.type === 'in'
                      ? 'bg-green-50 text-green-600'
                      : 'bg-[#f2f5f9] text-ink-soft')
                  }
                >
                  {t.type === 'in' ? '↓' : '↑'}
                </span>
                <div className="flex-1">
                  <div className="font-semibold">
                    {t.type === 'in' ? 'Received' : 'Sent'}
                  </div>
                  <div className="font-mono text-[13px] text-muted">
                    {t.counterparty}
                  </div>
                </div>
                <div className="text-right">
                  <div
                    className={
                      'font-semibold ' +
                      (t.type === 'in' ? 'text-green-600' : 'text-ink')
                    }
                  >
                    {t.type === 'in' ? '+' : '−'}
                    {t.amount} {t.symbol}
                  </div>
                  <div className="mt-0.5">
                    <TxStatusBadge status={t.status} />
                  </div>
                </div>
              </button>
            ))}
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
