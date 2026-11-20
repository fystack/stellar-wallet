import { useEffect, useState } from 'react'
import { api } from '../api.ts'

type Node = { name: string; region: string; online: boolean }
type Cluster = { nodes: Node[]; signers: number }

const REFRESH_MS = 15_000

// "2-of-3" -> 2; signing needs that many nodes, keygen needs all of them.
function signersNeeded(threshold: string) {
  const needed = Number.parseInt(threshold, 10)
  return Number.isFinite(needed) && needed > 0 ? needed : 2
}

function summary(online: number, total: number, signers: number) {
  if (online === total) return { tone: 'bg-success', text: `All ${total} nodes online`, detail: `Any ${signers} can sign a transaction.` }
  if (online >= signers) {
    return { tone: 'bg-warning', text: `${online} of ${total} nodes online`, detail: `Signing works. New wallets need all ${total} nodes.` }
  }
  return { tone: 'bg-danger', text: `${online} of ${total} nodes online`, detail: `Signing and new wallets are paused until ${signers} nodes are online.` }
}

export default function ClusterStatus({ className = '' }: { className?: string }) {
  const [cluster, setCluster] = useState<Cluster | null>(null)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let current = true
    const load = () =>
      api
        .cluster()
        .then((response) => {
          if (!current) return
          setCluster({ nodes: response.nodes, signers: signersNeeded(response.threshold) })
          setFailed(false)
        })
        .catch(() => current && setFailed(true))
    void load()
    const timer = setInterval(() => void load(), REFRESH_MS)
    return () => {
      current = false
      clearInterval(timer)
    }
  }, [])

  if (!cluster) {
    return (
      <section aria-label="MPC cluster" className={'surface flex items-center gap-3 !py-4 ' + className}>
        {failed ? (
          <p role="status" className="flex items-center gap-2 text-sm text-ink-soft">
            <span className="dot bg-line-strong" /> Cluster status unavailable
          </p>
        ) : (
          <div role="status" aria-label="Loading cluster status" className="skeleton h-5 w-56" />
        )}
      </section>
    )
  }

  const total = cluster.nodes.length || 3
  const online = cluster.nodes.filter((node) => node.online).length
  const state = summary(online, total, cluster.signers)

  return (
    <section
      aria-label="MPC cluster"
      className={'surface flex flex-col gap-3 !py-4 sm:flex-row sm:items-center sm:justify-between ' + className}
    >
      <div className="min-w-0">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted">MPC cluster · {cluster.signers}-of-{total}</p>
        <p role="status" className="mt-1 flex items-center gap-2 text-sm">
          <span className={'dot ' + state.tone} />
          <span className="font-semibold text-ink">{state.text}</span>
          <span className="hidden text-muted sm:inline">{state.detail}</span>
        </p>
        <p className="mt-0.5 text-xs text-muted sm:hidden">{state.detail}</p>
      </div>
      <ul className="flex flex-wrap gap-2" aria-label="Nodes">
        {cluster.nodes.map((node) => (
          <li
            key={node.name}
            className="flex items-center gap-2 border border-line bg-page px-2.5 py-1 font-mono text-xs text-ink-soft"
          >
            <span className={'dot ' + (node.online ? 'bg-success' : 'bg-line-strong')} />
            {node.name}
            <span className="sr-only">{node.online ? 'online' : 'offline'}</span>
          </li>
        ))}
      </ul>
    </section>
  )
}
