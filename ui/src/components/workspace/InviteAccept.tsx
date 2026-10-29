import { useState } from 'react'
import { api } from '../../api.ts'
import type { InvitePreview, JoinResult } from '../../types.ts'

const statusMessages: Record<Exclude<InvitePreview['status'], 'pending'>, string> = {
  accepted: 'This invite has already been used.',
  revoked: 'This invite was revoked. Ask the workspace owner for a new one.',
  expired: 'This invite has expired. Ask the workspace owner for a new one.',
}

export default function InviteAccept({
  token,
  userEmail,
  onJoined,
  onDismiss,
  onSignOut,
  preview,
  loadError,
}: {
  token: string
  userEmail: string
  onJoined: (result: JoinResult) => void
  onDismiss: () => void
  onSignOut: () => void
  preview: InvitePreview | null
  loadError: string
}) {
  const [acceptError, setAcceptError] = useState('')
  const [busy, setBusy] = useState(false)
  const error = loadError || acceptError

  async function accept() {
    setBusy(true)
    setAcceptError('')
    try {
      onJoined(await api.acceptInvite(token))
    } catch (failure) {
      setAcceptError(failure instanceof Error ? failure.message : 'Unable to accept the invite.')
      setBusy(false)
    }
  }

  const wrongAccount = preview && preview.email.toLowerCase() !== userEmail.toLowerCase()
  if (!preview && !error) return null

  return (
    <section className="border-b border-line px-5 py-5" aria-labelledby="invite-title">
      <div className="mb-1 text-xs font-semibold uppercase tracking-wide text-brand">Workspace invite</div>
      {preview && (
        <>
          <h2 id="invite-title" className="mb-1 text-xl font-bold">Join {preview.workspaceName || 'a workspace'}</h2>
          <p className="mb-4 text-sm text-muted">
            {preview.invitedBy} invited {preview.email}. You'll see the workspace's wallets when you join.
          </p>
          {preview.status !== 'pending' && <p className="mb-4 text-sm text-ink-soft">{statusMessages[preview.status]}</p>}
          {preview.status === 'pending' && wrongAccount && (
            <p className="notice notice-warning mb-4">
              You're signed in as {userEmail}. Sign in as {preview.email} to accept this invite.
            </p>
          )}
        </>
      )}
      {error && <div role="alert" className="mb-4 bg-danger-soft px-3 py-2 text-sm text-danger">{error}</div>}
      <div className="flex flex-wrap gap-2">
        {preview?.status === 'pending' && !wrongAccount && (
          <button className="btn btn-primary btn-lg" disabled={busy} onClick={() => void accept()}>
            {busy ? 'Joining…' : 'Accept invite'}
          </button>
        )}
        {preview?.status === 'pending' && wrongAccount && (
          <button className="btn btn-primary btn-lg" onClick={onSignOut}>
            Sign out and switch account
          </button>
        )}
        <button
          className="btn btn-secondary"
          onClick={onDismiss}
        >
          {preview?.status === 'pending' && !wrongAccount ? 'Not now' : 'Dismiss'}
        </button>
      </div>
    </section>
  )
}
