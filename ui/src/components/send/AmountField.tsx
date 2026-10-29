import AssetSelect from '../AssetSelect.tsx'
import { TokenLogo } from '../../logos.tsx'
import { formatAmount } from '../../lib/format.ts'
import { assetKey } from '../../hooks/useBalances.ts'
import type { AssetBalance } from '../../hooks/useBalances.ts'

export default function AmountField({
  amount,
  onAmountChange,
  assets,
  assetId,
  onAssetChange,
  maxSend,
  overBalance,
}: {
  amount: string
  onAmountChange: (amount: string) => void
  assets: AssetBalance[]
  assetId: string
  onAssetChange: (assetId: string) => void
  maxSend: number
  overBalance: boolean
}) {
  const selected = assets.find((a) => assetKey(a) === assetId)
  const asset = selected?.symbol ?? assetId
  return (
    <div className="flex flex-col gap-2.5 bg-card px-4 pb-3.5 pt-3.5">
      <div className="flex items-center justify-between text-sm">
        <span className="font-semibold text-ink-soft">Amount</span>
        {selected && (
          <button
            type="button"
            onClick={() => onAmountChange(String(maxSend))}
            className="text-xs font-semibold text-ink-soft transition hover:text-brand"
          >
            Balance: {formatAmount(selected.balance)} · <span className="text-brand">Max</span>
          </button>
        )}
      </div>
      <div className="flex items-center gap-3">
        <input
          className={
            'min-w-0 flex-1 border-0 bg-transparent p-0 text-[34px] font-bold leading-none tracking-tight text-ink outline-none placeholder:text-line-strong ' +
            (overBalance ? 'text-danger' : '')
          }
          type="number"
          aria-label="Amount"
          min="0"
          step="any"
          placeholder="0"
          value={amount}
          onChange={(e) => onAmountChange(e.target.value)}
        />
        {assets.length > 1 ? (
          <AssetSelect
            value={assetId}
            options={assets.map((a) => ({ label: a.symbol, value: assetKey(a) }))}
            onChange={onAssetChange}
          />
        ) : (
          <div className="flex shrink-0 items-center gap-2 border border-line bg-white py-2.5 pl-3 pr-4 text-base font-bold text-ink">
            <TokenLogo symbol={asset} size={22} />
            {asset}
          </div>
        )}
      </div>
      {selected && (
        <span className={'text-xs ' + (overBalance ? 'text-danger' : 'text-muted')}>
          {overBalance
            ? `Insufficient balance — max ${formatAmount(String(maxSend))} ${asset}`
            : !selected.issuer
              ? 'Keeps 1 XLM reserve + network fee'
              : `Available ${formatAmount(selected.balance)} ${asset}`}
        </span>
      )}
    </div>
  )
}
