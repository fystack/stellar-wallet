const storageKey = 'pending_invite'

// The invite token must survive sign-in, including the OAuth round trip that drops the hash.
export function capturePendingInvite(): string {
  const match = window.location.hash.match(/^#\/invite\/([^/?#]+)/)
  if (match) {
    const token = decodeURIComponent(match[1])
    try {
      sessionStorage.setItem(storageKey, token)
    } catch {
      // Without storage the invite still works when accepted in this page load.
    }
    window.history.replaceState(null, '', window.location.pathname + window.location.search + '#/wallets')
    return token
  }
  try {
    return sessionStorage.getItem(storageKey) ?? ''
  } catch {
    return ''
  }
}

export function clearPendingInvite() {
  try {
    sessionStorage.removeItem(storageKey)
  } catch {
    // Nothing stored.
  }
}
