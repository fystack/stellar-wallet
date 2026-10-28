import { test, expect, type Page } from '@playwright/test'
import { Keypair } from '@stellar/stellar-sdk'
import {
  RECIPIENT,
  approvalTeam,
  errorCode,
  insertReadyWallet,
  latestEmailText,
  mailpitOnline,
  openRequest,
  openWorkspace,
  paymentRequest,
  personalWorkspaceId,
  requestPayment,
  signIn,
  sql,
} from './helpers.ts'
import { fundKeypair } from './fund.ts'

// Held payments never sign, so these use a wallet row on the funded e2e account and run without MPC nodes.

test.describe.configure({ mode: 'serial', timeout: 300_000 })

const USDC = { asset_code: 'USDC', asset_issuer: 'GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5' }

function heldWallet(team: Awaited<ReturnType<typeof approvalTeam>>) {
  const walletId = insertReadyWallet(team.workspaceId, team.owner.id, 'Held wallet', fundKeypair().publicKey())
  sql(`UPDATE wallets SET balance = '1000' WHERE id = '${walletId}';`)
  return walletId
}

async function openSend(page: Page, walletId: string) {
  await page.getByRole('navigation').getByRole('button', { name: 'Send' }).click()
  await page.getByLabel('From wallet').selectOption(walletId)
  await page.getByRole('textbox', { name: /Recipient/ }).fill(RECIPIENT)
}

test('operators cannot swap or add trustlines and members cannot send', async () => {
  const team = await approvalTeam()
  const walletId = heldWallet(team)

  const swap = await team.operator.api.post(`/api/v1/wallets/${walletId}/swap`, { data: { from: 'XLM', to: `USDC:${USDC.asset_issuer}`, amount: '1', slippage_bps: 100 } })
  expect(await errorCode(swap)).toBe('workspace_owner_required')
  const trustline = await team.operator.api.post(`/api/v1/wallets/${walletId}/trustline`, { data: { code: 'USDC', issuer: USDC.asset_issuer } })
  expect(await errorCode(trustline)).toBe('workspace_owner_required')
  const send = await team.viewer.api.post('/api/v1/transactions', { data: { wallet_id: walletId, to: RECIPIENT, amount: '1' } })
  expect(await errorCode(send)).toBe('send_role_required')
})

test('the last owner cannot step down until ownership is handed over', async () => {
  const team = await approvalTeam()
  const members = `/api/v1/workspaces/${team.workspaceId}/members`

  expect(await errorCode(await team.owner.api.patch(`${members}/${team.owner.id}`, { data: { role: 'member' } }))).toBe('last_owner')
  // With one approval required, stepping down would leave nobody to approve the new owner's payments.
  expect((await team.owner.api.delete(`/api/v1/workspaces/${team.workspaceId}/approval-policy`)).status()).toBe(204)
  expect((await team.owner.api.patch(`${members}/${team.operator.id}`, { data: { role: 'owner' } })).status()).toBe(204)
  expect((await team.owner.api.patch(`${members}/${team.owner.id}`, { data: { role: 'member' } })).status()).toBe(204)

  const policy = await team.owner.api.put(`/api/v1/workspaces/${team.workspaceId}/approval-policy`, {
    data: { thresholdXlm: '5', requiredApprovals: 1, belowThreshold: 'send' },
  })
  expect(await errorCode(policy)).toBe('workspace_owner_required')
})

test('invalid approval policies are rejected', async () => {
  const team = await approvalTeam()
  const put = (data: object) => team.owner.api.put(`/api/v1/workspaces/${team.workspaceId}/approval-policy`, { data })
  const valid = { thresholdXlm: '10', requiredApprovals: 1, belowThreshold: 'send' }

  for (const thresholdXlm of ['-1', '1.12345678', 'abc']) {
    expect(await errorCode(await put({ ...valid, thresholdXlm })), thresholdXlm).toBe('approval_policy_invalid')
  }
  expect(await errorCode(await put({ ...valid, requiredApprovals: 0 }))).toBe('approval_policy_invalid')
  expect(await errorCode(await put({ ...valid, requiredApprovals: 2 }))).toBe('approval_policy_invalid')
  expect(await errorCode(await put({ ...valid, belowThreshold: 'later' }))).toBe('approval_policy_invalid')
  expect(await errorCode(await team.viewer.api.put(`/api/v1/workspaces/${team.workspaceId}/approval-policy`, { data: valid }))).toBe('workspace_owner_required')
})

