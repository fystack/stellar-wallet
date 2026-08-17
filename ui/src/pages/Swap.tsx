import { useEffect, useRef, useState } from 'react'
import { SwapIcon, ShieldIcon } from '../icons.tsx'
import { TokenLogo } from '../logos.tsx'
import { formatAmount, shortAddress } from '../format.ts'
import { api, ApiError } from '../api.ts'
import { toast } from '../toast.tsx'
import Modal from '../components/Modal.tsx'
import type { SwapQuote, Wallet } from '../types.ts'

type AssetOption = { label: string; value: string; balance?: string }

// "XLM" or "CODE:ISSUER" → a short human label.
function assetLabel(value: string): string {
  return value === 'XLM' ? 'XLM' : value.split(':')[0]
}

function ChevronDown() {
  return (
    <svg width={14} height={14} viewBox="0 0 24 24" fill="none">
      <path
        d="M6 9l6 6 6-6"
        stroke="currentColor"
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

// A token picker: logo + native select styled to match the sharp-cornered UI.
function AssetSelect({
  value,
  options,
  onChange,
}: {
  value: string
  options: AssetOption[]
  onChange: (value: string) => void
}) {
  return (
    <div className="relative flex shrink-0 items-center">
      <span className="pointer-events-none absolute left-3 z-10">
        <TokenLogo symbol={assetLabel(value)} size={22} />
      </span>
      <select
        className="w-[140px] cursor-pointer appearance-none border border-line bg-white py-2.5 pl-11 pr-8 text-base font-bold text-ink outline-none transition focus:border-brand"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      >
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
      <span className="pointer-events-none absolute right-3 text-muted">
        <ChevronDown />
      </span>
    </div>
  )
}

// A few testnet assets with live DEX liquidity, so the receive list is never
// empty on a fresh wallet. On mainnet these come from Settings instead.
// Stellar testnet assets with live DEX liquidity from XLM (and real logos),
// so the receive list is useful on a fresh wallet.
const CIRCLE_USDC =
  'USDC:GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5'
const TESTNET_MM = 'GAHPYWLK6YRN7CVYZOO4H3VDRZ7PVF5UJGLZCSPAEIKJE2XSWF5LAGER'
const SUGGESTED_TESTNET_ASSETS: AssetOption[] = [
  { label: 'USDC', value: CIRCLE_USDC },
  { label: 'USDT', value: `USDT:${TESTNET_MM}` },
  { label: 'BTC', value: `BTC:${TESTNET_MM}` },
  { label: 'ETH', value: `ETH:${TESTNET_MM}` },
]

const SLIPPAGE_OPTIONS = [
  { label: '0.5%', bps: 50 },
  { label: '1%', bps: 100 },
  { label: '2%', bps: 200 },
]

export default function Swap({
  wallets: allWallets,
  preselectId,
  onGoCreate,
  onSubmit,
}: {
  wallets: Wallet[]
  preselectId?: string
  onGoCreate: () => void
  onSubmit: (data: {
    walletId: string
    from: string
    to: string
    amount: string
    slippageBps: number
  }) => Promise<void>
}) {
  const wallets = allWallets.filter(
    (w) => w.status === 'ready' && w.chain === 'stellar',
  )
  const [from, setFrom] = useState(preselectId ?? wallets[0]?.id ?? '')
  useEffect(() => {
    setFrom(preselectId ?? wallets[0]?.id ?? '')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [preselectId])

  // Asset universe: held assets fund the "pay" side; held + configured
  // trustline assets are valid "receive" targets.
  const [held, setHeld] = useState<AssetOption[]>([])
  const [toOptions, setToOptions] = useState<AssetOption[]>([])
  const [payAsset, setPayAsset] = useState('XLM')
  const [receiveAsset, setReceiveAsset] = useState('')
  const [amount, setAmount] = useState('')
  const [slippage, setSlippage] = useState(100)
  const [submitting, setSubmitting] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)
  const [addingTrustline, setAddingTrustline] = useState(false)
  // Bumped after adding a trustline to re-fetch balances.
  const [reload, setReload] = useState(0)

  useEffect(() => {
    if (!from) return
    Promise.all([api.balance(from), api.getConfig().catch(() => null)]).then(
      ([bal, config]) => {
        const heldOptions: AssetOption[] = bal.assets.map((a) => ({
          label: a.symbol,
          value: a.issuer ? `${a.symbol}:${a.issuer}` : 'XLM',
          balance: a.balance,
        }))
        setHeld(heldOptions)
        setPayAsset(heldOptions[0]?.value ?? 'XLM')

        const targets = new Map<string, AssetOption>()
        targets.set('XLM', { label: 'XLM', value: 'XLM' })
        heldOptions.forEach((o) => targets.set(o.value, o))
        config?.assets.forEach((a) =>
          targets.set(`${a.code}:${a.issuer}`, {
            label: a.code,
            value: `${a.code}:${a.issuer}`,
          }),
        )
        SUGGESTED_TESTNET_ASSETS.forEach((a) => {
          if (!targets.has(a.value)) targets.set(a.value, a)
        })
        setToOptions([...targets.values()])
      },
    )
  }, [from, reload])

  // Default the receive asset to the first option that differs from pay.
  useEffect(() => {
    const firstDifferent = toOptions.find((o) => o.value !== payAsset)
    setReceiveAsset((prev) =>
      prev && prev !== payAsset ? prev : firstDifferent?.value ?? '',
    )
  }, [toOptions, payAsset])

  const payBalance = Number(
    held.find((h) => h.value === payAsset)?.balance ?? 0,
  )
  const amtNum = Number(amount) || 0
  const isXlm = payAsset === 'XLM'
  const maxPay = isXlm ? Math.max(0, payBalance - 1 - 0.00001) : payBalance
  const overBalance = amtNum > maxPay + 1e-9

  // Live quote fetching, debounced.
  const [quote, setQuote] = useState<SwapQuote | null>(null)
  const [quoteErr, setQuoteErr] = useState('')
  const [quoting, setQuoting] = useState(false)
  const quoteSeq = useRef(0)

  useEffect(() => {
    setQuote(null)
    setQuoteErr('')
    if (!from || !receiveAsset || payAsset === receiveAsset || amtNum <= 0) {
      setQuoting(false)
      return
    }
    setQuoting(true)
    const seq = ++quoteSeq.current
    const timer = setTimeout(async () => {
      try {
        const q = await api.swapQuote(from, payAsset, receiveAsset, amount)
        if (seq !== quoteSeq.current) return
        setQuote(q)
      } catch (e) {
        if (seq !== quoteSeq.current) return
        setQuoteErr(e instanceof ApiError ? e.message : 'No quote available')
      } finally {
        if (seq === quoteSeq.current) setQuoting(false)
      }
    }, 450)
    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [from, payAsset, receiveAsset, amount])

  const rate =
    quote && amtNum > 0
      ? Number(quote.dest_amount) / amtNum
      : 0
  const minReceive = quote
    ? Number(quote.dest_amount) * (1 - slippage / 10000)
    : 0

  // Receiving a credit asset requires a trustline to it first (Stellar rule).
  const needsTrustline =
    receiveAsset !== '' &&
    receiveAsset !== 'XLM' &&
    !held.some((h) => h.value === receiveAsset)

  const canSubmit = Boolean(
    from &&
      quote &&
      !overBalance &&
      amtNum > 0 &&
      payAsset !== receiveAsset &&
      !needsTrustline &&
      !submitting,
  )

  async function handleAddTrustline() {
    if (addingTrustline) return
    const [code, issuer] = receiveAsset.split(':')
    if (!code || !issuer) return
    // Stay disabled through the whole signing + confirmation window (not just
    // the dispatch) so a second click can't submit a duplicate trustline.
    setAddingTrustline(true)
    try {
      await api.addTrustline(from, code, issuer)
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to add trustline')
      setAddingTrustline(false)
      return
    }
    toast.success(`Adding ${code} trustline — signing…`)
    // Trustline is signed by the cluster asynchronously; poll balances until the
    // new asset shows up, then unlock the swap.
    let tries = 0
    const poll = setInterval(async () => {
      tries++
      const bal = await api.balance(from).catch(() => null)
      if (bal?.assets.some((a) => a.symbol === code && a.issuer === issuer)) {
        clearInterval(poll)
        setReload((n) => n + 1)
        setAddingTrustline(false)
        toast.success(`${code} trustline ready`)
      } else if (tries >= 15) {
        clearInterval(poll)
        setAddingTrustline(false)
        toast.error(`${code} trustline is taking longer than expected — try again shortly`)
      }
    }, 2000)
  }

  // Flip only works when we actually hold the asset we'd start paying with.
  const canFlip = held.some((h) => h.value === receiveAsset)
  function flip() {
    if (!canFlip) return
    setPayAsset(receiveAsset)
    setReceiveAsset(payAsset)
    setAmount('')
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!canSubmit) return
    // Confirm before signing/broadcasting.
    setShowConfirm(true)
  }

  async function confirmSwap() {
    setSubmitting(true)
    try {
      await onSubmit({ walletId: from, from: payAsset, to: receiveAsset, amount, slippageBps: slippage })
      setShowConfirm(false)
    } finally {
      setSubmitting(false)
    }
  }

  if (wallets.length === 0) {
    return (
      <div className="mx-auto mt-10 max-w-[460px] text-center sm:mt-16">
        <div className="mx-auto mb-6 grid h-16 w-16 place-items-center bg-brand text-white sm:mb-8 sm:h-[78px] sm:w-[78px]">
          <SwapIcon size={32} />
        </div>
        <h2 className="mb-2.5 text-2xl font-bold">No Stellar wallet to swap</h2>
        <p className="mb-7 text-base leading-relaxed text-muted">
          Swaps use the built-in Stellar DEX. Create a Stellar wallet to get
          started.
        </p>
        <button className="cta w-full !px-6 sm:w-auto" onClick={onGoCreate}>
          Create a wallet
        </button>
      </div>
    )
  }

  return (
    <div className="mx-auto mt-2.5 max-w-[560px]">
      <header className="mb-6">
        <h1 className="mb-2 text-2xl font-bold sm:text-3xl">Swap</h1>
        <p className="text-base leading-relaxed text-muted">
          Trade assets on the Stellar DEX. The path payment is signed by 2 of 3
          nodes.
        </p>
      </header>

      <form className="surface flex flex-col gap-[18px]" onSubmit={handleSubmit}>
        <label className="flex flex-col gap-2">
          <span className="text-sm font-semibold text-ink-soft">Wallet</span>
          <select
            className="field-input"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
          >
            {wallets.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name} · {shortAddress(w.address)}
              </option>
            ))}
          </select>
        </label>

        {/* Pay / Receive panels with an overlapping flip control */}
        <div className="relative flex flex-col gap-1">
          {/* Pay */}
          <div className="flex flex-col gap-2 border border-line bg-card px-4 pb-3 pt-3.5">
            <div className="flex items-center justify-between text-sm">
              <span className="font-semibold text-ink-soft">You pay</span>
              <button
                type="button"
                onClick={() => setAmount(String(maxPay))}
                className="text-xs font-semibold text-ink-soft transition hover:text-brand"
              >
                Balance: {formatAmount(String(payBalance))} ·{' '}
                <span className="text-brand">Max</span>
              </button>
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
              <AssetSelect
                value={payAsset}
                options={held}
                onChange={setPayAsset}
              />
            </div>
            {overBalance && (
              <span className="text-xs font-medium text-danger">
                Insufficient balance — max {formatAmount(String(maxPay))}{' '}
                {assetLabel(payAsset)}
              </span>
            )}
          </div>

          {/* Flip — sits on the divider between the two panels */}
          <button
            type="button"
            onClick={flip}
            disabled={!canFlip}
            aria-label="Flip assets"
            className="absolute left-1/2 top-1/2 z-10 grid h-9 w-9 -translate-x-1/2 -translate-y-1/2 place-items-center border border-line bg-white text-ink-soft ring-4 ring-white transition hover:border-brand hover:text-brand disabled:cursor-not-allowed disabled:text-muted disabled:hover:border-line"
          >
            <SwapIcon size={16} />
          </button>

          {/* Receive */}
          <div className="flex flex-col gap-2 border border-line bg-card px-4 pb-3.5 pt-3">
            <span className="text-sm font-semibold text-ink-soft">
              You receive{' '}
              <span className="font-normal text-muted">(estimated)</span>
            </span>
            <div className="flex items-center gap-3">
              <div className="min-w-0 flex-1 truncate text-3xl font-bold text-ink">
                {quoting ? (
                  <span className="text-muted">…</span>
                ) : quote ? (
                  formatAmount(quote.dest_amount)
                ) : (
                  <span className="text-muted">0</span>
                )}
              </div>
              <AssetSelect
                value={receiveAsset}
                options={toOptions.filter((o) => o.value !== payAsset)}
                onChange={setReceiveAsset}
              />
            </div>
          </div>
        </div>

        {quoteErr && (
          <div className="bg-danger-soft px-3 py-2 text-sm font-medium text-danger">
            {quoteErr}
          </div>
        )}

        {/* Slippage */}
        <div className="flex items-center justify-between">
          <span className="text-sm font-semibold text-ink-soft">
            Slippage tolerance
          </span>
          <div className="flex gap-1.5">
            {SLIPPAGE_OPTIONS.map((s) => (
              <button
                key={s.bps}
                type="button"
                onClick={() => setSlippage(s.bps)}
                className={
                  'border px-3 py-1.5 text-sm font-semibold transition ' +
                  (slippage === s.bps
                    ? 'border-brand bg-brand text-white'
                    : 'border-line bg-white text-ink-soft hover:bg-card')
                }
              >
                {s.label}
              </button>
            ))}
          </div>
        </div>

        {quote && (
          <div className="flex flex-col gap-2 border border-line px-3.5 py-3 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-muted">Rate</span>
              <span className="font-semibold">
                1 {assetLabel(payAsset)} ≈ {formatAmount(String(rate))}{' '}
                {assetLabel(receiveAsset)}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-muted">Minimum received</span>
              <span className="font-semibold">
                {formatAmount(String(minReceive))} {assetLabel(receiveAsset)}
              </span>
            </div>
            <div className="flex items-center justify-between gap-3">
              <span className="shrink-0 text-muted">Route</span>
              <span className="flex flex-wrap items-center justify-end gap-1.5">
                {[
                  assetLabel(payAsset),
                  ...quote.path.map((p) => p.code || 'XLM'),
                  assetLabel(receiveAsset),
                ].map((code, i, arr) => (
                  <span key={i} className="flex items-center gap-1.5">
                    <span className="bg-card px-2 py-0.5 text-xs font-semibold text-ink-soft">
                      {code}
                    </span>
                    {i < arr.length - 1 && (
                      <span className="text-muted">→</span>
                    )}
                  </span>
                ))}
              </span>
            </div>
          </div>
        )}

        <div className="flex items-center gap-2.5 text-sm text-muted">
          <span className="shrink-0 text-brand">
            <ShieldIcon size={16} />
          </span>
          <span>Signed by 2 of 3 nodes — your key is never assembled.</span>
        </div>

        {needsTrustline ? (
          <>
            <div className="flex items-start gap-2 bg-warning-soft px-3.5 py-3 text-sm text-warning">
              <span className="shrink-0">⚠</span>
              <span>
                You need a trustline to{' '}
                <strong>{assetLabel(receiveAsset)}</strong> before you can
                receive it. Add it once, then swap.
              </span>
            </div>
            <button
              type="button"
              className="cta w-full"
              disabled={addingTrustline}
              onClick={handleAddTrustline}
            >
              {addingTrustline
                ? 'Adding trustline…'
                : `Add ${assetLabel(receiveAsset)} trustline`}
            </button>
          </>
        ) : (
          <button className="cta w-full" type="submit" disabled={!canSubmit}>
            {submitting ? 'Submitting…' : 'Review & sign swap'}
          </button>
        )}
      </form>

      {showConfirm && quote && (
        <Modal
          title="Confirm swap"
          onClose={() => !submitting && setShowConfirm(false)}
        >
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1 border border-line bg-card px-4 py-4">
              <div className="flex items-center gap-2 text-lg font-bold text-ink-soft">
                <TokenLogo symbol={assetLabel(payAsset)} size={22} />−
                {formatAmount(amount)} {assetLabel(payAsset)}
              </div>
              <div className="pl-1 text-muted">↓</div>
              <div className="flex items-center gap-2 text-2xl font-extrabold text-success">
                <TokenLogo symbol={assetLabel(receiveAsset)} size={24} />≈
                {formatAmount(quote.dest_amount)} {assetLabel(receiveAsset)}
              </div>
            </div>

            <div className="divide-y divide-line bg-card px-4 text-sm">
              <div className="flex items-center justify-between py-2.5">
                <span className="text-muted">Minimum received</span>
                <span className="font-semibold">
                  {formatAmount(String(minReceive))} {assetLabel(receiveAsset)}
                </span>
              </div>
              <div className="flex items-center justify-between py-2.5">
                <span className="text-muted">Slippage</span>
                <span className="font-semibold">{slippage / 100}%</span>
              </div>
              <div className="flex items-center justify-between py-2.5">
                <span className="text-muted">Venue</span>
                <span className="font-semibold">Stellar DEX</span>
              </div>
            </div>

            <div className="flex items-center gap-2.5 text-sm text-muted">
              <span className="shrink-0 text-brand">
                <ShieldIcon size={16} />
              </span>
              <span>
                Rate may move slightly. Signed by 2 of 3 nodes and broadcast
                on-chain — can't be undone.
              </span>
            </div>

            <div className="flex gap-3">
              <button
                type="button"
                className="cta-secondary flex-1"
                disabled={submitting}
                onClick={() => setShowConfirm(false)}
              >
                Cancel
              </button>
              <button
                type="button"
                className="cta flex-1 !px-6 !py-3 !text-base"
                disabled={submitting}
                onClick={confirmSwap}
              >
                {submitting ? 'Submitting…' : 'Confirm & swap'}
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}
