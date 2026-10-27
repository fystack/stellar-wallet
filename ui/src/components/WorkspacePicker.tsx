import { useState } from 'react'
import { ChevronDownIcon } from '../icons.tsx'
import { FystackLogo } from '../logos.tsx'
import type { Workspace, WorkspaceInvite } from '../types.ts'
import CreateWorkspaceModal from './CreateWorkspaceModal.tsx'
import InvitationList from './InvitationList.tsx'

export function workspaceLabel(workspace: Workspace) {
  if (workspace.personal) return 'Personal'
  return workspace.role === 'owner' ? 'Owner' : 'Member'
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
}) {
  const [pickingId, setPickingId] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [tab, setTab] = useState<'workspaces' | 'invitations'>(invitations.length > 0 ? 'invitations' : 'workspaces')

  async function pick(id: string) {
    setPickingId(id)
    try {
      await onPick(id)
    } finally {
      setPickingId('')
    }
  }

  return (
    <div className="flex min-h-full items-center justify-center bg-[#f7f8fa] p-4">
      <div className="w-full max-w-[440px]">
        <div className="mb-6 text-center sm:mb-8">
          <div className="mx-auto mb-4 w-fit sm:mb-5">
            <FystackLogo height={44} />
          </div>
          <h1 className="text-2xl font-extrabold">Choose a workspace</h1>
          <p className="mt-1.5 text-[15px] text-muted">
            Signed in as {userEmail} ·{' '}
            <button
              onClick={onSignOut}
              disabled={signingOut}
              className="font-semibold text-brand hover:underline disabled:opacity-50"
            >
              {signingOut ? 'Signing out…' : 'Sign out'}
            </button>
          </p>
        </div>

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
              {key === 'invitations' && invitations.length > 0 && (
                <span className="grid h-5 min-w-5 place-items-center bg-brand px-1.5 text-xs font-bold text-white">
                  {invitations.length}
                </span>
              )}
            </button>
          ))}
        </div>

        {tab === 'invitations' && (
          <div id="picker-panel-invitations" role="tabpanel" aria-labelledby="picker-tab-invitations" className="border border-line bg-white">
            <InvitationList
              invitations={invitations}
              verified={invitationsVerified}
              onAccept={onAcceptInvitation}
              onDecline={onDeclineInvitation}
            />
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
                    <span className="grid h-10 w-10 shrink-0 place-items-center bg-brand-soft text-sm font-bold uppercase text-brand">
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

        <p className="mt-5 text-center text-sm text-muted">
          Need a new team?{' '}
          <button onClick={() => setShowCreate(true)} className="font-semibold text-brand hover:underline">
            Create a workspace
          </button>
        </p>
      </div>
      {showCreate && <CreateWorkspaceModal onClose={() => setShowCreate(false)} onCreate={onCreate} />}
    </div>
  )
}
