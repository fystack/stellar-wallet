import { test, expect, type APIRequestContext } from '@playwright/test'
import { Horizon } from '@stellar/stellar-sdk'
import { RECIPIENT, approvalTeam, clusterOnline, openRequest, openWorkspace, paymentRequest, requestPayment } from './helpers.ts'
import { HORIZON, fundAccount } from './fund.ts'

// Real MPC signing and testnet broadcast; skipped when the cluster is offline.

test.describe.configure({ mode: 'serial', timeout: 300_000 })

const USDC_ISSUER = 'GBBD47IF6LWK7P7MDEVSCWR7DPUWV3NY3DTQEVFL4NAT4AQH3ZLLFLA5'

let team: Awaited<ReturnType<typeof approvalTeam>>
let walletId = ''

test.beforeAll(async () => {
  test.setTimeout(300_000)
  if (!(await clusterOnline())) return
  team = await approvalTeam({ extraOperators: 1 })
  const wallets: { id: string; address: string; status: string }[] = (await (await team.owner.api.get(`/api/v1/wallets?workspace_id=${team.workspaceId}`)).json()).data
  const wallet = wallets.find((entry) => entry.status === 'ready')!
  walletId = wallet.id
  await fundAccount(wallet.address, '120')
})

test.beforeEach(async () => {
  test.skip(!walletId, 'needs all MPC nodes online for keygen and signing')
})

async function confirmed(api: APIRequestContext, transactionId: string) {
  await expect
    .poll(async () => (await (await api.get(`/api/v1/transactions/${transactionId}`)).json()).data.status, { timeout: 120_000 })
    .toBe('confirmed')
  return (await (await api.get(`/api/v1/transactions/${transactionId}`)).json()).data as { txHash: string }
}

async function send(api: APIRequestContext, data: Record<string, string>) {
  const response = await api.post('/api/v1/transactions', { data: { wallet_id: walletId, to: RECIPIENT, ...data } })
  expect(response.status(), await response.text()).toBe(201)
  return (await response.json()).data.transaction as { id: string }
}

test('operator payments at or below the threshold send immediately', async () => {
  await confirmed(team.operator.api, (await send(team.operator.api, { amount: '10' })).id)
})

test('a payment approved in the app is broadcast and the open request links to it', async ({ page }) => {
  const request = await requestPayment(team.operator.api, walletId, RECIPIENT, '20')
  await openWorkspace(page, team.owner.email, team.name)
  await openRequest(page, '20')
  await page.getByRole('button', { name: 'Approve payment' }).click()

  await expect.poll(async () => (await paymentRequest(team.owner.api, request.id)).status, { timeout: 120_000 }).toBe('executed')
  const { txHash } = await paymentRequest(team.owner.api, request.id)
  await expect(page.getByRole('link', { name: txHash })).toHaveAttribute('href', new RegExp(`/transactions/${txHash}$`))
  await expect(page.getByText('1 of 1')).toBeVisible()
})

test('two approvals at the same moment send the payment once', async () => {
  const amount = '12.3456789'
  const request = await requestPayment(team.operator.api, walletId, RECIPIENT, amount)
  const approve = (api: APIRequestContext) => api.post(`/api/v1/payment-requests/${request.id}/approve`)
  await Promise.all([approve(team.owner.api), approve(team.operators[0].api)])

  await expect.poll(async () => (await paymentRequest(team.owner.api, request.id)).status, { timeout: 120_000 }).toBe('executed')
  // The next test sends from the same wallet; overlapping sends reuse a sequence number and fail with tx_bad_seq.
  await confirmed(team.owner.api, (await paymentRequest(team.owner.api, request.id)).transactionId)
  const transactions: { amount: string }[] = (await (await team.owner.api.get(`/api/v1/wallets/${walletId}/transactions?limit=100`)).json()).data
  expect(transactions.filter((transaction) => transaction.amount === amount)).toHaveLength(1)
})

test('a 7-decimal amount and a memo land on chain exactly', async () => {
  const memo = `e2e ${Date.now()}`
  const { txHash } = await confirmed(team.owner.api, (await send(team.owner.api, { amount: '0.0000001', memo, memo_type: 'text' })).id)

  const server = new Horizon.Server(HORIZON)
  const transaction = await server.transactions().transaction(txHash).call()
  expect(transaction.memo).toBe(memo)
  const operations = await server.operations().forTransaction(txHash).call()
  expect(operations.records[0]).toMatchObject({ type: 'payment', amount: '0.0000001' })
})

test('the owner adds a trustline and swaps XLM into it', async () => {
  const trustline = await team.owner.api.post(`/api/v1/wallets/${walletId}/trustline`, { data: { code: 'USDC', issuer: USDC_ISSUER } })
  expect(trustline.status(), await trustline.text()).toBe(201)
  await confirmed(team.owner.api, (await trustline.json()).data.id)

  const to = `USDC:${USDC_ISSUER}`
  const quote = await team.owner.api.get(`/api/v1/wallets/${walletId}/swap/quote?from=XLM&to=${encodeURIComponent(to)}&amount=1`)
  expect(quote.status(), await quote.text()).toBe(200)
  const swap = await team.owner.api.post(`/api/v1/wallets/${walletId}/swap`, { data: { from: 'XLM', to, amount: '1', slippage_bps: 300 } })
  expect(swap.status(), await swap.text()).toBe(201)
  await confirmed(team.owner.api, (await swap.json()).data.transaction.id)
})
