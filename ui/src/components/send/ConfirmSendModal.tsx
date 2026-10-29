import Modal from '../Modal.tsx'
import ConfirmActions from '../ConfirmActions.tsx'
import { ShieldIcon } from '../../icons.tsx'
import { TokenLogo } from '../../logos.tsx'
import { formatAmount, shortAddress } from '../../lib/format.ts'
import { BASE_FEE_XLM } from '../../lib/stellar.ts'
import type { ResolvedRecipient } from '../../types.ts'

export default function ConfirmSendModal({
  amount,
  asset,
  fromName,
  recipient,
  memo,
  onCancel,
  onConfirm,
}: {
  amount: string
  asset: string
  fromName: string
  recipient: ResolvedRecipient
  memo: string
  onCancel: () => void
  onConfirm: () => void
}) {
  return (
    <Modal title="Confirm transaction" onClose={onCancel}>
      <div className="flex flex-col gap-4">
        <div className="flex flex-col items-center gap-1.5 border border-line bg-card py-5">
          <TokenLogo symbol={asset} size={34} />
          <div className="text-3xl font-extrabold">
            {formatAmount(amount)} <span className="text-lg text-muted">{asset}</span>
          </div>
        </div>

        <div className="divide-y divide-line bg-card px-4 text-sm">
          <div className="flex items-center justify-between gap-3 py-2.5">
            <span className="text-muted">From</span>
            <span className="font-semibold">{fromName}</span>
          </div>
          <div className="flex items-center justify-between gap-3 py-2.5">
            <span className="shrink-0 text-muted">To</span>
            <span className="truncate text-right font-mono text-[13px]">
              {recipient.federation || shortAddress(recipient.address, 8, 8)}
            </span>
          </div>
          {memo && (
            <div className="flex items-center justify-between gap-3 py-2.5">
              <span className="shrink-0 text-muted">Memo</span>
              <span className="truncate text-right">{memo}</span>
            </div>
          )}
          <div className="flex items-center justify-between gap-3 py-2.5">
            <span className="text-muted">Network fee</span>
            <span className="font-semibold">{BASE_FEE_XLM} XLM</span>
          </div>
        </div>

        <ConfirmActions confirmLabel="Confirm & sign" onCancel={onCancel} onConfirm={onConfirm} />

        <p className="flex items-center justify-center gap-1.5 text-xs text-muted">
          <ShieldIcon size={14} />
          Signed by 2 of 3 nodes. Broadcasts on-chain and can't be undone.
        </p>
      </div>
    </Modal>
  )
}
