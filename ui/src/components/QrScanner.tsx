import { useEffect, useRef, useState } from 'react'
import Modal from './Modal.tsx'

// The browser BarcodeDetector API isn't in the TS DOM lib yet.
type BarcodeDetectorLike = {
  detect: (source: CanvasImageSource) => Promise<{ rawValue: string }[]>
}
declare const BarcodeDetector:
  | {
      new (options?: { formats?: string[] }): BarcodeDetectorLike
      getSupportedFormats?: () => Promise<string[]>
    }
  | undefined

// Stellar QR codes may embed the address in a SEP-7 URI (web+stellar:pay?...).
// Pull out the payable string so the caller gets a bare G/M/federation value.
function extractRecipient(raw: string): string {
  const value = raw.trim()
  const match = value.match(/destination=([^&\s]+)/i)
  if (match) return decodeURIComponent(match[1])
  return value.replace(/^web\+stellar:(pay\?)?/i, '')
}

export default function QrScanner({
  onResult,
  onClose,
}: {
  onResult: (text: string) => void
  onClose: () => void
}) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const fileRef = useRef<HTMLInputElement>(null)
  const [error, setError] = useState('')
  const supported = typeof BarcodeDetector !== 'undefined'

  useEffect(() => {
    if (!supported) return
    let stream: MediaStream | null = null
    let raf = 0
    let stopped = false
    const detector = new BarcodeDetector!({ formats: ['qr_code'] })

    async function start() {
      try {
        stream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: 'environment' },
        })
        if (stopped) return
        const video = videoRef.current
        if (!video) return
        video.srcObject = stream
        await video.play()
        scan()
      } catch {
        setError('Camera unavailable — upload a QR image instead.')
      }
    }

    async function scan() {
      const video = videoRef.current
      if (!video || stopped) return
      try {
        const codes = await detector.detect(video)
        if (codes.length > 0) {
          onResult(extractRecipient(codes[0].rawValue))
          onClose()
          return
        }
      } catch {
        /* transient decode error — keep scanning */
      }
      raf = requestAnimationFrame(scan)
    }

    start()
    return () => {
      stopped = true
      cancelAnimationFrame(raf)
      stream?.getTracks().forEach((track) => track.stop())
    }
  }, [supported, onResult, onClose])

  async function onFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    if (!supported) {
      setError('QR decoding is not supported in this browser.')
      return
    }
    try {
      const bitmap = await createImageBitmap(file)
      const detector = new BarcodeDetector!({ formats: ['qr_code'] })
      const codes = await detector.detect(bitmap)
      if (codes.length > 0) {
        onResult(extractRecipient(codes[0].rawValue))
        onClose()
      } else {
        setError('No QR code found in that image.')
      }
    } catch {
      setError('Could not read that image.')
    }
  }

  return (
    <Modal title="Scan QR code" onClose={onClose} maxWidth={420}>
      <div className="flex flex-col gap-4">
        {supported ? (
          <div className="relative aspect-square w-full overflow-hidden bg-black">
            <video
              ref={videoRef}
              className="h-full w-full object-cover"
              muted
              playsInline
            />
            <div className="pointer-events-none absolute inset-6 border-2 border-white/70" />
          </div>
        ) : (
          <p className="bg-warning-soft px-3 py-2 text-sm text-warning">
            Live camera scanning isn't supported here. Upload a QR image below.
          </p>
        )}

        {error && (
          <p className="bg-danger-soft px-3 py-2 text-sm text-danger">{error}</p>
        )}

        <button
          type="button"
          className="cta-secondary w-full"
          onClick={() => fileRef.current?.click()}
        >
          Upload QR image
        </button>
        <input
          ref={fileRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={onFile}
        />
      </div>
    </Modal>
  )
}
