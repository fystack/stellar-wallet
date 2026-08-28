import { useState } from 'react'
import { FystackLogo } from '../logos.tsx'
import { api, ApiError } from '../api.ts'

export default function Login({ onLogin }: { onLogin: (email: string) => void }) {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const canSubmit = /\S+@\S+\.\S+/.test(email) && password.length >= 6

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
            {mode === 'login' ? 'Sign in' : 'Create your account'}
          </h1>
          <p className="mt-1.5 text-[15px] text-muted">
            {mode === 'login'
              ? 'Access your MPC Stellar wallet.'
              : 'Set up your MPC Stellar wallet.'}
          </p>
        </div>

        <form
          onSubmit={handleSubmit}
          className="flex flex-col gap-4 border border-line bg-white p-5 sm:p-7"
        >
          <label className="flex flex-col gap-2">
            <span className="text-sm font-semibold text-ink-soft">Email</span>
            <input
              className="field-input"
              type="email"
              placeholder="you@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoFocus
            />
          </label>

          <label className="flex flex-col gap-2">
            <span className="text-sm font-semibold text-ink-soft">Password</span>
            <input
              className="field-input"
              type="password"
              placeholder="At least 6 characters"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </label>

          {error && (
            <div className="bg-[#fdecec] px-3 py-2 text-sm text-[#d33a3a]">
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

        <p className="mt-5 text-center text-sm text-muted">
          {mode === 'login' ? "No account? " : 'Already have an account? '}
          <button
            onClick={() => {
              setMode(mode === 'login' ? 'register' : 'login')
              setError('')
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
