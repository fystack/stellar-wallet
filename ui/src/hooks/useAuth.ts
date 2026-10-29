import { useCallback, useEffect, useState } from 'react'
import { api, ApiError, clearLegacySession } from '../api.ts'
import { consumeAuthError } from '../lib/authErrors.ts'
import { toast } from '../toast.tsx'

export function useAuth() {
  const [user, setUser] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [signingOut, setSigningOut] = useState(false)

  const restore = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const account = await api.me()
      setUser(account.email)
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) setUser(null)
      else setError('Unable to reach the wallet service. Please try again.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    const authError = consumeAuthError()
    if (authError) setNotice(authError)
    clearLegacySession()
    void restore()
  }, [restore])

  const signIn = useCallback((email: string) => {
    setNotice('')
    setUser(email)
  }, [])

  const expire = useCallback(() => {
    setNotice('')
    setUser(null)
  }, [])

  // Returns whether the server session was ended; the caller then clears its own state.
  const signOut = useCallback(async () => {
    setSigningOut(true)
    try {
      await api.logout()
      return true
    } catch {
      toast.error('Unable to sign out. Please try again.')
      return false
    } finally {
      setSigningOut(false)
    }
  }, [])

  return {
    user,
    loading,
    error,
    notice,
    signingOut,
    restore,
    signIn,
    signOut,
    expire,
    dismissNotice: () => setNotice(''),
  }
}
