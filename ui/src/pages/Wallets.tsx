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
    title: 'Three key shares',
    body: 'Keygen runs across the 3 nodes. Each node stores one share; none of them stores the full key.',
  },
  {
    icon: <CheckShieldIcon />,
    title: 'Any 2 of 3 sign',
    body: 'A transaction is signed when 2 nodes take part, so one node being offline does not block payments.',
  },
  {
    icon: <ShieldIcon />,
    title: 'One share cannot sign',
    body: 'A single node, or anyone who gets its share, cannot sign a transaction on its own.',
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
    <div className="grid gap-[18px] [grid-template-columns:repeat(auto-fill,minmax(min(100%,280px),1fr))]">
      {wallets.map((w) => (
        <button
          type="button"
          onClick={() => w.status === 'ready' && onOpen(w.id)}
          disabled={w.status !== 'ready'}
          className="flex flex-col border border-line bg-white p-5 text-left transition duration-200 enabled:hover:-translate-y-0.5 enabled:hover:border-brand disabled:cursor-default"
          key={w.id}
        >
          <div className="mb-5 flex flex-wrap items-center justify-between gap-2">
            <ChainLogo chain={w.chain} size={36} />
            <span className="badge badge-brand text-[11px] uppercase tracking-wide">
              {w.chain} · 2-of-3
            </span>
          </div>
          <div className="truncate text-base font-semibold">{w.name}</div>
          {w.status === 'generating' ? (
            <div className="mb-4 mt-1 flex items-center gap-2 text-sm text-brand">
              <span className="h-2 w-2 animate-pulse rounded-full bg-brand" />
              Generating key across 3 nodes…
            </div>
          ) : w.status === 'failed' ? (
            <div className="mb-4 mt-1 text-sm text-danger">Keygen failed</div>
          ) : (
            <div className="mb-5 mt-1 font-mono text-xs text-muted">
              {shortAddress(w.address, 8, 8)}
            </div>
          )}
          <div className="mt-auto flex min-w-0 items-center gap-2 border-t border-line pt-4">
            <TokenLogo symbol={w.symbol} size={20} />
            <div className="flex min-w-0 flex-wrap items-baseline gap-x-1.5 leading-tight">
              <span className="break-all text-[22px] font-bold tabular-nums tracking-tight">
                {hidden ? '••••' : formatAmount(w.balance)}
              </span>
              <span className="text-sm font-medium text-muted">{w.symbol}</span>
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
      <section className="mx-auto max-w-[1360px]">
        <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
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
      <ClusterStatus className="mx-auto max-w-[940px]" />

      <section className="mx-auto mt-8 max-w-[940px] text-center sm:mt-12">
        <div className="mx-auto mb-6 grid h-16 w-16 place-items-center bg-brand text-white sm:mb-8 sm:h-[78px] sm:w-[78px]">
          <ShieldIcon size={34} />
        </div>
        <h1 className="mb-4 text-3xl font-bold tracking-tight sm:text-[36px]">
          No wallets yet
        </h1>
        <p className="mx-auto mb-8 max-w-[620px] text-base leading-relaxed text-muted sm:mb-11 sm:text-lg">
          Each wallet's private key is created as three key shares, one on each MPC node.
          The full key is never assembled on a server or in the browser.
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

      </section>
    </>
  )
}
