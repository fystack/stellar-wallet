const storageKey = 'pending_approval'

// Approval email links must survive sign-in, including the OAuth round trip that drops the hash.
export function capturePendingApproval() {
  const match = window.location.hash.match(/^#\/approvals\/([^/?#]+)/)
  if (!match) return
  try {
    sessionStorage.setItem(storageKey, decodeURIComponent(match[1]))
  } catch {
    // Without storage the link still opens when the user is already signed in.
  }
}

export function takePendingApproval(): string {
  try {
    const requestId = sessionStorage.getItem(storageKey) ?? ''
    sessionStorage.removeItem(storageKey)
    return requestId
  } catch {
    return ''
  }
}
