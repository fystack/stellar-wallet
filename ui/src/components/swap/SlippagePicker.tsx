const SLIPPAGE_OPTIONS = [
  { label: '0.5%', bps: 50 },
  { label: '1%', bps: 100 },
  { label: '2%', bps: 200 },
]

export default function SlippagePicker({
  value,
  onChange,
}: {
  value: number
  onChange: (bps: number) => void
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm font-semibold text-ink-soft">Slippage tolerance</span>
      <div className="flex gap-1.5">
        {SLIPPAGE_OPTIONS.map((option) => (
          <button
            key={option.bps}
            type="button"
            onClick={() => onChange(option.bps)}
            className={
              'border px-3 py-1.5 text-sm font-semibold transition ' +
              (value === option.bps
                ? 'border-brand bg-brand text-white'
                : 'border-line bg-white text-ink-soft hover:bg-card')
            }
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  )
}
