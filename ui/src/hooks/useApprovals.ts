import { useCallback, useEffect, useRef, useState } from 'react'
import { api, ApiError } from '../api.ts'
import { toast } from '../toast.tsx'
import type { ApprovalPolicySettings, PaymentRequest } from '../types.ts'

export function useApprovals(workspaceId: string, onUnauthorized: () => void) {
  const [policy, setPolicy] = useState<ApprovalPolicySettings | null>(null)
  const [pending, setPending] = useState<PaymentRequest[]>([])
  const [loadedFor, setLoadedFor] = useState('')
  const [failed, setFailed] = useState(false)
  // Latest copy of every request seen live, including ones that left the pending list.
  const [updates, setUpdates] = useState<Record<string, PaymentRequest>>({})
  // History loads on first view and then stays current from live updates.
  const [history, setHistory] = useState<PaymentRequest[]>([])
  const [historyFor, setHistoryFor] = useState('')
  const [historyFailed, setHistoryFailed] = useState(false)
  const [historyNext, setHistoryNext] = useState<string | null>(null)
  const [loadingMoreHistory, setLoadingMoreHistory] = useState(false)
  // A response for a workspace the user has since left must not overwrite the current one.
  const currentWorkspace = useRef(workspaceId)
  currentWorkspace.current = workspaceId

  const reload = useCallback(async () => {
    if (!workspaceId) return
    setFailed(false)
    try {
      const [settings, requests] = await Promise.all([
        api.approvalPolicy(workspaceId),
        api.paymentRequests(workspaceId, 'pending'),
      ])
      if (currentWorkspace.current !== workspaceId) return
      setPolicy(settings)
      setPending(requests.items)
      setLoadedFor(workspaceId)
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) onUnauthorized()
      else if (currentWorkspace.current === workspaceId) setFailed(true)
    }
  }, [workspaceId, onUnauthorized])

  const reloadPolicy = useCallback(async () => {
    if (!workspaceId) return
    try {
      const settings = await api.approvalPolicy(workspaceId)
      if (currentWorkspace.current === workspaceId) setPolicy(settings)
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) onUnauthorized()
      else if (currentWorkspace.current === workspaceId) setFailed(true)
    }
  }, [workspaceId, onUnauthorized])

  const loadHistory = useCallback(async () => {
    if (!workspaceId) return
    setHistoryFailed(false)
    try {
      const page = await api.paymentRequests(workspaceId, 'history')
      if (currentWorkspace.current !== workspaceId) return
      setHistory(page.items)
      setHistoryNext(page.nextCursor)
      setHistoryFor(workspaceId)
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) onUnauthorized()
      else if (currentWorkspace.current === workspaceId) setHistoryFailed(true)
    }
  }, [workspaceId, onUnauthorized])

  const loadMoreHistory = useCallback(async () => {
    if (!workspaceId || !historyNext) return
    setLoadingMoreHistory(true)
    try {
      const page = await api.paymentRequests(workspaceId, 'history', historyNext)
      if (currentWorkspace.current !== workspaceId) return
      setHistory((current) => [...current, ...page.items.filter((request) => !current.some((item) => item.id === request.id))])
      setHistoryNext(page.nextCursor)
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) onUnauthorized()
      else toast.error(error instanceof Error ? error.message : 'Unable to load older requests.')
    } finally {
      setLoadingMoreHistory(false)
    }
  }, [workspaceId, historyNext, onUnauthorized])

  useEffect(() => {
    setPolicy(null)
    setPending([])
    setUpdates({})
    setHistory([])
    setHistoryNext(null)
    setHistoryFor('')
    setLoadedFor('')
    void reload()
  }, [reload])

  const applyUpdate = useCallback(
    (request: PaymentRequest) => {
      if (request.workspaceId !== workspaceId) return
      setUpdates((current) => ({ ...current, [request.id]: request }))
      if (request.status !== 'pending') setHistory((current) => [request, ...current.filter((item) => item.id !== request.id)])
      setPending((current) => {
        const others = current.filter((item) => item.id !== request.id)
        return request.status === 'pending' ? [request, ...others] : others
      })
    },
    [workspaceId],
  )

  return {
    policy: loadedFor === workspaceId ? policy : null,
    setPolicy,
    pending: loadedFor === workspaceId ? pending : [],
    loaded: loadedFor === workspaceId,
    failed,
    reload,
    reloadPolicy,
    applyUpdate,
    updates,
    history: historyFor === workspaceId ? history : [],
    historyLoaded: historyFor === workspaceId,
    historyFailed,
    loadHistory,
    hasMoreHistory: historyFor === workspaceId && !!historyNext,
    loadingMoreHistory,
    loadMoreHistory,
  }
}
