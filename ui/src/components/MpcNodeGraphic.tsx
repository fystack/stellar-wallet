import { useId } from 'react'
import { KeyIcon } from '../icons.tsx'

type Point = [number, number]

const STELLAR_MARK_TOP =
  'M203,26.16l-28.46,14.5-137.43,70a82.49,82.49,0,0,1-.7-10.69A82.88,82.88,0,0,1,158.55,28.6l16.29-8.3,2.43-1.24A100,100,0,0,0,18.18,100q0,3.82.29,7.61a18.19,18.19,0,0,1-9.88,17.58L0,129.57V150l25.29-12.89,0,0,8.19-4.18,8.07-4.11v0L186.43,55l16.28-8.29,33.65-17.15V9.14Z'
const STELLAR_MARK_BOTTOM =
  'M236.36,50,49.78,145,33.5,153.31,0,170.38v20.41l33.27-16.95,28.46-14.5,137.57-70.1A83.45,83.45,0,0,1,200,100,82.87,82.87,0,0,1,77.79,171.36l-1,.53-17.66,9A100,100,0,0,0,218.18,100c0-2.57-.1-5.14-.29-7.68a18.2,18.2,0,0,1,9.87-17.58l8.6-4.38Z'

const lerp = (a: Point, b: Point, t: number): string => `${a[0] + (b[0] - a[0]) * t},${a[1] + (b[1] - a[1]) * t}`

function corners(cx: number, cy: number, w: number, h: number): Point[] {
  return [[cx, cy - h], [cx + w, cy], [cx, cy + h], [cx - w, cy]]
}

function diamond(cx: number, cy: number, w: number, h: number, t = 0.2): string {
  const p = corners(cx, cy, w, h)
  let d = `M${lerp(p[0], p[1], t)}`
  for (let i = 1; i <= 4; i++) {
    const corner = p[i % 4]
    d += ` L${lerp(corner, p[i - 1], t)} Q${corner[0]},${corner[1]} ${lerp(corner, p[(i + 1) % 4], t)}`
  }
  return d + 'Z'
}

// The two front edges, where the top face meets the sides.
function frontEdge(cx: number, cy: number, w: number, h: number, t = 0.2): string {
  const [, right, bottom, left] = corners(cx, cy, w, h)
  return `M${lerp(left, bottom, t)} L${lerp(bottom, left, t)} Q${bottom[0]},${bottom[1]} ${lerp(bottom, right, t)} L${lerp(right, bottom, t)}`
}

type Ids = Record<'top' | 'topHero' | 'bevel' | 'gloss' | 'side' | 'rim' | 'link' | 'floor' | 'glow' | 'soft' | 'mark', string>

type TileProps = { ids: Ids; cx: number; cy: number; w: number; h: number; depth: number; hero?: boolean }

function Tile({ ids, cx, cy, w, h, depth, hero }: TileProps) {
  const layers = Array.from({ length: depth }, (_, i) => depth - i)
  return (
    <g>
      <path
        className="mpc-pulse"
        d={diamond(cx, cy + depth, w, h)}
        fill="none"
        stroke="#38bdf8"
        strokeWidth={hero ? 8 : 4}
        opacity={hero ? 1 : 0.6}
        filter={`url(#${ids.glow})`}
      />
      {layers.map((offset) => (
        <path key={offset} d={diamond(cx, cy + offset, w, h)} fill={`url(#${ids.side})`} />
      ))}
      <path d={frontEdge(cx, cy + depth, w, h)} fill="none" stroke="#7dd3fc" strokeWidth={hero ? 1.6 : 1.1} strokeOpacity={hero ? 0.95 : 0.55} />
      <path d={frontEdge(cx, cy, w, h)} fill="none" stroke="#bfdbfe" strokeWidth="1.4" strokeOpacity="0.55" />
      <path d={diamond(cx, cy, w, h)} fill={`url(#${hero ? ids.topHero : ids.top})`} />
      <path d={diamond(cx, cy + 1.5, w * 0.86, h * 0.86)} fill={`url(#${ids.bevel})`} stroke="#ffffff" strokeOpacity="0.12" />
      <path d={diamond(cx, cy, w, h)} fill={`url(#${ids.gloss})`} />
      <path d={diamond(cx, cy, w, h)} fill="none" stroke={`url(#${ids.rim})`} strokeWidth="1.4" />
    </g>
  )
}

function Shadow({ ids, cx, cy, w, h, depth }: Omit<TileProps, 'hero'>) {
  return <ellipse cx={cx} cy={cy + depth + h + 14} rx={w * 0.95} ry={h * 0.38} fill="#01040f" opacity="0.7" filter={`url(#${ids.soft})`} />
}

const parties = [
  { label: 'Party A', cx: 280, cy: 96, delay: '0s' },
  { label: 'Party B', cx: 112, cy: 200, delay: '-1.4s' },
  { label: 'Party C', cx: 448, cy: 200, delay: '-2.8s' },
]
const party = { w: 80, h: 47, depth: 20 }
const hub = { cx: 280, cy: 292, w: 124, h: 72, depth: 32 }

