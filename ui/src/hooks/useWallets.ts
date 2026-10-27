import { useCallback, useEffect, useState } from 'react'
import { api, ApiError } from '../api.ts'
import type { Chain, Wallet } from '../types.ts'

// workspaceId re-fetches the list after the active workspace changes; empty waits for it.
export function useWallets(onUnauthorized: () => void, workspaceId: string) {
  const [wallets, setWallets] = useState<Wallet[]>([])
  const [loadedFor, setLoadedFor] = useState('')
  const [failedFor, setFailedFor] = useState('')

  const reload = useCallback(async () => {
    if (!workspaceId) return
    setFailedFor('')
    try {
      setWallets(await api.listWallets(workspaceId))
      setLoadedFor(workspaceId)
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) onUnauthorized()
      else setFailedFor(workspaceId)
    }
  }, [onUnauthorized, workspaceId])

  useEffect(() => {
    void reload()
  }, [reload])

  const applyUpdate = useCallback((wallet: Wallet) => {
    setWallets((current) => current.map((w) => (w.id === wallet.id ? wallet : w)))
  }, [])

  // Keygen runs asynchronously; the caller watches the returned wallet id until it is ready.
  const create = useCallback(
    async (name: string, chain: Chain) => {
      const wallet = await api.createWallet(workspaceId, name, chain)
      await reload()
      return wallet.id
    },
    [reload, workspaceId],
  )

  const remove = useCallback(
    async (id: string) => {
      await api.deleteWallet(id)
      await reload()
    },
    [reload],
  )

  // False until the active workspace's list arrives, so pages don't flash their empty state.
  const loaded = workspaceId !== '' && loadedFor === workspaceId
  const failed = workspaceId !== '' && failedFor === workspaceId
  return { wallets, loaded, failed, reload, applyUpdate, create, remove }
}
