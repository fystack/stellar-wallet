import { useEffect, useId, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'
import { CheckIcon, ChevronDownIcon } from '../../icons.tsx'
import { roleHints, roleLabels, roles } from '../../lib/roles.ts'
import type { WorkspaceRole } from '../../types.ts'

// Focus stays on the trigger; the highlighted option is announced through aria-activedescendant.
export default function RoleSelect({
  label,
  value,
  onChange,
  disabled,
  compact,
}: {
  label: string
  value: WorkspaceRole
  onChange: (role: WorkspaceRole) => void
  disabled?: boolean
  compact?: boolean
}) {
  const listId = useId()
  const rootRef = useRef<HTMLDivElement>(null)
  const [open, setOpen] = useState(false)
  const [active, setActive] = useState(roles.indexOf(value))

  useEffect(() => {
    if (!open) return
    const close = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [open])

  function show() {
    setActive(roles.indexOf(value))
    setOpen(true)
  }

  function choose(role: WorkspaceRole) {
    setOpen(false)
    if (role !== value) onChange(role)
  }

  function onKeyDown(event: KeyboardEvent<HTMLButtonElement>) {
    const last = roles.length - 1
    switch (event.key) {
      case 'ArrowDown':
      case 'ArrowUp': {
        event.preventDefault()
        if (!open) return show()
        const step = event.key === 'ArrowDown' ? 1 : -1
        setActive((index) => Math.min(last, Math.max(0, index + step)))
        return
      }
      case 'Home':
      case 'End':
        if (!open) return
        event.preventDefault()
        setActive(event.key === 'Home' ? 0 : last)
        return
      case 'Enter':
      case ' ':
        event.preventDefault()
        if (open) choose(roles[active])
        else show()
        return
      case 'Escape':
        if (open) {
          event.preventDefault()
          setOpen(false)
        }
        return
      case 'Tab':
        setOpen(false)
    }
  }

  return (
    <div ref={rootRef} className={'relative shrink-0 ' + (compact ? '' : 'w-full sm:w-auto')}>
      <button
        type="button"
        role="combobox"
        aria-label={label}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listId}
        aria-activedescendant={open ? `${listId}-${roles[active]}` : undefined}
        disabled={disabled}
        onClick={() => (open ? setOpen(false) : show())}
        onKeyDown={onKeyDown}
        className={
          'flex w-full items-center justify-between gap-3 border bg-white text-left text-ink outline-none transition ' +
          'hover:border-line-strong focus-visible:border-brand focus-visible:ring-4 focus-visible:ring-brand/15 ' +
          'disabled:cursor-not-allowed disabled:opacity-50 ' +
          (open ? 'border-brand ring-4 ring-brand/15 ' : 'border-line ') +
          (compact ? 'min-w-[124px] px-3 py-1.5 text-sm' : 'min-w-[148px] px-3.5 py-2 text-[15px]')
        }
      >
        <span className="font-medium">{roleLabels[value]}</span>
        <span className={'text-muted transition-transform ' + (open ? 'rotate-180' : '')}>
          <ChevronDownIcon />
        </span>
      </button>

      {open && (
        <ul
          id={listId}
          role="listbox"
          aria-label={label}
          className="absolute right-0 z-30 mt-1.5 w-72 border border-line bg-white py-1.5 shadow-card"
        >
          {roles.map((role, index) => {
            const selected = role === value
            return (
              <li
                key={role}
                id={`${listId}-${role}`}
                role="option"
                aria-selected={selected}
                onMouseDown={(event) => event.preventDefault()}
                onMouseEnter={() => setActive(index)}
                onClick={() => choose(role)}
                className={'flex cursor-pointer items-start gap-3 px-3.5 py-2.5 ' + (index === active ? 'bg-hover' : '')}
              >
                <span className="min-w-0 flex-1">
                  <span className={'block text-sm font-semibold ' + (selected ? 'text-brand' : 'text-ink')}>
                    {roleLabels[role]}
                  </span>
                  <span className="block text-xs leading-snug text-muted">{roleHints[role]}</span>
                </span>
                <span className={'mt-0.5 text-brand ' + (selected ? '' : 'invisible')}>
                  <CheckIcon size={16} />
                </span>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
