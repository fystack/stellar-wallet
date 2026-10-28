import type {
  ApprovalPolicy,
  ApprovalPolicySettings,
  CustomAsset,
  InvitePreview,
  JoinResult,
  Page,
  PaymentRequest,
  PaymentResult,
  ResolvedRecipient,
  SwapQuote,
  Transaction,
  Wallet,
  Workspace,
  WorkspaceInvite,
  WorkspaceMember,
  WorkspaceRole,
} from './types.ts'

export const BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8090'
export type Account = {
  id: string
  email: string
  has_password: boolean
  identities: { provider: string; email: string }[]
}
export type SocialProvider = { id: 'google' | 'github'; enabled: boolean }

export function clearLegacySession() {
  try {
    localStorage.removeItem('wallet_token')
    localStorage.removeItem('wallet_email')
  } catch {
    // Nothing to clear when storage is blocked.
  }
}

export type ApiResponse<T> = { data: T; meta?: Record<string, unknown> }

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public code: string = 'request_failed',
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function req<T>(path: string, opts: RequestInit & { json?: unknown } = {}): Promise<T> {
  return (await send<T>(path, opts)).data
}

async function reqPage<T>(path: string): Promise<Page<T>> {
  const body = await send<T[]>(path)
  const next = body.meta?.next_cursor
  return { items: body.data, nextCursor: typeof next === 'string' ? next : null }
}

async function send<T>(path: string, opts: RequestInit & { json?: unknown } = {}): Promise<ApiResponse<T>> {
  const { json, ...rest } = opts
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-CSRF-Protection': '1',
    ...(rest.headers as Record<string, string>),
  }

  const res = await fetch(`${BASE}${path}`, {
    ...rest,
    credentials: 'include',
    headers,
    body: json !== undefined ? JSON.stringify(json) : rest.body,
  })

  if (!res.ok) {
    let message = `Request failed (HTTP ${res.status}).`
    let code = 'request_failed'
    try {
      const body = await res.json() as { error?: { code?: unknown; message?: unknown } }
      if (typeof body.error?.code === 'string') code = body.error.code
      if (typeof body.error?.message === 'string') message = body.error.message
    } catch {
      // Gateways may return HTML or an empty body instead of the API contract.
    }
    throw new ApiError(res.status, message, code)
  }
  if (res.status === 204) return { data: undefined as T }
  let body: unknown
  try {
    body = await res.json()
  } catch {
    throw new ApiError(res.status, 'Invalid API response.', 'invalid_response')
  }
  if (typeof body !== 'object' || body === null || !('data' in body)) {
    throw new ApiError(res.status, 'Invalid API response.', 'invalid_response')
  }
  return body as ApiResponse<T>
}

async function auth(kind: 'login' | 'register', email: string, password: string) {
  const account = await req<Account>(`/api/v1/auth/${kind}`, {
    method: 'POST', json: { email, password },
  })
  return account.email
}

