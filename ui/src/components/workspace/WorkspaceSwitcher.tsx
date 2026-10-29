import { ChevronDownIcon } from '../../icons.tsx'
import type { Workspace } from '../../types.ts'
import { workspaceLabel } from './WorkspacePicker.tsx'

export default function WorkspaceSwitcher({
  active,
  invitationCount,
  onOpen,
}: {
  active: Workspace | null
  invitationCount: number
  onOpen: () => void
}) {
  return (
    <div className="ml-auto min-w-0 md:ml-0 md:mb-4">
      <button
        type="button"
        onClick={onOpen}
        className="btn btn-secondary w-full min-w-0 max-w-[180px] justify-start px-3 py-2 text-left font-normal md:max-w-none"
      >
        <span className="avatar avatar-brand h-7 w-7 text-xs">
          {active?.name.charAt(0) ?? '·'}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-semibold">{active?.name ?? 'Loading…'}</span>
          <span className="hidden text-xs text-muted md:block">
            {active ? workspaceLabel(active) : ''}
          </span>
        </span>
        {invitationCount > 0 && (
          <span
            aria-label={`${invitationCount} pending invitation${invitationCount === 1 ? '' : 's'}`}
            className="count"
          >
            {invitationCount}
          </span>
        )}
        <span className="shrink-0 -rotate-90 text-muted">
          <ChevronDownIcon />
        </span>
      </button>
    </div>
  )
}
