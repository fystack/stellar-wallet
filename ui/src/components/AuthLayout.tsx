import type { ReactNode } from 'react'
import { FystackLogo, GithubLogo, StellarLogo } from '../logos.tsx'

const REPO_URL = 'https://github.com/fystack/stellar-wallet'

const signers = [
  { name: 'Alice', status: 'partial sig ✓', signed: true },
  { name: 'Bob', status: 'partial sig ✓', signed: true },
  { name: 'Carol', status: 'not needed', signed: false },
]

const highlights = ['Key shares never combined', 'Owner / operator / member roles', 'Docker one-liner']

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

function ProductPanel() {
  return (
    <aside className="hidden w-[42%] max-w-[680px] flex-col justify-center bg-night px-16 py-12 text-white lg:flex">
      <p className="font-mono text-sm tracking-[0.15em] text-brand-soft">STELLAR MPC WALLET · OPEN-SOURCE</p>
      <h2 className="mt-4 text-[52px] font-extrabold leading-[1.08]">No single key.<br />No single point of failure.</h2>
      <p className="mt-6 text-lg leading-relaxed text-white/70">
        Threshold MPC signing, payment approvals and team roles for Stellar — running on infrastructure you control.
      </p>

      <div aria-hidden className="mt-10 border border-white/10 bg-white/[0.03]">
        <div className="flex items-center justify-between border-b border-white/10 px-6 py-4">
          <span className="font-semibold">Payment · 12,500 XLM</span>
          <span className="font-mono text-sm text-success">2 / 3 signed</span>
        </div>
        <ul className="px-6">
          {signers.map((signer) => (
            <li key={signer.name} className="flex items-center justify-between border-b border-white/10 py-3.5 last:border-none">
              <span className={`flex items-center gap-3 font-semibold ${signer.signed ? '' : 'text-white/50'}`}>
                <span className={`dot ${signer.signed ? 'bg-success' : 'bg-line-strong opacity-50'}`} />
                {signer.name}
              </span>
              <span className="text-sm text-white/60">{signer.status}</span>
            </li>
          ))}
        </ul>
        <p className="border-t border-white/10 px-6 py-4 font-mono text-sm text-brand-soft">→ submitted to Stellar · tx 7f3a…c91e</p>
      </div>

      <ul className="mt-10 flex flex-wrap gap-3">
        <li className="flex items-center gap-2 border border-white/20 px-4 py-2 text-sm">
          <StellarLogo size={16} />
          Built on Stellar
        </li>
        {highlights.map((highlight) => (
          <li key={highlight} className="border border-white/20 px-4 py-2 text-sm">{highlight}</li>
        ))}
      </ul>
    </aside>
  )
}
