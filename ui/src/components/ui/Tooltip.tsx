import { cloneElement, useId, useState } from 'react'
import type { ReactElement, ReactNode } from 'react'

// Shows on hover and keyboard focus, unlike the native title attribute; rendered only while open.
export default function Tooltip({ label, children }: { label: ReactNode; children: ReactElement }) {
  const id = useId()
  const [open, setOpen] = useState(false)
  return (
    <span
      className="relative inline-flex min-w-0 max-w-full"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      onFocus={() => setOpen(true)}
      onBlur={() => setOpen(false)}
      onKeyDown={(event) => event.key === 'Escape' && setOpen(false)}
    >
      {cloneElement(children, { 'aria-describedby': open ? id : undefined })}
      {open && (
        <span
          id={id}
          role="tooltip"
          className="pointer-events-none absolute bottom-full left-1/2 z-50 mb-2 w-max max-w-[260px] -translate-x-1/2 break-all bg-ink px-2.5 py-1.5 text-xs font-medium text-white"
        >
          {label}
        </span>
      )}
    </span>
  )
}
