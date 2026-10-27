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

export async function invite(owner: APIRequestContext, workspaceId: string, email: string): Promise<Invite> {
  const response = await owner.post(`/api/v1/workspaces/${workspaceId}/invites`, { data: { email } })
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
