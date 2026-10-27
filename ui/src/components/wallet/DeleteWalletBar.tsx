import { useState } from 'react'

export default function DeleteWalletBar({
  walletName,
  onCancel,
  onDelete,
}: {
  walletName: string
  onCancel: () => void
  onDelete: () => Promise<void>
}) {
  const [deleting, setDeleting] = useState(false)

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
      <span className="text-sm text-ink-soft">
        Delete <strong>{walletName}</strong> and its history? This can't be undone.
      </span>
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
