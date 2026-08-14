import { useState } from 'react'
import { CopyIcon, CheckIcon } from '../icons.tsx'

export default function CopyAddress({
  address,
  className = '',
  truncate = false,
}: {
  address: string
  className?: string
  truncate?: boolean
}) {
  const [copied, setCopied] = useState(false)

  function copy() {
    navigator.clipboard.writeText(address).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 1800)
    })
  }

  return (
    <button
      onClick={copy}
      title={address}
      className={
        'group inline-flex min-w-0 max-w-full items-center gap-1.5 font-mono text-sm text-muted transition-colors hover:text-ink ' +
        className
      }
    >
      <span className={truncate ? 'min-w-0 truncate' : ''}>{address}</span>
      {copied ? (
        <span className="shrink-0 text-green-600">
          <CheckIcon size={13} />
        </span>
      ) : (
        <span className="shrink-0">
          <CopyIcon size={13} />
        </span>
      )}
    </button>
  )
}
