import { useState } from 'react'
import { api } from '../../api.ts'
import { toast } from '../../toast.tsx'
import type { ApprovalPolicy, ApprovalPolicySettings } from '../../types.ts'

function describe(policy: ApprovalPolicy) {
  const approvals = `${policy.requiredApprovals} approval${policy.requiredApprovals === 1 ? '' : 's'}`
  if (policy.belowThreshold === 'approve') return `Every payment needs ${approvals} from another owner or operator.`
  return `Payments over ${policy.thresholdXlm} XLM, and all non-XLM payments, need ${approvals} from another owner or operator. Smaller XLM payments send immediately.`
}

export default function ApprovalPolicyCard({
  workspaceId,
  isOwner,
  settings,
  onChange,
}: {
  workspaceId: string
  isOwner: boolean
  settings: ApprovalPolicySettings
  onChange: (settings: ApprovalPolicySettings) => void
}) {
  const { policy, maxApprovals } = settings
  const [threshold, setThreshold] = useState(policy?.thresholdXlm ?? '100')
  const [required, setRequired] = useState(Math.min(policy?.requiredApprovals ?? 1, Math.max(maxApprovals, 1)))
  const [belowThreshold, setBelowThreshold] = useState<ApprovalPolicy['belowThreshold']>(policy?.belowThreshold ?? 'send')
  const [busy, setBusy] = useState(false)

  async function run(action: () => Promise<void>) {
    setBusy(true)
    try {
      await action()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Unable to update the approval policy.')
    } finally {
      setBusy(false)
    }
  }

  function save(event: React.FormEvent) {
    event.preventDefault()
    void run(async () => {
      onChange(await api.setApprovalPolicy(workspaceId, { thresholdXlm: threshold.trim(), requiredApprovals: required, belowThreshold }))
      toast.success('Approval policy saved')
    })
  }

  function disable() {
    void run(async () => {
      await api.disableApprovalPolicy(workspaceId)
      onChange({ policy: null, maxApprovals })
      toast.success('Approval policy turned off')
    })
  }

  return (
    <section className="surface mb-5" aria-labelledby="approval-policy-title">
      <h3 id="approval-policy-title" className="mb-1 text-[17px] font-bold">Payment approvals</h3>
      <p role="status" className="mb-[18px] text-sm text-muted">
        {policy ? describe(policy) : 'Off. Owners and operators send payments without a second sign-off.'}
      </p>

      {isOwner && maxApprovals === 0 && (
        <p className="text-sm text-ink-soft">Make another member an operator or owner to require approvals.</p>
      )}

      {isOwner && maxApprovals > 0 && (
        <form onSubmit={save} className="flex flex-col gap-4">
          <div className="flex flex-col gap-4 sm:flex-row">
            <label className="flex min-w-0 flex-1 flex-col gap-2">
              <span className="text-sm font-semibold text-ink-soft">Threshold (XLM)</span>
              <input
                className="field-input"
                inputMode="decimal"
                value={threshold}
                onChange={(event) => setThreshold(event.target.value)}
              />
            </label>
            <label className="flex min-w-0 flex-1 flex-col gap-2">
              <span className="text-sm font-semibold text-ink-soft">Approvals required</span>
              <select className="field-input" value={required} onChange={(event) => setRequired(Number(event.target.value))}>
                {Array.from({ length: maxApprovals }, (_, index) => index + 1).map((count) => (
                  <option key={count} value={count}>{count}</option>
                ))}
              </select>
            </label>
          </div>
          <fieldset className="flex flex-col gap-2">
            <legend className="mb-2 text-sm font-semibold text-ink-soft">XLM payments at or below the threshold</legend>
            <label className="flex items-center gap-2 text-sm">
              <input type="radio" name="below-threshold" checked={belowThreshold === 'send'} onChange={() => setBelowThreshold('send')} />
              Send immediately
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input type="radio" name="below-threshold" checked={belowThreshold === 'approve'} onChange={() => setBelowThreshold('approve')} />
              Require approval too
            </label>
          </fieldset>
          <div className="flex flex-col gap-2 sm:flex-row">
            <button type="submit" className="btn btn-primary btn-lg" disabled={busy || !threshold.trim()}>
              {busy ? 'Saving…' : policy ? 'Save policy' : 'Turn on approvals'}
            </button>
            {policy && (
              <button type="button" className="btn btn-secondary" disabled={busy} onClick={disable}>
                Turn off approvals
              </button>
            )}
          </div>
        </form>
      )}
    </section>
  )
}
