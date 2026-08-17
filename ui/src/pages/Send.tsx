import { useEffect, useRef, useState } from 'react'
import { SendIcon, ShieldIcon } from '../icons.tsx'
import { TokenLogo } from '../logos.tsx'
import { formatAmount, shortAddress } from '../format.ts'
import { api } from '../api.ts'
import QrScanner from '../components/QrScanner.tsx'
import Modal from '../components/Modal.tsx'
import type { Wallet } from '../types.ts'

// Stellar base fee: 100 stroops per operation = 0.00001 XLM.
const STELLAR_FEE = '0.00001'

type Asset = { symbol: string; balance: string; issuer?: string }

type Props = {
  wallets: Wallet[]
  preselectId?: string
  onGoCreate: () => void
  onSubmit: (data: {
    walletId: string
    to: string
    amount: string
    memo: string
    memoType?: string
    asset?: { code: string; issuer: string }
  }) => void
}

type Resolved = {
  address: string
  memo_type?: string
  memo?: string
  federation?: string
}

// A recipient worth resolving: G/M address or a name*domain federation string.
function looksResolvable(value: string): boolean {
  const v = value.trim()
  if (v === '') return false
  if (v.includes('*')) return v.split('*').length === 2 && v.split('*')[1].includes('.')
  return (v[0] === 'G' || v[0] === 'M') && v.length >= 12
}

