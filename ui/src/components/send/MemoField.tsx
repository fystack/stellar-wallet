export default function MemoField({
  value,
  onChange,
  requiredType,
}: {
  value: string
  onChange: (value: string) => void
  requiredType?: string
}) {
  const missing = Boolean(requiredType) && value.trim() === ''
  return (
    <label className="flex flex-col gap-2">
      <span className="text-sm font-semibold text-ink-soft">
        Memo{' '}
        {requiredType ? (
          <em className="font-semibold not-italic text-warning">(required by recipient)</em>
        ) : (
          <em className="font-normal not-italic text-muted">(optional)</em>
        )}
      </span>
      <input
        className={'field-input ' + (missing ? 'ring-2 ring-danger' : '')}
        type="text"
        placeholder={
          requiredType
            ? `This recipient needs a "${requiredType}" memo`
            : 'Note attached to the transaction'
        }
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
      {requiredType && (
        <span className="flex items-center gap-1.5 bg-warning-soft px-2.5 py-1.5 text-xs font-medium text-warning">
          ⚠ This recipient (often an exchange) requires a memo — sending without it can lose
          your funds.
        </span>
      )}
    </label>
  )
}
