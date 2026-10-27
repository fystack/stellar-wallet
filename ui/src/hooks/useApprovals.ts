import { useCallback, useEffect, useRef, useState } from 'react'
import { api, ApiError } from '../api.ts'
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
      setPending(requests)
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
      const requests = await api.paymentRequests(workspaceId, 'history')
      if (currentWorkspace.current !== workspaceId) return
      setHistory(requests)
      setHistoryFor(workspaceId)
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) onUnauthorized()
      else if (currentWorkspace.current === workspaceId) setHistoryFailed(true)
    }
  }, [workspaceId, onUnauthorized])

  useEffect(() => {
    setPolicy(null)
    setPending([])
    setUpdates({})
    setHistory([])
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
  }
}
