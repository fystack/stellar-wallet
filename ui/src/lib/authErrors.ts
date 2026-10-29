export const authErrorCode = {
  accountLinkRequired: 'account_link_required',
  identityAlreadyLinked: 'identity_already_linked',
  invalidOAuthState: 'invalid_oauth_state',
  oauthCancelled: 'oauth_cancelled',
  reauthenticationRequired: 'reauthentication_required',
  authenticationRequired: 'authentication_required',
  invalidLinkSession: 'invalid_link_session',
  providerUnavailable: 'provider_unavailable',
} as const

const messages: Record<string, string> = {
  [authErrorCode.accountLinkRequired]: 'This email already has an account. Sign in with your existing method, then link Google or GitHub in Settings.',
  [authErrorCode.identityAlreadyLinked]: 'This login method is already linked to an account.',
  [authErrorCode.invalidOAuthState]: 'This sign-in request expired or was started in another browser. Please try again.',
  [authErrorCode.oauthCancelled]: 'Social sign-in was cancelled. Please try again.',
  [authErrorCode.reauthenticationRequired]: 'Sign in again or confirm your password in Settings before linking a login method.',
  [authErrorCode.authenticationRequired]: 'Sign in again before linking a login method.',
  [authErrorCode.invalidLinkSession]: 'Your session changed. Start account linking again from Settings.',
  [authErrorCode.providerUnavailable]: 'This login provider is not configured on this deployment.',
}

export function consumeAuthError(): string {
  const url = new URL(window.location.href)
  const code = url.searchParams.get('auth_error')
  if (!code) return ''
  url.searchParams.delete('auth_error')
  window.history.replaceState(null, '', url.pathname + url.search + url.hash)
  return messages[code] ?? 'Unable to complete social sign-in. Please try again.'
}
