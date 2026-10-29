import { readPreference, writePreference } from '../lib/preferences.ts'
import Tooltip from '../components/ui/Tooltip.tsx'
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
import { formatAmount, shortAddress } from '../lib/format.ts'
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
  loading: boolean
  failed: boolean
  onRetry: () => void
  wallets: Wallet[]
  workspaceName: string
  // Undefined for members: only workspace owners create wallets.
  onCreate?: () => void
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
          className="flex flex-col border border-line bg-white p-5 text-left transition duration-200 enabled:hover:-translate-y-0.5 enabled:hover:border-brand disabled:cursor-default"
          key={w.id}
        >
          <div className="mb-4 flex items-center justify-between">
            <ChainLogo chain={w.chain} size={40} />
            <span className="badge badge-brand uppercase">
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
            <div className="mb-4 mt-1 text-sm text-danger">Keygen failed</div>
          ) : (
            <div className="mb-4 mt-1 font-mono text-[13px] text-muted">
              {shortAddress(w.address, 8, 8)}
            </div>
          )}
          <div className="mt-auto flex items-center gap-2">
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

export default function Wallets({ loading, failed, onRetry, wallets, workspaceName, onCreate, onOpen }: Props) {
  const hasWallets = wallets.length > 0
  // Shared with the Settings "Hide balances" pref, and persisted.
  const [hidden, setHidden] = useState(
    () => readPreference('pref_hide') === '1',
  )
  const toggleHidden = () =>
    setHidden((h) => {
      const next = !h
      writePreference('pref_hide', next ? '1' : '0')
      return next
    })

  if (failed) {
    return (
      <div role="alert" className="mx-auto mt-10 max-w-[1040px] text-center">
        <p className="text-ink-soft">Unable to load wallets for {workspaceName || 'this workspace'}.</p>
        <button className="btn btn-primary btn-lg mt-4" onClick={onRetry}>
          Try again
        </button>
      </div>
    )
  }

  if (loading) {
    return (
      <p role="status" className="mx-auto mt-10 max-w-[1040px] text-center text-muted">
        Loading wallets…
      </p>
    )
  }

  // With at least one wallet, skip the onboarding hero — just show the list.
  if (hasWallets) {
    return (
      <section className="mx-auto max-w-[1040px]">
        <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
          <div className="min-w-0">
            <h2 className="text-2xl font-bold">Wallets</h2>
            {workspaceName && <p className="truncate text-sm text-muted">{workspaceName}</p>}
          </div>
          <div className="flex items-center gap-2">
            <Tooltip label={hidden ? 'Show balances' : 'Hide balances'}>
              <button
                onClick={toggleHidden}
                className="btn btn-secondary p-2 text-muted"
                aria-label={hidden ? 'Show balances' : 'Hide balances'}
              >
                {hidden ? <EyeOffIcon /> : <EyeIcon />}
              </button>
            </Tooltip>
            {onCreate && (
              <button
                onClick={onCreate}
                className="btn btn-primary"
              >
                + New wallet
              </button>
            )}
          </div>
        </div>
        <WalletGrid wallets={wallets} onOpen={onOpen} hidden={hidden} />
      </section>
    )
  }

  return (
    <>
      <ClusterStatus />

      <section className="mx-auto mt-6 max-w-[940px] text-center sm:mt-10">
        <div className="mx-auto mb-6 grid h-16 w-16 place-items-center bg-brand text-white sm:mb-8 sm:h-[78px] sm:w-[78px]">
          <ShieldIcon size={34} />
        </div>
        <h1 className="mb-4 text-3xl font-extrabold tracking-tight sm:text-[40px]">
          Your keys, split — never whole
        </h1>
        <p className="mx-auto mb-8 max-w-[620px] text-base leading-relaxed text-muted sm:mb-11 sm:text-[19px]">
          This wallet uses threshold cryptography. No server, no browser, no one
          ever holds your complete private key.
        </p>

        <div className="grid grid-cols-1 gap-5 text-left md:grid-cols-3">
          {features.map((f) => (
            <div className="border border-line bg-white p-5 sm:p-6" key={f.title}>
              <div className="mb-5 grid h-11 w-11 place-items-center bg-brand-soft text-brand">
                {f.icon}
              </div>
              <h3 className="mb-3 text-[19px] font-bold">{f.title}</h3>
              <p className="text-[15px] leading-relaxed text-muted">{f.body}</p>
            </div>
          ))}
        </div>

        {onCreate ? (
          <button className="btn btn-primary btn-lg mt-8 w-full sm:mt-11 sm:w-auto" onClick={onCreate}>
            Create your first wallet
          </button>
        ) : (
          <p className="mt-8 text-base text-ink-soft sm:mt-11">
            No wallets in {workspaceName || 'this workspace'} yet. Ask a workspace owner to create one.
          </p>
        )}

        <p className="mt-6 text-sm leading-relaxed text-muted sm:mt-7 sm:text-[15px]">
          Secured by{' '}
          <strong className="text-ink-soft">2-of-3 threshold signatures</strong> ·
          No seed phrase · No single server risk
        </p>
      </section>
    </>
  )
}
