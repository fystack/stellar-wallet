# Team workspaces and invites

Wallets belong to workspaces. Every account has a **Personal** workspace; teams
create a separate workspace and invite members by email. Key shares stay on the
3 MPC nodes (2-of-3); each member signs in with their own login, and every
member gets their own MPC wallet in the workspace when they join.

## Roles

| Action | Owner | Member |
|---|---|---|
| See workspace wallets and history | ✅ | ✅ |
| Send, swap, add trustlines | ✅ | ❌ until payment approvals (D3) |
| Create or delete wallets | ✅ | ❌ |
| Invite, revoke invites, remove members | ✅ | ❌ |

A workspace always keeps at least one owner. Personal workspaces cannot invite;
create a team workspace instead.

## Invites

- Owners invite from **Settings → Members**. The invite link is single-use and
  expires after 7 days. Re-inviting the same email replaces the previous link.
- The link is emailed when SMTP is configured, and is always shown to the owner
  with a **Copy** button, so invites work without an email provider.
- The invitee opens the link, signs in or registers (email/password, Google or
  GitHub) with the **invited email**, then accepts. Accepting switches them to
  the workspace and starts keygen for their wallet. If the MPC cluster is below
  quorum, they still join and an owner can create the wallet later.

## In-app invitations

Pending invites also appear inside the app, with **Accept** and **Decline**:

- On the **Invitations** tab of the workspace picker. After sign-in the picker
  opens on that tab whenever there are pending invitations.
- In the sidebar workspace switcher, with a count badge.

They are listed only for emails confirmed by Google or GitHub (any linked
identity, not just the account's main email). Email/password sign-up does not
verify the address, so those accounts are told to open the emailed link
instead. Declining revokes the invite; the owner sees it leave Pending invites
and can re-invite.

## Email setup

With `smtp.host` empty and a localhost `frontend_url`, the backend writes each
invite email, link included, to its log:

```
INFO  email not sent (SMTP disabled, local log mailer)  {"to": "teammate@example.com", "subject": "...", "body": "...http://localhost:5173/#/invite/..."}
```

Outside localhost an empty `smtp.host` disables email. To see real emails
locally, run Mailpit from `docker-compose.infra.yaml` and read them at
http://localhost:8025:

```yaml
smtp:
  host: "localhost"
  port: 1025
  from: "Stellar MPC Wallet <no-reply@localhost>"
```

Any SMTP provider works in production (Resend, SES, Postmark, Gmail); set
`username`, `password` and the provider's port.

## Reproduction checks

1. Sign in as the owner. The sidebar switcher shows **Personal**.
2. Open the switcher → create workspace "Treasury". A wallet starts generating
   for the owner in Treasury.
3. Settings → Members → invite `teammate@example.com`. Copy the link (or open it
   from Mailpit).
4. In a private window, open the link. The login page shows who invited you.
   Register or sign in with Google/GitHub as `teammate@example.com`, then accept.
5. The teammate lands in Treasury, sees the owner's wallet and their own new
   wallet, and appears as **Member** in Settings → Members on both sides.
6. The teammate has no **New wallet** or **Delete** buttons, and cannot invite.
7. Opening the same link again shows "already used". Inviting with a different
   signed-in email shows the email-mismatch message.
8. The owner removes the teammate; the teammate's next request falls back to
   their Personal workspace and Treasury wallets are no longer visible.

## API

All routes are under `/api/v1` and use the standard `{data}` / `{error}` envelope.

| Method | Route | Auth | Behavior |
|---|---|---|---|
| GET | `/workspaces` | session | Member workspaces and the session's active id |
| POST | `/workspaces` | session | Create a team workspace, switch to it, provision the owner's wallet |
| POST | `/workspaces/:id/switch` | member | Set the session's active workspace |
| GET | `/workspaces/:id/members` | member | Members and pending invites |
| DELETE | `/workspaces/:id/members/:userId` | owner | Remove a member (`last_owner` protects the final owner) |
| POST | `/workspaces/:id/invites` | owner | Create an invite; returns the link and whether it was emailed |
| DELETE | `/workspaces/:id/invites/:inviteId` | owner | Revoke a pending invite |
| GET | `/invites/:token` | public | Workspace name, inviter, invited email and status |
| POST | `/invites/:token/accept` | session | Join, switch, provision the member's wallet |
| GET | `/invitations` | session | Pending invites for the account's verified emails, and whether any email is verified |
| POST | `/invitations/:id/accept` | session | Accept an invite addressed to a verified email of the account |
| POST | `/invitations/:id/decline` | session | Decline (revoke) such an invite |

Invitations addressed to someone else return `invite_not_found`, so IDs
reveal nothing.

## End-to-end tests

`ui/e2e/invite.spec.ts` (links) and `ui/e2e/invitations.spec.ts` (in-app
invitations) run against a live backend and vite. MPC nodes are optional.
The invitations spec marks test accounts as Google-verified by inserting into
the backend's SQLite file, so it needs the `sqlite3` CLI and `E2E_DB`:

```bash
cd ui
E2E_DB=../backend/wallet.db npx playwright test e2e/invite.spec.ts e2e/invitations.spec.ts
```

Set `auth_rate_limit` high (e.g. 1000) in the backend config used for e2e;
the default of 30 sign-ins per minute per IP is below what the suites need.

Error codes: `workspace_not_found`, `workspace_owner_required`,
`personal_workspace`, `already_member`, `invite_not_found`,
`invite_unavailable`, `invite_email_mismatch`, `member_not_found`,
`last_owner`.
