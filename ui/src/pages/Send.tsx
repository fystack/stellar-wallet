import { useEffect, useState } from 'react'
import { SendIcon, ShieldIcon } from '../icons.tsx'
import { TokenLogo } from '../logos.tsx'
import { formatAmount, shortAddress } from '../format.ts'
import { api } from '../api.ts'
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
    asset?: { code: string; issuer: string }
  }) => void
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
  const fromWallet = wallets.find((w) => w.id === from)
  const isSelfSend =
    !!fromWallet && to.trim() !== '' && to.trim() === fromWallet.address

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
  const addrValid = /^G[A-Z2-7]{55}$/.test(to.trim())
  const overBalance = amtNum > maxSend + 1e-9
  const addrError = to.trim() !== '' && !addrValid

  if (wallets.length === 0) {
    return (
      <div className="mx-auto mt-16 max-w-[460px] text-center">
        <div className="mx-auto mb-8 grid h-[78px] w-[78px] place-items-center  bg-brand text-white">
          <SendIcon size={32} />
        </div>
        <h2 className="mb-2.5 text-2xl font-bold">No wallet to send from</h2>
        <p className="mb-7 text-base leading-relaxed text-muted">
          Create a wallet first, then you can send assets across the Stellar
          network.
        </p>
        <button className="cta" onClick={onGoCreate}>
          Create your first wallet
        </button>
      </div>
    )
  }

  const canSubmit = Boolean(
    from && addrValid && amtNum > 0 && !overBalance,
  )

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const isNative = !selectedAsset?.issuer
    onSubmit({
      walletId: from,
      to: to.trim(),
      amount,
      memo: memo.trim(),
      asset: isNative
        ? undefined
        : { code: selectedAsset!.symbol, issuer: selectedAsset!.issuer! },
    })
  }

  return (
    <div className="mx-auto mt-2.5 max-w-[760px]">
      <header className="mb-6">
        <h1 className="mb-2 text-3xl font-bold">Send</h1>
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
            Recipient address
          </span>
          <input
            className="field-input font-mono"
            type="text"
            placeholder="G..."
            value={to}
            onChange={(e) => setTo(e.target.value)}
            spellCheck={false}
          />
          {addrError && (
            <span className="bg-[#fdecec] px-2.5 py-1.5 text-xs font-medium text-[#d33a3a]">
              Not a valid Stellar address (starts with G, 56 characters).
            </span>
          )}
          {isSelfSend && (
            <span className="flex items-center gap-1.5 bg-[#fff7ed] px-2.5 py-1.5 text-xs font-medium text-[#c2620b]">
              ⚠ This is the sender's own address — you'd only pay the network
              fee.
            </span>
          )}
        </label>

        <div className="flex gap-3.5">
          <label className="flex flex-1 flex-col gap-2">
            <div className="flex items-center justify-between">
              <span className="text-sm font-semibold text-ink-soft">Amount</span>
              <button
                type="button"
                onClick={() => setAmount(String(maxSend))}
                className="text-xs font-semibold text-brand hover:underline"
              >
                Max
              </button>
            </div>
            <input
              className={
                'field-input ' + (overBalance ? 'ring-2 ring-[#d33a3a]' : '')
              }
              type="number"
              min="0"
              step="any"
              placeholder="0.00"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
          </label>
          <label className="flex flex-col gap-2">
            <span className="text-sm font-semibold text-ink-soft">Asset</span>
            {assets.length > 1 ? (
              <div className="relative flex items-center">
                <span className="pointer-events-none absolute left-3">
                  <TokenLogo symbol={asset} size={20} />
                </span>
                <select
                  className="field-input w-full pl-10 font-semibold"
                  value={asset}
                  onChange={(e) => setAsset(e.target.value)}
                >
                  {assets.map((a) => (
                    <option key={a.symbol} value={a.symbol}>
                      {a.symbol} · {formatAmount(a.balance)}
                    </option>
                  ))}
                </select>
              </div>
            ) : (
              <div className="field-input flex items-center gap-2 bg-card font-semibold text-ink-soft">
                <TokenLogo symbol={asset} size={20} />
                {asset}
              </div>
            )}
          </label>
        </div>

        {selectedAsset && (
          <div
            className={
              '-mt-2 text-xs ' + (overBalance ? 'text-[#d33a3a]' : 'text-muted')
            }
          >
            {overBalance
              ? `Insufficient balance — max ${formatAmount(String(maxSend))} ${asset}`
              : `Available: ${formatAmount(selectedAsset.balance)} ${asset}${isXlm ? ' (keeps 1 XLM reserve + fee)' : ''}`}
          </div>
        )}

        <label className="flex flex-col gap-2">
          <span className="text-sm font-semibold text-ink-soft">
            Memo <em className="font-normal not-italic text-muted">(optional)</em>
          </span>
          <input
            className="field-input"
            type="text"
            placeholder="Note attached to the transaction"
            value={memo}
            onChange={(e) => setMemo(e.target.value)}
          />
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

        <div className="flex items-center gap-2.5  bg-card px-3.5 py-3 text-sm text-ink-soft">
          <span className="shrink-0 text-brand">
            <ShieldIcon size={18} />
          </span>
          <span>Requires approval from 2 of 3 nodes to sign.</span>
        </div>

        <button className="cta w-full" type="submit" disabled={!canSubmit}>
          Review &amp; sign
        </button>
      </form>
    </div>
  )
}
