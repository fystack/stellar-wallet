import { useEffect, useState } from 'react'
import { CheckIcon } from '../icons.tsx'

// Triangle node positions inside the 280x250 viewBox.
const N = [
  { x: 140, y: 46 }, // N1 top
  { x: 226, y: 190 }, // N2 bottom-right
  { x: 54, y: 190 }, // N3 bottom-left
]

// Full mesh — every node talks to every other node.
const EDGES: [number, number][] = [
  [0, 1],
  [1, 2],
  [2, 0],
]

const STEPS = ['Commit', 'Share', 'Verify']
const STEP_HINT = [
  'broadcasting commitments',
  'exchanging secret shares',
  'verifying & finalizing',
]

const BRAND = '#1f6bff'
const BRAND_LT = '#7aa7ff'
const GREEN = '#16a34a'

function Packet({
  edgeId,
  reverse,
  begin,
  color,
  r = 3.4,
}: {
  edgeId: string
  reverse?: boolean
  begin: string
  color: string
  r?: number
}) {
  return (
    <circle r={r} fill={color} filter="url(#glow)">
      <animateMotion
        dur="1.9s"
        begin={begin}
        repeatCount="indefinite"
        keyPoints={reverse ? '1;0' : '0;1'}
        keyTimes="0;1"
        calcMode="linear"
      >
        <mpath href={`#${edgeId}`} />
      </animateMotion>
      <animate
        attributeName="opacity"
        values="0;1;1;0"
        keyTimes="0;0.15;0.85;1"
        dur="1.9s"
        begin={begin}
        repeatCount="indefinite"
      />
    </circle>
  )
}

