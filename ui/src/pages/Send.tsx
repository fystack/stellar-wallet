import { useState } from 'react'
import { SendIcon, ShieldIcon } from '../icons.tsx'
import { formatAmount, shortAddress } from '../lib/format.ts'
import { useSendForm } from '../hooks/useSendForm.ts'
import EmptyState from '../components/EmptyState.tsx'
import QrScanner from '../components/QrScanner.tsx'
import RecipientField from '../components/send/RecipientField.tsx'
import AmountField from '../components/send/AmountField.tsx'
import MemoField from '../components/send/MemoField.tsx'
import SendSummary from '../components/send/SendSummary.tsx'
import ConfirmSendModal from '../components/send/ConfirmSendModal.tsx'
import type { SendInput, Wallet } from '../types.ts'

export default function Send({
  wallets: allWallets,
  preselectId,
  onGoCreate,
  onSubmit,
  readOnly = false,
}: {
  readOnly?: boolean
  wallets: Wallet[]
  preselectId?: string
  onGoCreate: () => void
  onSubmit: (input: SendInput) => void
}) {
  // Only wallets whose keygen finished can sign.
  const wallets = allWallets.filter((w) => w.status === 'ready')
  const form = useSendForm(wallets, preselectId, onSubmit)
  const [showScanner, setShowScanner] = useState(false)

  if (readOnly) {
    return (
      <EmptyState
        icon={<SendIcon size={32} />}
        title="View-only access"
        body="Only workspace owners can send, swap or add assets for now. Payment approvals for members are coming next."
      />
    )
  }

  if (wallets.length === 0) {
    return (
      <EmptyState
        icon={<SendIcon size={32} />}
        title="No wallet to send from"
        body="Create a wallet first, then you can send assets across the Stellar network."
        action="Create your first wallet"
        onAction={onGoCreate}
      />
    )
  }

  const { resolved } = form.recipient

  return (
    <div className="mx-auto mt-2.5 max-w-[760px]">
      <header className="mb-6">
        <h1 className="mb-2 text-2xl font-bold sm:text-3xl">Send</h1>
        <p className="text-base leading-relaxed text-muted">
          Transfer assets. The transaction is signed by 2 of 3 nodes — your key is never
          assembled.
        </p>
      </header>

      <form className="surface flex flex-col gap-[18px]" onSubmit={(e) => {
          e.preventDefault()
          form.review()
        }}>
        <label className="flex flex-col gap-2">
          <span className="text-sm font-semibold text-ink-soft">From wallet</span>
          <select className="field-input" value={form.from} onChange={(e) => form.setFrom(e.target.value)}>
            {wallets.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name} · {shortAddress(w.address)} · {formatAmount(w.balance)} {w.symbol}
              </option>
            ))}
          </select>
        </label>

        <RecipientField
          value={form.to}
          onChange={form.setTo}
          ownWallets={wallets.filter((w) => w.id !== form.from)}
          onScan={() => setShowScanner(true)}
          resolved={resolved}
          resolving={form.recipient.resolving}
          error={form.recipient.error}
          isSelfSend={form.isSelfSend}
        />

        <AmountField
          amount={form.amount}
          onAmountChange={form.setAmount}
          assets={form.assets}
          assetId={form.assetId}
          onAssetChange={form.setAssetId}
          maxSend={form.maxSend}
          overBalance={form.overBalance}
        />

        <MemoField
          value={form.memo}
          onChange={form.editMemo}
          requiredType={resolved?.memo_type}
        />

        <SendSummary amount={form.amount} asset={form.assetSymbol} />

        <div className="flex items-start gap-2.5 bg-card px-3.5 py-3 text-sm text-ink-soft sm:items-center">
          <span className="shrink-0 text-brand">
            <ShieldIcon size={18} />
          </span>
          <span>Requires approval from 2 of 3 nodes to sign.</span>
        </div>

        <button className="cta w-full" type="submit" disabled={!form.canSubmit}>
          Review &amp; sign
        </button>
      </form>

      {showScanner && <QrScanner onResult={form.setTo} onClose={() => setShowScanner(false)} />}

      {form.confirming && resolved && (
        <ConfirmSendModal
          amount={form.amount}
          asset={form.assetSymbol}
          fromName={form.fromWallet?.name ?? ''}
          recipient={resolved}
          memo={form.memo.trim()}
          onCancel={form.cancelReview}
          onConfirm={form.confirm}
        />
      )}
    </div>
  )
}
