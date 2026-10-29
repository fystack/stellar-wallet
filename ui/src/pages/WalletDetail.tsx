import { useEffect, useState } from 'react'
import { SendIcon, ShieldIcon, TrashIcon } from '../icons.tsx'
import { explorerAddress } from '../api.ts'
import { ChainLogo } from '../logos.tsx'
import { assetKey } from '../hooks/useBalances.ts'
import { useWalletDetail } from '../hooks/useWalletDetail.ts'
import CopyAddress from '../components/CopyAddress.tsx'
import ReceiveModal from '../components/ReceiveModal.tsx'
import TxDetailModal from '../components/transaction/TxDetailModal.tsx'
import DeleteWalletBar from '../components/wallet/DeleteWalletBar.tsx'
import BalanceList from '../components/wallet/BalanceList.tsx'
import AddAssetChips from '../components/wallet/AddAssetChips.tsx'
import TransactionList from '../components/wallet/TransactionList.tsx'
import type { Transaction, Wallet } from '../types.ts'

export default function WalletDetail({
  wallet,
  liveTxns,
  focusTxId,
  onFocusHandled,
  onBack,
  onSend,
  onDelete,
  readOnly = false,
}: {
  wallet: Wallet
  liveTxns: Record<string, Transaction>
  focusTxId?: string | null
  onFocusHandled?: () => void
  onBack: () => void
  onSend?: () => void
  // Undefined for members: only workspace owners delete wallets.
  onDelete?: (id: string) => Promise<void>
  readOnly?: boolean
}) {
  const [showReceive, setShowReceive] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const [openTxId, setOpenTxId] = useState<string | null>(null)
  const detail = useWalletDetail(wallet, liveTxns)

  // Auto-open a just-submitted transaction so the signing animation is visible.
  useEffect(() => {
    if (focusTxId) {
      setOpenTxId(focusTxId)
      onFocusHandled?.()
    }
  }, [focusTxId, onFocusHandled])

  const openTx = detail.transactions.find((t) => t.id === openTxId) ?? null

  return (
    <div className="mx-auto max-w-[1040px]">
      <div className="mb-5 flex items-center justify-between">
        <button onClick={onBack} className="text-sm font-semibold text-ink-soft hover:text-ink">
          ← Back to wallets
        </button>
        {onDelete && (
          <button
            onClick={() => setConfirmDelete(true)}
            className="btn btn-ghost-danger text-sm"
          >
            <TrashIcon size={16} /> Delete
          </button>
        )}
      </div>

      {confirmDelete && onDelete && (
        <DeleteWalletBar
          walletName={wallet.name}
          assets={detail.assets}
          onCancel={() => setConfirmDelete(false)}
          onDelete={() => onDelete(wallet.id)}
        />
      )}

      <section className="surface mb-5">
        <div className="mb-5 flex flex-col items-start gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex w-full min-w-0 items-center gap-3 sm:flex-1">
            <ChainLogo chain={wallet.chain} size={44} />
            <div className="min-w-0 flex-1">
              <div className="text-lg font-bold">{wallet.name}</div>
              <CopyAddress address={wallet.address} truncate />
            </div>
          </div>
          <a
            href={explorerAddress(wallet.chain, wallet.address)}
            target="_blank"
            rel="noreferrer"
            className="shrink-0 whitespace-nowrap bg-brand-soft px-2.5 py-1 text-xs font-semibold uppercase text-brand hover:underline"
          >
            {wallet.chain} · explorer ↗
          </a>
        </div>

        <div className="mb-1 text-sm text-muted">Balances</div>
        <BalanceList assets={detail.assets} fallbackSymbol={wallet.symbol} usd={detail.usd} />

        {wallet.chain === 'stellar' && !readOnly && (
          <AddAssetChips
            registry={detail.registry}
            heldKeys={new Set((detail.assets ?? []).map(assetKey))}
            pendingKeys={new Set(detail.pendingTrustlines)}
            onAdd={({ code, issuer }) => void detail.addTrustline(code, issuer)}
          />
        )}

        {!onSend && (
          <p className="notice notice-info mb-3">Only workspace owners and operators can send. Ask an owner to make you an operator.</p>
        )}
        <div className="grid grid-cols-2 gap-2 sm:flex sm:gap-3">
          {onSend && (
            <button
              onClick={onSend}
              className="btn btn-primary"
            >
              <SendIcon size={18} /> Send
            </button>
          )}
          <button
            onClick={() => setShowReceive(true)}
            className="btn btn-secondary"
          >
            Receive
          </button>
          <button
            onClick={() => void detail.fund()}
            disabled={detail.funding}
            className="btn btn-secondary col-span-2 sm:col-auto"
          >
            {detail.funding ? 'Funding…' : 'Fund (testnet)'}
          </button>
        </div>
      </section>

      <div className="mb-1 flex items-start gap-2 px-1 text-sm text-ink-soft sm:items-center">
        <span className="text-brand">
          <ShieldIcon size={16} />
        </span>
        Every outgoing transaction is signed by 2 of 3 nodes.
      </div>

      <TransactionList
        key={wallet.id}
        transactions={detail.transactions}
        hasMore={detail.hasMoreHistory}
        loadingMore={detail.loadingMoreHistory}
        onLoadMore={detail.loadMoreHistory}
        refreshing={detail.refreshing}
        onRefresh={() => void detail.refresh()}
        onOpen={setOpenTxId}
      />

      {showReceive && <ReceiveModal wallet={wallet} onClose={() => setShowReceive(false)} />}
      {openTx && (
        <TxDetailModal tx={openTx} chain={wallet.chain} onClose={() => setOpenTxId(null)} />
      )}
    </div>
  )
}
