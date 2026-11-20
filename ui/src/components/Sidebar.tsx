import type { ReactNode } from 'react'
import { CheckShieldIcon, GridIcon, SendIcon, SettingsIcon, SwapIcon } from '../icons.tsx'
import { AppLogo } from '../logos.tsx'
import type { PageKey } from '../hooks/useRoute.ts'

const nav: { key: PageKey; label: string; icon: ReactNode }[] = [
  { key: 'wallets', label: 'Wallets', icon: <GridIcon /> },
  { key: 'send', label: 'Send', icon: <SendIcon /> },
  { key: 'swap', label: 'Swap', icon: <SwapIcon /> },
  { key: 'approvals', label: 'Approvals', icon: <CheckShieldIcon size={20} /> },
  { key: 'settings', label: 'Settings', icon: <SettingsIcon /> },
]

export default function Sidebar({
  user,
  activePage,
  signingOut,
  onNavigate,
  onSignOut,
  workspaceSwitcher,
  pendingApprovals,
}: {
  user: string
  activePage: PageKey | null
  signingOut: boolean
  onNavigate: (page: PageKey) => void
  onSignOut: () => void
  workspaceSwitcher: ReactNode
  pendingApprovals: number
}) {
  return (
    <aside className="sticky top-0 z-40 flex h-16 w-full flex-shrink-0 items-center gap-3 border-b border-night-line bg-night px-4 text-white md:h-screen md:w-[264px] md:flex-col md:items-stretch md:gap-0 md:overflow-y-auto md:border-b-0 md:px-4 md:py-5 xl:w-[280px]">
      <button
        type="button"
        onClick={() => onNavigate('wallets')}
        aria-label="Go to wallets"
        className="flex min-w-0 items-center gap-3 text-left transition-opacity hover:opacity-80 md:px-2 md:pb-6"
      >
        <AppLogo size={32} />
        <div className="min-w-0">
          <div className="truncate text-[15px] font-semibold leading-tight tracking-tight">Stellar MPC Wallet</div>
          <div className="hidden truncate text-xs text-white/50 sm:block">Keys never assembled</div>
        </div>
      </button>

      {workspaceSwitcher}

      <p className="hidden px-3 pb-2 text-[11px] font-semibold uppercase tracking-[0.12em] text-white/40 md:block">Menu</p>
      <nav className="fixed inset-x-0 bottom-0 z-40 grid grid-cols-5 border-t border-night-line bg-night px-2 pb-[max(0.5rem,env(safe-area-inset-bottom))] pt-2 md:static md:flex md:flex-col md:gap-0.5 md:border-0 md:p-0">
        {nav.map((item) => {
          const badge = item.key === 'approvals' && pendingApprovals > 0 ? pendingApprovals : 0
          const active = activePage === item.key
          return (
            <button
              key={item.key}
              onClick={() => onNavigate(item.key)}
              aria-label={badge ? `Approvals, ${badge} pending` : undefined}
              aria-current={active ? 'page' : undefined}
              className={
                'relative flex min-w-0 flex-col items-center gap-1 px-1 py-2 text-xs font-medium transition md:flex-row md:gap-3 md:px-3 md:py-2.5 md:text-[14px] ' +
                (active
                  ? 'bg-white/[0.08] text-white md:before:absolute md:before:inset-y-1.5 md:before:left-0 md:before:w-[3px] md:before:bg-brand'
                  : 'text-white/60 hover:bg-white/[0.04] hover:text-white')
              }
            >
              <span className={'relative grid place-items-center ' + (active ? 'text-brand-bright' : '')}>
                {item.icon}
                {badge > 0 && (
                  <span className="absolute -right-2.5 -top-1.5 min-w-[18px] bg-danger px-1 text-center text-[11px] font-bold leading-[18px] text-white md:hidden">
                    {badge}
                  </span>
                )}
              </span>
              {item.label}
              {badge > 0 && (
                <span className="ml-auto hidden min-w-[22px] bg-danger px-1.5 text-center text-xs font-bold leading-[22px] text-white md:inline-block">
                  {badge}
                </span>
              )}
            </button>
          )
        })}
      </nav>

      <button
        onClick={onSignOut}
        disabled={signingOut}
        aria-label={`Sign out ${user}`}
        className="flex shrink-0 items-center gap-3 text-left text-white/60 transition-colors hover:text-white md:mt-auto md:border-t md:border-night-line md:px-2 md:pt-4"
      >
        <span className="avatar bg-brand md:h-9 md:w-9">
          {user.charAt(0)}
        </span>
        <span className="hidden min-w-0 md:block">
          <span className="block truncate text-[13px] font-semibold text-white">
            {user}
          </span>
          <span className="block text-xs text-white/50">Sign out</span>
        </span>
      </button>
    </aside>
  )
}
