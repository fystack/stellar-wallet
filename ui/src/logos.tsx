import { useId } from 'react'

type Props = { size?: number }

// App brand mark — a shield (security) holding 3 nodes (the MPC cluster).
export function AppLogo({ size = 46 }: Props) {
  const id = useId()
  return (
    <svg width={size} height={size} viewBox="0 0 44 44">
      <defs>
        <linearGradient id={id} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#3b82ff" />
          <stop offset="100%" stopColor="#0b53e6" />
        </linearGradient>
      </defs>
      <rect width="44" height="44" rx="11" fill={`url(#${id})`} />
      <path
        d="M22 9.5l8.5 3.2v7c0 6.4-4.1 9.7-8.5 11.3-4.4-1.6-8.5-4.9-8.5-11.3v-7L22 9.5z"
        fill="rgba(255,255,255,0.16)"
        stroke="#fff"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
      <g fill="#fff">
        <circle cx="18" cy="21" r="1.7" />
        <circle cx="22" cy="21" r="1.7" />
        <circle cx="26" cy="21" r="1.7" />
        <path
          d="M18 21h8"
          stroke="#fff"
          strokeWidth="1.2"
          strokeOpacity="0.55"
        />
      </g>
    </svg>
  )
}

// --- Chain / native marks ---

export function StellarLogo({ size = 28 }: Props) {
  // Black disc with a clean 4-point sparkle.
  return (
    <svg width={size} height={size} viewBox="0 0 40 40">
      <circle cx="20" cy="20" r="20" fill="#0d0d12" />
      <path
        d="M20 8c1 7.2 3.8 10 11 11-7.2 1-10 3.8-11 11-1-7.2-3.8-10-11-11 7.2-1 10-3.8 11-11z"
        fill="#fff"
      />
    </svg>
  )
}

export function SolanaLogo({ size = 28 }: Props) {
  const id = useId()
  return (
    <svg width={size} height={size} viewBox="0 0 40 40">
      <defs>
        <linearGradient id={id} x1="6" y1="28" x2="34" y2="12" gradientUnits="userSpaceOnUse">
          <stop offset="0%" stopColor="#9945FF" />
          <stop offset="100%" stopColor="#14F195" />
        </linearGradient>
      </defs>
      <circle cx="20" cy="20" r="20" fill="#131316" />
      <g fill={`url(#${id})`}>
        <path d="M11 11h20l-4 4H7z" />
        <path d="M7 17.5h20l4 4H11z" />
        <path d="M11 24h20l-4 4H7z" />
      </g>
    </svg>
  )
}

// --- Token marks ---

function CoinBadge({
  size,
  bg,
  children,
}: {
  size: number
  bg: string
  children: React.ReactNode
}) {
  return (
    <svg width={size} height={size} viewBox="0 0 40 40">
      <circle cx="20" cy="20" r="20" fill={bg} />
      {children}
    </svg>
  )
}

export function UsdcLogo({ size = 28 }: Props) {
  return (
    <CoinBadge size={size} bg="#2775CA">
      {/* USDC: dollar sign inside a broken ring */}
      <circle
        cx="20"
        cy="20"
        r="12.5"
        fill="none"
        stroke="#fff"
        strokeWidth="2.2"
        strokeDasharray="55 20"
        strokeLinecap="round"
        transform="rotate(-45 20 20)"
      />
      <text
        x="20"
        y="26.5"
        textAnchor="middle"
        fontSize="17"
        fontWeight="700"
        fill="#fff"
        fontFamily="Arial, sans-serif"
      >
        $
      </text>
    </CoinBadge>
  )
}

export function UsdtLogo({ size = 28 }: Props) {
  return (
    <CoinBadge size={size} bg="#26A17B">
      {/* Tether ₮ drawn as strokes (font-independent) */}
      <g fill="#fff">
        <rect x="10" y="12" width="20" height="3.6" rx="0.5" />
        <rect x="18.1" y="12" width="3.8" height="16" rx="0.5" />
        <rect x="13" y="18.5" width="14" height="3" rx="0.5" />
      </g>
    </CoinBadge>
  )
}

// --- Dispatchers ---

export function ChainLogo({ chain, size = 28 }: { chain: string; size?: number }) {
  if (chain === 'solana') return <SolanaLogo size={size} />
  return <StellarLogo size={size} />
}

export function TokenLogo({ symbol, size = 28 }: { symbol: string; size?: number }) {
  switch (symbol) {
    case 'SOL':
      return <SolanaLogo size={size} />
    case 'USDC':
      return <UsdcLogo size={size} />
    case 'USDT':
      return <UsdtLogo size={size} />
    case 'XLM':
      return <StellarLogo size={size} />
    default:
      // Unknown custom asset — neutral monogram badge.
      return <MonogramLogo symbol={symbol} size={size} />
  }
}

function MonogramLogo({ symbol, size = 28 }: { symbol: string; size?: number }) {
  return (
    <CoinBadge size={size} bg="#5b6b82">
      <text
        x="20"
        y="26"
        textAnchor="middle"
        fontSize="16"
        fontWeight="700"
        fill="#fff"
        fontFamily="Arial, sans-serif"
      >
        {(symbol[0] ?? '?').toUpperCase()}
      </text>
    </CoinBadge>
  )
}

// Tokens available per chain (native first).
export const chainTokens: Record<string, string[]> = {
  stellar: ['XLM', 'USDC', 'USDT'],
  solana: ['SOL', 'USDC', 'USDT'],
}