export default function KeygenMesh({ done = false }: { done?: boolean }) {
  const [step, setStep] = useState(0)

  useEffect(() => {
    if (done) return
    // Advance through the rounds once, then hold on the last until keygen completes.
    const t = setInterval(
      () =>
        setStep((s) => {
          if (s >= STEPS.length - 1) {
            clearInterval(t)
            return s
          }
          return s + 1
        }),
      1300,
    )
    return () => clearInterval(t)
  }, [done])

  return (
    <div className="flex flex-col items-center gap-4 py-1">
      <div className="text-xs font-semibold uppercase tracking-[0.14em] text-muted">
        Distributed key generation
      </div>

      <svg viewBox="0 0 280 250" className="w-[300px] max-w-full">
        <defs>
          <linearGradient
            id="edgeGrad"
            gradientUnits="userSpaceOnUse"
            x1="0"
            y1="0"
            x2="280"
            y2="250"
          >
            <stop offset="0%" stopColor={BRAND} />
            <stop offset="100%" stopColor={BRAND_LT} />
          </linearGradient>
          <filter id="glow" x="-60%" y="-60%" width="220%" height="220%">
            <feGaussianBlur stdDeviation="2.2" result="b" />
            <feMerge>
              <feMergeNode in="b" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
          <filter id="soft" x="-80%" y="-80%" width="260%" height="260%">
            <feGaussianBlur stdDeviation="6" />
          </filter>
        </defs>

        {/* edges: flowing dashed base + glowing packets on top */}
        {EDGES.map(([a, b], i) => (
          <path
            key={`e${i}`}
            id={`edge-${i}`}
            d={`M${N[a].x} ${N[a].y} L${N[b].x} ${N[b].y}`}
            fill="none"
            stroke={done ? GREEN : 'url(#edgeGrad)'}
            strokeWidth="2"
            strokeLinecap="round"
            strokeDasharray={done ? undefined : '2 7'}
            opacity={done ? 0.9 : 0.75}
          >
            {!done && (
              <animate
                attributeName="stroke-dashoffset"
                values="0;-18"
                dur="0.9s"
                repeatCount="indefinite"
              />
            )}
          </path>
        ))}

        {!done &&
          EDGES.map((_, i) => (
            <g key={`p${i}`}>
              <Packet edgeId={`edge-${i}`} begin={`${-i * 0.5}s`} color={BRAND} r={4} />
              <Packet
                edgeId={`edge-${i}`}
                reverse
                begin={`${-i * 0.5 - 1.1}s`}
                color={BRAND_LT}
                r={4}
              />
            </g>
          ))}

        {/* soft glow that fills the triangle once complete */}
        {done && (
          <polygon
            points={N.map((p) => `${p.x},${p.y}`).join(' ')}
            fill={GREEN}
            opacity="0.1"
            filter="url(#soft)"
          />
        )}

        {/* nodes */}
        {N.map((p, i) => (
          <g
            key={`n${i}`}
            style={
              done
                ? {
                    transformOrigin: `${p.x}px ${p.y}px`,
                    animation: `nodePop .45s ${i * 0.08}s cubic-bezier(.34,1.56,.64,1) both`,
                  }
                : undefined
            }
          >
            {/* soft halo */}
            <circle
              cx={p.x}
              cy={p.y}
              r="20"
              fill={done ? GREEN : BRAND}
              opacity={done ? 0.16 : 0.18}
              filter="url(#soft)"
            >
              {!done && (
                <animate
                  attributeName="opacity"
                  values="0.28;0.08;0.28"
                  dur="2.1s"
                  begin={`${i * 0.5}s`}
                  repeatCount="indefinite"
                />
              )}
            </circle>

            {/* node tile */}
            <rect
              x={p.x - 18}
              y={p.y - 18}
              width="36"
              height="36"
              rx="9"
              fill={done ? '#dcfce7' : '#eef4ff'}
              stroke={done ? GREEN : BRAND}
              strokeWidth="2"
            >
              {!done && (
                <animate
                  attributeName="stroke-opacity"
                  values="1;0.45;1"
                  dur="2.1s"
                  begin={`${i * 0.5}s`}
                  repeatCount="indefinite"
                />
              )}
            </rect>

            {done ? (
              /* checkmark inside the tile */
              <path
                d={`M${p.x - 6} ${p.y} l4 4 l8 -9`}
                stroke={GREEN}
                strokeWidth="2.4"
                fill="none"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            ) : (
              <>
                {/* number inside the tile */}
                <text
                  x={p.x}
                  y={p.y + 4.5}
                  textAnchor="middle"
                  fontSize="13"
                  fontWeight="800"
                  fill={BRAND}
                  fontFamily="ui-monospace, monospace"
                >
                  {i + 1}
                </text>
                {/* clear label placed OUTSIDE the triangle */}
                <text
                  x={p.x}
                  y={i === 0 ? p.y - 26 : p.y + 34}
                  textAnchor="middle"
                  fontSize="11.5"
                  fontWeight="600"
                  fill="#5b6b82"
                >
                  Node {i + 1}
                </text>
              </>
            )}
          </g>
        ))}
      </svg>

      {/* round stepper */}
      {done ? (
        <p className="flex items-center gap-1.5 text-center text-sm font-semibold text-green-600">
          <CheckIcon size={15} /> Key split across 3 nodes — never assembled
        </p>
      ) : (
        <div className="flex flex-col items-center gap-2">
          <div className="flex flex-wrap items-center justify-center gap-2">
            {STEPS.map((label, i) => (
              <div key={label} className="flex items-center gap-2">
                <div
                  className={
                    'flex items-center gap-1.5 text-xs font-semibold transition-colors ' +
                    (i === step ? 'text-brand' : 'text-muted')
                  }
                >
                  <span
                    className={
                      'h-1.5 w-1.5 rounded-full transition-all ' +
                      (i === step
                        ? 'scale-125 bg-brand'
                        : i < step
                          ? 'bg-brand/40'
                          : 'bg-[#d5dbe4]')
                    }
                  />
                  {label}
                </div>
                {i < STEPS.length - 1 && (
                  <span className="hidden h-px w-4 bg-line sm:block" />
                )}
              </div>
            ))}
          </div>
          <p className="text-center text-xs text-muted">{STEP_HINT[step]}</p>
        </div>
      )}
    </div>
  )
}
