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
          className="absolute right-2 text-xs font-semibold text-brand hover:underline"
        >
          Scan QR
        </button>
      </div>
      {ownWallets.length > 0 && (
        <select
          className="field-input text-sm"
          value=""
          onChange={(e) => {
            const wallet = ownWallets.find((w) => w.id === e.target.value)
            if (wallet) onChange(wallet.address)
          }}
        >
          <option value="">Send to one of my wallets…</option>
          {ownWallets.map((w) => (
            <option key={w.id} value={w.id}>
              {w.name} · {shortAddress(w.address)} · {formatAmount(w.balance)} {w.symbol}
            </option>
          ))}
        </select>
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
