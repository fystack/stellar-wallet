export default function ConfirmActions({
  confirmLabel,
  busy = false,
  onCancel,
  onConfirm,
}: {
  confirmLabel: string
  busy?: boolean
  onCancel: () => void
  onConfirm: () => void
}) {
  return (
    <div className="flex gap-3">
      <button type="button" className="btn btn-secondary flex-1" disabled={busy} onClick={onCancel}>
        Cancel
      </button>
      <button
        type="button"
        className="btn btn-primary flex-1"
        disabled={busy}
        onClick={onConfirm}
      >
        {confirmLabel}
      </button>
    </div>
  )
}
