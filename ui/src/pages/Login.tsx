import { useEffect, useState } from 'react'
import { FystackLogo, GithubLogo, GoogleLogo } from '../logos.tsx'
import { api, ApiError, type SocialProvider } from '../api.ts'
import type { InvitePreview } from '../types.ts'

function providerName(provider: string) {
  return provider === 'github' ? 'GitHub' : 'Google'
}

const inviteUnavailable: Record<Exclude<InvitePreview['status'], 'pending'>, string> = {
  accepted: 'This invite has already been used. Sign in to continue.',
  revoked: 'This invite was revoked. Ask the workspace owner for a new one.',
  expired: 'This invite has expired. Ask the workspace owner for a new one.',
}

export default function Login({
  onLogin,
  initialError = '',
  inviteToken = '',
}: {
  onLogin: (email: string) => void
  initialError?: string
  inviteToken?: string
}) {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState(initialError)
  const [providers, setProviders] = useState<SocialProvider[]>([])
  const [providerError, setProviderError] = useState('')
  useEffect(() => {
    void api.providers().then(setProviders).catch(() => setProviderError('Social sign-in is unavailable. You can still sign in with email.'))
  }, [])
  const [busy, setBusy] = useState(false)
  const [invite, setInvite] = useState<InvitePreview | null>(null)
  useEffect(() => {
    if (!inviteToken) return
    api.previewInvite(inviteToken).then((preview) => {
      setInvite(preview)
      if (preview.status === 'pending') {
        setEmail(preview.email)
        setMode(preview.hasAccount ? 'login' : 'register')
      } else {
        setError(inviteUnavailable[preview.status])
      }
    }).catch(() => setError('This invite link is invalid. Ask the workspace owner for a new one.'))
  }, [inviteToken])
  const pendingInvite = invite?.status === 'pending' ? invite : null
  // The invited account signs in with Google/GitHub only, so a password form would be a dead end.
  const inviteProviders = pendingInvite?.providers ?? []
  const socialOnly = !!pendingInvite?.hasAccount && !pendingInvite.hasPassword && inviteProviders.length > 0
  const socialProviders = (['google', 'github'] as const).filter(
    (provider) => !socialOnly || inviteProviders.includes(provider),
  )

  const isRegister = mode === 'register'
  const passwordMismatch = isRegister && confirmPassword.length > 0 && confirmPassword !== password
  const canSubmit =
    /\S+@\S+\.\S+/.test(email) && password.length >= 6 && (!isRegister || confirmPassword === password)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setBusy(true)
    try {
      const em =
        mode === 'login'
          ? await api.login(email.trim(), password)
          : await api.register(email.trim(), password)
      onLogin(em)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong')
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-full items-center justify-center bg-page p-4">
      <div className="w-full max-w-[400px]">
        <div className="mb-6 text-center sm:mb-8">
          <div className="mx-auto mb-4 w-fit sm:mb-5">
            <FystackLogo height={44} />
          </div>
          {pendingInvite ? (
            <>
              <h1 className="text-2xl font-extrabold">Join {pendingInvite.workspaceName}</h1>
              <p className="mt-1.5 text-[15px] text-muted">
                {pendingInvite.invitedBy} invited you.{' '}
                {socialOnly
                  ? `Continue with ${inviteProviders.map(providerName).join(' or ')} as`
                  : isRegister ? 'Create an account with' : 'Sign in with'}{' '}
                <strong className="font-semibold text-ink">{pendingInvite.email}</strong> to accept.
              </p>
            </>
          ) : (
            <>
              <h1 className="text-2xl font-extrabold">
                {isRegister ? 'Create your account' : 'Welcome back'}
              </h1>
              <p className="mt-1.5 text-[15px] text-muted">
                {isRegister
                  ? 'Set up your MPC Stellar wallet in a minute.'
                  : 'Sign in to access your MPC Stellar wallet.'}
              </p>
            </>
          )}
        </div>

        <div className="border border-line bg-white p-5 sm:p-7">
          {socialOnly && error && (
            <div role="alert" className="mb-4 bg-danger-soft px-3 py-2 text-sm text-danger">
              {error}
            </div>
          )}
          {!socialOnly && (
            <>
              <form onSubmit={handleSubmit} className="flex flex-col gap-4">
                <label className="flex flex-col gap-2">
                  <span className="text-sm font-semibold text-ink-soft">Email</span>
                  <input
                    className="field-input"
                    type="email"
                    placeholder="you@example.com"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    autoComplete="email"
                    autoFocus
                  />
                </label>

                <div className="flex flex-col gap-2">
                  <label htmlFor="login-password" className="text-sm font-semibold text-ink-soft">Password</label>
                  <input
                    id="login-password"
                    className="field-input"
                    type="password"
                    placeholder={isRegister ? 'Create a password' : 'Your password'}
                    autoComplete={isRegister ? 'new-password' : 'current-password'}
                    aria-describedby={isRegister ? 'login-password-hint' : undefined}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                  />
                  {isRegister && <span id="login-password-hint" className="text-xs text-muted">At least 6 characters.</span>}
                </div>

                {isRegister && (
                  <div className="flex flex-col gap-2">
                    <label htmlFor="login-confirm-password" className="text-sm font-semibold text-ink-soft">Confirm password</label>
                    <input
                      id="login-confirm-password"
                      className="field-input"
                      type="password"
                      placeholder="Repeat your password"
                      autoComplete="new-password"
                      value={confirmPassword}
                      onChange={(e) => setConfirmPassword(e.target.value)}
                      aria-invalid={passwordMismatch}
                      aria-describedby={passwordMismatch ? 'login-confirm-password-error' : undefined}
                    />
                    {passwordMismatch && <span id="login-confirm-password-error" className="text-xs text-danger">Passwords don't match.</span>}
                  </div>
                )}

                {error && (
                  <div role="alert" className="bg-danger-soft px-3 py-2 text-sm text-danger">
                    {error}
                  </div>
                )}

                <button
                  className="btn btn-primary btn-lg w-full !py-3.5"
                  type="submit"
                  disabled={!canSubmit || busy}
                >
                  {busy ? 'Please wait…' : mode === 'login' ? 'Sign in' : 'Create account'}
                </button>
              </form>

              <div className="my-5 flex items-center gap-3 text-xs text-muted">
                <span className="h-px flex-1 bg-line" />
                or
                <span className="h-px flex-1 bg-line" />
              </div>
            </>
          )}

          <div className="flex flex-col gap-3" aria-label="Social sign-in">
            {socialProviders.map((provider) => {
              const enabled = providers.some((entry) => entry.id === provider && entry.enabled)
              return <button key={provider} type="button" className="btn btn-secondary py-3" disabled={!enabled || busy} onClick={() => window.location.assign(api.oauthStartURL(provider))}>
                {provider === 'google' ? <GoogleLogo /> : <GithubLogo />}
                {isRegister && !socialOnly ? 'Sign up' : 'Continue'} with {providerName(provider)}
              </button>
            })}
            {providerError && <p role="status" className="text-sm text-muted">{providerError}</p>}
            {!providerError && providers.length > 0 && providers.every((provider) => !provider.enabled) && <p className="text-sm text-muted">Social sign-in is not configured on this deployment.</p>}
          </div>
        </div>

        {!socialOnly && (
          <p className="mt-5 text-center text-sm text-muted">
            {mode === 'login' ? "No account? " : 'Already have an account? '}
            <button
              onClick={() => {
                setMode(mode === 'login' ? 'register' : 'login')
                setError('')
                setConfirmPassword('')
              }}
              className="btn btn-link"
            >
              {mode === 'login' ? 'Create one' : 'Sign in'}
            </button>
          </p>
        )}
      </div>
    </div>
  )
}
