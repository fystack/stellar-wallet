import { formatAmount } from '../../lib/format.ts'
import { BASE_FEE_XLM } from '../../lib/stellar.ts'

export default function SendSummary({ amount, asset }: { amount: string; asset: string }) {
  return (
    <div className="flex flex-col gap-2.5 bg-card px-4 py-3.5 text-sm">
      <div className="flex items-center justify-between">
        <span className="text-muted">Amount</span>
        <span className="font-semibold">
          {amount ? formatAmount(amount) : '0'} {asset}
        </span>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-muted">Network fee (est.)</span>
        <span className="font-semibold">{BASE_FEE_XLM} XLM</span>
      </div>
      {asset === 'XLM' && (
        <div className="mt-0.5 flex items-center justify-between border-t border-line pt-3">
          <span className="font-semibold text-ink-soft">Total</span>
          <span className="text-base font-bold">
            {formatAmount(String((Number(amount) || 0) + Number(BASE_FEE_XLM)))} XLM
          </span>
        </div>
      )}
    </div>
  )
}