export default function Send({
  wallets: allWallets,
  preselectId,
  onGoCreate,
  onSubmit,
}: Props) {
  // Only wallets whose keygen finished can sign.
  const wallets = allWallets.filter((w) => w.status === 'ready')
  const [from, setFrom] = useState(preselectId ?? wallets[0]?.id ?? '')
  // Keep the dropdown in sync with the wallet the user opened Send from.
  useEffect(() => {
    setFrom(preselectId ?? wallets[0]?.id ?? '')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [preselectId])
  const [to, setTo] = useState('')
  const [amount, setAmount] = useState('')
  const [memo, setMemo] = useState('')
  const [memoTouched, setMemoTouched] = useState(false)
  const [showScanner, setShowScanner] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)

  // Recipient resolution: turns a G/M/federation string into a concrete
  // destination and surfaces any memo the recipient requires.
  const [resolved, setResolved] = useState<Resolved | null>(null)
  const [resolving, setResolving] = useState(false)
  const [resolveErr, setResolveErr] = useState('')
  const resolveSeq = useRef(0)

  useEffect(() => {
    const value = to.trim()
    setResolved(null)
    setResolveErr('')
    if (!looksResolvable(value)) {
      setResolving(false)
      return
    }
    setResolving(true)
    const seq = ++resolveSeq.current
    const timer = setTimeout(async () => {
      try {
        const r = await api.resolve(value)
        if (seq !== resolveSeq.current) return
        setResolved(r)
        if (r.memo && !memoTouched) setMemo(r.memo)
      } catch (e) {
        if (seq !== resolveSeq.current) return
        setResolveErr(e instanceof Error ? e.message : 'Could not resolve address')
      } finally {
        if (seq === resolveSeq.current) setResolving(false)
      }
    }, 400)
    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [to])

  const fromWallet = wallets.find((w) => w.id === from)
  const isSelfSend =
    !!fromWallet && !!resolved && resolved.address === fromWallet.address
  const memoRequired = !!resolved?.memo_type

  // Assets the selected wallet actually holds — populates the Asset dropdown.
  const [assets, setAssets] = useState<Asset[]>([])
  const [asset, setAsset] = useState('XLM')
  useEffect(() => {
    if (!from) return
    api
      .balance(from)
      .then((r) => {
        setAssets(r.assets)
        setAsset(r.assets[0]?.symbol ?? 'XLM')
      })
      .catch(() => setAssets([]))
  }, [from])
  const selectedAsset = assets.find((a) => a.symbol === asset)
  const isXlm = asset === 'XLM'
  const bal = Number(selectedAsset?.balance ?? 0)
  const feeXlm = 0.00001
  // Max sendable: XLM keeps fee + 1 XLM base reserve; tokens send full balance.
  const maxSend = isXlm ? Math.max(0, bal - feeXlm - 1) : bal
  const amtNum = Number(amount) || 0
  const addrValid = resolved !== null
  const overBalance = amtNum > maxSend + 1e-9
  const memoMissing = memoRequired && memo.trim() === ''

  if (wallets.length === 0) {
    return (
      <div className="mx-auto mt-10 max-w-[460px] text-center sm:mt-16">
        <div className="mx-auto mb-6 grid h-16 w-16 place-items-center bg-brand text-white sm:mb-8 sm:h-[78px] sm:w-[78px]">
          <SendIcon size={32} />
        </div>
        <h2 className="mb-2.5 text-2xl font-bold">No wallet to send from</h2>
        <p className="mb-7 text-base leading-relaxed text-muted">
          Create a wallet first, then you can send assets across the Stellar
          network.
        </p>
        <button className="cta w-full !px-6 sm:w-auto" onClick={onGoCreate}>
          Create your first wallet
        </button>
      </div>
    )
  }

  const canSubmit = Boolean(
    from && addrValid && amtNum > 0 && !overBalance && !memoMissing,
  )

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!canSubmit) return
    // Don't sign yet — open a confirmation step first.
    setShowConfirm(true)
  }

  function confirmSend() {
    if (!resolved) return
    const isNative = !selectedAsset?.issuer
    onSubmit({
      walletId: from,
      to: resolved.address,
      amount,
      memo: memo.trim(),
      memoType: resolved.memo_type,
      asset: isNative
        ? undefined
        : { code: selectedAsset!.symbol, issuer: selectedAsset!.issuer! },
    })
    setShowConfirm(false)
  }

  return (
    <div className="mx-auto mt-2.5 max-w-[760px]">
      <header className="mb-6">
        <h1 className="mb-2 text-2xl font-bold sm:text-3xl">Send</h1>
        <p className="text-base leading-relaxed text-muted">
          Transfer assets. The transaction is signed by 2 of 3 nodes — your key
          is never assembled.
        </p>
      </header>

      <form className="surface flex flex-col gap-[18px]" onSubmit={handleSubmit}>
        <label className="flex flex-col gap-2">
          <span className="text-sm font-semibold text-ink-soft">From wallet</span>
          <select
            className="field-input"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
          >
            {wallets.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name} · {shortAddress(w.address)} · {formatAmount(w.balance)}{' '}
                {w.symbol}
              </option>
            ))}
          </select>
        </label>

        <label className="flex flex-col gap-2">
          <span className="text-sm font-semibold text-ink-soft">
            Recipient
          </span>
          <div className="relative flex items-center">
            <input
              className="field-input pr-20 font-mono"
              type="text"
              placeholder="G…, M…, or name*domain.com"
              value={to}
              onChange={(e) => setTo(e.target.value)}
              spellCheck={false}
            />
            <button
              type="button"
              onClick={() => setShowScanner(true)}
              className="absolute right-2 text-xs font-semibold text-brand hover:underline"
            >
              Scan QR
            </button>
          </div>
          {resolving && (
            <span className="text-xs text-muted">Resolving address…</span>
          )}
          {resolveErr && (
            <span className="bg-danger-soft px-2.5 py-1.5 text-xs font-medium text-danger">
              {resolveErr}
            </span>
          )}
          {resolved?.federation && (
            <span className="bg-brand-soft px-2.5 py-1.5 text-xs font-medium text-brand">
              ✓ {resolved.federation} → {shortAddress(resolved.address)}
            </span>
          )}
          {resolved && !resolved.federation && to.trim()[0] === 'M' && (
            <span className="bg-brand-soft px-2.5 py-1.5 text-xs font-medium text-brand">
              ✓ Muxed address (memo encoded in address)
            </span>
          )}
          {isSelfSend && (
            <span className="flex items-center gap-1.5 bg-warning-soft px-2.5 py-1.5 text-xs font-medium text-warning">
              ⚠ This is the sender's own address — you'd only pay the network
              fee.
            </span>
          )}
        </label>

        {/* Amount */}
        <div className="flex flex-col gap-2 border border-line bg-card px-4 pb-3 pt-3.5">
          <div className="flex items-center justify-between text-sm">
            <span className="font-semibold text-ink-soft">Amount</span>
            {selectedAsset && (
              <button
                type="button"
                onClick={() => setAmount(String(maxSend))}
                className="text-xs font-semibold text-ink-soft transition hover:text-brand"
              >
                Balance: {formatAmount(selectedAsset.balance)} ·{' '}
                <span className="text-brand">Max</span>
              </button>
            )}
          </div>
          <div className="flex items-center gap-3">
            <input
              className={
                'min-w-0 flex-1 border-0 bg-transparent p-0 text-3xl font-bold text-ink outline-none placeholder:text-muted ' +
                (overBalance ? 'text-danger' : '')
              }
              type="number"
              min="0"
              step="any"
              placeholder="0"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
            {assets.length > 1 ? (
              <div className="relative flex shrink-0 items-center">
                <span className="pointer-events-none absolute left-3 z-10">
                  <TokenLogo symbol={asset} size={22} />
                </span>
                <select
                  className="w-[130px] cursor-pointer appearance-none border border-line bg-white py-2.5 pl-11 pr-8 text-base font-bold text-ink outline-none transition focus:border-brand"
                  value={asset}
                  onChange={(e) => setAsset(e.target.value)}
                >
                  {assets.map((a) => (
                    <option key={a.symbol} value={a.symbol}>
                      {a.symbol}
                    </option>
                  ))}
                </select>
                <span className="pointer-events-none absolute right-3 text-muted">
                  <svg width={14} height={14} viewBox="0 0 24 24" fill="none">
                    <path
                      d="M6 9l6 6 6-6"
                      stroke="currentColor"
                      strokeWidth={2}
                      strokeLinecap="round"
                      strokeLinejoin="round"
                    />
                  </svg>
                </span>
              </div>
            ) : (
              <div className="flex shrink-0 items-center gap-2 border border-line bg-white py-2.5 pl-3 pr-4 text-base font-bold text-ink">
                <TokenLogo symbol={asset} size={22} />
                {asset}
              </div>
            )}
          </div>
          {selectedAsset && (
            <span
              className={
                'text-xs ' + (overBalance ? 'text-danger' : 'text-muted')
              }
            >
              {overBalance
                ? `Insufficient balance — max ${formatAmount(String(maxSend))} ${asset}`
                : isXlm
                  ? 'Keeps 1 XLM reserve + network fee'
                  : `Available ${formatAmount(selectedAsset.balance)} ${asset}`}
            </span>
          )}
        </div>

        <label className="flex flex-col gap-2">
          <span className="text-sm font-semibold text-ink-soft">
            Memo{' '}
            {memoRequired ? (
              <em className="font-semibold not-italic text-warning">
                (required by recipient)
              </em>
            ) : (
              <em className="font-normal not-italic text-muted">(optional)</em>
            )}
          </span>
          <input
            className={
              'field-input ' + (memoMissing ? 'ring-2 ring-danger' : '')
            }
            type="text"
            placeholder={
              memoRequired
                ? `This recipient needs a "${resolved?.memo_type}" memo`
                : 'Note attached to the transaction'
            }
            value={memo}
            onChange={(e) => {
              setMemo(e.target.value)
              setMemoTouched(true)
            }}
          />
          {memoRequired && (
            <span className="flex items-center gap-1.5 bg-warning-soft px-2.5 py-1.5 text-xs font-medium text-warning">
              ⚠ This recipient (often an exchange) requires a memo — sending
              without it can lose your funds.
            </span>
          )}
        </label>

        {/* Fee / total summary */}
        <div className="flex flex-col gap-1.5 bg-card px-3.5 py-3 text-sm">
          <div className="flex items-center justify-between">
            <span className="text-muted">Amount</span>
            <span className="font-semibold">
              {amount ? formatAmount(amount) : '0'} {asset}
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-muted">Network fee (est.)</span>
            <span className="font-semibold">{STELLAR_FEE} XLM</span>
          </div>
          {asset === 'XLM' && (
            <div className="flex items-center justify-between border-t border-line pt-1.5">
              <span className="text-muted">Total</span>
              <span className="font-bold">
                {formatAmount(String((Number(amount) || 0) + Number(STELLAR_FEE)))}{' '}
                XLM
              </span>
            </div>
          )}
        </div>

        <div className="flex items-start gap-2.5 bg-card px-3.5 py-3 text-sm text-ink-soft sm:items-center">
          <span className="shrink-0 text-brand">
            <ShieldIcon size={18} />
          </span>
          <span>Requires approval from 2 of 3 nodes to sign.</span>
        </div>

        <button className="cta w-full" type="submit" disabled={!canSubmit}>
          Review &amp; sign
        </button>
      </form>

      {showScanner && (
        <QrScanner
          onResult={(text) => setTo(text)}
          onClose={() => setShowScanner(false)}
        />
      )}

      {showConfirm && resolved && (
        <Modal title="Confirm transaction" onClose={() => setShowConfirm(false)}>
          <div className="flex flex-col gap-4">
            <div className="flex flex-col items-center gap-1.5 border border-line bg-card py-5">
              <TokenLogo symbol={asset} size={34} />
              <div className="text-3xl font-extrabold">
                {formatAmount(amount)}{' '}
                <span className="text-lg text-muted">{asset}</span>
              </div>
            </div>

            <div className="divide-y divide-line bg-card px-4 text-sm">
              <div className="flex items-center justify-between gap-3 py-2.5">
                <span className="text-muted">From</span>
                <span className="font-semibold">{fromWallet?.name}</span>
              </div>
              <div className="flex items-center justify-between gap-3 py-2.5">
                <span className="shrink-0 text-muted">To</span>
                <span className="truncate text-right font-mono text-[13px]">
                  {resolved.federation
                    ? resolved.federation
                    : shortAddress(resolved.address, 8, 8)}
                </span>
              </div>
              {memo.trim() && (
                <div className="flex items-center justify-between gap-3 py-2.5">
                  <span className="shrink-0 text-muted">Memo</span>
                  <span className="truncate text-right">{memo.trim()}</span>
                </div>
              )}
              <div className="flex items-center justify-between gap-3 py-2.5">
                <span className="text-muted">Network fee</span>
                <span className="font-semibold">{STELLAR_FEE} XLM</span>
              </div>
            </div>

            <div className="flex items-center gap-2.5 text-sm text-muted">
              <span className="shrink-0 text-brand">
                <ShieldIcon size={16} />
              </span>
              <span>
                Signed by 2 of 3 nodes. This action broadcasts on-chain and
                can't be undone.
              </span>
            </div>

            <div className="flex gap-3">
              <button
                type="button"
                className="cta-secondary flex-1"
                onClick={() => setShowConfirm(false)}
              >
                Cancel
              </button>
              <button
                type="button"
                className="cta flex-1 !px-6 !py-3 !text-base"
                onClick={confirmSend}
              >
                Confirm &amp; sign
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}
