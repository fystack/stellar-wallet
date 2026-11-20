import type { ReactNode } from 'react'
import { FystackLogo, GithubLogo, StellarLogo } from '../logos.tsx'
import { BoltIcon, ShieldIcon, UsersIcon } from '../icons.tsx'
import MpcNodeGraphic from './MpcNodeGraphic.tsx'

const REPO_URL = 'https://github.com/fystack/stellar-wallet'

export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-full bg-white">
      <div className="flex min-w-0 flex-1 flex-col px-5 py-6 sm:px-10 lg:px-16">
        <header className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <FystackLogo height={28} />
            <span className="flex items-center gap-1.5 border border-line px-2 py-1 text-sm font-semibold text-ink">
              <StellarLogo size={16} />
              Stellar MPC Wallet
            </span>
          </div>
          <a className="btn btn-secondary btn-sm" href={REPO_URL} target="_blank" rel="noreferrer" aria-label="Star on GitHub">
            <GithubLogo />
            <span className="hidden sm:inline">Star on GitHub</span>
          </a>
        </header>

        <main className="flex flex-1 items-center justify-center py-10">
          <div className="w-full max-w-[440px]">{children}</div>
        </main>

        <footer className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-muted">
          <span>Apache 2.0 licensed</span>
          <a className="text-ink-soft hover:text-ink" href={`${REPO_URL}#readme`} target="_blank" rel="noreferrer">Docs</a>
          <a className="text-ink-soft hover:text-ink" href={`${REPO_URL}#run-one-command`} target="_blank" rel="noreferrer">Self-host guide</a>
        </footer>
      </div>
      <ProductPanel />
    </div>
  )
}

const features = [
  { icon: <ShieldIcon size={22} />, title: '2-of-3 signing', body: 'Each key is held as 3 shares; any 2 nodes sign together' },
  { icon: <BoltIcon size={22} />, title: 'Payment approvals', body: 'Payments over a USD threshold wait for a teammate to approve' },
  { icon: <UsersIcon size={22} />, title: 'Roles', body: 'Owners, operators and read-only members' },
]

function ProductPanel() {
  return (
    <aside className="relative hidden w-1/2 min-w-0 shrink-0 flex-col overflow-y-auto bg-night-deep text-white lg:flex">
      <div aria-hidden className="pointer-events-none absolute inset-0 overflow-hidden">
        <div className="absolute -right-40 -top-40 h-[480px] w-[480px] rounded-full bg-brand/25 blur-[120px]" />
        <div className="absolute -bottom-48 -left-24 h-[420px] w-[420px] rounded-full bg-brand/15 blur-[120px]" />
      </div>

      <div className="relative mx-auto my-auto w-full max-w-[640px] px-10 py-12 xl:px-12">
        <div className="flex flex-wrap items-center gap-3">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-white/80">Stellar MPC Wallet</p>
          <span className="border border-brand-bright/60 px-2 py-0.5 text-[11px] font-semibold uppercase tracking-[0.16em] text-brand-bright">
            Open source
          </span>
        </div>
        <h2 className="mt-5 text-4xl font-bold leading-[1.1] tracking-tight xl:text-[44px]">
          Multi-party computation wallet for Stellar
        </h2>
        <p className="mt-4 max-w-[520px] text-base leading-relaxed text-white/70">
          Signing keys are split across three MPC nodes and never assembled. You run the nodes, the backend and
          the app on your own infrastructure.
        </p>

        <MpcNodeGraphic className="mx-auto my-8 block w-full max-w-[560px]" />

        <ul className="space-y-4">
          {features.map((feature) => (
            <li key={feature.title} className="flex items-center gap-4">
              <span className="grid h-12 w-12 shrink-0 place-items-center border border-white/10 bg-brand/15 text-brand-bright">
                {feature.icon}
              </span>
              <span className="min-w-0">
                <span className="block font-semibold">{feature.title}</span>
                <span className="block text-sm text-white/60">{feature.body}</span>
              </span>
            </li>
          ))}
        </ul>

        <p className="mt-8 flex items-center gap-2 text-[13px] text-white/50">
          <StellarLogo size={14} />
          Stellar testnet · Self-hosted with Docker Compose
        </p>
      </div>
    </aside>
  )
}
