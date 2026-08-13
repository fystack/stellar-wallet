import { useState } from 'react'
import type { ReactNode } from 'react'
import {
  ShieldIcon,
  SplitIcon,
  CheckShieldIcon,
  EyeIcon,
  EyeOffIcon,
} from '../icons.tsx'
import { ChainLogo, TokenLogo } from '../logos.tsx'
import { formatAmount } from '../format.ts'
import ClusterStatus from '../components/ClusterStatus.tsx'
import type { Wallet } from '../types.ts'

type Feature = {
  icon: ReactNode
  title: string
  body: string
}

const features: Feature[] = [
  {
    icon: <SplitIcon />,
    title: 'Key split across 3 nodes',
    body: 'Your private key is divided into shares. Each node holds one fragment — never the whole key.',
  },
  {
    icon: <ShieldIcon />,
    title: 'No single point of failure',
    body: 'Compromising one node reveals nothing. An attacker needs 2 of 3 nodes to sign anything.',
  },
  {
    icon: <CheckShieldIcon />,
    title: '2-of-3 must cooperate',
    body: 'Every transaction requires agreement from 2 nodes. One offline node never blocks your funds.',
  },
]

type Props = {
  wallets: Wallet[]
  onCreate: () => void
  onOpen: (id: string) => void
}

function WalletGrid({
  wallets,
  onOpen,
  hidden,
}: {
  wallets: Wallet[]
  onOpen: (id: string) => void
  hidden: boolean
}) {
  return (
    <div className="grid gap-[18px] [grid-template-columns:repeat(auto-fill,minmax(240px,1fr))]">
      {wallets.map((w) => (
        <button
          type="button"
          onClick={() => w.status === 'ready' && onOpen(w.id)}
          disabled={w.status !== 'ready'}
          className="border border-line bg-white p-5 text-left transition-colors enabled:hover:border-brand disabled:cursor-default"
          key={w.id}
        >
          <div className="mb-4 flex items-center justify-between">
            <ChainLogo chain={w.chain} size={40} />
            <span className="bg-brand-soft px-2.5 py-1 text-xs font-semibold uppercase text-brand">
              {w.chain} · 2-of-3
            </span>
          </div>
          <div className="text-[17px] font-bold">{w.name}</div>
          {w.status === 'generating' ? (
            <div className="mb-4 mt-1 flex items-center gap-2 text-sm text-brand">
              <span className="h-2 w-2 animate-pulse rounded-full bg-brand" />
              Generating key across 3 nodes…
            </div>
          ) : w.status === 'failed' ? (
            <div className="mb-4 mt-1 text-sm text-[#d33a3a]">Keygen failed</div>
          ) : (
            <div className="mb-4 mt-1 break-all font-mono text-sm text-muted">
              {w.address}
            </div>
          )}
          <div className="flex items-center gap-2">
            <TokenLogo symbol={w.symbol} size={22} />
            <div className="truncate text-2xl font-extrabold leading-none">
              {hidden ? '••••' : formatAmount(w.balance)}{' '}
              <span className="text-[15px] font-semibold text-muted">
                {w.symbol}
              </span>
            </div>
          </div>
        </button>
      ))}
    </div>
  )
}

export default function Wallets({ wallets, onCreate, onOpen }: Props) {
  const hasWallets = wallets.length > 0
  const [hidden, setHidden] = useState(false)

  // With at least one wallet, skip the onboarding hero — just show the list.
  if (hasWallets) {
    return (
      <section className="mx-auto max-w-[1040px]">
        <div className="mb-5 flex items-center justify-between">
          <h2 className="text-2xl font-bold">My Wallets</h2>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setHidden((h) => !h)}
              className="border border-line p-2 text-muted hover:bg-[#f2f5f9]"
              aria-label={hidden ? 'Show balances' : 'Hide balances'}
              title={hidden ? 'Show balances' : 'Hide balances'}
            >
              {hidden ? <EyeOffIcon /> : <EyeIcon />}
            </button>
            <button
              onClick={onCreate}
              className="bg-brand px-4 py-2.5 font-semibold text-white transition-colors hover:bg-brand-deep"
            >
              + New wallet
            </button>
          </div>
        </div>
        <WalletGrid wallets={wallets} onOpen={onOpen} hidden={hidden} />
      </section>
    )
  }

  return (
    <>
      <ClusterStatus />

      <section className="mx-auto mt-10 max-w-[940px] text-center">
        <div className="mx-auto mb-8 grid h-[78px] w-[78px] place-items-center bg-brand text-white">
          <ShieldIcon size={34} />
        </div>
        <h1 className="mb-4 text-[40px] font-extrabold tracking-tight">
          Your keys, split — never whole
        </h1>
        <p className="mx-auto mb-11 max-w-[620px] text-[19px] leading-relaxed text-muted">
          This wallet uses threshold cryptography. No server, no browser, no one
          ever holds your complete private key.
        </p>

        <div className="grid grid-cols-1 gap-5 text-left md:grid-cols-3">
          {features.map((f) => (
            <div className="border border-line bg-white p-6" key={f.title}>
              <div className="mb-5 grid h-11 w-11 place-items-center bg-brand-soft text-brand">
                {f.icon}
              </div>
              <h3 className="mb-3 text-[19px] font-bold">{f.title}</h3>
              <p className="text-[15px] leading-relaxed text-muted">{f.body}</p>
            </div>
          ))}
        </div>

        <button className="cta mt-11" onClick={onCreate}>
          Create your first wallet
        </button>

        <p className="mt-7 text-[15px] text-muted">
          Secured by{' '}
          <strong className="text-ink-soft">2-of-3 threshold signatures</strong> ·
          No seed phrase · No single server risk
        </p>
      </section>
    </>
  )
}
