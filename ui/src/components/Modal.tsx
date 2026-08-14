import { useEffect } from 'react'
import type { ReactNode } from 'react'
import { CloseIcon } from '../icons.tsx'

export default function Modal({
  title,
  onClose,
  children,
  maxWidth = 520,
}: {
  title: string
  onClose: () => void
  children: ReactNode
  maxWidth?: number
}) {
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose])

  return (
    <div
      className="fixed inset-0 z-50 flex justify-center overflow-y-auto bg-black/40 p-3 sm:items-center sm:p-4"
      onClick={onClose}
    >
      <div
        className="my-auto max-h-[calc(100dvh-1.5rem)] w-full overflow-y-auto border border-line bg-white p-4 shadow-2xl sm:max-h-[calc(100dvh-2rem)] sm:p-7"
        style={{ maxWidth }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between sm:mb-6">
          <h2 className="text-xl font-bold sm:text-2xl">{title}</h2>
          <button
            onClick={onClose}
            aria-label="Close"
            className="grid h-9 w-9 place-items-center text-muted hover:bg-[#f2f5f9]"
          >
            <CloseIcon />
          </button>
        </div>
        {children}
      </div>
    </div>
  )
}
