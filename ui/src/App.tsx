import { useEffect, useRef, useState } from 'react'
import Login from './pages/Login.tsx'
import AppShell from './components/AppShell.tsx'
import { Toaster } from './toast.tsx'
import { useAuth } from './hooks/useAuth.ts'
import { capturePendingInvite, clearPendingInvite } from './lib/pendingInvite.ts'
import { capturePendingApproval } from './lib/pendingApproval.ts'

capturePendingApproval()

export default function App() {
  const auth = useAuth()
  const [inviteToken, setInviteToken] = useState(capturePendingInvite)
  const signedIn = useRef(false)
  signedIn.current = !!auth.user

  // Pasting an invite or approval link into an open tab only changes the hash, without a reload.
  // Signed in, the app's own navigation also writes #/approvals/:id, so only a signed-out tab keeps it for later.
  useEffect(() => {
    function onHashChange() {
      if (window.location.hash.startsWith('#/invite/')) setInviteToken(capturePendingInvite())
      if (!signedIn.current) capturePendingApproval()
    }
    window.addEventListener('hashchange', onHashChange)
    return () => window.removeEventListener('hashchange', onHashChange)
  }, [])

  function dismissInvite() {
    clearPendingInvite()
    setInviteToken('')
  }

  return (
    <>
      {auth.loading ? (
        <div className="grid min-h-screen place-items-center" role="status">
          Checking your session…
        </div>
      ) : auth.error ? (
        <div className="grid min-h-screen place-items-center p-6">
          <div role="alert">
            <p>{auth.error}</p>
            <button className="btn btn-primary btn-lg mt-4" onClick={() => void auth.restore()}>
              Try again
            </button>
          </div>
        </div>
      ) : auth.user ? (
        <AppShell
          user={auth.user}
          notice={auth.notice}
          onDismissNotice={auth.dismissNotice}
          signingOut={auth.signingOut}
          onSignOut={auth.signOut}
          onSessionExpired={auth.expire}
          inviteToken={inviteToken}
          onInviteHandled={dismissInvite}
        />
      ) : (
        <Login initialError={auth.notice} onLogin={auth.signIn} inviteToken={inviteToken} />
      )}
      <Toaster />
    </>
  )
}
