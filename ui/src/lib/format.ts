// Format a raw chain balance string into a compact, human amount.
// "10000.0000000" -> "10,000", "0.00" -> "0", "25.5000000" -> "25.5"
export function formatAmount(raw: string): string {
  const n = Number(raw)
  if (!Number.isFinite(n)) return raw
  return n.toLocaleString('en-US', { maximumFractionDigits: 4 })
}

// Format a USD value: 1234.5 -> "$1,234.50".
export function formatUsd(n: number): string {
  if (!Number.isFinite(n)) return ''
  return (
    '$' +
    n.toLocaleString('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })
  )
}

// Trim an address in the middle, keeping head and tail: "GABC12…WXYZ89".
export function shortAddress(a: string, head = 6, tail = 6): string {
  if (!a || a.length <= head + tail + 1) return a
  return `${a.slice(0, head)}…${a.slice(-tail)}`
}
