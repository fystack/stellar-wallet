import { useId } from 'react'
import stellarLogo from './assets/stellar-logo.png'

type Props = { size?: number }

// App brand mark — the Fystack logo icon.
export function AppLogo({ size = 46 }: Props) {
  const id = useId()
  return (
    <svg
      width={(size * 802.9) / 999.9}
      height={size}
      viewBox="0 0 802.9 999.9"
      fill="none"
    >
      <defs>
        <linearGradient
          id={id}
          x1="-54.99"
          y1="278.79"
          x2="733.89"
          y2="734.28"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stopColor="#0e70df" />
          <stop offset="1" stopColor="#00b9e6" />
        </linearGradient>
      </defs>
      <path
        fill={`url(#${id})`}
        d="M802.9,250.9C802.9,112.3,690.6,0,552,0H250.9C112.3,0,0,112.3,0,250.9v698.8c0,27.7,22.5,50.2,50.2,50.2,11.4,0,22.4-3.9,31.3-10.9l184.9-147c35.5-28.3,79.6-43.7,125-43.7h160.7c138.5,0,250.8-112.3,250.8-250.9V250.9Z"
      />
    </svg>
  )
}

// --- Chain / native marks ---

export function StellarLogo({ size = 28 }: Props) {
  // Official Stellar mark, exactly as published in trustwallet/assets.
  return (
    <img
      src={stellarLogo}
      width={size}
      height={size}
      alt="Stellar"
      style={{ display: 'block' }}
    />
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
  // Official Circle USD Coin mark.
  return (
    <svg width={size} height={size} viewBox="0 0 2000 2000" role="img" aria-label="USDC">
      <path
        d="M1000 2000c554.17 0 1000-445.83 1000-1000S1554.17 0 1000 0 0 445.83 0 1000s445.83 1000 1000 1000z"
        fill="#2775CA"
      />
      <path
        d="M1275 1158.33c0-145.83-87.5-195.83-262.5-216.66-125-16.67-150-50-150-108.34s41.67-95.83 125-95.83c75 0 116.67 25 137.5 87.5 4.17 12.5 16.67 20.83 29.17 20.83h66.66c16.67 0 29.17-12.5 29.17-29.16v-4.17c-16.67-91.67-91.67-162.5-187.5-170.83v-100c0-16.67-12.5-29.17-33.33-33.34h-62.5c-16.67 0-29.17 12.5-33.34 33.34v95.83c-125 16.67-204.16 100-204.16 204.17 0 137.5 83.33 191.66 258.33 212.5 116.67 20.83 154.17 45.83 154.17 112.5s-58.34 112.5-137.5 112.5c-108.34 0-145.84-45.84-158.34-108.34-4.16-16.66-16.66-25-29.16-25h-70.84c-16.66 0-29.16 12.5-29.16 29.17v4.17c16.66 104.16 83.33 179.16 220.83 200v100c0 16.66 12.5 29.16 33.33 33.33h62.5c16.67 0 29.17-12.5 33.34-33.33v-100c125-20.84 208.33-108.34 208.33-220.84z"
        fill="#fff"
      />
      <path
        d="M787.5 1595.83c-325-116.66-491.67-479.16-370.83-800 62.5-175 200-308.33 370.83-370.83 16.67-8.33 25-20.83 25-41.67V325c0-16.67-8.33-29.17-25-33.33-4.17 0-12.5 0-16.67 4.16-395.83 125-612.5 545.84-487.5 941.67 75 233.33 254.17 412.5 487.5 487.5 16.67 8.33 33.34 0 37.5-16.67 4.17-4.16 4.17-8.33 4.17-16.66v-58.34c0-12.5-12.5-29.16-25-37.5zM1229.17 295.83c-16.67-8.33-33.34 0-37.5 16.67-4.17 4.17-4.17 8.33-4.17 16.67v58.33c0 16.67 12.5 33.33 25 41.67 325 116.66 491.67 479.16 370.83 800-62.5 175-200 308.33-370.83 370.83-16.67 8.33-25 20.83-25 41.67V1700c0 16.67 8.33 29.17 25 33.33 4.17 0 12.5 0 16.67-4.16 395.83-125 612.5-545.84 487.5-941.67-75-237.5-258.34-416.67-487.5-491.67z"
        fill="#fff"
      />
    </svg>
  )
}

export function UsdtLogo({ size = 28 }: Props) {
  // Official Tether USD mark.
  return (
    <svg width={size} height={size} viewBox="0 0 2000 2000" role="img" aria-label="USDT">
      <path
        d="M1000 0c552.26 0 1000 447.74 1000 1000s-447.74 1000-1000 1000S0 1552.38 0 1000 447.62 0 1000 0z"
        fill="#26A17B"
      />
      <path
        d="M1123.42 866.76V718h340.18V489.24H537.28V718h340.18v148.77c-276.6 12.72-484.5 67.42-484.5 133 0 65.55 207.9 120.25 484.5 133v476.5h246v-476.6c276-12.72 483.5-67.36 483.5-132.9 0-65.53-207.5-120.17-483.5-132.9m0 225.65v-.02c-6.94.5-42.61 2.62-122 2.62-63.48 0-108.13-1.86-123.87-2.62v.03C633.34 1081.71 451 1039.09 451 988c0-51.09 182.32-93.7 426.55-103.25v166.72c16 1.1 61.75 3.8 124.95 3.8 75.83 0 114.13-3.13 120.92-3.75v-166.7c243.75 9.5 425.63 52.13 425.63 103.15 0 51.02-181.88 93.63-425.63 103.13"
        fill="#fff"
      />
    </svg>
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
