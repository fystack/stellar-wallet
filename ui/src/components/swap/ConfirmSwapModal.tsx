import Modal from '../Modal.tsx'
import ConfirmActions from '../ConfirmActions.tsx'
import { ShieldIcon } from '../../icons.tsx'
import { TokenLogo } from '../../logos.tsx'
import { formatAmount } from '../../lib/format.ts'
import { assetLabel } from '../../lib/stellar.ts'

export default function ConfirmSwapModal({
  amount,
  payAsset,
  receiveAsset,
  receiveAmount,
  minReceive,
  slippageBps,
  submitting,
  onCancel,
  onConfirm,
}: {
  amount: string
  payAsset: string
  receiveAsset: string
  receiveAmount: string
  minReceive: number
  slippageBps: number
  submitting: boolean
  onCancel: () => void
  onConfirm: () => void
}) {
  return (
    <Modal title="Confirm swap" onClose={() => !submitting && onCancel()}>
      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-1 border border-line bg-card px-4 py-4">
          <div className="flex items-center gap-2 text-lg font-bold text-ink-soft">
            <TokenLogo symbol={assetLabel(payAsset)} size={22} />−{formatAmount(amount)}{' '}
            {assetLabel(payAsset)}
          </div>
          <div className="pl-1 text-muted">↓</div>
          <div className="flex items-center gap-2 text-2xl font-extrabold text-success">
            <TokenLogo symbol={assetLabel(receiveAsset)} size={24} />≈
            {formatAmount(receiveAmount)} {assetLabel(receiveAsset)}
          </div>
        </div>

        <div className="divide-y divide-line bg-card px-4 text-sm">
          <div className="flex items-center justify-between py-2.5">
            <span className="text-muted">Minimum received</span>
            <span className="font-semibold">
              {formatAmount(String(minReceive))} {assetLabel(receiveAsset)}
            </span>
          </div>
          <div className="flex items-center justify-between py-2.5">
            <span className="text-muted">Slippage</span>
            <span className="font-semibold">{slippageBps / 100}%</span>
          </div>
          <div className="flex items-center justify-between py-2.5">
            <span className="text-muted">Venue</span>
            <span className="font-semibold">Stellar DEX</span>
          </div>
        </div>

        <ConfirmActions
          confirmLabel={submitting ? 'Submitting…' : 'Confirm & swap'}
          busy={submitting}
          onCancel={onCancel}
          onConfirm={onConfirm}
        />

        <p className="flex items-center justify-center gap-1.5 text-xs text-muted">
          <ShieldIcon size={14} />
          Rate may move slightly. Signed by 2 of 3 nodes and can't be undone.
        </p>
      </div>
    </Modal>
  )
}
