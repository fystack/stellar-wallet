import { useState } from 'react'
import type { WorkspaceInvite } from '../types.ts'

export default function InvitationList({
  invitations,
  verified,
  compact = false,
  onAccept,
  onDecline,
}: {
  invitations: WorkspaceInvite[]
  verified: boolean
  compact?: boolean
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
      <p className={'text-sm text-muted ' + (compact ? 'px-3 py-2' : 'px-5 py-6 text-center')}>
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
          className={'flex items-center gap-3 border-b border-line last:border-none ' + (compact ? 'px-3 py-2.5' : 'px-5 py-4')}
        >
          {!compact && (
            <span className="grid h-10 w-10 shrink-0 place-items-center bg-brand-soft text-sm font-bold uppercase text-brand">
              {invitation.workspaceName?.charAt(0) ?? '·'}
            </span>
          )}
          <span className="min-w-0 flex-1">
            <span className={'block truncate font-semibold ' + (compact ? 'text-sm' : '')}>{invitation.workspaceName}</span>
            <span className="block truncate text-[13px] text-muted">
              From {invitation.invitedBy}
              {!compact && ` · expires ${new Date(invitation.expiresAt * 1000).toLocaleDateString()}`}
            </span>
          </span>
          <span className="flex shrink-0 gap-1.5">
            <button
              type="button"
              disabled={!!busy}
              onClick={() => void run(invitation.id, 'decline')}
              aria-label={`Decline invitation to ${invitation.workspaceName}`}
              className={'border border-line bg-white font-semibold text-ink-soft hover:bg-hover disabled:opacity-50 ' + (compact ? 'px-2 py-1 text-xs' : 'px-3 py-2 text-sm')}
            >
              Decline
            </button>
            <button
              type="button"
              disabled={!!busy}
              onClick={() => void run(invitation.id, 'accept')}
              aria-label={`Accept invitation to ${invitation.workspaceName}`}
              className={'bg-brand font-semibold text-white hover:bg-brand-deep disabled:opacity-50 ' + (compact ? 'px-2 py-1 text-xs' : 'px-3 py-2 text-sm')}
            >
              {busy?.id === invitation.id && busy.action === 'accept' ? 'Joining…' : 'Accept'}
            </button>
          </span>
        </li>
      ))}
    </ul>
  )
}
