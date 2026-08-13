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

export type Transaction = {
  id: string
  walletId: string
  type: 'in' | 'out'
  counterparty: string
  amount: string
  symbol: string
  memo?: string
  status: TxStatus
  signature?: string
  txHash?: string
  error?: string
  createdAt: string
}
