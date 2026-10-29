import { useEffect, useState } from 'react'
import { GithubLogo, GoogleLogo } from '../logos.tsx'
import AuthLayout from '../components/AuthLayout.tsx'
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

  const switchMode = () => {
    setMode(mode === 'login' ? 'register' : 'login')
    setError('')
    setConfirmPassword('')
  }

  return (
    <AuthLayout>
      {pendingInvite ? (
        <>
          <h1 className="text-[34px] font-extrabold leading-tight">Join {pendingInvite.workspaceName}</h1>
          <p className="mt-2 text-[17px] text-ink-soft">
            {pendingInvite.invitedBy} invited you.{' '}
            {socialOnly
              ? `Continue with ${inviteProviders.map(providerName).join(' or ')} as`
              : isRegister ? 'Create an account with' : 'Sign in with'}{' '}
            <strong className="font-semibold text-ink">{pendingInvite.email}</strong> to accept.
          </p>
        </>
      ) : (
        <>
          <h1 className="text-[34px] font-extrabold leading-tight">{isRegister ? 'Create your account' : 'Sign in'}</h1>
          <p className="mt-2 text-[17px] text-ink-soft">
            The self-hosted MPC wallet for Stellar. Your keys stay split across your own nodes.
          </p>
        </>
      )}

      <div className={`mt-7 grid gap-3 ${socialProviders.length > 1 ? 'grid-cols-2' : ''}`} aria-label="Social sign-in">
        {socialProviders.map((provider) => {
          const enabled = providers.some((entry) => entry.id === provider && entry.enabled)
          return (
            <button
              key={provider}
              type="button"
              className="btn btn-secondary btn-lg"
              aria-label={`${isRegister && !socialOnly ? 'Sign up' : 'Continue'} with ${providerName(provider)}`}
              disabled={!enabled || busy}
              onClick={() => window.location.assign(api.oauthStartURL(provider))}
            >
              {provider === 'google' ? <GoogleLogo /> : <GithubLogo />}
              {providerName(provider)}
            </button>
          )
        })}
      </div>
      {providerError && <p role="status" className="mt-3 text-sm text-muted">{providerError}</p>}
      {!providerError && providers.length > 0 && providers.every((provider) => !provider.enabled) && (
        <p className="mt-3 text-sm text-muted">Social sign-in is not configured on this deployment.</p>
      )}

      {socialOnly && error && (
        <div role="alert" className="mt-5 bg-danger-soft px-3 py-2 text-sm text-danger">
          {error}
        </div>
      )}

      {!socialOnly && (
        <>
          <div className="my-6 flex items-center gap-3 text-sm text-muted">
            <span className="h-px flex-1 bg-line" />
            or with email
            <span className="h-px flex-1 bg-line" />
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-5">
            <label className="flex flex-col gap-2">
              <span className="text-[15px] font-semibold text-ink">Email</span>
              <input
                className="field-input"
                type="email"
                placeholder="you@company.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                autoComplete="email"
                autoFocus
              />
            </label>

            <div className="flex flex-col gap-2">
              <label htmlFor="login-password" className="text-[15px] font-semibold text-ink">Password</label>
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
                <label htmlFor="login-confirm-password" className="text-[15px] font-semibold text-ink">Confirm password</label>
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

            <button className="btn btn-primary btn-lg mt-2 w-full" type="submit" disabled={!canSubmit || busy}>
              {busy ? 'Please wait…' : mode === 'login' ? 'Sign in' : 'Create account'}
            </button>
          </form>

          <p className="mt-6 text-center text-[15px] text-ink-soft">
            {mode === 'login' ? 'New here? ' : 'Already have an account? '}
            <button onClick={switchMode} className="btn btn-link font-bold">
              {mode === 'login' ? 'Create an account' : 'Sign in'}
            </button>
          </p>
        </>
      )}
    </AuthLayout>
  )
}
