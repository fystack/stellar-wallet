import { useEffect, useState } from 'react'
import { CheckIcon, SpinnerIcon } from '../icons.tsx'

type NodeState = 'idle' | 'active' | 'done'

type Props = {
  // "keygen" while a wallet is generating, "sign" while a tx is signing.
  mode: 'keygen' | 'sign'
  // When true the ceremony is finished — show all nodes done.
  done?: boolean
}

const labels = {
  keygen: {
    title: 'MPC key generation',
    running: (n: number) => `Node ${n} generating key share…`,
    connect: 'Connecting to signing cluster…',
    finished: 'Key split across 3 nodes — never assembled.',
  },
  sign: {
    title: 'MPC threshold signing',
    running: (n: number) => `Node ${n} signing…`,
    connect: 'Requesting signatures from cluster…',
    finished: 'Signed by 2 of 3 nodes.',
  },
}

export default function MpcCeremony({ mode, done = false }: Props) {
  const [nodes, setNodes] = useState<NodeState[]>(['idle', 'idle', 'idle'])
  const [msg, setMsg] = useState(labels[mode].connect)

  useEffect(() => {
    if (done) return
    const L = labels[mode]
    setNodes(['idle', 'idle', 'idle'])
    setMsg(L.connect)
    const timers = [
      setTimeout(() => {
        setNodes(['active', 'idle', 'idle'])
        setMsg(L.running(1))
      }, 700),
      setTimeout(() => {
        setNodes(['done', 'active', 'idle'])
        setMsg(L.running(2))
      }, 2200),
      setTimeout(() => {
        setNodes(['done', 'done', 'active'])
        setMsg(L.running(3))
      }, 3700),
      setTimeout(() => setNodes(['done', 'done', 'done']), 5200),
    ]
    // Loop the animation until the real result arrives.
    const loop = setInterval(() => {
      setNodes(['active', 'idle', 'idle'])
      setMsg(L.running(1))
      setTimeout(() => setNodes(['done', 'active', 'idle']), 1500)
      setTimeout(() => setNodes(['done', 'done', 'active']), 3000)
      setTimeout(() => setNodes(['done', 'done', 'done']), 4500)
    }, 6000)
    return () => {
      timers.forEach(clearTimeout)
      clearInterval(loop)
    }
  }, [mode, done])

  const display: NodeState[] = done ? ['done', 'done', 'done'] : nodes
  const L = labels[mode]

  return (
    <div className="flex flex-col items-center gap-4 py-2">
      <div className="text-xs font-semibold uppercase tracking-wide text-muted">
        {L.title}
      </div>

      <div className="flex items-center">
        {display.map((state, i) => (
          <div key={i} className="flex items-center">
            <div className="flex flex-col items-center gap-1.5">
              <div
                className={
                  'grid h-12 w-12 place-items-center border-2 transition-all duration-500 ' +
                  (state === 'done'
                    ? 'border-green-500 bg-green-50 text-green-600'
                    : state === 'active'
                      ? 'scale-110 border-brand bg-brand-soft text-brand'
                      : 'border-line bg-[#f5f6f8] text-muted')
                }
              >
                {state === 'done' ? (
                  <CheckIcon size={18} />
                ) : state === 'active' ? (
                  <SpinnerIcon size={16} />
                ) : (
                  <span className="text-xs font-bold">{i + 1}</span>
                )}
              </div>
              <span className="text-[11px] text-muted">Node {i + 1}</span>
            </div>
            {i < 2 && (
              <div
                className={
                  'mb-4 h-0.5 w-8 transition-colors duration-500 ' +
                  (display[i] === 'done' && display[i + 1] !== 'idle'
                    ? 'bg-green-400'
                    : 'bg-line')
                }
              />
            )}
          </div>
        ))}
      </div>

      <p className="text-center text-sm text-ink-soft">
        {done ? L.finished : msg}
      </p>
    </div>
  )
}
