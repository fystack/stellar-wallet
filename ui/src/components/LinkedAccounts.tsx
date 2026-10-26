import { useEffect, useState } from 'react'
import { api, ApiError, type Account, type SocialProvider } from '../api.ts'
import { authErrorCode } from '../authErrors.ts'

export default function LinkedAccounts() {
  const [account, setAccount] = useState<Account | null>(null)
  const [providers, setProviders] = useState<SocialProvider[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [reauthProvider, setReauthProvider] = useState<SocialProvider['id'] | null>(null)
  const [password, setPassword] = useState('')

  async function load() {
    setError('')
    try {
      const [user, available] = await Promise.all([api.me(), api.providers()])
      setAccount(user)
      setProviders(available)
    } catch { setError('Unable to load linked accounts. Please try again.') }
  }
  useEffect(() => { void load() }, [])

  async function link(provider: SocialProvider['id']) {
    setBusy(true)
    setError('')
    try {
      const { url } = await api.linkProvider(provider)
      window.location.assign(url)
    } catch (error) {
      if (error instanceof ApiError && error.code === authErrorCode.reauthenticationRequired) setReauthProvider(provider)
      setError(error instanceof Error ? error.message : 'Unable to link account.')
      setBusy(false)
    }
  }

  async function confirmPassword(event: React.FormEvent) {
    event.preventDefault()
    if (!reauthProvider) return
    setBusy(true)
    try {
      await api.reauthenticate(password)
      setPassword('')
      await link(reauthProvider)
    } catch (error) {
      setError(error instanceof Error ? error.message : 'Unable to confirm password.')
      setBusy(false)
    }
  }

  return <section className="mb-6 border border-line bg-white p-5 sm:p-6" aria-labelledby="linked-accounts-title">
    <h2 id="linked-accounts-title" className="text-lg font-bold">Linked accounts</h2>
    <p className="mt-1 text-sm text-muted">Connect a social login to access the same account and wallets.</p>
    {account && <p className="mt-3 text-sm">{account.email}{account.has_password ? ' · Password login enabled' : ' · Social login'}</p>}
    {!account && !error && <p className="mt-4 text-sm" role="status">Loading linked accounts…</p>}
    <div className="mt-4 flex flex-col gap-3 sm:flex-row">
      {providers.map((provider) => {
        const identity = account?.identities.find((entry) => entry.provider === provider.id)
        const name = provider.id === 'google' ? 'Google' : 'GitHub'
        return <button key={provider.id} className="border border-line px-4 py-2 text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-50" disabled={busy || !provider.enabled || !!identity} onClick={() => void link(provider.id)}>
          {identity ? `${name} connected` : provider.enabled ? `Connect ${name}` : `${name} not configured`}
        </button>
      })}
    </div>
    {error && <p role="alert" className="mt-3 text-sm text-danger">{error}</p>}
    {!account && error && <button className="mt-3 text-sm font-semibold underline" onClick={() => void load()}>Try again</button>}
    {reauthProvider && account?.has_password && <form className="mt-4 flex max-w-md flex-col gap-3" onSubmit={(event) => void confirmPassword(event)}>
      <label className="text-sm font-semibold">Confirm your password
        <input className="field-input mt-2 w-full" type="password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} required />
      </label>
      <button className="cta" type="submit" disabled={busy || !password}>Confirm and connect</button>
    </form>}
    {reauthProvider && !account?.has_password && <p className="mt-3 text-sm text-muted">Sign out and sign in again, then return here to connect another provider.</p>}
  </section>
}
