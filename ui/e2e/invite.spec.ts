import { test, expect, type Page } from '@playwright/test'
import { PASSWORD, invite, memberSwitcher, registerAccount, teamWithInvite, uniqueEmail } from './helpers.ts'

// Invite links against a live backend and vite; MPC nodes are optional.

async function apiSession(email: string) {
  return (await registerAccount(email)).api
}

async function expectInviteLogin(page: Page, workspaceName: string, email: string) {
  await expect(page.getByRole('heading', { level: 1, name: `Join ${workspaceName}` })).toBeVisible()
  await expect(page.getByLabel('Email')).toHaveValue(email)
}

async function acceptAndExpectJoined(page: Page, workspaceName: string) {
  const panel = page.getByRole('region', { name: `Join ${workspaceName}` })
  await expect(panel).toBeVisible()
  await panel.getByRole('button', { name: 'Accept invite' }).click()
  await expect(page.getByText(`You joined ${workspaceName}`)).toBeVisible()
  await expect(panel).toBeHidden()
  await expect(memberSwitcher(page, workspaceName)).toBeVisible()
}

async function signInFromLogin(page: Page, password = PASSWORD) {
  await page.getByLabel('Password', { exact: true }).fill(password)
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
}

test('logged out, new email: invite page opens in sign-up mode, then joins', async ({ page }) => {
  const email = uniqueEmail('newcomer')
  const team = await teamWithInvite(email)

  await page.goto(team.invite.path)
  await expectInviteLogin(page, team.name, email)
  await expect(page.getByRole('button', { name: 'Create account' })).toBeVisible()

  await page.getByLabel('Password', { exact: true }).fill(PASSWORD)
  await page.getByLabel('Confirm password').fill(PASSWORD)
  await page.getByRole('button', { name: 'Create account' }).click()

  await acceptAndExpectJoined(page, team.name)

  // The session remembers the joined workspace, so a reload skips the picker.
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeHidden()
  await expect(memberSwitcher(page, team.name)).toBeVisible()

  await page.getByRole('button', { name: 'Settings' }).click()
  await expect(page.getByRole('heading', { name: `Members of ${team.name}` })).toBeVisible()
  await expect(page.getByText(`${email} (you)`)).toBeVisible()
})

test('logged out, existing password account: invite page opens in sign-in mode', async ({ page }) => {
  const email = uniqueEmail('existing')
  await (await apiSession(email)).dispose()
  const team = await teamWithInvite(email)

  await page.goto(team.invite.path)
  await expectInviteLogin(page, team.name, email)
  await expect(page.getByLabel('Confirm password')).toBeHidden()

  await signInFromLogin(page)
  await acceptAndExpectJoined(page, team.name)
})

test('invite link pasted into an already-open login tab is picked up', async ({ page }) => {
  const email = uniqueEmail('pasted')
  const team = await teamWithInvite(email)

  await page.goto('/')
  await expect(page.getByRole('heading', { level: 1, name: 'Sign in' })).toBeVisible()
  await page.evaluate((path) => {
    window.location.hash = path.slice(1)
  }, team.invite.path)

  await expectInviteLogin(page, team.name, email)
})

test('invitee with several workspaces sees the invite on the picker Invitations tab', async ({ page }) => {
  const email = uniqueEmail('multi')
  const invitee = await apiSession(email)
  const first = await teamWithInvite(email)
  const accepted = await invitee.post(`/api/v1/invites/${first.invite.token}/accept`)
  expect(accepted.status(), await accepted.text()).toBe(200)
  await invitee.dispose()
  const second = await teamWithInvite(email)

  await page.goto(second.invite.path)
  await signInFromLogin(page)

  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeVisible()
  await expect(page.getByRole('tab', { name: /Invitations/ })).toHaveAttribute('aria-selected', 'true')
  await acceptAndExpectJoined(page, second.name)
})

test('signed in as a different account: mismatch is explained and switching keeps the invite', async ({ page }) => {
  const invited = uniqueEmail('invited')
  const team = await teamWithInvite(invited)
  const other = uniqueEmail('other')

  await page.goto('/')
  await page.getByRole('button', { name: 'Create an account' }).click()
  await page.getByLabel('Email').fill(other)
  await page.getByLabel('Password', { exact: true }).fill(PASSWORD)
  await page.getByLabel('Confirm password').fill(PASSWORD)
  await page.getByRole('button', { name: 'Create account' }).click()
  await expect(page.getByRole('button', { name: `Sign out ${other}` })).toBeVisible()

  await page.goto(team.invite.path)
  const panel = page.getByRole('region', { name: `Join ${team.name}` })
  await expect(panel.getByText(`You're signed in as ${other}`)).toBeVisible()
  await expect(panel.getByRole('button', { name: 'Accept invite' })).toBeHidden()

  await panel.getByRole('button', { name: 'Sign out and switch account' }).click()
  await expectInviteLogin(page, team.name, invited)
})

