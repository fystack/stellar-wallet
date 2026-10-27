import { useEffect, useRef, useState } from 'react'
import { CheckIcon, ChevronDownIcon } from '../icons.tsx'
import type { Workspace, WorkspaceInvite } from '../types.ts'
import CreateWorkspaceModal from './CreateWorkspaceModal.tsx'
import InvitationList from './InvitationList.tsx'
import { workspaceLabel } from './WorkspacePicker.tsx'

export default function WorkspaceSwitcher({
  workspaces,
  active,
  onSwitch,
  onCreate,
  invitations,
  invitationsVerified,
  onAcceptInvitation,
  onDeclineInvitation,
}: {
  workspaces: Workspace[]
  active: Workspace | null
  onSwitch: (id: string) => Promise<void>
  onCreate: (name: string) => Promise<void>
  invitations: WorkspaceInvite[]
  invitationsVerified: boolean
  onAcceptInvitation: (id: string) => Promise<void>
  onDeclineInvitation: (id: string) => Promise<void>
}) {
  const [open, setOpen] = useState(false)
  const [showCreate, setShowCreate] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function onPointerDown(event: PointerEvent) {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false)
    }
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div ref={containerRef} className="relative ml-auto min-w-0 md:ml-0 md:mb-4">
      <button
        type="button"
        onClick={() => setOpen((current) => !current)}
        aria-haspopup="listbox"
        aria-expanded={open}
        className="flex w-full min-w-0 max-w-[180px] items-center gap-2 border border-line bg-white px-3 py-2 text-left hover:bg-hover md:max-w-none"
      >
        <span className="grid h-7 w-7 shrink-0 place-items-center bg-brand-soft text-xs font-bold uppercase text-brand">
          {active?.name.charAt(0) ?? '·'}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-semibold">{active?.name ?? 'Loading…'}</span>
          <span className="hidden text-xs text-muted md:block">
            {active ? workspaceLabel(active) : ''}
          </span>
        </span>
        {invitations.length > 0 && (
          <span
            aria-label={`${invitations.length} pending invitation${invitations.length === 1 ? '' : 's'}`}
            className="grid h-5 min-w-5 shrink-0 place-items-center bg-brand px-1.5 text-xs font-bold text-white"
          >
            {invitations.length}
          </span>
        )}
        <span className="shrink-0 text-muted">
          <ChevronDownIcon />
        </span>
      </button>

      {open && (
        <div className="absolute left-0 right-0 top-full z-50 mt-1 min-w-[300px] border border-line bg-white shadow-lg md:right-auto md:w-full">
          {invitations.length > 0 && (
            <div className="border-b border-line">
              <div className="px-3 pb-1 pt-3 text-xs font-semibold uppercase tracking-wide text-muted">Invitations</div>
              <InvitationList
                compact
                invitations={invitations}
                verified={invitationsVerified}
                onAccept={async (id) => {
                  setOpen(false)
                  await onAcceptInvitation(id)
                }}
                onDecline={onDeclineInvitation}
              />
            </div>
          )}
          <div className="px-3 pb-1 pt-3 text-xs font-semibold uppercase tracking-wide text-muted">Workspaces</div>
          <ul role="listbox" aria-label="Workspaces" className="max-h-64 overflow-y-auto py-1">
            {workspaces.map((workspace) => (
              <li key={workspace.id}>
                <button
                  type="button"
                  role="option"
                  aria-selected={workspace.id === active?.id}
                  onClick={() => {
                    setOpen(false)
                    if (workspace.id !== active?.id) void onSwitch(workspace.id)
                  }}
                  className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm hover:bg-hover"
                >
                  <span className="min-w-0 flex-1 truncate font-medium">{workspace.name}</span>
                  <span className="shrink-0 text-xs text-muted">{workspaceLabel(workspace)}</span>
                  <span className={'shrink-0 text-brand ' + (workspace.id === active?.id ? '' : 'invisible')}>
                    <CheckIcon />
                  </span>
                </button>
              </li>
            ))}
          </ul>
          <button
            type="button"
            onClick={() => {
              setOpen(false)
              setShowCreate(true)
            }}
            className="flex w-full items-center gap-2.5 border-t border-line px-3 py-3 text-left text-sm font-semibold text-brand hover:bg-hover"
          >
            <span className="grid h-5 w-5 place-items-center text-base leading-none">+</span>
            New workspace
          </button>
        </div>
      )}
      {showCreate && <CreateWorkspaceModal onClose={() => setShowCreate(false)} onCreate={onCreate} />}
    </div>
  )
}
