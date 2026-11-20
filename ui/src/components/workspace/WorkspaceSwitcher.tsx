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
    <div className="ml-auto min-w-0 md:ml-0 md:mb-6">
      <button
        type="button"
        onClick={onOpen}
        className="btn w-full min-w-0 max-w-[180px] justify-start border border-night-line bg-white/[0.04] px-3 py-2 text-left font-normal text-white hover:border-white/20 hover:bg-white/[0.08] focus-visible:ring-offset-night md:max-w-none"
      >
        <span className="avatar h-7 w-7 bg-brand text-xs font-bold">
          {active?.name.charAt(0) ?? '·'}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-semibold">{active?.name ?? 'Loading…'}</span>
          <span className="hidden truncate text-xs text-white/50 md:block">
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
        <span className="shrink-0 -rotate-90 text-white/40">
          <ChevronDownIcon />
        </span>
      </button>
    </div>
  )
}
