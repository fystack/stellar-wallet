import { CheckIcon, ClockIcon, CloseIcon, SpinnerIcon } from '../icons.tsx'
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
    <ol>
      {steps.map((step, i) => {
        const state = stepState(step.status, status)
        const isLast = i === steps.length - 1
        const nextDone = stepState(steps[i + 1]?.status, status) === 'done'
        return (
          <li key={step.status} className="flex gap-4">
            <div className="flex flex-col items-center">
              <div className="relative grid place-items-center">
                {/* pulsing ring on the in-progress step */}
                {state === 'active' && (
                  <span className="absolute inline-flex h-7 w-7 animate-ping rounded-md bg-brand/25" />
                )}
                <div
                  style={{
                    animation: 'tlPop .45s cubic-bezier(.34,1.56,.64,1) both',
                    animationDelay: `${i * 0.09}s`,
                  }}
                  className={
                    'relative grid h-7 w-7 shrink-0 place-items-center rounded-md border-2 shadow-sm transition-colors duration-300 ' +
                    (state === 'done'
                      ? 'border-green-500 bg-green-500 text-white'
                      : state === 'active'
                        ? 'border-brand bg-white text-brand'
                        : state === 'failed'
                          ? 'border-red-500 bg-red-500 text-white'
                          : 'border-line bg-white text-[#cbd3de] shadow-none')
                  }
                >
                  {state === 'done' && <CheckIcon size={13} />}
                  {state === 'active' && <SpinnerIcon size={13} />}
                  {state === 'pending' && <ClockIcon size={13} />}
                  {state === 'failed' && <CloseIcon size={13} />}
                </div>
              </div>
              {!isLast && (
                <div className="relative mt-1 h-8 w-0.5 overflow-hidden rounded-full bg-line">
                  {/* connector fills green once this step is done — drawn top→down, staggered */}
                  <div
                    className="absolute inset-x-0 top-0 bg-green-400 transition-all duration-500"
                    style={{
                      height: state === 'done' ? '100%' : '0%',
                      transitionDelay: `${i * 0.09 + 0.15}s`,
                    }}
                  />
                  {state === 'done' && !nextDone && (
                    <div className="absolute inset-x-0 top-0 h-2 animate-pulse bg-brand" />
                  )}
                </div>
              )}
            </div>
            <div className="pb-7">
              <p
                className={
                  'text-sm font-semibold transition-colors ' +
                  (state === 'pending' ? 'text-[#cbd3de]' : 'text-ink')
                }
              >
                {step.label}
              </p>
              <p className="mt-0.5 text-xs text-muted">{step.description}</p>
            </div>
          </li>
        )
      })}
    </ol>
  )
}