export default function MpcNodeGraphic({ className = '' }: { className?: string }) {
  const base = useId().replace(/:/g, '')
  const ids = Object.fromEntries(
    ['top', 'topHero', 'bevel', 'gloss', 'side', 'rim', 'link', 'floor', 'glow', 'soft', 'mark'].map((key) => [key, `${base}-${key}`]),
  ) as Ids

  return (
    <svg viewBox="0 24 560 400" className={className} aria-hidden fill="none">
      <defs>
        <linearGradient id={ids.top} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#5b9bff" />
          <stop offset="0.55" stopColor="#2563eb" />
          <stop offset="1" stopColor="#1e40af" />
        </linearGradient>
        <linearGradient id={ids.topHero} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#60a5fa" />
          <stop offset="0.5" stopColor="#2563eb" />
          <stop offset="1" stopColor="#1d3fa8" />
        </linearGradient>
        <linearGradient id={ids.bevel} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#1d4ed8" stopOpacity="0" />
          <stop offset="1" stopColor="#0b1f5c" stopOpacity="0.35" />
        </linearGradient>
        <radialGradient id={ids.gloss} cx="0.3" cy="0.15" r="0.65">
          <stop offset="0" stopColor="#ffffff" stopOpacity="0.45" />
          <stop offset="0.5" stopColor="#ffffff" stopOpacity="0.08" />
          <stop offset="1" stopColor="#ffffff" stopOpacity="0" />
        </radialGradient>
        {/* Hard stop at the centre splits the extrusion into a lit left face and a shaded right face. */}
        <linearGradient id={ids.side} x1="0" y1="0" x2="1" y2="0">
          <stop offset="0" stopColor="#2a5bd0" />
          <stop offset="0.5" stopColor="#1c46b0" />
          <stop offset="0.5" stopColor="#102a74" />
          <stop offset="1" stopColor="#0c2160" />
        </linearGradient>
        <linearGradient id={ids.rim} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#e0f2fe" stopOpacity="0.9" />
          <stop offset="0.5" stopColor="#bfdbfe" stopOpacity="0.25" />
          <stop offset="1" stopColor="#bfdbfe" stopOpacity="0" />
        </linearGradient>
        <linearGradient id={ids.link} x1="0" y1="0" x2="1" y2="0">
          <stop offset="0" stopColor="#38bdf8" />
          <stop offset="1" stopColor="#818cf8" />
        </linearGradient>
        <radialGradient id={ids.floor} cx="0.5" cy="0.5" r="0.5">
          <stop offset="0" stopColor="#1d4ed8" stopOpacity="0.35" />
          <stop offset="1" stopColor="#1d4ed8" stopOpacity="0" />
        </radialGradient>
        <filter id={ids.glow} x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="6" />
        </filter>
        <filter id={ids.soft} x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="9" />
        </filter>
        <filter id={ids.mark} x="-50%" y="-50%" width="200%" height="200%">
          <feDropShadow dx="0" dy="0" stdDeviation="5" floodColor="#bae6fd" floodOpacity="0.9" />
        </filter>
      </defs>

      <ellipse cx="280" cy="250" rx="270" ry="150" fill={`url(#${ids.floor})`} />

      {parties.map((p) => (
        <Shadow key={p.label} ids={ids} cx={p.cx} cy={p.cy} {...party} />
      ))}
      <Shadow ids={ids} cx={hub.cx} cy={hub.cy} w={hub.w} h={hub.h} depth={hub.depth} />

      <ellipse cx="280" cy="206" rx="168" ry="108" stroke="#1e3a8a" strokeOpacity="0.7" strokeWidth="7" />
      <ellipse cx="280" cy="206" rx="168" ry="108" stroke="#60a5fa" strokeOpacity="0.45" strokeWidth="1.5" />
      <ellipse
        className="mpc-flow"
        cx="280"
        cy="206"
        rx="168"
        ry="108"
        pathLength={1000}
        stroke={`url(#${ids.link})`}
        strokeWidth="5"
        strokeLinecap="round"
        strokeDasharray="70 180"
        filter={`url(#${ids.glow})`}
      />
      <ellipse
        className="mpc-flow"
        cx="280"
        cy="206"
        rx="168"
        ry="108"
        pathLength={1000}
        stroke="#e0f2fe"
        strokeWidth="2"
        strokeLinecap="round"
        strokeDasharray="70 180"
      />

      {parties.map((p) => (
        <g key={p.label} className="mpc-float" style={{ animationDelay: p.delay }}>
          <Tile ids={ids} cx={p.cx} cy={p.cy} {...party} />
          <g transform={`translate(${p.cx - 12} ${p.cy - 33})`} style={{ color: '#e0f2fe' }}>
            <KeyIcon size={24} />
          </g>
          <text x={p.cx} y={p.cy + 15} textAnchor="middle" fill="#0b1f5c" fillOpacity="0.5" fontSize="16" fontWeight="600" fontFamily="Inter, sans-serif" dy="1">
            {p.label}
          </text>
          <text x={p.cx} y={p.cy + 15} textAnchor="middle" fill="#ffffff" fontSize="16" fontWeight="600" fontFamily="Inter, sans-serif">
            {p.label}
          </text>
        </g>
      ))}

      <Tile ids={ids} cx={hub.cx} cy={hub.cy} w={hub.w} h={hub.h} depth={hub.depth} hero />
      <g transform={`translate(${hub.cx - 33} ${hub.cy - 29}) scale(0.28)`} fill="#ffffff" filter={`url(#${ids.mark})`}>
        <path d={STELLAR_MARK_TOP} />
        <path d={STELLAR_MARK_BOTTOM} />
      </g>
    </svg>
  )
}
