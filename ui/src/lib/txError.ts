// Maps raw Stellar/Horizon result codes (as stored in tx.error, e.g.
// "broadcast: tx_failed (op_no_trust)") to a human-readable explanation plus,
// where possible, an action the user can take to resolve it themselves.
//
// The raw code is still shown underneath so it stays useful for debugging.

export type FriendlyTxError = {
  title: string // short, human message
  hint?: string // optional actionable suggestion
}

// Keyed by the Horizon result code. Operation codes (op_*) are checked before
// transaction codes (tx_*) because they carry the more specific reason.
const CODES: Record<string, FriendlyTxError> = {
  // --- operation-level codes (most specific) ---
  op_no_trust: {
    title: 'The recipient has not added a trustline for this asset.',
    hint: 'They must add a trustline for this token before they can receive it. Native XLM never needs one.',
  },
  op_underfunded: {
    title: 'Not enough balance to cover this amount.',
    hint: 'Reduce the amount, or top up the wallet and try again.',
  },
  op_no_destination: {
    title: 'The destination account does not exist yet.',
    hint: 'For a brand-new account, send at least 1 XLM first to create it on-chain.',
  },
  op_line_full: {
    title: "The recipient's trustline is already at its limit.",
    hint: 'The recipient must raise their trustline limit for this asset.',
  },
  op_no_issuer: {
    title: 'The issuer of this asset does not exist.',
    hint: 'Double-check the asset issuer address.',
  },
  op_malformed: {
    title: 'The transaction was rejected as malformed.',
    hint: 'Check the destination address and asset details.',
  },
  op_low_reserve: {
    title: 'The account would drop below the minimum XLM reserve.',
    hint: 'Keep enough XLM to cover the base reserve, or send a smaller amount.',
  },
  // --- transaction-level codes ---
  tx_insufficient_balance: {
    title: 'Not enough XLM to cover the amount plus the network fee.',
    hint: 'Top up XLM and try again.',
  },
  tx_insufficient_fee: {
    title: 'The network fee was too low.',
    hint: 'The network is busy — retry, fees may have risen.',
  },
  tx_bad_seq: {
    title: 'The transaction used a stale sequence number.',
    hint: 'This usually resolves on retry.',
  },
  tx_no_source_account: {
    title: 'The sending account does not exist on-chain yet.',
    hint: 'Fund the wallet with XLM first to activate it.',
  },
  tx_too_late: {
    title: 'The transaction expired before it was submitted.',
    hint: 'Just try sending again.',
  },
  tx_bad_auth: {
    title: 'The transaction signature was not accepted.',
    hint: 'Try sending again.',
  },
}

// friendlyTxError turns a stored error string into a readable message.
// Returns null when there is nothing to map, so callers can fall back to the raw
// string (or an "Unknown error" placeholder).
export function friendlyTxError(raw?: string): FriendlyTxError | null {
  if (!raw) return null
  const lower = raw.toLowerCase()
  // Prefer operation codes (op_*), then transaction codes (tx_*).
  const opMatch = Object.keys(CODES).find(
    (k) => k.startsWith('op_') && lower.includes(k),
  )
  if (opMatch) return CODES[opMatch]
  const txMatch = Object.keys(CODES).find(
    (k) => k.startsWith('tx_') && lower.includes(k),
  )
  if (txMatch) return CODES[txMatch]
  return null
}
