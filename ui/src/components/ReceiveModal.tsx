import { useEffect } from 'react'
import QRCode from 'react-qr-code'
import Modal from './Modal.tsx'
import CopyAddress from './CopyAddress.tsx'
import { api } from '../api.ts'
import type { Wallet } from '../types.ts'

export default function ReceiveModal({
  wallet,
  onClose,
}: {
  wallet: Wallet
  onClose: () => void
}) {
  // While the QR is open, poll chain history so incoming payments show up live.
  useEffect(() => {
    const poll = () => api.sync(wallet.id).catch(() => {})
    poll()
    const t = setInterval(poll, 5000)
    return () => clearInterval(t)
  }, [wallet.id])

  return (
    <Modal title="Receive" onClose={onClose} maxWidth={380}>
      <div className="text-center">
        <span className="inline-block bg-brand-soft px-3 py-1 text-xs font-semibold uppercase text-brand">
          {wallet.chain}
        </span>

        <div className="my-6 flex justify-center">
          <div className="border border-line bg-white p-4">
            <QRCode value={wallet.address} size={176} />
          </div>
        </div>

        <div className="bg-card px-4 py-3">
          <p className="mb-1.5 text-xs font-medium uppercase tracking-wide text-muted">
            Address
          </p>
          <CopyAddress
            address={wallet.address}
            className="justify-center break-all"
          />
        </div>

        <p className="mt-4 text-xs leading-relaxed text-muted">
          Only send {wallet.symbol} on {wallet.chain} to this address.
        </p>

        <div className="mt-3 flex items-center justify-center gap-2 text-xs font-medium text-brand">
          <span className="h-2 w-2 animate-pulse rounded-full bg-brand" />
          Listening for incoming payments…
        </div>
      </div>
    </Modal>
  )
}