export const api = {
  me: () => req<Account>('/api/v1/auth/me'),
  logout: () => req<void>('/api/v1/auth/logout', { method: 'POST' }),
  logoutEverywhere: () => req<void>('/api/v1/auth/logout-all', { method: 'POST' }),
  providers: () => req<SocialProvider[]>('/api/v1/auth/providers'),
  oauthStartURL: (provider: SocialProvider['id']) => `${BASE}/api/v1/auth/${provider}/start`,
  linkProvider: (provider: SocialProvider['id']) => req<{ url: string }>(`/api/v1/auth/${provider}/link`, { method: 'POST' }),
  reauthenticate: (password: string) => req<void>('/api/v1/auth/reauthenticate', { method: 'POST', json: { password } }),
  login: (email: string, password: string) => auth('login', email, password),
  register: (email: string, password: string) => auth('register', email, password),

  workspaces: () => req<{ workspaces: Workspace[]; activeId: string; selected: boolean }>('/api/v1/workspaces'),
  createWorkspace: (name: string) =>
    req<JoinResult>('/api/v1/workspaces', { method: 'POST', json: { name } }),
  switchWorkspace: (id: string) =>
    req<Workspace>(`/api/v1/workspaces/${id}/switch`, { method: 'POST' }),
  members: (id: string) =>
    req<{ members: WorkspaceMember[]; invites: WorkspaceInvite[] }>(`/api/v1/workspaces/${id}/members`),
  removeMember: (id: string, userId: string) =>
    req<void>(`/api/v1/workspaces/${id}/members/${userId}`, { method: 'DELETE' }),
  invite: (id: string, email: string, role: WorkspaceRole) =>
    req<{ invite: WorkspaceInvite; link: string; emailSent: boolean }>(`/api/v1/workspaces/${id}/invites`, {
      method: 'POST',
      json: { email, role },
    }),
  changeRole: (id: string, userId: string, role: WorkspaceRole) =>
    req<void>(`/api/v1/workspaces/${id}/members/${userId}`, { method: 'PATCH', json: { role } }),
  approvalPolicy: (id: string) =>
    req<ApprovalPolicySettings>(`/api/v1/workspaces/${id}/approval-policy`),
  setApprovalPolicy: (id: string, policy: Pick<ApprovalPolicy, 'thresholdXlm' | 'requiredApprovals' | 'belowThreshold'>) =>
    req<ApprovalPolicySettings>(`/api/v1/workspaces/${id}/approval-policy`, { method: 'PUT', json: policy }),
  disableApprovalPolicy: (id: string) =>
    req<void>(`/api/v1/workspaces/${id}/approval-policy`, { method: 'DELETE' }),
  paymentRequests: (workspaceId: string, status: 'pending' | 'history', cursor?: string | null) =>
    reqPage<PaymentRequest>(
      `/api/v1/workspaces/${workspaceId}/payment-requests?status=${status}${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`,
    ),
  paymentRequest: (id: string) => req<PaymentRequest>(`/api/v1/payment-requests/${id}`),
  approvePayment: (id: string) => req<PaymentRequest>(`/api/v1/payment-requests/${id}/approve`, { method: 'POST' }),
  rejectPayment: (id: string) => req<PaymentRequest>(`/api/v1/payment-requests/${id}/reject`, { method: 'POST' }),
  cancelPayment: (id: string) => req<PaymentRequest>(`/api/v1/payment-requests/${id}/cancel`, { method: 'POST' }),
  revokeInvite: (id: string, inviteId: string) =>
    req<void>(`/api/v1/workspaces/${id}/invites/${inviteId}`, { method: 'DELETE' }),
  invitations: () => req<{ invites: WorkspaceInvite[]; verified: boolean }>('/api/v1/invitations'),
  acceptInvitation: (id: string) =>
    req<JoinResult>(`/api/v1/invitations/${id}/accept`, { method: 'POST' }),
  declineInvitation: (id: string) =>
    req<void>(`/api/v1/invitations/${id}/decline`, { method: 'POST' }),
  previewInvite: (token: string) =>
    req<InvitePreview>(`/api/v1/invites/${encodeURIComponent(token)}`),
  acceptInvite: (token: string) =>
    req<JoinResult>(`/api/v1/invites/${encodeURIComponent(token)}/accept`, { method: 'POST' }),

  listWallets: (workspaceId: string) =>
    req<Wallet[]>(`/api/v1/wallets?workspace_id=${encodeURIComponent(workspaceId)}`),
  createWallet: (workspaceId: string, name: string, chain: string) =>
    req<Wallet>('/api/v1/wallets', { method: 'POST', json: { workspace_id: workspaceId, name, chain } }),
  walletTxns: (id: string, cursor?: string | null) =>
    reqPage<Transaction>(`/api/v1/wallets/${id}/transactions${cursor ? `?cursor=${encodeURIComponent(cursor)}` : ''}`),
  createTxn: (
    walletId: string,
    to: string,
    amount: string,
    memo: string,
    asset?: { code: string; issuer: string },
    memoType?: string,
  ) =>
    req<PaymentResult>('/api/v1/transactions', {
      method: 'POST',
      json: {
        wallet_id: walletId,
        to,
        amount,
        memo,
        memo_type: memoType ?? '',
        asset_code: asset?.code ?? '',
        asset_issuer: asset?.issuer ?? '',
      },
    }),
  resolve: (q: string) =>
    req<ResolvedRecipient>(`/api/v1/resolve?q=${encodeURIComponent(q)}`),
  swapQuote: (walletId: string, from: string, to: string, amount: string) =>
    req<SwapQuote>(
      `/api/v1/wallets/${walletId}/swap/quote?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}&amount=${encodeURIComponent(amount)}`,
    ),
  createSwap: (
    walletId: string,
    from: string,
    to: string,
    amount: string,
    slippageBps: number,
  ) =>
    req<{ transaction: Transaction; quote: SwapQuote; dest_min: string }>(
      `/api/v1/wallets/${walletId}/swap`,
      {
        method: 'POST',
        json: { from, to, amount, slippage_bps: slippageBps },
      },
    ),
  balance: (id: string) =>
    req<{
      assets: { symbol: string; balance: string; issuer?: string }[]
      symbol: string
      explorer: string
    }>(`/api/v1/wallets/${id}/balance`),
  fund: (id: string) =>
    req<{ balance: string }>(`/api/v1/wallets/${id}/fund`, { method: 'POST' }),
  deleteWallet: (id: string) =>
    req<{ ok: boolean }>(`/api/v1/wallets/${id}`, { method: 'DELETE' }),
  sync: (id: string) =>
    req<{ synced: number }>(`/api/v1/wallets/${id}/sync`),
  cluster: () =>
    req<{
      threshold: string
      nodes: { name: string; region: string; online: boolean }[]
    }>(`/api/v1/cluster`),
  prices: () => req<Record<string, number>>(`/api/v1/prices`),
  txChain: (hash: string) =>
    req<{
      fee: string
      ledger: number
      operations: number
      source: string
      memo: string
      successful: boolean
    }>(`/api/v1/tx/${hash}/chain`),
  chains: () =>
    req<
      {
        chain: string
        name: string
        symbol: string
        network: string
        rpc: string
        online: boolean
      }[]
    >(`/api/v1/chains`),
  getConfig: () => req<{ horizonUrl: string }>(`/api/v1/config`),
  listAssets: (workspaceId: string) =>
    req<{ assets: CustomAsset[] }>(`/api/v1/workspaces/${workspaceId}/assets`),
  addAsset: (workspaceId: string, code: string, issuer: string) =>
    req<{ assets: CustomAsset[] }>(`/api/v1/workspaces/${workspaceId}/assets`, {
      method: 'POST',
      json: { code, issuer },
    }),
  removeAsset: (workspaceId: string, code: string, issuer: string) =>
    req<{ assets: CustomAsset[] }>(
      `/api/v1/workspaces/${workspaceId}/assets?code=${encodeURIComponent(code)}&issuer=${encodeURIComponent(issuer)}`,
      { method: 'DELETE' },
    ),
  addTrustline: (id: string, code: string, issuer: string) =>
    req<Transaction>(`/api/v1/wallets/${id}/trustline`, {
      method: 'POST',
      json: { code, issuer },
    }),
}

export function explorerTx(chain: string, hash: string): string {
  if (chain === 'stellar')
    return `https://testnet.stellarchain.io/transactions/${hash}`
  return ''
}

export function explorerAddress(chain: string, address: string): string {
  if (chain === 'stellar')
    return `https://testnet.stellarchain.io/accounts/${address}`
  return ''
}
