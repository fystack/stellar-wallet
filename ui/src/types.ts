export type Chain = 'stellar' | 'solana'

export type WalletStatus = 'generating' | 'ready' | 'failed'

export type Wallet = {
  id: string
  name: string
  chain: Chain
  symbol: string
  address: string
  pubkey: string
  balance: string
  status: WalletStatus
}

// Lifecycle of an outgoing MPC transaction (mirrors the mpcium flow).
export type TxStatus =
  | 'policy_check'
  | 'signing'
  | 'broadcast'
  | 'confirmed'
  | 'failed'

export type SwapAsset = { code: string; issuer: string }

export type SwapQuote = {
  send_asset: SwapAsset
  send_amount: string
  dest_asset: SwapAsset
  dest_amount: string
  path: SwapAsset[]
}

export type Transaction = {
  id: string
  walletId: string
  type: 'in' | 'out' | 'swap'
  counterparty: string
  amount: string
  symbol: string
  recvAmount?: string
  recvSymbol?: string
  memo?: string
  status: TxStatus
  signature?: string
  txHash?: string
  error?: string
  createdAt: string
}
