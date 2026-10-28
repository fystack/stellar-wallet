import { useState } from 'react'
import TransactionRow from './TransactionRow.tsx'
import { RefreshIcon, SpinnerIcon } from '../../icons.tsx'
import type { Transaction } from '../../types.ts'

const PAGE_SIZE = 8

export default function TransactionList({
  transactions,
  hasMore,
  loadingMore,
  onLoadMore,
  refreshing,
  onRefresh,
  onOpen,
}: {
  transactions: Transaction[]
  hasMore: boolean
  loadingMore: boolean
  onLoadMore: () => Promise<void>
  refreshing: boolean
  onRefresh: () => void
  onOpen: (id: string) => void
}) {
  const [visibleCount, setVisibleCount] = useState(PAGE_SIZE)

  async function showMore() {
    if (visibleCount >= transactions.length && hasMore) await onLoadMore()
    setVisibleCount((count) => count + PAGE_SIZE)
  }

  return (
    <section className="surface">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-[17px] font-bold">Transactions</h3>
        <button
          type="button"
          onClick={onRefresh}
          disabled={refreshing}
          className="btn btn-sm text-ink-soft hover:bg-card"
        >
          {refreshing ? <SpinnerIcon size={14} /> : <RefreshIcon size={14} />}
          {refreshing ? 'Refreshing…' : 'Refresh'}
        </button>
      </div>
      {transactions.length === 0 ? (
        <div className="py-10 text-center text-sm text-muted">No transactions yet.</div>
      ) : (
        <div className="flex flex-col">
          {transactions.slice(0, visibleCount).map((transaction) => (
            <TransactionRow
              key={transaction.id}
              transaction={transaction}
              onOpen={() => onOpen(transaction.id)}
            />
          ))}
          {(transactions.length > PAGE_SIZE || hasMore) && (
            <div className="flex items-center justify-between gap-3 pt-4 text-sm">
              <span className="text-muted">
                Showing {Math.min(visibleCount, transactions.length)}
                {hasMore ? '' : ` of ${transactions.length}`}
              </span>
              <div className="flex gap-2">
                {visibleCount > PAGE_SIZE && (
                  <button
                    type="button"
                    onClick={() => setVisibleCount(PAGE_SIZE)}
                    className="border border-line px-3 py-1.5 font-semibold text-ink-soft transition-colors hover:bg-hover"
                  >
                    Show less
                  </button>
                )}
                {(visibleCount < transactions.length || hasMore) && (
                  <button
                    type="button"
                    onClick={() => void showMore()}
                    disabled={loadingMore}
                    className="border border-line px-3 py-1.5 font-semibold text-brand transition-colors hover:bg-brand-soft"
                  >
                    {loadingMore ? 'Loading…' : 'Show more'}
                  </button>
                )}
              </div>
            </div>
          )}
        </div>
      )}
    </section>
  )
}
