// Status reads as a dot plus a word; dot is a background class such as bg-success.
export default function StatusLabel({ label, dot }: { label: string; dot: string }) {
  return (
    <span className="inline-flex shrink-0 items-center gap-1.5 text-xs font-semibold text-ink-soft">
      <span aria-hidden className={'dot ' + dot} />
      {label}
    </span>
  )
}
