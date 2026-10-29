import { useState } from 'react'
import { formatAmount } from '../../lib/format.ts'
import type { AssetBalance } from '../../hooks/useBalances.ts'

export default function DeleteWalletBar({
  walletName,
  assets,
  onCancel,
  onDelete,
}: {
  walletName: string
  assets: AssetBalance[] | null
  onCancel: () => void
  onDelete: () => Promise<void>
}) {
  const [deleting, setDeleting] = useState(false)
  const held = (assets ?? []).filter((a) => Number(a.balance) > 0)

  async function confirmDelete() {
    setDeleting(true)
    try {
      await onDelete()
    } finally {
      setDeleting(false)
    }
  }

  return (
    <div className="mb-5 flex flex-col items-start gap-3 border border-danger-line bg-danger-soft px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
      <div className="flex flex-col gap-1.5 text-sm text-ink-soft">
        <span>
          Delete <strong>{walletName}</strong> and its history? This can't be undone.
        </span>
        {held.length > 0 && (
          <span role="alert" className="font-semibold text-danger">
            This wallet still holds{' '}
            {held.map((a) => `${formatAmount(a.balance)} ${a.symbol}`).join(', ')}. You won't be
            able to send these funds after deleting it.
          </span>
        )}
      </div>
      <div className="flex w-full shrink-0 gap-2 sm:w-auto">
        <button
          onClick={onCancel}
          className="btn btn-secondary btn-sm flex-1 sm:flex-none"
        >
          Cancel
        </button>
        <button
          disabled={deleting}
          onClick={() => void confirmDelete()}
          className="btn btn-danger btn-sm flex-1 sm:flex-none"
        >
          {deleting ? 'Deleting…' : 'Delete'}
        </button>
      </div>
    </div>
  )
}