test('personal workspaces have no approval policy', async ({ page }) => {
  const team = await approvalTeam()
  const personal = await personalWorkspaceId(team.owner.api)
  expect((await (await team.owner.api.get(`/api/v1/workspaces/${personal}/approval-policy`)).json()).data.policy).toBeNull()
  const set = await team.owner.api.put(`/api/v1/workspaces/${personal}/approval-policy`, {
    data: { thresholdXlm: '10', requiredApprovals: 1, belowThreshold: 'send' },
  })
  expect(await errorCode(set)).toBe('personal_workspace')

  await openWorkspace(page, team.owner.email, 'Personal')
  await page.getByRole('navigation').getByRole('button', { name: 'Settings' }).click()
  await expect(page.getByRole('heading', { name: 'Linked accounts' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Payment approvals' })).toBeHidden()
})

test('operators see the policy read-only and owners can turn it off', async ({ page }) => {
  const team = await approvalTeam()
  await openWorkspace(page, team.operator.email, team.name)
  await page.getByRole('navigation').getByRole('button', { name: 'Settings' }).click()
  const policy = page.getByRole('region', { name: 'Payment approvals' })
  await expect(policy.getByRole('status')).toContainText('Payments over 10 XLM')
  await expect(policy.getByRole('button', { name: 'Save policy' })).toBeHidden()
  await expect(policy.getByLabel('Threshold (XLM)')).toBeHidden()

  await page.getByRole('button', { name: `Sign out ${team.operator.email}` }).click()
  await openWorkspace(page, team.owner.email, team.name)
  await page.getByRole('navigation').getByRole('button', { name: 'Settings' }).click()
  await policy.getByRole('button', { name: 'Turn off approvals' }).click()
  await expect(policy.getByRole('status')).toContainText('Off.')
  expect((await (await team.owner.api.get(`/api/v1/workspaces/${team.workspaceId}/approval-policy`)).json()).data.policy).toBeNull()
})

test('the send form explains when a payment needs approval and holds it', async ({ page }) => {
  const team = await approvalTeam()
  const walletId = heldWallet(team)
  await openWorkspace(page, team.operator.email, team.name)
  await openSend(page, walletId)

  await page.getByLabel('Amount', { exact: true }).fill('10')
  await expect(page.getByText(/This payment needs 1 approval/)).toBeHidden()
  await expect(page.getByRole('button', { name: 'Review & sign' })).toBeVisible()

  await page.getByLabel('Amount', { exact: true }).fill('50')
  await expect(page.getByText('This payment needs 1 approval from another owner or operator before it is signed.')).toBeVisible()
  await page.getByRole('button', { name: 'Review & request approval' }).click()
  await page.getByRole('button', { name: 'Confirm & sign' }).click()

  await expect(page.getByText('Sent for approval. It needs 1 approval.')).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Payment request' })).toBeVisible()
  await expect(page.getByText('0 of 1')).toBeVisible()
})

test('small payments are held when the policy requires approval below the threshold', async () => {
  const team = await approvalTeam()
  const walletId = heldWallet(team)
  const update = await team.owner.api.put(`/api/v1/workspaces/${team.workspaceId}/approval-policy`, {
    data: { thresholdXlm: '10', requiredApprovals: 1, belowThreshold: 'approve' },
  })
  expect(update.status()).toBe(200)
  await requestPayment(team.operator.api, walletId, RECIPIENT, '1')
})

test('non-XLM payments always need approval', async () => {
  const team = await approvalTeam({ thresholdXlm: '1000000' })
  await requestPayment(team.operator.api, heldWallet(team), RECIPIENT, '0.01', USDC)
})

test('approvers see a new request live and the approval email links to it', async ({ page }) => {
  const team = await approvalTeam()
  const walletId = heldWallet(team)
  await openWorkspace(page, team.owner.email, team.name)
  await page.getByRole('button', { name: /^Approvals/ }).click()
  await expect(page.getByText('Nothing to approve')).toBeVisible()

  const request = await requestPayment(team.operator.api, walletId, RECIPIENT, '41')

  await expect(page.getByRole('button', { name: 'Approvals, 1 pending' })).toBeVisible()
  await expect(page.getByRole('button', { name: /Review 41 XLM/ })).toBeVisible()
  if (await mailpitOnline()) expect(await latestEmailText(team.owner.email)).toContain(`#/approvals/${request.id}`)
})

test('an approval link opens the request in its workspace after sign-in', async ({ page }) => {
  const team = await approvalTeam()
  const request = await requestPayment(team.operator.api, heldWallet(team), RECIPIENT, '42')

  await page.goto(`/#/approvals/${request.id}`)
  await signIn(page, team.owner.email)

  await expect(page.getByRole('heading', { name: 'Payment request' })).toBeVisible()
  await expect(page.getByText('42 XLM').first()).toBeVisible()
  await expect(page.getByRole('button', { name: new RegExp(`${team.name}\\s+Owner`) })).toBeVisible()
})

test('a failed workspace switch from an approval link keeps the picker open', async ({ page }) => {
  const team = await approvalTeam()
  const request = await requestPayment(team.operator.api, heldWallet(team), RECIPIENT, '43')
  await page.route('**/api/v1/workspaces/*/switch', (route) =>
    route.fulfill({ status: 500, json: { error: { code: 'internal_error', message: 'An unexpected error occurred.' } } }),
  )

  await page.goto(`/#/approvals/${request.id}`)
  await signIn(page, team.owner.email)

  await expect(page.getByText('An unexpected error occurred.')).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Payment request' })).toBeHidden()
})

test('only other owners and operators can vote, once each', async () => {
  const team = await approvalTeam({ extraOperators: 1, requiredApprovals: 2 })
  const request = await requestPayment(team.operator.api, heldWallet(team), RECIPIENT, '44')
  const approve = (api: typeof team.owner.api) => api.post(`/api/v1/payment-requests/${request.id}/approve`)

  expect(await errorCode(await approve(team.operator.api))).toBe('approval_self_not_allowed')
  expect(await errorCode(await approve(team.viewer.api))).toBe('approval_role_required')

  const statuses = (await Promise.all([approve(team.owner.api), approve(team.owner.api)])).map((response) => response.status())
  expect(statuses.sort()).toEqual([200, 409])
  const stored = await paymentRequest(team.owner.api, request.id)
  expect(stored.status).toBe('pending')
  expect(stored.votes).toHaveLength(1)
})

test('owners can cancel any request, members cannot, and closed requests take no votes', async () => {
  const team = await approvalTeam()
  const request = await requestPayment(team.operator.api, heldWallet(team), RECIPIENT, '45')

  expect(await errorCode(await team.viewer.api.post(`/api/v1/payment-requests/${request.id}/cancel`))).toBe('payment_request_cancel_forbidden')
  const cancelled = await team.owner.api.post(`/api/v1/payment-requests/${request.id}/cancel`)
  expect((await cancelled.json()).data.status).toBe('cancelled')
  expect(await errorCode(await team.owner.api.post(`/api/v1/payment-requests/${request.id}/approve`))).toBe('payment_request_closed')
})

test('demoting an approver below the required approvals is refused', async () => {
  const team = await approvalTeam({ extraOperators: 1, requiredApprovals: 2 })
  const demote = await team.owner.api.patch(`/api/v1/workspaces/${team.workspaceId}/members/${team.operators[0].id}`, { data: { role: 'member' } })
  expect(await errorCode(demote)).toBe('approval_policy_conflict')
})

test('a demoted approver no longer counts towards a pending request', async () => {
  const team = await approvalTeam({ extraOperators: 2, requiredApprovals: 2 })
  const request = await requestPayment(team.operator.api, heldWallet(team), RECIPIENT, '46')
  const approved = await team.operators[0].api.post(`/api/v1/payment-requests/${request.id}/approve`)
  expect(approved.status(), await approved.text()).toBe(200)

  const demote = await team.owner.api.patch(`/api/v1/workspaces/${team.workspaceId}/members/${team.operators[0].id}`, { data: { role: 'member' } })
  expect(demote.status(), await demote.text()).toBe(204)

  const stored = await paymentRequest(team.owner.api, request.id)
  expect(stored.status).toBe('pending')
  expect(stored.votes).toEqual([expect.objectContaining({ userId: team.operators[0].id, counted: false })])
})

test('an expired request cannot be approved', async ({ page }) => {
  const team = await approvalTeam()
  const request = await requestPayment(team.operator.api, heldWallet(team), RECIPIENT, '48')
  sql(`UPDATE payment_requests SET expires_at = strftime('%s','now') - 60 WHERE id = '${request.id}';`)

  expect((await paymentRequest(team.owner.api, request.id)).status).toBe('expired')
  expect(await errorCode(await team.owner.api.post(`/api/v1/payment-requests/${request.id}/approve`))).toBe('payment_request_closed')

  await openWorkspace(page, team.owner.email, team.name)
  await page.getByRole('button', { name: /^Approvals/ }).click()
  await expect(page.getByText('Nothing to approve')).toBeVisible()
  await page.getByRole('tab', { name: 'History' }).click()
  await expect(page.getByRole('button', { name: /Review 48 XLM/ })).toContainText('Expired')
})

test('a payment that can no longer be built fails with a reason', async () => {
  const team = await approvalTeam()
  const walletId = heldWallet(team)
  const request = await requestPayment(team.operator.api, walletId, RECIPIENT, '49')
  sql(`UPDATE wallets SET address = '${Keypair.random().publicKey()}' WHERE id = '${walletId}';`)

  const approved = await team.owner.api.post(`/api/v1/payment-requests/${request.id}/approve`)
  expect(approved.status(), await approved.text()).toBe(200)
  await expect.poll(async () => (await paymentRequest(team.owner.api, request.id)).status, { timeout: 60_000 }).toBe('failed')
  expect((await paymentRequest(team.owner.api, request.id)).reason).toBeTruthy()
})

test('request history pages by cursor without gaps or repeats', async () => {
  const team = await approvalTeam()
  const walletId = heldWallet(team)
  const created: string[] = []
  for (let index = 0; index < 7; index++) {
    const request = await requestPayment(team.operator.api, walletId, RECIPIENT, `${50 + index}`)
    await team.operator.api.post(`/api/v1/payment-requests/${request.id}/cancel`)
    created.push(request.id)
  }

  const seen: string[] = []
  let cursor = ''
  do {
    const page = await (await team.owner.api.get(`/api/v1/workspaces/${team.workspaceId}/payment-requests?status=history&limit=3&cursor=${cursor}`)).json()
    expect(page.data.length).toBeLessThanOrEqual(3)
    seen.push(...page.data.map((request: { id: string }) => request.id))
    cursor = page.meta?.next_cursor ?? ''
  } while (cursor)

  expect(new Set(seen).size).toBe(seen.length)
  expect(seen.sort()).toEqual(created.sort())
})

test('Escape closes the payment request dialog', async ({ page }) => {
  const team = await approvalTeam()
  await requestPayment(team.operator.api, heldWallet(team), RECIPIENT, '60')
  await openWorkspace(page, team.owner.email, team.name)
  await openRequest(page, '60')
  await page.keyboard.press('Escape')
  await expect(page.getByRole('heading', { name: 'Payment request' })).toBeHidden()
})
