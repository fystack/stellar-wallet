import { test, expect, request, type Page } from '@playwright/test'
import { API, PASSWORD, createWorkspace, registerAccount, signIn, sql, uniqueEmail } from './helpers.ts'

// Google and GitHub consent can't run headless, so their pages are intercepted and answer the callback themselves.

const PROVIDER_PAGES = /^https:\/\/(accounts\.google\.com|github\.com\/login\/oauth)\//

type Authorization = { url: URL; state: string }

// Captures the provider authorization request; with `deny`, the provider page sends the user back as if consent was refused.
async function interceptProvider(page: Page, { deny }: { deny: boolean }) {
  const seen: Authorization[] = []
  // Routes don't see redirect hops, so the backend's 302 to the provider becomes a page navigation the route can catch.
  await page.route('**/api/v1/auth/*/start', async (route) => {
    const response = await route.fetch({ maxRedirects: 0 })
    const location = response.headers()['location']
    await route.fulfill({
      response,
      status: 200,
      headers: { ...response.headers(), 'content-type': 'text/html' },
      body: `<script>location.replace(${JSON.stringify(location)})</script>`,
    })
  })
  await page.route(PROVIDER_PAGES, async (route) => {
    const url = new URL(route.request().url())
    const state = url.searchParams.get('state') ?? ''
    seen.push({ url, state })
    const callback = `${url.searchParams.get('redirect_uri')}?error=access_denied&state=${encodeURIComponent(state)}`
    await route.fulfill({
      contentType: 'text/html',
      body: deny ? `<script>location.replace(${JSON.stringify(callback)})</script>` : '<p>provider</p>',
    })
  })
  return seen
}

function authErrorRedirect(page: Page) {
  return page.waitForRequest((sent) => sent.url().includes('auth_error='))
}

function authError(url: string) {
  return new URL(url).searchParams.get('auth_error')
}

for (const provider of [
  { id: 'google', name: 'Google', scope: 'openid email profile' },
  { id: 'github', name: 'GitHub', scope: 'read:user user:email' },
]) {
  test(`${provider.name} sign-in uses PKCE and a refused consent returns a friendly message`, async ({ page }) => {
    const seen = await interceptProvider(page, { deny: true })
    await page.goto('/')
    const redirected = authErrorRedirect(page)
    await page.getByRole('button', { name: `Continue with ${provider.name}` }).click()

    expect(authError((await redirected).url())).toBe('oauth_cancelled')
    const authorization = seen[0].url
    expect(authorization.searchParams.get('scope')).toBe(provider.scope)
    expect(authorization.searchParams.get('code_challenge_method')).toBe('S256')
    expect(authorization.searchParams.get('code_challenge')).toBeTruthy()
    expect(authorization.searchParams.get('redirect_uri')).toBe(`${API}/api/v1/auth/${provider.id}/callback`)
    await expect(page.getByRole('alert')).toContainText('Social sign-in was cancelled')
  })
}

test('a replayed OAuth state is rejected', async ({ page }) => {
  const seen = await interceptProvider(page, { deny: false })
  await page.goto('/')
  await page.getByRole('button', { name: 'Continue with Google' }).click()
  await expect.poll(() => seen.length).toBe(1)
  const callback = `${API}/api/v1/auth/google/callback?error=access_denied&state=${encodeURIComponent(seen[0].state)}`

  let redirected = authErrorRedirect(page)
  await page.goto(callback)
  expect(authError((await redirected).url())).toBe('oauth_cancelled')

  redirected = authErrorRedirect(page)
  await page.goto(callback)
  expect(authError((await redirected).url())).toBe('invalid_oauth_state')
  await expect(page.getByRole('alert')).toContainText('expired or was started in another browser')
})

test('an OAuth state started in another browser is rejected', async ({ page, browser }) => {
  const seen = await interceptProvider(page, { deny: false })
  await page.goto('/')
  await page.getByRole('button', { name: 'Continue with GitHub' }).click()
  await expect.poll(() => seen.length).toBe(1)

  const other = await browser.newPage()
  const redirected = authErrorRedirect(other)
  await other.goto(`${API}/api/v1/auth/github/callback?code=stolen&state=${encodeURIComponent(seen[0].state)}`)
  expect(authError((await redirected).url())).toBe('invalid_oauth_state')
  await other.close()
})

