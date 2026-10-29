import { useState } from 'react'
import type { ReactNode } from 'react'
import { ChevronDownIcon } from '../../icons.tsx'
import AuthLayout from '../AuthLayout.tsx'
import type { Workspace, WorkspaceInvite } from '../../types.ts'
import CreateWorkspaceModal from './CreateWorkspaceModal.tsx'
import InvitationList from './InvitationList.tsx'
import { roleLabels } from '../../lib/roles.ts'

export function workspaceLabel(workspace: Workspace) {
  if (workspace.personal) return 'Personal'
  return roleLabels[workspace.role]
}

export default function WorkspacePicker({
  userEmail,
  workspaces,
  onPick,
  onCreate,
  signingOut,
  onSignOut,
  invitations,
  invitationsVerified,
  onAcceptInvitation,
  onDeclineInvitation,
  onBack,
  linkInvite,
}: {
  userEmail: string
  workspaces: Workspace[]
  invitations: WorkspaceInvite[]
  invitationsVerified: boolean
  onAcceptInvitation: (id: string) => Promise<void>
  onDeclineInvitation: (id: string) => Promise<void>
  onPick: (id: string) => Promise<void>
  onCreate: (name: string) => Promise<void>
  signingOut: boolean
  onSignOut: () => void
  onBack?: () => void
  linkInvite?: ReactNode
}) {
  const [pickingId, setPickingId] = useState('')
  const invitationCount = invitations.length + (linkInvite ? 1 : 0)
  const [showCreate, setShowCreate] = useState(false)
  const [tab, setTab] = useState<'workspaces' | 'invitations'>(linkInvite || invitations.length > 0 ? 'invitations' : 'workspaces')

  async function pick(id: string) {
    setPickingId(id)
    try {
      await onPick(id)
    } finally {
      setPickingId('')
    }
  }

  return (
    <AuthLayout>
      {onBack && (
        <button type="button" onClick={onBack} className="btn btn-link mb-4 text-sm">
          ← Back
        </button>
      )}
      <h1 className="text-[34px] font-extrabold leading-tight">Choose a workspace</h1>
      <p className="mt-2 mb-7 text-[17px] text-ink-soft">
        Signed in as <span className="break-all font-semibold text-ink">{userEmail}</span> ·{' '}
        <button onClick={onSignOut} disabled={signingOut} className="btn btn-link">
          {signingOut ? 'Signing out…' : 'Sign out'}
        </button>
      </p>

      <div role="tablist" aria-label="Workspaces and invitations" className="flex border border-b-0 border-line bg-white">
        {(['workspaces', 'invitations'] as const).map((key) => (
          <button
            key={key}
            type="button"
            role="tab"
            id={`picker-tab-${key}`}
            aria-selected={tab === key}
            aria-controls={`picker-panel-${key}`}
            onClick={() => setTab(key)}
            className={
              'flex flex-1 items-center justify-center gap-2 border-b-2 px-4 py-3 text-sm font-semibold transition-colors ' +
              (tab === key ? 'border-brand text-brand' : 'border-line text-muted hover:text-ink')
            }
          >
            {key === 'workspaces' ? 'Workspaces' : 'Invitations'}
            {key === 'invitations' && invitationCount > 0 && (
              <span className="count">
                {invitationCount}
              </span>
            )}
          </button>
        ))}
      </div>

      {tab === 'invitations' && (
        <div id="picker-panel-invitations" role="tabpanel" aria-labelledby="picker-tab-invitations" className="border border-line bg-white">
          {linkInvite}
          {(!linkInvite || invitations.length > 0) && (
            <InvitationList
              invitations={invitations}
              verified={invitationsVerified}
              onAccept={onAcceptInvitation}
              onDecline={onDeclineInvitation}
            />
          )}
        </div>
      )}

      {tab === 'workspaces' && (
        <div id="picker-panel-workspaces" role="tabpanel" aria-labelledby="picker-tab-workspaces" className="border border-line bg-white">
          <ul aria-label="Your workspaces">
            {workspaces.map((workspace) => (
              <li key={workspace.id} className="border-b border-line last:border-none">
                <button
                  type="button"
                  disabled={!!pickingId}
                  onClick={() => void pick(workspace.id)}
                  className="flex w-full items-center gap-3 px-5 py-4 text-left transition-colors hover:bg-hover disabled:cursor-wait disabled:opacity-60"
                >
                  <span className="avatar avatar-brand h-10 w-10">
                    {workspace.name.charAt(0)}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-semibold">{workspace.name}</span>
                    <span className="block text-[13px] text-muted">{workspaceLabel(workspace)}</span>
                  </span>
                  <span className="shrink-0 -rotate-90 text-muted">
                    {pickingId === workspace.id ? '…' : <ChevronDownIcon />}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      <p className="mt-6 text-center text-[15px] text-ink-soft">
        Need a new team?{' '}
        <button onClick={() => setShowCreate(true)} className="btn btn-link font-bold">
          Create a workspace
        </button>
      </p>
    {showCreate && <CreateWorkspaceModal onClose={() => setShowCreate(false)} onCreate={onCreate} />}
    </AuthLayout>
  )
}
