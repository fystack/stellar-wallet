import { test, expect } from '@playwright/test'
import {
  acceptByLink,
  markGoogleVerified,
  memberSwitcher,
  registerAccount,
  signIn,
  teamWithInvite,
  uniqueEmail,
} from './helpers.ts'

// In-app invitations, listed only for Google/GitHub-verified emails.

test('verified invitee lands on the Invitations tab after sign-in and accepts', async ({ page }) => {
  const invitee = await registerAccount(uniqueEmail('inbox'))
  markGoogleVerified(invitee)
  const team = await teamWithInvite(invitee.email)

  await signIn(page, invitee.email)

  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeVisible()
  await expect(page.getByRole('tab', { name: /Invitations/ })).toHaveAttribute('aria-selected', 'true')
  const row = page.getByRole('tabpanel').getByRole('listitem').filter({ hasText: team.name })
  await expect(row).toContainText(/From e2e_owner_/)

  await row.getByRole('button', { name: `Accept invitation to ${team.name}` }).click()
  await expect(page.getByText(`You joined ${team.name}`)).toBeVisible()
  await expect(memberSwitcher(page, team.name)).toBeVisible()

  await page.reload()
  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeHidden()
  await expect(memberSwitcher(page, team.name)).toBeVisible()
})

test('declining removes the invitation for both sides', async ({ page }) => {
  const invitee = await registerAccount(uniqueEmail('decline'))
  markGoogleVerified(invitee)
  const kept = await teamWithInvite(invitee.email)
  const declined = await teamWithInvite(invitee.email)

  await signIn(page, invitee.email)
  const panel = page.getByRole('tabpanel')
  await expect(panel.getByRole('listitem')).toHaveCount(2)
  await panel.getByRole('button', { name: `Decline invitation to ${declined.name}` }).click()

  await expect(page.getByText('Invitation declined')).toBeVisible()
  await expect(panel.getByRole('listitem')).toHaveCount(1)
  await expect(panel.getByRole('listitem')).toContainText(kept.name)
  await expect(page.getByRole('tab', { name: /Invitations/ })).toContainText('1')

  const members = await declined.owner.get(`/api/v1/workspaces/${declined.workspaceId}/members`)
  expect((await members.json()).data.invites).toEqual([])
})

test('declining the last invitation with only a personal workspace goes straight into the app', async ({ page }) => {
  const invitee = await registerAccount(uniqueEmail('decline_last'))
  markGoogleVerified(invitee)
  const team = await teamWithInvite(invitee.email)

  await signIn(page, invitee.email)
  await page.getByRole('button', { name: `Decline invitation to ${team.name}` }).click()

  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeHidden()
  await expect(page.getByRole('button', { name: `Sign out ${invitee.email}` })).toBeVisible()
})

test('unverified password account is pointed to the emailed link', async ({ page }) => {
  const invitee = await registerAccount(uniqueEmail('unverified'))
  const joined = await teamWithInvite(invitee.email)
  await acceptByLink(invitee.api, joined.invite.token)
  await teamWithInvite(invitee.email)

  await signIn(page, invitee.email)

  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeVisible()
  await expect(page.getByRole('tab', { name: 'Workspaces' })).toHaveAttribute('aria-selected', 'true')
  await page.getByRole('tab', { name: 'Invitations' }).click()
  await expect(page.getByRole('tabpanel')).toContainText('open the invite link from your email')
})

test('switcher shows a badge for new invitations and accepts from the dropdown', async ({ page }) => {
  const invitee = await registerAccount(uniqueEmail('badge'))
  markGoogleVerified(invitee)
  const first = await teamWithInvite(invitee.email)
  await acceptByLink(invitee.api, first.invite.token)

  await signIn(page, invitee.email)
  await page.getByRole('button', { name: /Personal\s+Personal/ }).click()
  await expect(page.getByRole('button', { name: `Sign out ${invitee.email}` })).toBeVisible()

  const second = await teamWithInvite(invitee.email)
  await page.reload()
  await expect(page.getByRole('heading', { name: 'Choose a workspace' })).toBeHidden()
  await expect(page.getByLabel('1 pending invitation')).toBeVisible()

  await page.getByRole('button', { name: /Personal\s+Personal\s+1/ }).click()
  await page.getByRole('button', { name: `Accept invitation to ${second.name}` }).click()

  await expect(page.getByText(`You joined ${second.name}`)).toBeVisible()
  await expect(memberSwitcher(page, second.name)).toBeVisible()
  await expect(page.getByLabel(/pending invitation/)).toBeHidden()
})
