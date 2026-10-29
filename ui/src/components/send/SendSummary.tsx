import { formatAmount } from '../../lib/format.ts'
import { BASE_FEE_XLM } from '../../lib/stellar.ts'

export default function SendSummary({ amount, asset }: { amount: string; asset: string }) {
  return (
    <div className="flex flex-col gap-1.5 bg-card px-3.5 py-3 text-sm">
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
        <div className="flex items-center justify-between border-t border-line pt-1.5">
          <span className="text-muted">Total</span>
          <span className="font-bold">
            {formatAmount(String((Number(amount) || 0) + Number(BASE_FEE_XLM)))} XLM
          </span>
        </div>
      )}
    </div>
  )
}
