import { test, expect } from '@playwright/test'
import {
  PASSWORD,
  acceptByLink,
  clusterOnline,
  createWorkspace,
  errorCode,
  latestEmailText,
  mailpitOnline,
  markGoogleVerified,
  memberSwitcher,
  openWorkspace,
  personalWorkspaceId,
  registerAccount,
  sql,
  teamWithInvite,
  uniqueEmail,
} from './helpers.ts'
import { fundKeypair } from './fund.ts'

test('the emailed invite link joins the workspace', async ({ page }) => {
  test.skip(!(await mailpitOnline()), 'needs Mailpit and smtp.host pointing at it')
  const email = uniqueEmail('mailed')
  const team = await teamWithInvite(email)

  const link = (await latestEmailText(email)).match(/https?:\/\/\S+#\/invite\/[\w-]+/)?.[0]
  expect(link).toBeTruthy()
  await page.goto(link!)
  await page.getByLabel('Password', { exact: true }).fill(PASSWORD)
  await page.getByLabel('Confirm password').fill(PASSWORD)
  await page.getByRole('button', { name: 'Create account' }).click()
  await page.getByRole('region', { name: `Join ${team.name}` }).getByRole('button', { name: 'Accept invite' }).click()
  await expect(memberSwitcher(page, team.name)).toBeVisible()
})

test('joining a workspace does not create a wallet', async () => {
  const invitee = await registerAccount(uniqueEmail('nowallet'))
  const team = await teamWithInvite(invitee.email)
  const before = (await (await team.owner.get(`/api/v1/wallets?workspace_id=${team.workspaceId}`)).json()).data.length

  await acceptByLink(invitee.api, team.invite.token)

  expect((await (await team.owner.get(`/api/v1/wallets?workspace_id=${team.workspaceId}`)).json()).data).toHaveLength(before)
  const personal = await personalWorkspaceId(invitee.api)
  expect((await (await invitee.api.get(`/api/v1/wallets?workspace_id=${personal}`)).json()).data).toHaveLength(0)
})

test('a removed member falls back to Personal and loses the team wallets', async ({ page }) => {
  const invitee = await registerAccount(uniqueEmail('removed'))
  const team = await teamWithInvite(invitee.email)
  await acceptByLink(invitee.api, team.invite.token)
  await openWorkspace(page, invitee.email, team.name)
  await expect(memberSwitcher(page, team.name)).toBeVisible()

  expect((await team.owner.delete(`/api/v1/workspaces/${team.workspaceId}/members/${invitee.id}`)).status()).toBe(204)

  await page.reload()
  await expect(page.getByRole('button', { name: /Personal\s+Personal/ })).toBeVisible()
  const wallets = await invitee.api.get(`/api/v1/wallets?workspace_id=${team.workspaceId}`)
  expect(await errorCode(wallets)).toBe('workspace_not_found')
})

test('an expired invite link says it expired', async ({ page }) => {
  const team = await teamWithInvite(uniqueEmail('expired'))
  sql(`UPDATE workspace_invites SET expires_at = strftime('%s','now') - 60 WHERE id = '${team.invite.inviteId}';`)

  await page.goto(team.invite.path)
  await expect(page.getByRole('alert')).toHaveText(/invite has expired/)
  const accept = await (await registerAccount(uniqueEmail('late'))).api.post(`/api/v1/invites/${team.invite.token}/accept`)
  expect(await errorCode(accept)).toBe('invite_unavailable')
})

test('an invitation to a secondary verified email is listed and can be accepted', async () => {
  const account = await registerAccount(uniqueEmail('primary'))
  const secondary = uniqueEmail('secondary')
  markGoogleVerified(account, secondary)
  const team = await teamWithInvite(secondary)

  const listed = (await (await account.api.get('/api/v1/invitations')).json()).data
  const invitation = listed.invites.find((entry: { workspaceId: string }) => entry.workspaceId === team.workspaceId)
  expect(invitation).toBeTruthy()
  const accepted = await account.api.post(`/api/v1/invitations/${invitation.id}/accept`)
  expect(accepted.status(), await accepted.text()).toBe(200)
})

test('custom assets belong to one workspace and only owners change them', async () => {
  const invitee = await registerAccount(uniqueEmail('assets'))
  const team = await teamWithInvite(invitee.email)
  await acceptByLink(invitee.api, team.invite.token)
  const asset = { code: 'E2E', issuer: fundKeypair().publicKey() }

  const added = await team.owner.post(`/api/v1/workspaces/${team.workspaceId}/assets`, { data: asset })
  expect(added.status(), await added.text()).toBe(200)

  const shared = (await (await invitee.api.get(`/api/v1/workspaces/${team.workspaceId}/assets`)).json()).data.assets
  expect(shared).toContainEqual(asset)
  const ownerPersonal = await personalWorkspaceId(team.owner)
  expect((await (await team.owner.get(`/api/v1/workspaces/${ownerPersonal}/assets`)).json()).data.assets).not.toContainEqual(asset)

  const denied = await invitee.api.post(`/api/v1/workspaces/${team.workspaceId}/assets`, { data: { code: 'NOPE', issuer: asset.issuer } })
  expect(await errorCode(denied)).toBe('workspace_owner_required')
  const removed = await invitee.api.delete(`/api/v1/workspaces/${team.workspaceId}/assets?code=E2E&issuer=${asset.issuer}`)
  expect(await errorCode(removed)).toBe('workspace_owner_required')
})

test('a workspace created while the cluster is below quorum still works', async () => {
  test.skip(await clusterOnline(), 'stop at least two MPC nodes to run this')
  const owner = await registerAccount(uniqueEmail('quorum'))
  const workspaceId = await createWorkspace(owner.api, 'Below quorum')

  const members = await owner.api.get(`/api/v1/workspaces/${workspaceId}/members`)
  expect(members.status()).toBe(200)
  const wallet = await owner.api.post('/api/v1/wallets', { data: { workspace_id: workspaceId, name: 'Later', chain: 'stellar' } })
  expect(await errorCode(wallet)).toBe('mpc_unavailable')
})
