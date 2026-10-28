import { test, expect, type Page } from '@playwright/test'
import { RECIPIENT, approvalTeam, insertReadyWallet, openWorkspace, requestPayment, signIn, sql } from './helpers.ts'
import { fundKeypair } from './fund.ts'

test.describe.configure({ mode: 'serial', timeout: 300_000 })

async function teamWithWallet() {
  const team = await approvalTeam()
  const walletId = insertReadyWallet(team.workspaceId, team.owner.id, 'Layout wallet', fundKeypair().publicKey())
  sql(`UPDATE wallets SET balance = '1000' WHERE id = '${walletId}';`)
  return { team, walletId }
}

async function expectNoHorizontalScroll(page: Page, label: string) {
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
  expect(overflow, `${label} overflows by ${overflow}px`).toBeLessThanOrEqual(0)
}

test('every page fits a 360px screen without horizontal scroll', async ({ page }) => {
  const { team, walletId } = await teamWithWallet()
  await requestPayment(team.operator.api, walletId, RECIPIENT, '70')
  await page.setViewportSize({ width: 360, height: 780 })

  await signIn(page, team.owner.email)
  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeVisible()
  await expectNoHorizontalScroll(page, 'workspace picker')
  await page.getByRole('list', { name: 'Your workspaces' }).getByRole('button', { name: new RegExp(team.name) }).click()

  for (const [hash, heading] of [
    ['#/wallets', 'Wallets'],
    ['#/send', 'Send'],
    ['#/swap', 'Swap'],
    ['#/approvals', 'Approvals'],
    ['#/settings', 'Settings'],
  ]) {
    await page.goto(`/${hash}`)
    await expect(page.getByRole('heading', { name: heading, exact: true }).first()).toBeVisible()
    await expectNoHorizontalScroll(page, hash)
  }

  await page.goto(`/#/wallet/${walletId}`)
  await expect(page.getByRole('button', { name: 'Receive' })).toBeVisible()
  await expectNoHorizontalScroll(page, 'wallet detail')

  await page.goto('/#/approvals')
  await page.getByRole('button', { name: /Review 70 XLM/ }).click()
  await expect(page.getByRole('heading', { name: 'Payment request' })).toBeVisible()
  await expectNoHorizontalScroll(page, 'payment request dialog')
})

test('the app works when browser storage is blocked', async ({ page }) => {
  const { team } = await teamWithWallet()
  await page.addInitScript(() => {
    for (const name of ['localStorage', 'sessionStorage']) {
      Object.defineProperty(window, name, {
        get() {
          throw new DOMException('blocked', 'SecurityError')
        },
      })
    }
  })

  await openWorkspace(page, team.owner.email, team.name)
  await page.getByRole('navigation').getByRole('button', { name: 'Settings' }).click()
  await expect(page.getByRole('region', { name: 'Payment approvals' })).toBeVisible()
  await page.getByRole('navigation').getByRole('button', { name: 'Wallets' }).click()
  await page.getByRole('button', { name: 'Hide balances' }).click()
  await expect(page.getByText('••••').first()).toBeVisible()
})

test('opening Send directly picks the first wallet once wallets load', async ({ page }) => {
  const { team } = await teamWithWallet()
  await openWorkspace(page, team.owner.email, team.name)
  await page.goto('/#/send')
  await page.reload()

  const from = page.getByLabel('From wallet')
  await expect(from).toBeVisible()
  await expect(from).not.toHaveValue('')
})

test('wallet history pages by cursor and the list loads older pages', async ({ page }) => {
  const { team, walletId } = await teamWithWallet()
  const rows = Array.from({ length: 60 }, (_, index) => {
    const createdAt = new Date(Date.UTC(2026, 0, 1, 0, index)).toISOString()
    return `('e2e-tx-${walletId}-${index}', '${walletId}', '${team.owner.id}', 'out', '${RECIPIENT}', '${index + 1}', 'XLM', 'confirmed', '${createdAt}')`
  })
  sql(`INSERT INTO transactions (id, wallet_id, user_id, type, counterparty, amount, symbol, status, created_at) VALUES ${rows.join(',')};`)

  const seen: string[] = []
  let cursor = ''
  do {
    const response = await (await team.owner.api.get(`/api/v1/wallets/${walletId}/transactions?limit=25&cursor=${cursor}`)).json()
    expect(response.data.length).toBeLessThanOrEqual(25)
    seen.push(...response.data.map((transaction: { id: string }) => transaction.id))
    cursor = response.meta?.next_cursor ?? ''
  } while (cursor)
  expect(seen).toHaveLength(60)
  expect(new Set(seen).size).toBe(60)

  await openWorkspace(page, team.owner.email, team.name)
  await page.goto(`/#/wallet/${walletId}`)
  const showMore = page.getByRole('button', { name: 'Show more' })
  while (await showMore.isVisible()) await showMore.click()
  await expect(page.getByText('−1 XLM', { exact: true })).toBeVisible()
})
