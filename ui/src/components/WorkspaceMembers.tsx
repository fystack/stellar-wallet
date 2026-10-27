import { useCallback, useEffect, useState } from 'react'
import { api } from '../api.ts'
import { CopyIcon } from '../icons.tsx'
import { toast } from '../toast.tsx'
import type { Workspace, WorkspaceInvite, WorkspaceMember } from '../types.ts'

function RoleBadge({ role }: { role: WorkspaceMember['role'] }) {
  return (
    <span className={'shrink-0 px-2 py-0.5 text-xs font-semibold ' + (role === 'owner' ? 'bg-brand-soft text-brand' : 'bg-card text-ink-soft')}>
      {role === 'owner' ? 'Owner' : 'Member'}
    </span>
  )
}

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast.success('Invite link copied')
  } catch {
    toast.error('Unable to copy. Select the link and copy it manually.')
  }
}

export default function WorkspaceMembers({
  workspace,
  userEmail,
  onCreateWorkspace,
}: {
  workspace: Workspace | null
  userEmail: string
  onCreateWorkspace: (name: string) => Promise<void>
}) {
  const [members, setMembers] = useState<WorkspaceMember[]>([])
  const [invites, setInvites] = useState<WorkspaceInvite[]>([])
  const [error, setError] = useState('')
  const [email, setEmail] = useState('')
  const [teamName, setTeamName] = useState('')
  const [busy, setBusy] = useState(false)
  const [lastInvite, setLastInvite] = useState<{ email: string; link: string; emailSent: boolean } | null>(null)

  const workspaceId = workspace?.id ?? ''
  const isOwner = workspace?.role === 'owner'

  const load = useCallback(async () => {
    if (!workspaceId) return
    setError('')
    try {
      const result = await api.members(workspaceId)
      setMembers(result.members)
      setInvites(result.invites)
    } catch {
      setError('Unable to load members. Please try again.')
    }
  }, [workspaceId])

  useEffect(() => {
    setLastInvite(null)
    void load()
  }, [load])

  async function run(action: () => Promise<void>) {
    setBusy(true)
    try {
      await action()
    } catch (failure) {
      toast.error(failure instanceof Error ? failure.message : 'Something went wrong.')
    } finally {
      setBusy(false)
    }
  }

  async function sendInvite(event: React.FormEvent) {
    event.preventDefault()
    await run(async () => {
      const result = await api.invite(workspaceId, email.trim())
      setLastInvite({ email: result.invite.email, link: result.link, emailSent: result.emailSent })
      setEmail('')
      await load()
    })
  }

  async function createTeam(event: React.FormEvent) {
    event.preventDefault()
    await run(async () => {
      await onCreateWorkspace(teamName.trim())
      setTeamName('')
    })
  }

  if (!workspace) return null

  if (workspace.personal) {
    return (
      <section className="surface mb-5" aria-labelledby="team-title">
        <h3 id="team-title" className="mb-1 text-[17px] font-bold">Team</h3>
        <p className="mb-[18px] text-sm text-muted">
          Your personal workspace is just for you. Create a team workspace to invite teammates, share wallets and give each member their own login.
        </p>
        <form onSubmit={(event) => void createTeam(event)} className="flex flex-col gap-2 sm:flex-row">
          <input
            className="field-input min-w-0 flex-1"
            placeholder="Team name (e.g. Treasury)"
            aria-label="Team workspace name"
            maxLength={60}
            value={teamName}
            onChange={(event) => setTeamName(event.target.value)}
          />
          <button
            type="submit"
            disabled={busy || !teamName.trim()}
            className="bg-brand px-4 py-2.5 font-semibold text-white transition-colors hover:bg-brand-deep disabled:cursor-not-allowed disabled:opacity-50"
          >
            Create team workspace
          </button>
        </form>
      </section>
    )
  }

  return (
    <section className="surface mb-5" aria-labelledby="team-title">
      <h3 id="team-title" className="mb-1 text-[17px] font-bold">Members of {workspace.name}</h3>
      <p className="mb-[18px] text-sm text-muted">
        Everyone here can see this workspace's wallets and send from them. Each member gets their own wallet when they join.
      </p>

      {error && <div role="alert" className="mb-4 bg-danger-soft px-3 py-2 text-sm text-danger">{error}</div>}

      <div className="flex flex-col">
        {members.map((member) => (
          <div key={member.userId} className="flex items-center gap-3 border-b border-line py-3 last:border-none">
            <span className="grid h-9 w-9 shrink-0 place-items-center bg-[#1b1c1f] text-sm font-semibold uppercase text-white">
              {member.email.charAt(0)}
            </span>
            <div className="min-w-0 flex-1">
              <div className="truncate font-semibold">
                {member.email}
                {member.email === userEmail && <span className="font-normal text-muted"> (you)</span>}
              </div>
              <div className="text-[13px] text-muted">Joined {new Date(member.joinedAt).toLocaleDateString()}</div>
            </div>
            <RoleBadge role={member.role} />
            {isOwner && member.email !== userEmail && (
              <button
                disabled={busy}
                onClick={() => void run(async () => {
                  await api.removeMember(workspaceId, member.userId)
                  toast.success(`${member.email} removed`)
                  await load()
                })}
                className="shrink-0 text-sm font-semibold text-muted hover:text-danger disabled:opacity-50"
              >
                Remove
              </button>
            )}
          </div>
        ))}
      </div>

      {invites.length > 0 && (
        <div className="mt-5">
          <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-muted">Pending invites</div>
          {invites.map((invite) => (
            <div key={invite.id} className="flex items-center gap-3 border-b border-line py-2.5 last:border-none">
              <div className="min-w-0 flex-1">
                <div className="truncate font-medium">{invite.email}</div>
                <div className="text-[13px] text-muted">
                  Expires {new Date(invite.expiresAt * 1000).toLocaleDateString()}
                </div>
              </div>
              {isOwner && (
                <button
                  disabled={busy}
                  onClick={() => void run(async () => {
                    await api.revokeInvite(workspaceId, invite.id)
                    if (lastInvite?.email === invite.email) setLastInvite(null)
                    await load()
                  })}
                  className="shrink-0 text-sm font-semibold text-muted hover:text-danger disabled:opacity-50"
                >
                  Revoke
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      {isOwner && (
        <form onSubmit={(event) => void sendInvite(event)} className="mt-5 flex flex-col gap-2 border-t border-line pt-5">
          <label htmlFor="invite-email" className="text-sm font-semibold text-ink-soft">Invite a teammate</label>
          <div className="flex flex-col gap-2 sm:flex-row">
            <input
              id="invite-email"
              className="field-input min-w-0 flex-1"
              type="email"
              placeholder="teammate@company.com"
              autoComplete="off"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
            <button
              type="submit"
              disabled={busy || !/\S+@\S+\.\S+/.test(email)}
              className="bg-brand px-4 py-2.5 font-semibold text-white transition-colors hover:bg-brand-deep disabled:cursor-not-allowed disabled:opacity-50"
            >
              Send invite
            </button>
          </div>
        </form>
      )}

      {lastInvite && (
        <div className="mt-4 bg-card px-3.5 py-3 text-sm" role="status">
          <div className="mb-2 text-ink-soft">
            {lastInvite.emailSent
              ? `Invite emailed to ${lastInvite.email}. You can also share this link:`
              : `Email isn't set up on this server. Send this link to ${lastInvite.email}:`}
          </div>
          <div className="flex items-center gap-2">
            <input
              readOnly
              aria-label="Invite link"
              value={lastInvite.link}
              onFocus={(event) => event.currentTarget.select()}
              className="field-input min-w-0 flex-1 !py-2 font-mono text-xs"
            />
            <button
              type="button"
              onClick={() => void copyText(lastInvite.link)}
              className="flex shrink-0 items-center gap-1.5 border border-line bg-white px-3 py-2 text-sm font-semibold hover:border-brand hover:text-brand"
            >
              <CopyIcon /> Copy
            </button>
          </div>
          <div className="mt-2 text-xs text-muted">Works once and expires in 7 days. The teammate must sign in with {lastInvite.email}.</div>
        </div>
      )}
    </section>
  )
}
