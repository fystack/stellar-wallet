import { ChevronDownIcon } from '../../icons.tsx'
import { formatAmount, shortAddress } from '../../lib/format.ts'
import type { ResolvedRecipient, Wallet } from '../../types.ts'

export default function RecipientField({
  value,
  onChange,
  ownWallets,
  onScan,
  resolved,
  resolving,
  error,
  isSelfSend,
}: {
  value: string
  onChange: (value: string) => void
  ownWallets: Wallet[]
  onScan: () => void
  resolved: ResolvedRecipient | null
  resolving: boolean
  error: string
  isSelfSend: boolean
}) {
  return (
    <label className="flex flex-col gap-2">
      <span className="text-sm font-semibold text-ink-soft">Recipient</span>
      <div className="relative flex items-center">
        <input
          className="field-input pr-20 font-mono"
          type="text"
          placeholder="G…, M…, or name*domain.com"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          spellCheck={false}
        />
        <button
          type="button"
          onClick={onScan}
          className="btn btn-link absolute right-2 text-xs"
        >
          Scan QR
        </button>
      </div>
      {ownWallets.length > 0 && (
        <span className="relative inline-flex w-fit items-center gap-1 text-xs font-semibold text-brand">
          Or pick one of my wallets
          <ChevronDownIcon size={12} />
          <select
            className="absolute inset-0 cursor-pointer opacity-0"
            aria-label="Send to one of my wallets"
            value=""
            onChange={(e) => {
              const wallet = ownWallets.find((w) => w.id === e.target.value)
              if (wallet) onChange(wallet.address)
            }}
          >
            <option value="" disabled>Pick a wallet</option>
            {ownWallets.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name} · {shortAddress(w.address)} · {formatAmount(w.balance)} {w.symbol}
              </option>
            ))}
          </select>
        </span>
      )}
      {resolving && <span className="text-xs text-muted">Resolving address…</span>}
      {error && (
        <span className="bg-danger-soft px-2.5 py-1.5 text-xs font-medium text-danger">
          {error}
        </span>
      )}
      {resolved?.federation && (
        <span className="bg-brand-soft px-2.5 py-1.5 text-xs font-medium text-brand">
          ✓ {resolved.federation} → {shortAddress(resolved.address)}
        </span>
      )}
      {resolved && !resolved.federation && value.trim()[0] === 'M' && (
        <span className="bg-brand-soft px-2.5 py-1.5 text-xs font-medium text-brand">
          ✓ Muxed address (memo encoded in address)
        </span>
      )}
      {isSelfSend && (
        <span className="flex items-center gap-1.5 bg-warning-soft px-2.5 py-1.5 text-xs font-medium text-warning">
          ⚠ This is the sender's own address — you'd only pay the network fee.
        </span>
      )}
    </label>
  )
}
