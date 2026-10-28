import { useCallback, useEffect, useState } from 'react'
import { api } from '../api.ts'
import { exceedsMax, maxSendable } from '../lib/stellar.ts'
import { useDebouncedQuery } from './useDebouncedQuery.ts'
import { useSwapAssets } from './useSwapAssets.ts'
import { useTrustlineRequest } from './useTrustlineRequest.ts'
import type { SwapInput, Wallet } from '../types.ts'

const QUOTE_DEBOUNCE_MS = 450

export function useSwapForm(
  wallets: Wallet[],
  preselectId: string | undefined,
  onSubmit: (input: SwapInput) => Promise<void>,
) {
  const [from, setFrom] = useState(preselectId ?? wallets[0]?.id ?? '')
  const [payAsset, setPayAsset] = useState('XLM')
  const [receiveAsset, setReceiveAsset] = useState('')
  const [amount, setAmount] = useState('')
  const [slippage, setSlippage] = useState(100)
  const [submitting, setSubmitting] = useState(false)
  const [confirming, setConfirming] = useState(false)

  useEffect(() => {
    setFrom(preselectId ?? wallets[0]?.id ?? '')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [preselectId])

  const { held, receivable, reload } = useSwapAssets(from, wallets.find((wallet) => wallet.id === from)?.workspaceId ?? '')
  const reloadAssets = useCallback(() => {
    reload().catch(() => {})
  }, [reload])
  const trustline = useTrustlineRequest(from, reloadAssets)

  useEffect(() => {
    setPayAsset(held[0]?.value ?? 'XLM')
  }, [held])

  useEffect(() => {
    const firstDifferent = receivable.find((option) => option.value !== payAsset)
    setReceiveAsset((current) =>
      current && current !== payAsset ? current : (firstDifferent?.value ?? ''),
    )
  }, [receivable, payAsset])

  const amountValue = Number(amount) || 0
  const quoteReady = Boolean(from && receiveAsset && payAsset !== receiveAsset && amountValue > 0)
  const quote = useDebouncedQuery(
    quoteReady ? () => api.swapQuote(from, payAsset, receiveAsset, amount) : null,
    [from, payAsset, receiveAsset, amount],
    QUOTE_DEBOUNCE_MS,
    'No quote available',
  )

  const payBalance = Number(held.find((h) => h.value === payAsset)?.balance ?? 0)
  const maxPay = maxSendable(payBalance, payAsset === 'XLM')
  const overBalance = exceedsMax(amountValue, maxPay)
  const receiveAmount = quote.data?.dest_amount ?? null
  const rate = receiveAmount && amountValue > 0 ? Number(receiveAmount) / amountValue : 0
  const minReceive = receiveAmount ? Number(receiveAmount) * (1 - slippage / 10000) : 0
  // Stellar only credits an asset to an account that already trusts it.
  const needsTrustline =
    receiveAsset !== '' && receiveAsset !== 'XLM' && !held.some((h) => h.value === receiveAsset)
  const canFlip = held.some((h) => h.value === receiveAsset)
  const canSubmit = Boolean(quote.data && !overBalance && quoteReady && !needsTrustline && !submitting)

  function flip() {
    if (!canFlip) return
    setPayAsset(receiveAsset)
    setReceiveAsset(payAsset)
    setAmount('')
  }

  function addTrustline() {
    const [code, issuer] = receiveAsset.split(':')
    if (code && issuer) void trustline.add(code, issuer)
  }

  function review() {
    if (canSubmit) setConfirming(true)
  }

  async function confirm() {
    setSubmitting(true)
    try {
      await onSubmit({ walletId: from, from: payAsset, to: receiveAsset, amount, slippageBps: slippage })
      setConfirming(false)
    } finally {
      setSubmitting(false)
    }
  }

  return {
    from,
    setFrom,
    amount,
    setAmount,
    payAsset,
    setPayAsset,
    payOptions: held,
    payBalance,
    maxPay,
    overBalance,
    receiveAsset,
    setReceiveAsset,
    receiveOptions: receivable.filter((option) => option.value !== payAsset),
    quote,
    receiveAmount,
    rate,
    minReceive,
    slippage,
    setSlippage,
    canFlip,
    flip,
    needsTrustline,
    addingTrustline: trustline.pending.includes(receiveAsset),
    addTrustline,
    canSubmit,
    submitting,
    confirming,
    review,
    cancelReview: () => setConfirming(false),
    confirm,
  }
}
