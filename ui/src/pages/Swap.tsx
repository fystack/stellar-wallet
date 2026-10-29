import { SwapIcon, ShieldIcon } from '../icons.tsx'
import { shortAddress } from '../lib/format.ts'
import { assetLabel } from '../lib/stellar.ts'
import { useSwapForm } from '../hooks/useSwapForm.ts'
import EmptyState from '../components/EmptyState.tsx'
import SwapPanels from '../components/swap/SwapPanels.tsx'
import SlippagePicker from '../components/swap/SlippagePicker.tsx'
import QuoteDetails from '../components/swap/QuoteDetails.tsx'
import ConfirmSwapModal from '../components/swap/ConfirmSwapModal.tsx'
import type { SwapInput, Wallet } from '../types.ts'

export default function Swap({
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
  onSubmit: (input: SwapInput) => Promise<void>
}) {
  const wallets = allWallets.filter((w) => w.status === 'ready' && w.chain === 'stellar')
  const form = useSwapForm(wallets, preselectId, onSubmit)

  if (readOnly) {
    return (
      <EmptyState
        icon={<SwapIcon size={32} />}
        title="View-only access"
        body="Only workspace owners can swap. Operators can send payments, which may need approval."
      />
    )
  }

  if (wallets.length === 0) {
    return (
      <EmptyState
        icon={<SwapIcon size={32} />}
        title="No Stellar wallet to swap"
        body="Swaps use the built-in Stellar DEX. Create a Stellar wallet to get started."
        action="Create a wallet"
        onAction={onGoCreate}
      />
    )
  }

  const { quote } = form

  return (
    <div className="mx-auto mt-2.5 max-w-[560px]">
      <header className="mb-6">
        <h1 className="mb-2 text-2xl font-bold sm:text-3xl">Swap</h1>
        <p className="text-base leading-relaxed text-muted">
          Trade assets on the Stellar DEX. The path payment is signed by 2 of 3 nodes.
        </p>
      </header>

      <form className="surface flex flex-col gap-5" onSubmit={(e) => {
          e.preventDefault()
          form.review()
        }}>
        <label className="flex flex-col gap-2">
          <span className="text-sm font-semibold text-ink-soft">Wallet</span>
          <select className="field-input" value={form.from} onChange={(e) => form.setFrom(e.target.value)}>
            {wallets.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name} · {shortAddress(w.address)}
              </option>
            ))}
          </select>
        </label>

        <SwapPanels
          amount={form.amount}
          onAmountChange={form.setAmount}
          payAsset={form.payAsset}
          payOptions={form.payOptions}
          onPayAssetChange={form.setPayAsset}
          payBalance={form.payBalance}
          maxPay={form.maxPay}
          overBalance={form.overBalance}
          receiveAsset={form.receiveAsset}
          receiveOptions={form.receiveOptions}
          onReceiveAssetChange={form.setReceiveAsset}
          receiveAmount={form.receiveAmount}
          quoting={quote.loading}
          canFlip={form.canFlip}
          onFlip={form.flip}
        />

        {quote.error && (
          <div className="bg-danger-soft px-3 py-2 text-sm font-medium text-danger">
            {quote.error}
          </div>
        )}

        <SlippagePicker value={form.slippage} onChange={form.setSlippage} />

        {quote.data && (
          <QuoteDetails
            quote={quote.data}
            payAsset={form.payAsset}
            receiveAsset={form.receiveAsset}
            rate={form.rate}
            minReceive={form.minReceive}
          />
        )}

        {form.needsTrustline ? (
          <>
            <div className="notice notice-warning flex items-start gap-2">
              <span className="shrink-0">⚠</span>
              <span>
                You need a trustline to <strong>{assetLabel(form.receiveAsset)}</strong> before you
                can receive it. Add it once, then swap.
              </span>
            </div>
            <button
              type="button"
              className="btn btn-primary btn-lg w-full"
              disabled={form.addingTrustline}
              onClick={form.addTrustline}
            >
              {form.addingTrustline
                ? 'Adding trustline…'
                : `Add ${assetLabel(form.receiveAsset)} trustline`}
            </button>
          </>
        ) : (
          <button className="btn btn-primary btn-lg w-full" type="submit" disabled={!form.canSubmit}>
            {form.submitting ? 'Submitting…' : 'Review & sign swap'}
          </button>
        )}

        <p className="flex items-center justify-center gap-1.5 text-xs text-muted">
          <ShieldIcon size={14} />
          Signed by 2 of 3 nodes. Your key is never assembled.
        </p>
      </form>

      {form.confirming && quote.data && (
        <ConfirmSwapModal
          amount={form.amount}
          payAsset={form.payAsset}
          receiveAsset={form.receiveAsset}
          receiveAmount={quote.data.dest_amount}
          minReceive={form.minReceive}
          slippageBps={form.slippage}
          submitting={form.submitting}
          onCancel={form.cancelReview}
          onConfirm={() => void form.confirm()}
        />
      )}
    </div>
  )
}
