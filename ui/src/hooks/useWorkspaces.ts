import { useCallback, useEffect, useState } from 'react'
import { api, ApiError } from '../api.ts'
import { toast } from '../toast.tsx'
import type { Workspace, WorkspaceInvite } from '../types.ts'

export function useWorkspaces(onUnauthorized: () => void) {
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [activeId, setActiveId] = useState('')
  const [selected, setSelected] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState('')
  const [invitations, setInvitations] = useState<WorkspaceInvite[]>([])
  const [invitationsVerified, setInvitationsVerified] = useState(false)

  // Invitations are secondary: a failure here must not block loading workspaces.
  const reloadInvitations = useCallback(async () => {
    try {
      const result = await api.invitations()
      setInvitations(result.invites)
      setInvitationsVerified(result.verified)
    } catch {
      setInvitations([])
    }
  }, [])

  const reload = useCallback(async () => {
    setError('')
    try {
      const [list] = await Promise.all([api.workspaces(), reloadInvitations()])
      setWorkspaces(list.workspaces)
      setActiveId(list.activeId)
      setSelected(list.selected)
      setLoaded(true)
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) onUnauthorized()
      else setError('Unable to load your workspaces. Please try again.')
    }
  }, [onUnauthorized, reloadInvitations])

  useEffect(() => {
    void reload()
  }, [reload])

  const switchTo = useCallback(async (id: string) => {
    try {
      await api.switchWorkspace(id)
      setActiveId(id)
      setSelected(true)
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Unable to switch workspace.')
    }
  }, [])

  const create = useCallback(
    async (name: string) => {
      const result = await api.createWorkspace(name)
      await reload()
      return result
    },
    [reload],
  )

  const acceptInvitation = useCallback(
    async (id: string) => {
      const result = await api.acceptInvitation(id)
      await reload()
      return result
    },
    [reload],
  )

  const declineInvitation = useCallback(
    async (id: string) => {
      await api.declineInvitation(id)
      await reloadInvitations()
    },
    [reloadInvitations],
  )

  const active = workspaces.find((workspace) => workspace.id === activeId) ?? null
  // Ask a freshly signed-in user to pick when they have a real choice or invitations to answer.
  const needsPick = loaded && !selected && (workspaces.length >= 2 || invitations.length > 0)
  return {
    workspaces,
    active,
    activeId,
    loaded,
    error,
    needsPick,
    invitations,
    invitationsVerified,
    reload,
    switchTo,
    create,
    acceptInvitation,
    declineInvitation,
  }
}
