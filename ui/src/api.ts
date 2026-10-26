import type { SwapQuote, Transaction, Wallet } from './types.ts'

export const BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8090'
export type Account = {
  id: string
  email: string
  has_password: boolean
  identities: { provider: string; email: string }[]
}
export type SocialProvider = { id: 'google' | 'github'; enabled: boolean }

export function clearLegacySession() {
  localStorage.removeItem('wallet_token')
  localStorage.removeItem('wallet_email')
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
  if (res.status === 204) return undefined as T
  let body: unknown
  try {
    body = await res.json()
  } catch {
    throw new ApiError(res.status, 'Invalid API response.', 'invalid_response')
  }
  if (typeof body !== 'object' || body === null || !('data' in body)) {
    throw new ApiError(res.status, 'Invalid API response.', 'invalid_response')
  }
  return (body as ApiResponse<T>).data
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
  providers: () => req<SocialProvider[]>('/api/v1/auth/providers'),
  oauthStartURL: (provider: SocialProvider['id']) => `${BASE}/api/v1/auth/${provider}/start`,
  linkProvider: (provider: SocialProvider['id']) => req<{ url: string }>(`/api/v1/auth/${provider}/link`, { method: 'POST' }),
  reauthenticate: (password: string) => req<void>('/api/v1/auth/reauthenticate', { method: 'POST', json: { password } }),
  login: (email: string, password: string) => auth('login', email, password),
  register: (email: string, password: string) => auth('register', email, password),

  listWallets: () => req<Wallet[]>('/api/v1/wallets'),
  createWallet: (name: string, chain: string) =>
    req<Wallet>('/api/v1/wallets', { method: 'POST', json: { name, chain } }),
  walletTxns: (id: string) =>
    req<Transaction[]>(`/api/v1/wallets/${id}/transactions`),
  createTxn: (
    walletId: string,
    to: string,
    amount: string,
    memo: string,
    asset?: { code: string; issuer: string },
    memoType?: string,
  ) =>
    req<Transaction>('/api/v1/transactions', {
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
    req<{
      address: string
      memo_type?: string
      memo?: string
      federation?: string
    }>(`/api/v1/resolve?q=${encodeURIComponent(q)}`),
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
  getConfig: () =>
    req<{
      horizonUrl: string
      assets: { code: string; issuer: string }[]
    }>(`/api/v1/config`),
  putConfig: (horizonUrl: string) =>
    req<{ assets: { code: string; issuer: string }[] }>(`/api/v1/config`, {
      method: 'PUT',
      json: { horizonUrl },
    }),
  addAsset: (code: string, issuer: string) =>
    req<{ assets: { code: string; issuer: string }[] }>(`/api/v1/assets`, {
      method: 'POST',
      json: { code, issuer },
    }),
  removeAsset: (code: string, issuer: string) =>
    req<{ assets: { code: string; issuer: string }[] }>(
      `/api/v1/assets?code=${encodeURIComponent(code)}&issuer=${encodeURIComponent(issuer)}`,
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