test('revoked invite shows a clear message', async ({ page }) => {
  const team = await teamWithInvite(uniqueEmail('revoked'))
  const revoked = await team.owner.delete(`/api/v1/workspaces/${team.workspaceId}/invites/${team.invite.inviteId}`)
  expect(revoked.status()).toBe(204)

  await page.goto(team.invite.path)
  await expect(page.getByRole('alert')).toHaveText(/invite was revoked/)
})

test('re-inviting the same email replaces the old link', async ({ page }) => {
  const email = uniqueEmail('replaced')
  const team = await teamWithInvite(email)
  const newer = await invite(team.owner, team.workspaceId, email)

  await page.goto(team.invite.path)
  await expect(page.getByRole('alert')).toHaveText(/invite was revoked/)

  await page.goto(newer.path)
  await expectInviteLogin(page, team.name, email)
})

test('used invite cannot be accepted again', async ({ page }) => {
  const email = uniqueEmail('used')
  const invitee = await apiSession(email)
  const team = await teamWithInvite(email)
  const accepted = await invitee.post(`/api/v1/invites/${team.invite.token}/accept`)
  expect(accepted.status(), await accepted.text()).toBe(200)
  await invitee.dispose()

  await page.goto(team.invite.path)
  await expect(page.getByRole('alert')).toHaveText(/already been used/)
})

test('garbled invite link shows an invalid-link message', async ({ page }) => {
  await page.goto('/#/invite/not-a-real-token')
  await expect(page.getByRole('alert')).toHaveText(/invite link is invalid/)
  await expect(page.getByRole('heading', { level: 1, name: 'Sign in' })).toBeVisible()
})

test('invitee who dismisses the invite is not asked again', async ({ page }) => {
  const email = uniqueEmail('dismiss')
  const team = await teamWithInvite(email)

  await page.goto(team.invite.path)
  await page.getByLabel('Password', { exact: true }).fill(PASSWORD)
  await page.getByLabel('Confirm password').fill(PASSWORD)
  await page.getByRole('button', { name: 'Create account' }).click()

  const panel = page.getByRole('region', { name: `Join ${team.name}` })
  await panel.getByRole('button', { name: 'Not now' }).click()
  await expect(panel).toBeHidden()
  await page.reload()
  await expect(page.getByRole('button', { name: `Sign out ${email}` })).toBeVisible()
  await expect(panel).toBeHidden()
})

test('members are view-only: no send, swap or wallet management', async ({ page }) => {
  const email = uniqueEmail('viewonly')
  const team = await teamWithInvite(email)

  await page.goto(team.invite.path)
  await page.getByLabel('Password', { exact: true }).fill(PASSWORD)
  await page.getByLabel('Confirm password').fill(PASSWORD)
  await page.getByRole('button', { name: 'Create account' }).click()
  await acceptAndExpectJoined(page, team.name)

  await expect(page.getByRole('button', { name: '+ New wallet' })).toBeHidden()
  await page.getByRole('navigation').getByRole('button', { name: 'Send' }).click()
  await expect(page.getByRole('heading', { name: 'View-only access' })).toBeVisible()
  await page.getByRole('navigation').getByRole('button', { name: 'Swap' }).click()
  await expect(page.getByRole('heading', { name: 'View-only access' })).toBeVisible()

  await page.getByRole('navigation').getByRole('button', { name: 'Settings' }).click()
  await expect(page.getByText('Only workspace owners can add or remove assets.')).toBeVisible()
  await expect(page.getByLabel('Asset code')).toBeHidden()
  await expect(page.getByRole('button', { name: /^Remove / })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Send invite' })).toBeHidden()
  await expect(page.getByRole('combobox', { name: /^Role for / })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Turn on approvals' })).toBeHidden()
  await expect(page.getByRole('switch', { name: /Transaction notifications/ })).toBeEnabled()
})


test('sign out of all devices ends every session of the account', async ({ page }) => {
  const account = await apiSession(uniqueEmail('everywhere'))
  await page.goto('/')
  await page.getByLabel('Email').fill((await (await account.get('/api/v1/auth/me')).json()).data.email)
  await page.getByLabel('Password', { exact: true }).fill(PASSWORD)
  await page.getByRole('button', { name: 'Sign in', exact: true }).click()
  await page.getByRole('navigation').getByRole('button', { name: 'Settings' }).click()
  await page.getByRole('button', { name: 'Sign out all' }).click()

  await expect(page.getByRole('heading', { level: 1, name: 'Sign in' })).toBeVisible()
  expect((await account.get('/api/v1/auth/me')).status()).toBe(401)
})
