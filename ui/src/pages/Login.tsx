import { useEffect, useState } from 'react'
import { FystackLogo, GithubLogo, GoogleLogo } from '../logos.tsx'
import { api, ApiError, type SocialProvider } from '../api.ts'

export default function Login({ onLogin, initialError = '' }: { onLogin: (email: string) => void; initialError?: string }) {
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
    <div className="flex min-h-full items-center justify-center bg-[#f7f8fa] p-4">
      <div className="w-full max-w-[400px]">
        <div className="mb-6 text-center sm:mb-8">
          <div className="mx-auto mb-4 w-fit sm:mb-5">
            <FystackLogo height={44} />
          </div>
          <h1 className="text-2xl font-extrabold">
            {isRegister ? 'Create your account' : 'Welcome back'}
          </h1>
          <p className="mt-1.5 text-[15px] text-muted">
            {isRegister
              ? 'Set up your MPC Stellar wallet in a minute.'
              : 'Sign in to access your MPC Stellar wallet.'}
          </p>
        </div>

        <div className="border border-line bg-white p-5 sm:p-7">
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

            <label className="flex flex-col gap-2">
              <span className="text-sm font-semibold text-ink-soft">Password</span>
              <input
                className="field-input"
                type="password"
                placeholder={isRegister ? 'Create a password' : 'Your password'}
                autoComplete={isRegister ? 'new-password' : 'current-password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
              {isRegister && <span className="text-xs text-muted">At least 6 characters.</span>}
            </label>

            {isRegister && (
              <label className="flex flex-col gap-2">
                <span className="text-sm font-semibold text-ink-soft">Confirm password</span>
                <input
                  className="field-input"
                  type="password"
                  placeholder="Repeat your password"
                  autoComplete="new-password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  aria-invalid={passwordMismatch}
                />
                {passwordMismatch && <span className="text-xs text-[#d33a3a]">Passwords don't match.</span>}
              </label>
            )}

            {error && (
              <div role="alert" className="bg-[#fdecec] px-3 py-2 text-sm text-[#d33a3a]">
                {error}
              </div>
            )}

            <button
              className="cta w-full !py-3.5"
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

          <div className="flex flex-col gap-3" aria-label="Social sign-in">
            {(['google', 'github'] as const).map((provider) => {
              const enabled = providers.some((entry) => entry.id === provider && entry.enabled)
              return <button key={provider} type="button" className="flex items-center justify-center gap-2.5 border border-line bg-white px-4 py-3 text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-50" disabled={!enabled || busy} onClick={() => window.location.assign(api.oauthStartURL(provider))}>
                {provider === 'google' ? <GoogleLogo /> : <GithubLogo />}
                {isRegister ? 'Sign up' : 'Continue'} with {provider === 'google' ? 'Google' : 'GitHub'}
              </button>
            })}
            {providerError && <p role="status" className="text-sm text-muted">{providerError}</p>}
            {!providerError && providers.length > 0 && providers.every((provider) => !provider.enabled) && <p className="text-sm text-muted">Social sign-in is not configured on this deployment.</p>}
          </div>
        </div>

        <p className="mt-5 text-center text-sm text-muted">
          {mode === 'login' ? "No account? " : 'Already have an account? '}
          <button
            onClick={() => {
              setMode(mode === 'login' ? 'register' : 'login')
              setError('')
              setConfirmPassword('')
            }}
            className="font-semibold text-brand hover:underline"
          >
            {mode === 'login' ? 'Create one' : 'Sign in'}
          </button>
        </p>
      </div>
    </div>
  )
}
