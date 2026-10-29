import { useEffect, useState } from 'react'
import { api } from '../api.ts'
import { exceedsMax, maxSendable } from '../lib/stellar.ts'
import { assetKey, useBalances } from './useBalances.ts'
import { useDebouncedQuery } from './useDebouncedQuery.ts'
import type { SendInput, Wallet } from '../types.ts'

// A recipient worth resolving: G/M address or a name*domain federation string.
function looksResolvable(value: string): boolean {
  if (value === '') return false
  if (value.includes('*')) {
    const parts = value.split('*')
    return parts.length === 2 && parts[1].includes('.')
  }
  return (value[0] === 'G' || value[0] === 'M') && value.length >= 12
}

export function useSendForm(
  wallets: Wallet[],
  preselectId: string | undefined,
  onSubmit: (input: SendInput) => void,
) {
  const [from, setFrom] = useState(preselectId ?? wallets[0]?.id ?? '')
  const [to, setTo] = useState('')
  const [amount, setAmount] = useState('')
  const [assetId, setAssetId] = useState('XLM')
  const [memo, setMemo] = useState('')
  const [memoTouched, setMemoTouched] = useState(false)
  const [confirming, setConfirming] = useState(false)

  useEffect(() => {
    setFrom(preselectId ?? wallets[0]?.id ?? '')
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [preselectId])

  useEffect(() => {
    if (wallets.length > 0 && !wallets.some((wallet) => wallet.id === from)) setFrom(wallets[0].id)
  }, [wallets, from])

  const { assets } = useBalances(from)
  useEffect(() => {
    if (assets) setAssetId(assets[0] ? assetKey(assets[0]) : 'XLM')
  }, [assets])

  const recipientQuery = to.trim()
  const lookup = useDebouncedQuery(
    looksResolvable(recipientQuery) ? () => api.resolve(recipientQuery) : null,
    [recipientQuery],
    400,
    'Could not resolve address',
  )
  const resolved = lookup.data
  // Partial G…/M… addresses and federation names are still being typed; anything else can't become valid.
  const malformed = recipientQuery !== '' && !looksResolvable(recipientQuery) && !/^[GM]|\*/.test(recipientQuery)
  const recipient = {
    resolved,
    error: malformed ? 'Enter a valid Stellar address (G… or M…) or a federation address (name*domain.com).' : lookup.error,
    resolving: lookup.loading,
  }
  useEffect(() => {
    if (resolved?.memo && !memoTouched) setMemo(resolved.memo)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resolved])

  const fromWallet = wallets.find((w) => w.id === from)
  const selectedAsset = assets?.find((a) => assetKey(a) === assetId)
  const maxSend = maxSendable(Number(selectedAsset?.balance ?? 0), !selectedAsset?.issuer)
  const amountValue = Number(amount) || 0
  const overBalance = exceedsMax(amountValue, maxSend)
  const memoMissing = Boolean(resolved?.memo_type) && memo.trim() === ''
  const canSubmit = Boolean(from && resolved && amountValue > 0 && !overBalance && !memoMissing)

  function editMemo(value: string) {
    setMemo(value)
    setMemoTouched(true)
  }

  function review() {
    if (canSubmit) setConfirming(true)
  }

  function confirm() {
    if (!resolved) return
    onSubmit({
      walletId: from,
      to: resolved.address,
      amount,
      memo: memo.trim(),
      memoType: resolved.memo_type,
      asset: selectedAsset?.issuer
        ? { code: selectedAsset.symbol, issuer: selectedAsset.issuer }
        : undefined,
    })
    setConfirming(false)
  }

  return {
    from,
    setFrom,
    fromWallet,
    to,
    setTo,
    recipient,
    isSelfSend: Boolean(fromWallet && resolved && resolved.address === fromWallet.address),
    amount,
    setAmount,
    assets: assets ?? [],
    assetId,
    setAssetId,
    assetSymbol: selectedAsset?.symbol ?? assetId,
    maxSend,
    overBalance,
    memo,
    editMemo,
    canSubmit,
    confirming,
    review,
    cancelReview: () => setConfirming(false),
    confirm,
  }
}
