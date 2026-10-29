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
    <div className="mb-5 flex flex-col items-start gap-3 border border-danger-line bg-[#fff7f7] px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
      <span className="text-sm text-ink-soft">
        Delete <strong>{walletName}</strong> and its history? This can't be undone.
      </span>
      <div className="flex w-full shrink-0 gap-2 sm:w-auto">
        <button
          onClick={onCancel}
          className="flex-1 border border-line px-3 py-1.5 text-sm font-semibold text-ink-soft hover:bg-hover sm:flex-none"
        >
          Cancel
        </button>
        <button
          disabled={deleting}
          onClick={() => void confirmDelete()}
          className="flex-1 bg-danger px-3 py-1.5 text-sm font-semibold text-white hover:bg-[#bf2f2f] disabled:opacity-50 sm:flex-none"
        >
          {deleting ? 'Deleting…' : 'Delete'}
        </button>
      </div>
    </div>
  )
}
