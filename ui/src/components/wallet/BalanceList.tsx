import { useState } from 'react'
import { TokenLogo } from '../../logos.tsx'
import { formatAmount } from '../../lib/format.ts'
import { assetKey } from '../../hooks/useBalances.ts'
import type { AssetBalance } from '../../hooks/useBalances.ts'

type SortKey = 'amount' | 'name'

// The native asset as a headline, then other held tokens in a sortable list.
export default function BalanceList({
  assets,
  fallbackSymbol,
  usd,
}: {
  assets: AssetBalance[] | null
  fallbackSymbol: string
  usd: (symbol: string, amount: string) => string
}) {
  const [sortBy, setSortBy] = useState<SortKey>('amount')

  if (assets === null) {
    return (
      <div className="mb-5 flex flex-col gap-2.5">
        <div className="flex items-center gap-2.5">
          <div className="skeleton h-[30px] w-[30px] rounded-full" />
          <div className="skeleton h-9 w-44" />
        </div>
        <div className="skeleton ml-[40px] h-4 w-24" />
      </div>
    )
  }

  const [native, ...tokens] = assets
  const nativeSymbol = native?.symbol ?? fallbackSymbol
  const nativeUsd = native ? usd(native.symbol, native.balance) : ''
  const sortedTokens = [...tokens].sort((a, b) =>
    sortBy === 'amount' ? Number(b.balance) - Number(a.balance) : a.symbol.localeCompare(b.symbol),
  )

  return (
    <div className="mb-5">
      <div className="mb-3">
        <div className="flex items-center gap-2.5">
          <TokenLogo symbol={nativeSymbol} size={30} />
          <div className="min-w-0 text-3xl font-extrabold leading-none sm:text-4xl">
            {formatAmount(native?.balance ?? '0')}{' '}
            <span className="text-xl font-semibold text-muted">{nativeSymbol}</span>
          </div>
        </div>
        {nativeUsd && <div className="mt-1.5 pl-[40px] text-sm text-muted">≈ {nativeUsd}</div>}
      </div>

      {tokens.length > 0 && (
        <div>
          {tokens.length > 1 && (
            <div className="mb-1.5 flex items-center justify-between">
              <span className="text-xs font-medium uppercase tracking-wide text-muted">
                {tokens.length} tokens
              </span>
              <select
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as SortKey)}
                className="border border-line bg-white px-2 py-1 text-xs font-semibold text-ink-soft"
              >
                <option value="amount">Sort: Amount</option>
                <option value="name">Sort: Name</option>
              </select>
            </div>
          )}
          <div className="flex flex-col divide-y divide-line border border-line">
            {sortedTokens.map((token) => {
              const value = usd(token.symbol, token.balance)
              return (
                <div
                  key={assetKey(token)}
                  className="flex min-h-[52px] items-center justify-between gap-3 px-3"
                >
                  <span className="flex items-center gap-2 font-semibold text-ink-soft">
                    <TokenLogo symbol={token.symbol} size={22} /> {token.symbol}
                  </span>
                  <span className="text-right">
                    <div className="font-mono text-sm font-semibold">
                      {formatAmount(token.balance)}
                    </div>
                    {value && <div className="text-xs text-muted">{value}</div>}
                  </span>
                </div>
              )
            })}
          </div>
        </div>
      )}
    </div>
  )
}
