import AssetSelect from '../AssetSelect.tsx'
import type { AssetOption } from '../AssetSelect.tsx'
import { SwapIcon } from '../../icons.tsx'
import { formatAmount } from '../../lib/format.ts'
import { assetLabel } from '../../lib/stellar.ts'

// Pay and receive panels with the flip control overlapping the divider between them.
export default function SwapPanels({
  amount,
  onAmountChange,
  payAsset,
  payOptions,
  onPayAssetChange,
  payBalance,
  maxPay,
  overBalance,
  receiveAsset,
  receiveOptions,
  onReceiveAssetChange,
  receiveAmount,
  quoting,
  canFlip,
  onFlip,
}: {
  amount: string
  onAmountChange: (amount: string) => void
  payAsset: string
  payOptions: AssetOption[]
  onPayAssetChange: (value: string) => void
  payBalance: number
  maxPay: number
  overBalance: boolean
  receiveAsset: string
  receiveOptions: AssetOption[]
  onReceiveAssetChange: (value: string) => void
  receiveAmount: string | null
  quoting: boolean
  canFlip: boolean
  onFlip: () => void
}) {
  return (
    <div className="relative flex flex-col gap-1">
      <div className="flex flex-col gap-2 border border-line bg-card px-4 pb-3 pt-3.5">
        <div className="flex items-center justify-between text-sm">
          <span className="font-semibold text-ink-soft">You pay</span>
          <button
            type="button"
            onClick={() => onAmountChange(String(maxPay))}
            className="text-xs font-semibold text-ink-soft transition hover:text-brand"
          >
            Balance: {formatAmount(String(payBalance))} · <span className="text-brand">Max</span>
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
            onChange={(e) => onAmountChange(e.target.value)}
          />
          <AssetSelect value={payAsset} options={payOptions} onChange={onPayAssetChange} />
        </div>
        {overBalance && (
          <span className="text-xs font-medium text-danger">
            Insufficient balance — max {formatAmount(String(maxPay))} {assetLabel(payAsset)}
          </span>
        )}
      </div>

      <button
        type="button"
        onClick={onFlip}
        disabled={!canFlip}
        aria-label="Flip assets"
        className="absolute left-1/2 top-1/2 z-10 grid h-9 w-9 -translate-x-1/2 -translate-y-1/2 place-items-center border border-line bg-white text-ink-soft ring-4 ring-white transition hover:border-brand hover:text-brand disabled:cursor-not-allowed disabled:text-muted disabled:hover:border-line"
      >
        <SwapIcon size={16} />
      </button>

      <div className="flex flex-col gap-2 border border-line bg-card px-4 pb-3.5 pt-3">
        <span className="text-sm font-semibold text-ink-soft">
          You receive <span className="font-normal text-muted">(estimated)</span>
        </span>
        <div className="flex items-center gap-3">
          <div className="min-w-0 flex-1 truncate text-3xl font-bold text-ink">
            {quoting ? (
              <span className="text-muted">…</span>
            ) : receiveAmount ? (
              formatAmount(receiveAmount)
            ) : (
              <span className="text-muted">0</span>
            )}
          </div>
          <AssetSelect
            value={receiveAsset}
            options={receiveOptions}
            onChange={onReceiveAssetChange}
          />
        </div>
      </div>
    </div>
  )
}
