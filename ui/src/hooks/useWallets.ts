import { useCallback, useEffect, useState } from 'react'
import { api, ApiError } from '../api.ts'
import { toast } from '../toast.tsx'
import type { Chain, Wallet } from '../types.ts'

export function useWallets(onUnauthorized: () => void) {
  const [wallets, setWallets] = useState<Wallet[]>([])

  const reload = useCallback(async () => {
    try {
      setWallets(await api.listWallets())
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) onUnauthorized()
      else toast.error('Unable to load wallets. Please try again.')
    }
  }, [onUnauthorized])

  useEffect(() => {
    void reload()
  }, [reload])

  const applyUpdate = useCallback((wallet: Wallet) => {
    setWallets((current) => current.map((w) => (w.id === wallet.id ? wallet : w)))
  }, [])

  // Keygen runs asynchronously; the caller watches the returned wallet id until it is ready.
  const create = useCallback(
    async (name: string, chain: Chain) => {
      const wallet = await api.createWallet(name, chain)
      await reload()
      return wallet.id
    },
    [reload],
  )

  const remove = useCallback(
    async (id: string) => {
      await api.deleteWallet(id)
      await reload()
    },
    [reload],
  )

  return { wallets, applyUpdate, create, remove }
}
