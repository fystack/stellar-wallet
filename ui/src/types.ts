export type Chain = 'stellar'

export type WalletStatus = 'generating' | 'ready' | 'failed'

export type WorkspaceRole = 'owner' | 'member'

export type Workspace = {
  id: string
  name: string
  role: WorkspaceRole
  personal: boolean
  createdAt: string
}

export type WorkspaceMember = {
  userId: string
  email: string
  role: WorkspaceRole
  joinedAt: string
}

export type WorkspaceInvite = {
  id: string
  workspaceId: string
  workspaceName?: string
  email: string
  invitedBy: string
  createdAt: string
  expiresAt: number
}

export type InviteStatus = 'pending' | 'accepted' | 'revoked' | 'expired'

export type InvitePreview = {
  workspaceName: string
  invitedBy: string
  email: string
  status: InviteStatus
  hasAccount: boolean
  hasPassword: boolean
  providers: string[]
}

export type JoinResult = {
  workspace: Workspace
  wallet?: Wallet
  walletError?: string
}

export type Wallet = {
  id: string
  workspaceId: string
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

export type ResolvedRecipient = {
  address: string
  memo_type?: string
  memo?: string
  federation?: string
}

export type SendInput = {
  walletId: string
  to: string
  amount: string
  memo: string
  memoType?: string
  asset?: { code: string; issuer: string }
}

export type SwapInput = {
  walletId: string
  from: string
  to: string
  amount: string
  slippageBps: number
}

export type CustomAsset = { code: string; issuer: string }
