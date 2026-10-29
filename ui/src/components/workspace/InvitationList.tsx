import { useState } from 'react'
import type { WorkspaceInvite } from '../../types.ts'

export default function InvitationList({
  invitations,
  verified,
  onAccept,
  onDecline,
}: {
  invitations: WorkspaceInvite[]
  verified: boolean
  onAccept: (id: string) => Promise<void>
  onDecline: (id: string) => Promise<void>
}) {
  const [busy, setBusy] = useState<{ id: string; action: 'accept' | 'decline' } | null>(null)

  async function run(id: string, action: 'accept' | 'decline') {
    setBusy({ id, action })
    try {
      await (action === 'accept' ? onAccept : onDecline)(id)
    } finally {
      setBusy(null)
    }
  }

  if (invitations.length === 0) {
    return (
      <p className="px-5 py-6 text-center text-sm text-muted">
        {verified
          ? 'No pending invitations.'
          : 'Invitations show up here once your email is confirmed by signing in with Google or GitHub. Until then, open the invite link from your email.'}
      </p>
    )
  }

  return (
    <ul aria-label="Invitations">
      {invitations.map((invitation) => (
        <li
          key={invitation.id}
          className="flex items-center gap-3 border-b border-line px-5 py-4 last:border-none"
        >
          <span className="avatar avatar-brand h-10 w-10">
            {invitation.workspaceName?.charAt(0) ?? '·'}
          </span>
          <span className="min-w-0 flex-1">
            <span className="block truncate font-semibold">{invitation.workspaceName}</span>
            <span className="block truncate text-[13px] text-muted">
              From {invitation.invitedBy}
              {` · expires ${new Date(invitation.expiresAt * 1000).toLocaleDateString()}`}
            </span>
          </span>
          <span className="flex shrink-0 gap-1.5">
            <button
              type="button"
              disabled={!!busy}
              onClick={() => void run(invitation.id, 'decline')}
              aria-label={`Decline invitation to ${invitation.workspaceName}`}
              className="btn btn-secondary btn-sm"
            >
              Decline
            </button>
            <button
              type="button"
              disabled={!!busy}
              onClick={() => void run(invitation.id, 'accept')}
              aria-label={`Accept invitation to ${invitation.workspaceName}`}
              className="btn btn-primary btn-sm"
            >
              {busy?.id === invitation.id && busy.action === 'accept' ? 'Joining…' : 'Accept'}
            </button>
          </span>
        </li>
      ))}
    </ul>
  )
}
