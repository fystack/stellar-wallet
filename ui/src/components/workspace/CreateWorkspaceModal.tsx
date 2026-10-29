import { useState } from 'react'
import Modal from '../Modal.tsx'

export default function CreateWorkspaceModal({
  onClose,
  onCreate,
}: {
  onClose: () => void
  onCreate: (name: string) => Promise<void>
}) {
  const [name, setName] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    setError('')
    setBusy(true)
    try {
      await onCreate(name.trim())
      onClose()
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : 'Unable to create workspace.')
      setBusy(false)
    }
  }

  return (
    <Modal title="Create a team workspace" onClose={onClose}>
      <p className="mb-5 text-[15px] text-muted">
        Invite teammates, share wallets and give each member their own login and wallet.
      </p>
      <form onSubmit={(event) => void handleSubmit(event)} className="flex flex-col gap-4">
        <label className="flex flex-col gap-2">
          <span className="text-sm font-semibold text-ink-soft">Workspace name</span>
          <input
            className="field-input"
            placeholder="e.g. Treasury"
            maxLength={60}
            autoFocus
            value={name}
            onChange={(event) => setName(event.target.value)}
          />
        </label>
        {error && (
          <div role="alert" className="bg-danger-soft px-3 py-2 text-sm text-danger">
            {error}
          </div>
        )}
        <button className="btn btn-primary btn-lg w-full" type="submit" disabled={busy || !name.trim()}>
          {busy ? 'Creating…' : 'Create workspace'}
        </button>
      </form>
    </Modal>
  )
}
