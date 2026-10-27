// localStorage can throw in private mode or with blocked storage; preferences then just don't persist.
export function readPreference(key: string): string | null {
  try {
    return localStorage.getItem(key)
  } catch {
    return null
  }
}

export function writePreference(key: string, value: string) {
  try {
    localStorage.setItem(key, value)
  } catch {
    // Kept in memory for this page load.
  }
}
