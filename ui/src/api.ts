import type { SwapQuote, Transaction, Wallet } from './types.ts'

export const BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8090'
const TOKEN_KEY = 'wallet_token'
const EMAIL_KEY = 'wallet_email'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}
export function getEmail(): string | null {
  return localStorage.getItem(EMAIL_KEY)
}
export function clearSession() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(EMAIL_KEY)
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message)
  }
}

async function req<T>(path: string, opts: RequestInit & { json?: unknown } = {}): Promise<T> {
  const { json, ...rest } = opts
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(rest.headers as Record<string, string>),
  }
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`

  const res = await fetch(`${BASE}${path}`, {
    ...rest,
    headers,
    body: json !== undefined ? JSON.stringify(json) : rest.body,
  })

  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      msg = (await res.json()).error ?? msg
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, msg)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

type AuthResp = { access_token: string; expires_in: number; email: string }

async function auth(kind: 'login' | 'register', email: string, password: string) {
  const data = await req<AuthResp>(`/api/v1/auth/${kind}`, {
    method: 'POST',
    json: { email, password },
  })
  localStorage.setItem(TOKEN_KEY, data.access_token)
  localStorage.setItem(EMAIL_KEY, data.email)
  return data.email
}

export const api = {
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
      solanaUrl: string
      assets: { code: string; issuer: string }[]
    }>(`/api/v1/config`),
  putConfig: (horizonUrl: string, solanaUrl: string) =>
    req<{ assets: { code: string; issuer: string }[] }>(`/api/v1/config`, {
      method: 'PUT',
      json: { horizonUrl, solanaUrl },
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
    return `https://stellar.expert/explorer/testnet/tx/${hash}`
  if (chain === 'solana')
    return `https://explorer.solana.com/tx/${hash}?cluster=devnet`
  return ''
}

export function explorerAddress(chain: string, address: string): string {
  if (chain === 'stellar')
    return `https://stellar.expert/explorer/testnet/account/${address}`
  if (chain === 'solana')
    return `https://explorer.solana.com/address/${address}?cluster=devnet`
  return ''
}
