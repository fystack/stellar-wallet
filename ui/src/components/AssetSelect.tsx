import { TokenLogo } from '../logos.tsx'
import { ChevronDownIcon } from '../icons.tsx'

export type AssetOption = { label: string; value: string; balance?: string }

// A token picker: logo + native select styled to match the sharp-cornered UI.
export default function AssetSelect({
  value,
  options,
  onChange,
}: {
  value: string
  options: AssetOption[]
  onChange: (value: string) => void
}) {
  const selected = options.find((option) => option.value === value)
  return (
    <div className="relative flex shrink-0 items-center">
      <span className="pointer-events-none absolute left-3 z-10">
        <TokenLogo symbol={selected?.label ?? value} size={22} />
      </span>
      <select
        className="w-[140px] cursor-pointer appearance-none bg-none border border-line bg-white py-2.5 pl-11 pr-8 text-base font-bold text-ink outline-none transition focus:border-brand"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
      <span className="pointer-events-none absolute right-3 text-muted">
        <ChevronDownIcon size={14} />
      </span>
    </div>
  )
}
