import type { ReactNode } from 'react'

export default function EmptyState({
  icon,
  title,
  body,
  action,
  onAction,
}: {
  icon: ReactNode
  title: string
  body: string
  action: string
  onAction: () => void
}) {
  return (
    <div className="mx-auto mt-10 max-w-[460px] text-center sm:mt-16">
      <div className="mx-auto mb-6 grid h-16 w-16 place-items-center bg-brand text-white sm:mb-8 sm:h-[78px] sm:w-[78px]">
        {icon}
      </div>
      <h2 className="mb-2.5 text-2xl font-bold">{title}</h2>
      <p className="mb-7 text-base leading-relaxed text-muted">{body}</p>
      <button className="cta w-full !px-6 sm:w-auto" onClick={onAction}>
        {action}
      </button>
    </div>
  )
}
