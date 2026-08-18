import { useId } from 'react'

type Props = { size?: number }

// App brand mark — a 3-node mesh forming a triangle: the 2-of-3 MPC cluster.
export function AppLogo({ size = 46 }: Props) {
  const id = useId()
  const glow = useId()
  // Triangle vertices (the three signing nodes).
  const nodes = [
    { x: 22, y: 13.5 },
    { x: 13, y: 29.5 },
    { x: 31, y: 29.5 },
  ]
  return (
    <svg width={size} height={size} viewBox="0 0 44 44">
      <defs>
        <linearGradient id={id} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#3b82ff" />
          <stop offset="100%" stopColor="#0b53e6" />
        </linearGradient>
        <radialGradient id={glow} cx="50%" cy="30%" r="75%">
          <stop offset="0%" stopColor="#fff" stopOpacity="0.22" />
          <stop offset="100%" stopColor="#fff" stopOpacity="0" />
        </radialGradient>
      </defs>
      <rect width="44" height="44" rx="12" fill={`url(#${id})`} />
      <rect width="44" height="44" rx="12" fill={`url(#${glow})`} />
      {/* edges */}
      <g stroke="#fff" strokeOpacity="0.55" strokeWidth="1.5">
        <path d={`M${nodes[0].x} ${nodes[0].y}L${nodes[1].x} ${nodes[1].y}`} />
        <path d={`M${nodes[1].x} ${nodes[1].y}L${nodes[2].x} ${nodes[2].y}`} />
        <path d={`M${nodes[2].x} ${nodes[2].y}L${nodes[0].x} ${nodes[0].y}`} />
      </g>
      {/* nodes */}
      <g fill="#fff">
        {nodes.map((n, i) => (
          <circle key={i} cx={n.x} cy={n.y} r="3.4" />
        ))}
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

export function BtcLogo({ size = 28 }: Props) {
  return (
    <CoinBadge size={size} bg="#F7931A">
      <text
        x="20"
        y="27"
        textAnchor="middle"
        fontSize="18"
        fontWeight="700"
        fill="#fff"
        fontFamily="Arial, sans-serif"
      >
        ₿
      </text>
    </CoinBadge>
  )
}

export function EthLogo({ size = 28 }: Props) {
  return (
    <CoinBadge size={size} bg="#627EEA">
      <g fill="#fff">
        <polygon points="20,6 28,20 20,24.5 12,20" opacity="0.95" />
        <polygon points="20,26 28,21.5 20,34 12,21.5" opacity="0.7" />
      </g>
    </CoinBadge>
  )
}

// --- Dispatchers ---

export function ChainLogo({ chain: _chain, size = 28 }: { chain: string; size?: number }) {
  return <StellarLogo size={size} />
}

export function TokenLogo({ symbol, size = 28 }: { symbol: string; size?: number }) {
  switch (symbol) {
    case 'USDC':
      return <UsdcLogo size={size} />
    case 'USDT':
      return <UsdtLogo size={size} />
    case 'BTC':
      return <BtcLogo size={size} />
    case 'ETH':
      return <EthLogo size={size} />
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
}
