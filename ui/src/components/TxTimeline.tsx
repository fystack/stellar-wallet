import { CheckIcon, CloseIcon } from '../icons.tsx'
import type { TxStatus } from '../types.ts'

const steps: { status: TxStatus; label: string; description: string }[] = [
  {
    status: 'policy_check',
    label: 'Policy check',
    description: 'Verifying limits and whitelist',
  },
  {
    status: 'signing',
    label: 'MPC signing',
    description: '2-of-3 nodes computing threshold signature',
  },
  {
    status: 'broadcast',
    label: 'Broadcast',
    description: 'Transaction submitted to network',
  },
  {
    status: 'confirmed',
    label: 'Confirmed',
    description: 'Included in a block on-chain',
  },
]

const ORDER: TxStatus[] = [
  'policy_check',
  'signing',
  'broadcast',
  'confirmed',
  'failed',
]

type State = 'done' | 'active' | 'pending' | 'failed'

function stepState(step: TxStatus, current: TxStatus): State {
  // Confirmed is terminal success — every step is complete.
  if (current === 'confirmed') return 'done'
  if (current === 'failed') {
    return ORDER.indexOf(step) < ORDER.indexOf(current) - 1 ? 'done' : 'failed'
  }
  const ci = ORDER.indexOf(current)
  const si = ORDER.indexOf(step)
  if (si < ci) return 'done'
  if (si === ci) return 'active'
  return 'pending'
}

export default function TxTimeline({ status }: { status: TxStatus }) {
  return (
    <ol className="flex flex-col">
      {steps.map((step, i) => {
        const state = stepState(step.status, status)
        const isLast = i === steps.length - 1
        return (
          <li key={step.status} className="flex gap-3.5">
            {/* rail: node + connector */}
            <div className="flex flex-col items-center">
              <div className="relative grid h-6 w-6 place-items-center">
                {state === 'active' && (
                  <span className="absolute inline-flex h-6 w-6 animate-ping rounded-full bg-brand/20" />
                )}
                <div
                  style={{
                    animation: 'tlPop .4s cubic-bezier(.34,1.56,.64,1) both',
                    animationDelay: `${i * 0.08}s`,
                  }}
                  className={
                    'relative grid h-6 w-6 place-items-center rounded-full transition-colors duration-300 ' +
                    (state === 'done'
                      ? 'bg-brand text-white'
                      : state === 'active'
                        ? 'border-2 border-brand bg-white'
                        : state === 'failed'
                          ? 'bg-danger text-white'
                          : 'border-2 border-line bg-white')
                  }
                >
                  {state === 'done' && <CheckIcon size={13} />}
                  {state === 'active' && (
                    <span className="h-2 w-2 animate-pulse rounded-full bg-brand" />
                  )}
                  {state === 'pending' && (
                    <span className="h-1.5 w-1.5 rounded-full bg-line" />
                  )}
                  {state === 'failed' && <CloseIcon size={13} />}
                </div>
              </div>
              {!isLast && (
                <div className="relative my-1 w-0.5 flex-1 overflow-hidden rounded-full bg-line">
                  <div
                    className="absolute inset-x-0 top-0 bg-brand transition-all duration-500"
                    style={{
                      height: state === 'done' ? '100%' : '0%',
                      transitionDelay: `${i * 0.08 + 0.12}s`,
                    }}
                  />
                </div>
              )}
            </div>

            {/* content */}
            <div className={isLast ? 'pb-0' : 'pb-6'}>
              <p
                className={
                  'text-sm font-semibold leading-6 transition-colors ' +
                  (state === 'pending'
                    ? 'text-muted'
                    : state === 'active'
                      ? 'text-brand'
                      : state === 'failed'
                        ? 'text-danger'
                        : 'text-ink')
                }
              >
                {step.label}
              </p>
              <p className="mt-0.5 text-xs leading-relaxed text-muted">
                {step.description}
              </p>
            </div>
          </li>
        )
      })}
    </ol>
  )
}
