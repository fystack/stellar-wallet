import { TokenLogo } from '../../logos.tsx'
import type { CustomAsset } from '../../types.ts'
import { trustlineKey } from '../../hooks/useTrustlineRequest.ts'

// Registry assets the wallet doesn't hold yet, offered as trustlines to add.
export default function AddAssetChips({
  registry,
  heldKeys,
  pendingKeys,
  onAdd,
}: {
  registry: CustomAsset[]
  heldKeys: Set<string>
  pendingKeys: Set<string>
  onAdd: (asset: CustomAsset) => void
}) {
  const notHeld = registry.filter((a) => !heldKeys.has(trustlineKey(a.code, a.issuer)))
  const available = notHeld.filter((a) => !pendingKeys.has(trustlineKey(a.code, a.issuer)))
  const adding = notHeld.filter((a) => pendingKeys.has(trustlineKey(a.code, a.issuer)))
  if (available.length === 0 && adding.length === 0) return null

  return (
    <div className="mb-5 flex flex-wrap items-center gap-2">
      <span className="text-xs font-medium uppercase tracking-wide text-muted">Add asset</span>
      {available.map((asset) => (
        <button
          key={asset.code + asset.issuer}
          onClick={() => onAdd(asset)}
          className="flex items-center gap-1.5 border border-line px-2.5 py-1 text-xs font-semibold text-ink-soft transition-colors hover:border-brand hover:text-brand"
        >
          <TokenLogo symbol={asset.code} size={16} />+ {asset.code}
        </button>
      ))}
      {adding.map((asset) => (
        <span
          key={asset.code + asset.issuer}
          className="flex items-center gap-1.5 border border-line px-2.5 py-1 text-xs font-semibold text-muted opacity-60"
        >
          <TokenLogo symbol={asset.code} size={16} /> Adding {asset.code}…
        </span>
      ))}
    </div>
  )
}
