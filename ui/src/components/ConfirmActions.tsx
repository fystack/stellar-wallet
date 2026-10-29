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
      <button type="button" className="cta-secondary flex-1" disabled={busy} onClick={onCancel}>
        Cancel
      </button>
      <button
        type="button"
        className="cta flex-1 !px-6 !py-3 !text-base"
        disabled={busy}
        onClick={onConfirm}
      >
        {confirmLabel}
      </button>
    </div>
  )
}
