import { useEffect, useState } from 'react'
import { ShieldIcon } from '../icons.tsx'
import Modal from './Modal.tsx'
import KeygenMesh from './KeygenMesh.tsx'
import CopyAddress from './CopyAddress.tsx'
import { ChainLogo, TokenLogo, chainTokens } from '../logos.tsx'
import type { Chain, Wallet } from '../types.ts'

type Props = {
  wallets: Wallet[]
  onClose: () => void
  onCreate: (data: { name: string; chain: Chain }) => Promise<string>
  onOpen: (id: string) => void
}

export default function CreateWalletModal({
  wallets,
  onClose,
  onCreate,
  onOpen,
}: Props) {
  const [name, setName] = useState('')
  const [creatingId, setCreatingId] = useState<string | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose])

  const created = wallets.find((w) => w.id === creatingId) ?? null
  const ready = created?.status === 'ready'

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    try {
      const id = await onCreate({
        name: name.trim() || 'Stellar Wallet',
        chain: 'stellar',
      })
      setCreatingId(id)
    } catch {
      setError('Failed to start keygen. Is the MPC cluster reachable?')
    }
  }

  // Ceremony view: keygen dispatched, waiting for (or showing) the result.
  if (creatingId) {
    return (
      <Modal title="Creating wallet" onClose={onClose}>
        <KeygenMesh done={ready} />
        {ready && created ? (
          <div className="mt-5">
            <div className="mb-1 text-xs font-medium uppercase tracking-wide text-muted">
              Address
            </div>
            <div className="mb-5 bg-card px-4 py-3">
              <CopyAddress address={created.address} className="break-all" />
            </div>
            <button
              className="btn btn-primary btn-lg w-full !py-3.5"
              onClick={() => onOpen(created.id)}
            >
              Open wallet
            </button>
          </div>
        ) : (
          <p className="mt-4 text-center text-xs text-muted">
            This takes a few seconds — the nodes never assemble the full key.
          </p>
        )}
      </Modal>
    )
  }

  return (
    <div
      className="fixed inset-0 z-50 flex justify-center overflow-y-auto bg-black/40 p-3 sm:items-center sm:p-4"
      onClick={onClose}
    >
      <div
        className="my-auto max-h-[calc(100dvh-1.5rem)] w-full max-w-[520px] overflow-y-auto border border-line bg-white p-4 shadow-2xl sm:max-h-[calc(100dvh-2rem)] sm:p-7"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-5 flex items-start justify-between gap-3 sm:mb-6">
          <div className="min-w-0">
            <h2 className="text-xl font-bold sm:text-2xl">Create a Stellar wallet</h2>
            <p className="mt-1 text-[15px] text-muted">
              Secured by 2-of-3 threshold signatures.
            </p>
          </div>
          <button
            onClick={onClose}
            aria-label="Close"
            className="grid h-9 w-9 place-items-center  text-muted hover:bg-hover"
          >
            ✕
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="mb-5 flex items-center gap-3 border border-line p-3.5">
            <ChainLogo chain="stellar" size={36} />
            <span className="min-w-0 flex-1">
              <span className="block font-semibold leading-tight">Stellar</span>
              <span className="mt-1 flex items-center gap-1">
                {chainTokens.stellar.map((token) => (
                  <TokenLogo key={token} symbol={token} size={16} />
                ))}
                <span className="ml-1 text-xs text-muted">and any Stellar asset</span>
              </span>
            </span>
            <span className="badge badge-neutral">Testnet</span>
          </div>

          <label className="mb-5 flex flex-col gap-2">
            <span className="text-sm font-semibold text-ink-soft">
              Wallet name <em className="font-normal not-italic text-muted">(optional)</em>
            </span>
            <input
              className="field-input"
              type="text"
              placeholder="Stellar Wallet"
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
          </label>

          <div className="mb-6 flex items-start gap-2.5 bg-card px-3.5 py-3 text-sm text-ink-soft sm:items-center">
            <span className="shrink-0 text-brand">
              <ShieldIcon size={18} />
            </span>
            <span>
              Key is generated in shares across 3 nodes — never assembled in one
              place.
            </span>
          </div>

          {error && (
            <div className="mb-4 bg-danger-soft px-3 py-2 text-sm text-danger">
              {error}
            </div>
          )}

          <div className="flex flex-col-reverse gap-2.5 sm:flex-row sm:gap-3">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 border border-line py-3.5 font-semibold text-ink-soft transition hover:bg-hover"
            >
              Cancel
            </button>
            <button type="submit" className="btn btn-primary btn-lg flex-1 !py-3.5">
              Create wallet
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
