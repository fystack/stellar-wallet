import { execFileSync } from 'node:child_process'
import { expect, request, type APIRequestContext, type Page } from '@playwright/test'

export const API = process.env.E2E_API ?? 'http://localhost:8090'
// The backend's SQLite file, used only to mark test accounts as Google-verified.
export const DB = process.env.E2E_DB ?? '../backend/wallet.db'
export const PASSWORD = 'secret123'

export type Invite = { token: string; path: string; inviteId: string }
export type Account = { email: string; id: string; api: APIRequestContext }

export function uniqueEmail(label: string) {
  return `e2e_${label}_${Date.now()}_${Math.floor(Math.random() * 1e6)}@test.io`
}

export async function registerAccount(email: string): Promise<Account> {
  const api = await request.newContext({ baseURL: API, extraHTTPHeaders: { 'X-CSRF-Protection': '1' } })
  const response = await api.post('/api/v1/auth/register', { data: { email, password: PASSWORD } })
  expect(response.status(), await response.text()).toBe(201)
  return { email, id: (await response.json()).data.id, api }
}

// Real Google sign-in can't run headless, so link a Google identity row the way a successful OAuth link would.
export function markGoogleVerified(account: Account) {
  const subject = `e2e-google-${account.id}`
  execFileSync('sqlite3', [
    DB,
    `INSERT INTO auth_identities (provider, subject, user_id, email) VALUES ('google', '${subject}', '${account.id}', '${account.email}');`,
  ])
}

export async function createWorkspace(owner: APIRequestContext, name: string): Promise<string> {
  const response = await owner.post('/api/v1/workspaces', { data: { name } })
  expect(response.status(), await response.text()).toBe(201)
  return (await response.json()).data.workspace.id
}

export async function invite(owner: APIRequestContext, workspaceId: string, email: string, role = 'member'): Promise<Invite> {
  const response = await owner.post(`/api/v1/workspaces/${workspaceId}/invites`, { data: { email, role } })
  expect(response.status(), await response.text()).toBe(201)
  const { invite, link } = (await response.json()).data
  const token = link.split('/#/invite/')[1]
  return { token, path: `/#/invite/${token}`, inviteId: invite.id }
}

export async function teamWithInvite(inviteeEmail: string) {
  const owner = await registerAccount(uniqueEmail('owner'))
  const name = `Team ${Math.floor(Math.random() * 1e6)}`
  const workspaceId = await createWorkspace(owner.api, name)
  return { owner: owner.api, name, workspaceId, invite: await invite(owner.api, workspaceId, inviteeEmail) }
}

export async function acceptByLink(invitee: APIRequestContext, token: string) {
  const response = await invitee.post(`/api/v1/invites/${token}/accept`)
  expect(response.status(), await response.text()).toBe(200)
}

export async function signIn(page: Page, email: string) {
  await page.goto('/')
  await page.getByLabel('Email').fill(email)
  await page.getByLabel('Password', { exact: true }).fill(PASSWORD)
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
}

export function memberSwitcher(page: Page, workspaceName: string) {
  return page.getByRole('button', { name: new RegExp(`${workspaceName}\\s+Member`) })
}

export async function setApprovalPolicy(owner: APIRequestContext, workspaceId: string, thresholdXlm: string, requiredApprovals = 1) {
  const response = await owner.put(`/api/v1/workspaces/${workspaceId}/approval-policy`, {
    data: { thresholdXlm, requiredApprovals, belowThreshold: 'send' },
  })
  expect(response.status(), await response.text()).toBe(200)
}

// Holding a payment never signs, so a ready wallet row is enough when the MPC nodes are down.
export function insertReadyWallet(workspaceId: string, ownerId: string, name: string, address: string) {
  const id = `e2e-wallet-${Date.now()}-${Math.floor(Math.random() * 1e6)}`
  execFileSync('sqlite3', [
    DB,
    `INSERT INTO wallets (id, user_id, workspace_id, name, chain, symbol, address, pubkey, balance, status, created_at)
     VALUES ('${id}', '${ownerId}', '${workspaceId}', '${name}', 'stellar', 'XLM', '${address}', '', '0', 'ready', '${new Date().toISOString()}');`,
  ])
  return id
}

export async function requestPayment(sender: APIRequestContext, walletId: string, to: string, amount: string) {
  const response = await sender.post('/api/v1/transactions', { data: { wallet_id: walletId, to, amount } })
  expect(response.status(), await response.text()).toBe(202)
  return (await response.json()).data.request as { id: string; requiredApprovals: number }
}

export async function clusterOnline(): Promise<boolean> {
  const api = await request.newContext({ baseURL: API })
  try {
    const response = await api.get('/api/v1/cluster')
    const nodes: { online: boolean }[] = (await response.json()).data.nodes
    return nodes.filter((node) => node.online).length >= 3
  } finally {
    await api.dispose()
  }
}
