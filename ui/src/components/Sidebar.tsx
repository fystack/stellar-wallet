import type { ReactNode } from 'react'
import { GridIcon, SendIcon, SettingsIcon, SwapIcon } from '../icons.tsx'
import { AppLogo } from '../logos.tsx'
import type { PageKey } from '../hooks/useRoute.ts'

const nav: { key: PageKey; label: string; icon: ReactNode }[] = [
  { key: 'wallets', label: 'Wallets', icon: <GridIcon /> },
  { key: 'send', label: 'Send', icon: <SendIcon /> },
  { key: 'swap', label: 'Swap', icon: <SwapIcon /> },
  { key: 'settings', label: 'Settings', icon: <SettingsIcon /> },
]

export default function Sidebar({
  user,
  activePage,
  signingOut,
  onNavigate,
  onSignOut,
}: {
  user: string
  activePage: PageKey | null
  signingOut: boolean
  onNavigate: (page: PageKey) => void
  onSignOut: () => void
}) {
  return (
    <aside className="sticky top-0 z-40 flex h-16 w-full flex-shrink-0 items-center border-b border-line bg-white px-4 md:top-0 md:h-screen md:w-[260px] md:flex-col md:items-stretch md:overflow-y-auto md:border-b-0 md:border-r md:p-4">
      <button
        type="button"
        onClick={() => onNavigate('wallets')}
        aria-label="Go to wallets"
        className="flex items-center gap-2.5 text-left transition-opacity hover:opacity-80 md:gap-3 md:px-2 md:pb-6 md:pt-1.5"
      >
        <AppLogo size={40} />
        <div>
          <div className="font-bold md:text-lg">MPC Stellar Wallet</div>
          <div className="hidden text-[13px] text-muted sm:block">
            Keys never assembled
          </div>
        </div>
      </button>

      <nav className="fixed inset-x-0 bottom-0 z-40 grid grid-cols-4 border-t border-line bg-white px-2 pb-[max(0.5rem,env(safe-area-inset-bottom))] pt-2 md:static md:flex md:flex-col md:gap-1 md:border-0 md:p-0">
        {nav.map((item) => (
          <button
            key={item.key}
            onClick={() => onNavigate(item.key)}
            className={
              'flex min-w-0 flex-col items-center gap-1 px-1 py-2 text-xs font-semibold transition md:flex-row md:gap-3 md:px-3.5 md:py-3 md:text-base ' +
              (activePage === item.key
                ? 'bg-brand-soft text-brand'
                : 'text-ink-soft hover:bg-hover')
            }
          >
            <span className="grid place-items-center">{item.icon}</span>
            {item.label}
          </button>
        ))}
      </nav>

      <button
        onClick={onSignOut}
        disabled={signingOut}
        aria-label={`Sign out ${user}`}
        className="ml-auto flex items-center gap-3 text-left text-[15px] text-ink-soft transition-colors hover:text-ink md:ml-0 md:mt-auto md:px-2 md:py-3"
      >
        <span className="grid h-9 w-9 shrink-0 place-items-center bg-[#1b1c1f] text-sm font-semibold uppercase text-white md:h-10 md:w-10">
          {user.charAt(0)}
        </span>
        <span className="hidden min-w-0 md:block">
          <span className="block truncate text-sm font-semibold text-ink">
            {user}
          </span>
          <span className="block text-[13px] text-muted">Sign out</span>
        </span>
      </button>
    </aside>
  )
}
