import { test, expect, type Page } from '@playwright/test'
import { fundAccount } from './fund.ts'

// Full user journey against the live stack:
// register → create wallet (real MPC keygen) → fund (from the e2e fund wallet) →
// create a second wallet → transfer XLM A→B → transaction confirms on-chain.

// Serial, with long waits: keygens queue on the shared local cluster.
test.describe.configure({ mode: 'serial', timeout: 300_000 })

// About 10 s per keygen, so a new wallet can wait behind other tests' wallets.
const KEYGEN_WAIT = 180_000

// Distinctive, so the balance check can't match other text on the page.
const FUNDED = '123.45'

async function register(page: Page): Promise<string> {
  const email = `e2e_${Date.now()}_${Math.floor(Math.random() * 1e4)}@test.io`
  await page.goto('/')
  await page.getByRole('button', { name: 'Create one' }).click()
  await page.getByPlaceholder('you@example.com').fill(email)
  await page.getByLabel('Password', { exact: true }).fill('secret123')
  await page.getByLabel('Confirm password').fill('secret123')
  await page.getByRole('button', { name: 'Create account' }).click()
  await expect(page.getByText('Your keys, split — never whole')).toBeVisible()
  return email
}

// Creates a wallet from the modal and opens it. Returns its address.
async function createWalletAndOpen(page: Page, first: boolean): Promise<string> {
  await page
    .getByRole('button', {
      name: first ? 'Create your first wallet' : '+ New wallet',
    })
    .click()
  // Stellar is selected by default in the modal.
  await page.getByRole('button', { name: 'Create wallet' }).click()
  // Keygen ceremony → wallet ready.
  const open = page.getByRole('button', { name: 'Open wallet' })
  await expect(open).toBeVisible({ timeout: KEYGEN_WAIT })
  await open.click()
  const label = await page.getByRole('button', { name: /^Copy address G/ }).first().getAttribute('aria-label')
  expect(label).toBeTruthy()
  return (label as string).replace('Copy address ', '')
}

test('register → create → fund → transfer → confirmed', async ({ page }) => {
  await register(page)

  // Wallet A + fund it.
  await fundAccount(await createWalletAndOpen(page, true), FUNDED)
  await expect(page.getByText(FUNDED).first()).toBeVisible({ timeout: 40_000 })

  // Wallet B (recipient) — grab its address.
  await page.getByRole('button', { name: 'Back to wallets' }).click()
  const dest = await createWalletAndOpen(page, false)
  await page.getByRole('button', { name: 'Back to wallets' }).click()

  // Open the funded wallet A and send to B.
  await page
    .getByRole('main')
    .locator('button', { hasText: FUNDED })
    .first()
    .click()
  await page.getByRole('main').getByRole('button', { name: 'Send' }).click()

  await page.getByRole('textbox', { name: /Recipient/ }).fill(dest)
  await page.getByLabel('Amount', { exact: true }).fill('5')
  await page.getByRole('button', { name: 'Review & sign' }).click()
  await page.getByRole('button', { name: 'Confirm & sign' }).click()

  // Transaction detail opens automatically and should reach Confirmed.
  await expect(page.getByRole('heading', { name: 'Transaction', exact: true })).toBeVisible()
  await expect(page.getByText('Confirmed').first()).toBeVisible({
    timeout: 90_000,
  })
})

test('self-send shows a warning', async ({ page }) => {
  await register(page)
  const addr = await createWalletAndOpen(page, true)
  await fundAccount(addr, FUNDED)
  await expect(page.getByText(FUNDED).first()).toBeVisible({ timeout: 40_000 })

  await page.getByRole('main').getByRole('button', { name: 'Send' }).click()
  await page.getByRole('textbox', { name: /Recipient/ }).fill(addr)
  await expect(page.getByText(/sender's own address/)).toBeVisible()
})

test('send validation: bad address + insufficient balance block submit', async ({
  page,
}) => {
  await register(page)
  await createWalletAndOpen(page, true) // unfunded wallet (balance 0)
  await page.getByRole('main').getByRole('button', { name: 'Send' }).click()

  const review = page.getByRole('button', { name: 'Review & sign' })

  // Invalid address → error + disabled submit.
  await page.getByRole('textbox', { name: /Recipient/ }).fill('not-a-valid-address')
  await expect(page.getByText(/valid Stellar address/)).toBeVisible()

  // Valid address but amount over (empty) balance → insufficient + disabled.
  await page
    .getByRole('textbox', { name: /Recipient/ })
    .fill('GAP5J5SWHUGNBXI5BALRU4HITVMWTAO2A52I6JNRWLYFEQ2W4YSBWEVJ')
  await page.getByLabel('Amount', { exact: true }).fill('5')
  await expect(page.getByText(/Insufficient balance/)).toBeVisible()
  await expect(review).toBeDisabled()
})

test('receive modal stays centered and listens for payments', async ({ page }) => {
  await register(page)
  const addr = await createWalletAndOpen(page, true)
  const viewportWidth = 639 // one pixel below Tailwind's sm breakpoint
  await page.setViewportSize({ width: viewportWidth, height: 900 })
  await page.getByRole('button', { name: 'Receive' }).click()
  const heading = page.getByRole('heading', { name: 'Receive' })
  await expect(heading).toBeVisible()

  const modal = heading.locator('../..')
  const box = await modal.boundingBox()
  expect(box).not.toBeNull()
  expect(
    Math.abs(box!.x + box!.width / 2 - viewportWidth / 2),
  ).toBeLessThanOrEqual(1)

  await expect(page.getByText(addr).first()).toBeVisible()
  await expect(page.getByText(/Listening for incoming/)).toBeVisible()
})

test('delete wallet returns to the empty state', async ({ page }) => {
  await register(page)
  await createWalletAndOpen(page, true)
  await page.getByRole('button', { name: 'Delete' }).click()
  await page
    .getByRole('button', { name: 'Delete', exact: true })
    .last()
    .click()
  await expect(
    page.getByRole('button', { name: 'Create your first wallet' }),
  ).toBeVisible()
})

test('hide balances persists across reload', async ({ page }) => {
  await register(page)
  await createWalletAndOpen(page, true)
  await page.getByRole('button', { name: 'Back to wallets' }).click()

  await page.getByRole('button', { name: 'Hide balances' }).click()
  await expect(page.getByText('••••').first()).toBeVisible()

  await page.reload()
  await expect(page.getByText('••••').first()).toBeVisible()
})

test('wallet detail survives reload (hash routing)', async ({ page }) => {
  await register(page)
  const addr = await createWalletAndOpen(page, true)
  await expect(page).toHaveURL(/#\/wallet\//)
  await page.reload()
  // Still on the detail page after reload.
  await expect(page.getByText(addr)).toBeVisible({ timeout: 40_000 })
  await expect(page.getByRole('button', { name: 'Receive' })).toBeVisible()
})

test('logout then login again', async ({ page }) => {
  const email = await register(page)
  await page.getByRole('button', { name: /Sign out/ }).click()
  await expect(page.getByText('Welcome back')).toBeVisible()

  await page.getByPlaceholder('you@example.com').fill(email)
  await page.getByLabel('Password', { exact: true }).fill('secret123')
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await expect(page.getByText('Your keys, split — never whole')).toBeVisible()
})
