import { test, expect, type APIRequestContext, type Page } from '@playwright/test'
import {
  acceptByLink,
  clusterOnline,
  createWorkspace,
  insertReadyWallet,
  invite,
  registerAccount,
  requestPayment,
  setApprovalPolicy,
  signIn,
  uniqueEmail,
} from './helpers.ts'
import { fundAccount } from './fund.ts'

// Only the last test needs MPC nodes; it is skipped when they are offline.

// Serial, with long waits: keygens queue on the shared local cluster.
test.describe.configure({ mode: 'serial', timeout: 300_000 })

const RECIPIENT = 'GAHPYWLK6YRN7CVYZOO4H3VDRZ7PVF5UJGLZCSPAEIKJE2XSWF5LAGER'

// Creating the team starts a keygen for its first wallet.
async function walletsSettled(owner: APIRequestContext, workspaceId: string) {
  await expect
    .poll(async () => {
      const wallets: { status: string }[] = (await (await owner.get(`/api/v1/wallets?workspace_id=${workspaceId}`)).json()).data
      return wallets.some((wallet) => wallet.status === 'generating')
    }, { timeout: 180_000 })
    .toBe(false)
}

async function approvalTeam() {
  const owner = await registerAccount(uniqueEmail('owner'))
  const operator = await registerAccount(uniqueEmail('operator'))
  const viewer = await registerAccount(uniqueEmail('viewer'))
  const name = `Treasury ${Math.floor(Math.random() * 1e6)}`
  const workspaceId = await createWorkspace(owner.api, name)
  await walletsSettled(owner.api, workspaceId)
  await acceptByLink(operator.api, (await invite(owner.api, workspaceId, operator.email, 'operator')).token)
  await acceptByLink(viewer.api, (await invite(owner.api, workspaceId, viewer.email)).token)
  await setApprovalPolicy(owner.api, workspaceId, '10')
  return { owner, operator, viewer, name, workspaceId }
}

async function openWorkspace(page: Page, email: string, workspaceName: string) {
  await signIn(page, email)
  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeVisible()
  await page.getByRole('list', { name: 'Your workspaces' }).getByRole('button', { name: new RegExp(workspaceName) }).click()
  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeHidden()
}

async function openRequest(page: Page, amount: string) {
  await page.getByRole('button', { name: /^Approvals/ }).click()
  await page.getByRole('button', { name: new RegExp(`Review ${amount} XLM`) }).click()
  await expect(page.getByRole('heading', { name: 'Payment request' })).toBeVisible()
}

test('operator payment above the threshold waits for approval and the owner rejects it', async ({ page }) => {
  const team = await approvalTeam()
  const walletId = insertReadyWallet(team.workspaceId, team.owner.id, 'Ops wallet', RECIPIENT)
  await requestPayment(team.operator.api, walletId, RECIPIENT, '20')

  await openWorkspace(page, team.operator.email, team.name)
  await openRequest(page, '20')
  await expect(page.getByText('Another owner or operator needs to approve your payment.')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Approve payment' })).toBeHidden()
  await expect(page.getByRole('button', { name: 'Cancel request' })).toBeVisible()
  await page.getByRole('button', { name: 'Close' }).click()

  await page.getByRole('button', { name: /^Sign out/ }).click()
  await openWorkspace(page, team.owner.email, team.name)
  await expect(page.getByRole('button', { name: 'Approvals, 1 pending' })).toBeVisible()
  await openRequest(page, '20')
  await expect(page.getByText('0 of 1')).toBeVisible()
  await page.getByRole('button', { name: 'Reject' }).click()
  await expect(page.getByText('Payment rejected')).toBeVisible()
  await expect(page.getByText('Rejected').first()).toBeVisible()
  await expect(page.getByRole('button', { name: 'Approve payment' })).toBeHidden()
  await page.getByRole('button', { name: 'Close' }).click()

  await page.getByRole('tab', { name: 'History' }).click()
  await expect(page.getByRole('button', { name: /Review 20 XLM/ })).toBeVisible()
  await expect(page.getByRole('button', { name: /^Approvals$/ })).toBeVisible()
})

test('the initiator cancels a held payment', async ({ page }) => {
  const team = await approvalTeam()
  const walletId = insertReadyWallet(team.workspaceId, team.owner.id, 'Ops wallet', RECIPIENT)
  await requestPayment(team.operator.api, walletId, RECIPIENT, '25')

  await openWorkspace(page, team.operator.email, team.name)
  await openRequest(page, '25')
  await page.getByRole('button', { name: 'Cancel request' }).click()
  await expect(page.getByText('Payment request cancelled')).toBeVisible()
  await expect(page.getByRole('button', { name: 'Cancel request' })).toBeHidden()
})

test('read-only members see requests but cannot send or approve', async ({ page }) => {
  const team = await approvalTeam()
  const walletId = insertReadyWallet(team.workspaceId, team.owner.id, 'Ops wallet', RECIPIENT)
  await requestPayment(team.operator.api, walletId, RECIPIENT, '30')

  await openWorkspace(page, team.viewer.email, team.name)
  await openRequest(page, '30')
  await expect(page.getByRole('button', { name: 'Approve payment' })).toBeHidden()
  await expect(page.getByRole('button', { name: 'Cancel request' })).toBeHidden()
  await page.getByRole('button', { name: 'Close' }).click()
  await page.getByRole('button', { name: 'Send' }).click()
  await expect(page.getByRole('heading', { name: 'View-only access' })).toBeVisible()
})

test('owner sets the approval policy and roles in Settings', async ({ page }) => {
  const team = await approvalTeam()
  await openWorkspace(page, team.owner.email, team.name)
  await page.getByRole('button', { name: 'Settings' }).click()
  const policy = page.getByRole('region', { name: 'Payment approvals' })
  await expect(policy.getByRole('status')).toContainText('Payments over 10 XLM')
  await policy.getByLabel('Threshold (XLM)').fill('250')
  await policy.getByRole('button', { name: 'Save policy' }).click()
  await expect(policy.getByRole('status')).toContainText('Payments over 250 XLM')

  await page.getByLabel(`Role for ${team.viewer.email}`).selectOption('operator')
  await expect(page.getByText(`${team.viewer.email} is now operator`)).toBeVisible()
  await expect(policy.getByLabel('Approvals required').getByRole('option')).toHaveCount(2)
})

test('approved payment is signed and broadcast on testnet', async ({ page }) => {
  test.skip(!(await clusterOnline()), 'needs all MPC nodes online for keygen and signing')
  const team = await approvalTeam()
  const walletResponse = await team.owner.api.post('/api/v1/wallets', { data: { workspace_id: team.workspaceId, name: 'Funded', chain: 'stellar' } })
  expect(walletResponse.status(), await walletResponse.text()).toBe(201)
  const walletId = (await walletResponse.json()).data.id
  await expect
    .poll(async () => (await (await team.owner.api.get(`/api/v1/wallets/${walletId}`)).json()).data.status, { timeout: 180_000 })
    .toBe('ready')
  await fundAccount((await (await team.owner.api.get(`/api/v1/wallets/${walletId}`)).json()).data.address, '50')
  const request = await requestPayment(team.operator.api, walletId, RECIPIENT, '20')

  await openWorkspace(page, team.owner.email, team.name)
  await openRequest(page, '20')
  await page.getByRole('button', { name: 'Approve payment' }).click()
  await expect(page.getByText('Payment approved')).toBeVisible()
  await expect
    .poll(async () => (await (await team.owner.api.get(`/api/v1/payment-requests/${request.id}`)).json()).data.txHash ?? '', { timeout: 90_000 })
    .not.toBe('')
})
