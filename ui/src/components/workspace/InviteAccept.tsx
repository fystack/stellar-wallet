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

  const name = preview?.workspaceName || 'a workspace'
  const pending = preview?.status === 'pending'

  return (
    <section className="border-b border-line last:border-none" aria-label={`Join ${name}`}>
      <div className="flex flex-wrap items-center gap-3 px-5 py-4">
        <span className="avatar avatar-brand h-10 w-10">{preview?.workspaceName?.charAt(0) || '·'}</span>
        <span className="min-w-[9rem] flex-1">
          <span className="block truncate font-semibold">{preview ? name : 'Workspace invite'}</span>
          {preview && (
            <span className="block truncate text-[13px] text-muted">
              From {preview.invitedBy} · for {preview.email}
            </span>
          )}
        </span>
        <span className="ml-auto flex shrink-0 gap-1.5">
          <button type="button" onClick={onDismiss} className="btn btn-secondary btn-sm">
            {pending && !wrongAccount ? 'Not now' : 'Dismiss'}
          </button>
          {pending && !wrongAccount && (
            <button
              type="button"
              disabled={busy}
              onClick={() => void accept()}
              aria-label={`Accept invite to ${name}`}
              className="btn btn-primary btn-sm"
            >
              {busy ? 'Joining…' : 'Accept'}
            </button>
          )}
        </span>
      </div>
      {(error || (preview && !pending) || (pending && wrongAccount)) && (
        <div className="flex flex-col items-start gap-3 px-5 pb-4">
          {preview && preview.status !== 'pending' && <p className="text-sm text-ink-soft">{statusMessages[preview.status]}</p>}
          {pending && wrongAccount && (
            <p className="notice notice-warning w-full [overflow-wrap:anywhere]">
              You're signed in as {userEmail}. Sign in as {preview.email} to accept this invite.{' '}
              <button type="button" className="btn-link font-semibold" onClick={onSignOut}>
                Sign out and switch account
              </button>
            </p>
          )}
          {error && <div role="alert" className="w-full bg-danger-soft px-3 py-2 text-sm text-danger">{error}</div>}
        </div>
      )}
    </section>
  )
}
