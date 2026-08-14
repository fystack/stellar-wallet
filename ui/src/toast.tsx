import { useEffect, useState } from 'react'

type Kind = 'success' | 'error'
type Toast = { id: string; kind: Kind; msg: string }

let toasts: Toast[] = []
const listeners = new Set<(t: Toast[]) => void>()

function emit() {
  listeners.forEach((l) => l([...toasts]))
}

export const toast = {
  show(kind: Kind, msg: string) {
    const id = Math.random().toString(36).slice(2)
    toasts = [...toasts, { id, kind, msg }]
    emit()
    setTimeout(() => {
      toasts = toasts.filter((t) => t.id !== id)
      emit()
    }, 3800)
  },
  success(msg: string) {
    this.show('success', msg)
  },
  error(msg: string) {
    this.show('error', msg)
  },
}

export function Toaster() {
  const [items, setItems] = useState<Toast[]>([])
  useEffect(() => {
    listeners.add(setItems)
    setItems([...toasts])
    return () => {
      listeners.delete(setItems)
    }
  }, [])

  return (
    <div className="fixed inset-x-4 top-4 z-[100] flex flex-col gap-2 sm:left-auto sm:right-5 sm:top-5 sm:w-[320px]">
      {items.map((t) => (
        <div
          key={t.id}
          className={
            'flex items-start gap-2.5 border px-4 py-3 text-sm shadow-lg ' +
            (t.kind === 'success'
              ? 'border-green-200 bg-green-50 text-green-800'
              : 'border-[#f0c2c2] bg-[#fff5f5] text-[#b32424]')
          }
        >
          <span className="mt-0.5 shrink-0 font-bold">
            {t.kind === 'success' ? '✓' : '!'}
          </span>
          <span className="min-w-0 break-words">{t.msg}</span>
        </div>
      ))}
    </div>
  )
}
