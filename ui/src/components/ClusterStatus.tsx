import { useEffect, useState } from 'react'
import { ShieldIcon } from '../icons.tsx'
import { api } from '../api.ts'

type Node = { name: string; region: string; online: boolean }

export default function ClusterStatus() {
  const [nodes, setNodes] = useState<Node[]>([])
  const [threshold, setThreshold] = useState('2-of-3')

  useEffect(() => {
    const load = () =>
      api
        .cluster()
        .then((r) => {
          setNodes(r.nodes)
          setThreshold(r.threshold)
        })
        .catch(() => {})
    load()
    const t = setInterval(load, 15000)
    return () => clearInterval(t)
  }, [])

  const online = nodes.filter((n) => n.online).length
  const total = nodes.length || 3
  const allOnline = online === total

  return (
    <div className="flex items-center gap-2.5 border border-line bg-white px-5 py-3.5 text-[15px] text-ink-soft">
      <span className={allOnline ? 'text-green-600' : 'text-[#d97706]'}>
        <ShieldIcon size={18} />
      </span>
      <span className="inline-flex gap-1.5">
        {Array.from({ length: total }).map((_, i) => (
          <i
            key={i}
            className={
              'inline-block h-2 w-2 rounded-full ' +
              (i < online ? 'bg-green-600' : 'bg-[#d5dbe4]')
            }
          />
        ))}
      </span>
      <strong className="text-ink">Protected by {threshold} MPC</strong>
      <span className="text-muted">
        · Key never assembled · {online} of {total} nodes online
      </span>
    </div>
  )
}
