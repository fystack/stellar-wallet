import { formatAmount } from '../../lib/format.ts'
import { assetLabel } from '../../lib/stellar.ts'
import type { SwapQuote } from '../../types.ts'

export default function QuoteDetails({
  quote,
  payAsset,
  receiveAsset,
  rate,
  minReceive,
}: {
  quote: SwapQuote
  payAsset: string
  receiveAsset: string
  rate: number
  minReceive: number
}) {
  const route = [
    assetLabel(payAsset),
    ...quote.path.map((hop) => hop.code || 'XLM'),
    assetLabel(receiveAsset),
  ]
  return (
    <div className="flex flex-col gap-2 border border-line px-3.5 py-3 text-sm">
      <div className="flex items-center justify-between">
        <span className="text-muted">Rate</span>
        <span className="font-semibold">
          1 {assetLabel(payAsset)} ≈ {formatAmount(String(rate))} {assetLabel(receiveAsset)}
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
          {route.map((code, index) => (
            <span key={index} className="flex items-center gap-1.5">
              <span className="badge badge-neutral">
                {code}
              </span>
              {index < route.length - 1 && <span className="text-muted">→</span>}
            </span>
          ))}
        </span>
      </div>
    </div>
  )
}
