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

export function sql(statement: string): string {
  return execFileSync('sqlite3', [DB, statement], { encoding: 'utf8' }).trim()
}

// Real Google sign-in can't run headless, so link a Google identity row the way a successful OAuth link would.
export function markGoogleVerified(account: Account, email = account.email) {
  sql(`INSERT INTO auth_identities (provider, subject, user_id, email) VALUES ('google', 'e2e-google-${account.id}', '${account.id}', '${email}');`)
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
  sql(`INSERT INTO wallets (id, user_id, workspace_id, name, chain, symbol, address, pubkey, balance, status, created_at)
     VALUES ('${id}', '${ownerId}', '${workspaceId}', '${name}', 'stellar', 'XLM', '${address}', '', '0', 'ready', '${new Date().toISOString()}');`)
  return id
}

export async function requestPayment(sender: APIRequestContext, walletId: string, to: string, amount: string, extra: Record<string, string> = {}) {
  const response = await sender.post('/api/v1/transactions', { data: { wallet_id: walletId, to, amount, ...extra } })
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

export const RECIPIENT = 'GAHPYWLK6YRN7CVYZOO4H3VDRZ7PVF5UJGLZCSPAEIKJE2XSWF5LAGER'

// Creating a team starts a keygen for its first wallet.
export async function walletsSettled(owner: APIRequestContext, workspaceId: string) {
  await expect
    .poll(async () => {
      const wallets: { status: string }[] = (await (await owner.get(`/api/v1/wallets?workspace_id=${workspaceId}`)).json()).data
      return wallets.some((wallet) => wallet.status === 'generating')
    }, { timeout: 180_000 })
    .toBe(false)
}

export async function approvalTeam({ extraOperators = 0, thresholdXlm = '10', requiredApprovals = 1 } = {}) {
  const owner = await registerAccount(uniqueEmail('owner'))
  const operator = await registerAccount(uniqueEmail('operator'))
  const viewer = await registerAccount(uniqueEmail('viewer'))
  const name = `Treasury ${Math.floor(Math.random() * 1e6)}`
  const workspaceId = await createWorkspace(owner.api, name)
  await walletsSettled(owner.api, workspaceId)
  await acceptByLink(operator.api, (await invite(owner.api, workspaceId, operator.email, 'operator')).token)
  await acceptByLink(viewer.api, (await invite(owner.api, workspaceId, viewer.email)).token)
  const operators: Account[] = []
  for (let index = 0; index < extraOperators; index++) {
    const extra = await registerAccount(uniqueEmail(`operator${index + 2}`))
    await acceptByLink(extra.api, (await invite(owner.api, workspaceId, extra.email, 'operator')).token)
    operators.push(extra)
  }
  await setApprovalPolicy(owner.api, workspaceId, thresholdXlm, requiredApprovals)
  return { owner, operator, viewer, operators, name, workspaceId }
}

export async function personalWorkspaceId(api: APIRequestContext): Promise<string> {
  const workspaces: { id: string; personal: boolean }[] = (await (await api.get('/api/v1/workspaces')).json()).data.workspaces
  return workspaces.find((workspace) => workspace.personal)!.id
}

export async function openWorkspace(page: Page, email: string, workspaceName: string) {
  await signIn(page, email)
  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeVisible()
  await page.getByRole('list', { name: 'Your workspaces' }).getByRole('button', { name: new RegExp(workspaceName) }).click()
  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeHidden()
}

export async function openRequest(page: Page, amount: string) {
  await page.getByRole('button', { name: /^Approvals/ }).click()
  await page.getByRole('button', { name: new RegExp(`Review ${amount} XLM`) }).click()
  await expect(page.getByRole('heading', { name: 'Payment request' })).toBeVisible()
}

export async function paymentRequest(api: APIRequestContext, id: string) {
  return (await (await api.get(`/api/v1/payment-requests/${id}`)).json()).data
}

export async function errorCode(response: { json(): Promise<unknown> }): Promise<string> {
  return ((await response.json()) as { error: { code: string } }).error.code
}

export const MAILPIT = process.env.E2E_MAILPIT ?? 'http://localhost:8025'

export async function mailpitOnline(): Promise<boolean> {
  const api = await request.newContext({ baseURL: MAILPIT })
  try {
    return (await api.get('/api/v1/messages?limit=1')).ok()
  } catch {
    return false
  } finally {
    await api.dispose()
  }
}

// Emails are sent in the background, so poll until the latest one to this address arrives.
export async function latestEmailText(to: string): Promise<string> {
  const api = await request.newContext({ baseURL: MAILPIT })
  try {
    let id = ''
    await expect
      .poll(async () => {
        const found = await (await api.get(`/api/v1/search?query=${encodeURIComponent(`to:"${to}"`)}`)).json()
        id = found.messages?.[0]?.ID ?? ''
        return id
      }, { timeout: 45_000 })
      .not.toBe('')
    return (await (await api.get(`/api/v1/message/${id}`)).json()).Text as string
  } finally {
    await api.dispose()
  }
}