test('linking from a fresh session goes straight to the provider', async ({ page }) => {
  const account = await registerAccount(uniqueEmail('link_fresh'))
  const seen = await interceptProvider(page, { deny: true })
  await signIn(page, account.email)
  await page.getByRole('navigation').getByRole('button', { name: 'Settings' }).click()

  const redirected = authErrorRedirect(page)
  await page.getByRole('region', { name: 'Linked accounts' }).getByRole('button', { name: 'Connect' }).first().click()

  expect(authError((await redirected).url())).toBe('oauth_cancelled')
  expect(seen[0].url.searchParams.get('state')).toBeTruthy()
  await expect(page.getByRole('button', { name: `Sign out ${account.email}` })).toBeVisible()
})

test('linking more than ten minutes after sign-in asks for the password first', async ({ page }) => {
  const account = await registerAccount(uniqueEmail('link_stale'))
  const seen = await interceptProvider(page, { deny: false })
  await signIn(page, account.email)
  await expect(page.getByRole('button', { name: `Sign out ${account.email}` })).toBeVisible()
  sql(`UPDATE auth_sessions SET created_at = created_at - 700 WHERE user_id = '${account.id}';`)

  await page.getByRole('navigation').getByRole('button', { name: 'Settings' }).click()
  const linked = page.getByRole('region', { name: 'Linked accounts' })
  await linked.getByRole('button', { name: 'Connect' }).first().click()
  await linked.getByLabel('Confirm your password to continue').fill(PASSWORD)
  await linked.getByRole('button', { name: 'Confirm and connect' }).click()

  await expect.poll(() => seen.length).toBe(1)
})

test('signing out during linking invalidates the link', async ({ page }) => {
  const account = await registerAccount(uniqueEmail('link_logout'))
  const seen = await interceptProvider(page, { deny: false })
  await signIn(page, account.email)
  await page.getByRole('navigation').getByRole('button', { name: 'Settings' }).click()
  await page.getByRole('region', { name: 'Linked accounts' }).getByRole('button', { name: 'Connect' }).first().click()
  await expect.poll(() => seen.length).toBe(1)

  await page.goto('/')
  await page.getByRole('button', { name: `Sign out ${account.email}` }).click()
  await signIn(page, account.email)
  await expect(page.getByRole('button', { name: `Sign out ${account.email}` })).toBeVisible()

  const redirected = authErrorRedirect(page)
  await page.goto(`${seen[0].url.searchParams.get('redirect_uri')}?error=access_denied&state=${encodeURIComponent(seen[0].state)}`)
  expect(authError((await redirected).url())).toBe('invalid_link_session')
})

test('signing out on one device keeps the other device signed in', async () => {
  const account = await registerAccount(uniqueEmail('two_devices'))
  const second = await request.newContext({ baseURL: API, extraHTTPHeaders: { 'X-CSRF-Protection': '1' } })
  const login = await second.post('/api/v1/auth/login', { data: { email: account.email, password: PASSWORD } })
  expect(login.status(), await login.text()).toBe(200)

  expect((await account.api.post('/api/v1/auth/logout')).status()).toBe(204)
  expect((await account.api.get('/api/v1/auth/me')).status()).toBe(401)
  expect((await second.get('/api/v1/auth/me')).status()).toBe(200)
  await second.dispose()
})

test('signing in again keeps the active workspace', async () => {
  const account = await registerAccount(uniqueEmail('resignin'))
  const workspaceId = await createWorkspace(account.api, 'Resign team')
  const login = await account.api.post('/api/v1/auth/login', { data: { email: account.email, password: PASSWORD } })
  expect(login.status(), await login.text()).toBe(200)
  expect((await (await account.api.get('/api/v1/workspaces')).json()).data.activeId).toBe(workspaceId)
})

test('providers without credentials are unavailable on the login page', async ({ page }) => {
  await page.route('**/api/v1/auth/providers', (route) =>
    route.fulfill({ json: { data: [{ id: 'google', enabled: false }, { id: 'github', enabled: false }] } }),
  )
  await page.goto('/')
  await expect(page.getByRole('button', { name: 'Continue with Google' })).toBeDisabled()
  await expect(page.getByRole('button', { name: 'Continue with GitHub' })).toBeDisabled()
  await expect(page.getByText('Social sign-in is not configured on this deployment.')).toBeVisible()
})
