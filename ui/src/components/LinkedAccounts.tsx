import { useEffect, useState } from 'react'
import { api, ApiError, type Account, type SocialProvider } from '../api.ts'
import { authErrorCode } from '../lib/authErrors.ts'
import { GithubLogo, GoogleLogo } from '../logos.tsx'
import { MailIcon } from '../icons.tsx'

function StatusDot({ on, label }: { on: boolean; label: string }) {
  return (
    <span className={'flex shrink-0 items-center gap-1.5 text-[13px] font-semibold ' + (on ? 'text-success' : 'text-muted')}>
      <span className={'h-2 w-2 rounded-full ' + (on ? 'bg-green-600' : 'bg-[#cbd3de]')} />
      {label}
    </span>
  )
}

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

  return (
    <section className="surface mb-5" aria-labelledby="linked-accounts-title">
      <h3 id="linked-accounts-title" className="mb-1 text-[17px] font-bold">Linked accounts</h3>
      <p className="mb-[18px] text-sm text-muted">Connect a social login to access the same account and wallets.</p>

      {!account && !error && <p className="mb-4 text-sm text-muted" role="status">Loading linked accounts…</p>}

      <div className="flex flex-col">
        {account && (
          <div className="flex items-center gap-3 border-b border-line py-3 last:border-none">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center border border-line bg-white text-ink-soft">
              <MailIcon />
            </span>
            <div className="min-w-0 flex-1">
              <div className="font-semibold">Email &amp; password</div>
              <div className="truncate text-[13px] text-muted">{account.email}</div>
            </div>
            <StatusDot on={account.has_password} label={account.has_password ? 'Enabled' : 'Not set'} />
          </div>
        )}
        {providers.map((provider) => {
          const identity = account?.identities.find((entry) => entry.provider === provider.id)
          const name = provider.id === 'google' ? 'Google' : 'GitHub'
          return (
            <div key={provider.id} className="flex items-center gap-3 border-b border-line py-3 last:border-none">
              <span className="flex h-9 w-9 shrink-0 items-center justify-center border border-line bg-white">
                {provider.id === 'google' ? <GoogleLogo /> : <GithubLogo />}
              </span>
              <div className="min-w-0 flex-1">
                <div className="font-semibold">{name}</div>
                <div className="text-[13px] text-muted">
                  {identity ? 'Signed in with ' + name : provider.enabled ? 'Not connected' : 'Not available on this deployment'}
                </div>
              </div>
              {identity ? (
                <StatusDot on label="Connected" />
              ) : provider.enabled ? (
                <button
                  className="shrink-0 border border-line bg-white px-4 py-2 text-sm font-semibold text-ink transition-colors hover:border-brand hover:text-brand disabled:cursor-not-allowed disabled:opacity-50"
                  disabled={busy}
                  onClick={() => void link(provider.id)}
                >
                  Connect
                </button>
              ) : (
                <StatusDot on={false} label="Unavailable" />
              )}
            </div>
          )
        })}
      </div>

      {error && <div role="alert" className="mt-4 bg-[#fdecec] px-3 py-2 text-sm text-[#d33a3a]">{error}</div>}
      {!account && error && (
        <button className="mt-3 text-sm font-semibold text-brand hover:underline" onClick={() => void load()}>Try again</button>
      )}

      {reauthProvider && account?.has_password && (
        <form className="mt-4 flex flex-col gap-2 border-t border-line pt-4" onSubmit={(event) => void confirmPassword(event)}>
          <label className="flex flex-col gap-2">
            <span className="text-sm font-semibold text-ink-soft">Confirm your password to continue</span>
            <div className="flex flex-col gap-2 sm:flex-row">
              <input className="field-input min-w-0 flex-1" type="password" autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} required />
              <button className="bg-brand px-4 py-2.5 font-semibold text-white transition-colors hover:bg-brand-deep disabled:cursor-not-allowed disabled:opacity-50" type="submit" disabled={busy || !password}>
                Confirm and connect
              </button>
            </div>
          </label>
        </form>
      )}
      {reauthProvider && !account?.has_password && (
        <p className="mt-4 text-sm text-muted">Sign out and sign in again, then return here to connect another provider.</p>
      )}
    </section>
  )
}
