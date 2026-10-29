import { useState } from 'react'
import { CopyIcon, CheckIcon } from '../icons.tsx'
import Tooltip from './ui/Tooltip.tsx'
import { shortAddress } from '../lib/format.ts'

export default function CopyAddress({
  address,
  className = '',
  truncate = false,
  tooltipAlign,
}: {
  address: string
  className?: string
  truncate?: boolean
  tooltipAlign?: 'center' | 'start' | 'end'
}) {
  const [copied, setCopied] = useState(false)

  function copy() {
    navigator.clipboard.writeText(address).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1800)
    })
  }

  return (
    <Tooltip label={copied ? 'Copied' : address} align={tooltipAlign}>
      <button
        onClick={copy}
        aria-label={`Copy address ${address}`}
        className={
          'inline-flex min-w-0 max-w-full items-center gap-1.5 font-mono text-sm text-muted transition-colors hover:text-ink ' +
          className
        }
      >
        <span className={truncate ? '' : 'break-all'}>{truncate ? shortAddress(address, 8, 8) : address}</span>
        {copied ? (
          <span className="shrink-0 text-success">
            <CheckIcon size={13} />
          </span>
        ) : (
          <span className="shrink-0">
            <CopyIcon size={13} />
          </span>
        )}
      </button>
    </Tooltip>
  )
}
