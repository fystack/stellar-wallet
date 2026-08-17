import { useEffect, useState } from 'react'
import { CheckShieldIcon } from '../icons.tsx'
import { TokenLogo, ChainLogo } from '../logos.tsx'
import { api } from '../api.ts'
import { toast } from '../toast.tsx'

type ChainStat = { chain: string; online: boolean }

function ChainBadge({
  chain,
  chains,
}: {
  chain: string
  chains: ChainStat[]
}) {
  const c = chains.find((x) => x.chain === chain)
  const online = c?.online
  return (
    <span
      className={
        'ml-auto flex items-center gap-1.5 text-xs font-semibold ' +
        (online === undefined
          ? 'text-muted'
          : online
            ? 'text-success'
            : 'text-danger')
      }
    >
      <span
        className={
          'h-2 w-2 rounded-full ' +
          (online === undefined
            ? 'bg-[#d5dbe4]'
            : online
              ? 'bg-green-600'
              : 'bg-danger')
        }
      />
      {online === undefined ? 'Checking…' : online ? 'Online' : 'Unreachable'}
    </span>
  )
}

type NodeStatus = {
  name: string
  region: string
  online: boolean
}

function Toggle({
  checked,
  onChange,
  disabled = false,
}: {
  checked: boolean
  onChange: (v: boolean) => void
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => !disabled && onChange(!checked)}
      className={
        'inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors ' +
        (disabled ? 'cursor-not-allowed opacity-60 ' : '') +
        (checked ? 'bg-brand' : 'bg-[#d5dbe4]')
      }
    >
      <span
        className={
          'inline-block h-5 w-5 transform rounded-full bg-white shadow-sm transition-transform ' +
          (checked ? 'translate-x-6' : 'translate-x-0.5')
        }
      />
    </button>
  )
}

function Pref({
  name,
  desc,
  checked,
  onChange,
  disabled = false,
}: {
  name: string
  desc: string
  checked: boolean
  onChange: (v: boolean) => void
  disabled?: boolean
}) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-line py-3.5 last:border-none sm:items-center">
      <div className="min-w-0">
        <div className="font-semibold">{name}</div>
        <div className="mt-0.5 text-sm text-muted">{desc}</div>
      </div>
      <Toggle checked={checked} onChange={onChange} disabled={disabled} />
    </div>
  )
}

export default function Settings() {
  const [hideBalances, setHideBalances] = useState(
    () => localStorage.getItem('pref_hide') === '1',
  )
  const [testnet, setTestnet] = useState(true)
  const [notifications, setNotifications] = useState(
    () => localStorage.getItem('pref_notif') !== '0',
  )
  const toggleHide = (v: boolean) => {
    setHideBalances(v)
    localStorage.setItem('pref_hide', v ? '1' : '0')
  }
  const toggleNotif = (v: boolean) => {
    setNotifications(v)
    localStorage.setItem('pref_notif', v ? '1' : '0')
  }
  const [nodes, setNodes] = useState<NodeStatus[]>([])
  const [threshold, setThreshold] = useState('2-of-3')

  // Chain / RPC config + custom asset registry.
  const [horizonUrl, setHorizonUrl] = useState('')
  const [solanaUrl, setSolanaUrl] = useState('')
  const [assets, setAssets] = useState<{ code: string; issuer: string }[]>([])
  const [savingRpc, setSavingRpc] = useState(false)
  const [newCode, setNewCode] = useState('')
  const [newIssuer, setNewIssuer] = useState('')
  const [chains, setChains] = useState<ChainStat[]>([])

  useEffect(() => {
    const load = () =>
      api
        .chains()
        .then((r) => setChains(r.map((c) => ({ chain: c.chain, online: c.online }))))
        .catch(() => {})
    load()
    const t = setInterval(load, 10000)
    return () => clearInterval(t)
  }, [])

  useEffect(() => {
    const load = () =>
      api
        .cluster()
        .then((r) => {
          setNodes(r.nodes)
          setThreshold(r.threshold)
        })
        .catch(() => {})
    load()
    const t = setInterval(load, 10000)
    return () => clearInterval(t)
  }, [])

  useEffect(() => {
    api
      .getConfig()
      .then((r) => {
        setHorizonUrl(r.horizonUrl)
        setSolanaUrl(r.solanaUrl)
        setAssets(r.assets)
      })
      .catch(() => {})
  }, [])

  async function saveRpc() {
    setSavingRpc(true)
    try {
      await api.putConfig(horizonUrl.trim(), solanaUrl.trim())
      toast.success('RPC endpoints saved')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Save failed')
    } finally {
      setSavingRpc(false)
    }
  }

  async function addAsset() {
    if (!newCode.trim() || !newIssuer.trim()) return
    try {
      const r = await api.addAsset(newCode.trim(), newIssuer.trim())
      setAssets(r.assets)
      setNewCode('')
      setNewIssuer('')
      toast.success('Asset added')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Could not add asset')
    }
  }

  async function removeAsset(code: string, issuer: string) {
    const r = await api.removeAsset(code, issuer)
    setAssets(r.assets)
  }

  return (
    <div className="mx-auto mt-2.5 max-w-[760px]">
      <header className="mb-6">
        <h1 className="mb-2 text-2xl font-bold sm:text-3xl">Settings</h1>
        <p className="text-base leading-relaxed text-muted">
          Manage your MPC cluster, security, and preferences.
        </p>
      </header>

      <section className="surface mb-5">
        <h3 className="mb-[18px] text-[17px] font-bold">MPC nodes</h3>
        <div className="mb-4 flex flex-col">
          {nodes.map((n) => (
            <div
              key={n.name}
              className="flex items-center gap-3 border-b border-line py-3 last:border-none"
            >
              <span
                className={
                  'h-2.5 w-2.5 rounded-full ' +
                  (n.online ? 'bg-green-600' : 'bg-[#cbd3de]')
                }
              />
              <div className="min-w-0 flex-1">
                <div className="font-semibold">{n.name}</div>
                <div className="truncate font-mono text-[13px] text-muted">{n.region}</div>
              </div>
              <span
                className={
                  'text-[13px] font-semibold ' +
                  (n.online ? 'text-success' : 'text-muted')
                }
              >
                {n.online ? 'Online' : 'Offline'}
              </span>
            </div>
          ))}
        </div>
        <div className="flex items-start gap-2.5 bg-card px-3.5 py-3 text-sm text-ink-soft sm:items-center">
          <span className="shrink-0 text-brand">
            <CheckShieldIcon size={18} />
          </span>
          <span>
            Threshold: {threshold} · Key never assembled on any single node.
          </span>
        </div>
      </section>

      <section className="surface mb-5">
        <h3 className="mb-[18px] text-[17px] font-bold">Chains & RPC</h3>

        <div className="mb-5 flex flex-col gap-4">
          {/* Stellar */}
          <div>
            <div className="mb-2 flex flex-wrap items-center gap-2.5">
              <ChainLogo chain="stellar" size={28} />
              <span className="font-semibold">Stellar</span>
              <span className="bg-card px-2 py-0.5 text-xs font-medium uppercase text-muted">
                testnet
              </span>
              <ChainBadge chain="stellar" chains={chains} />
            </div>
            <input
              className="field-input w-full font-mono text-sm"
              value={horizonUrl}
              onChange={(e) => setHorizonUrl(e.target.value)}
              spellCheck={false}
            />
          </div>

          {/* Solana */}
          <div>
            <div className="mb-2 flex flex-wrap items-center gap-2.5">
              <ChainLogo chain="solana" size={28} />
              <span className="font-semibold">Solana</span>
              <span className="bg-card px-2 py-0.5 text-xs font-medium uppercase text-muted">
                devnet
              </span>
              <ChainBadge chain="solana" chains={chains} />
            </div>
            <input
              className="field-input w-full font-mono text-sm"
              value={solanaUrl}
              onChange={(e) => setSolanaUrl(e.target.value)}
              spellCheck={false}
            />
          </div>
        </div>

        <button
          onClick={saveRpc}
          disabled={savingRpc}
          className="w-full bg-brand px-4 py-2.5 font-semibold text-white transition-colors hover:bg-brand-deep disabled:opacity-50 sm:w-auto"
        >
          {savingRpc ? 'Saving…' : 'Save endpoints'}
        </button>
      </section>

      <section className="surface mb-5">
        <h3 className="mb-[18px] text-[17px] font-bold">Custom assets</h3>
        {assets.length === 0 ? (
          <div className="mb-4 text-sm text-muted">
            No custom assets. Add a Stellar asset (code + issuer) to make it
            available to your wallets.
          </div>
        ) : (
          <div className="mb-4 flex flex-col">
            {assets.map((a) => (
              <div
                key={a.code + a.issuer}
                className="flex items-center gap-3 border-b border-line py-2.5 last:border-none"
              >
                <TokenLogo symbol={a.code} size={22} />
                <div className="min-w-0 flex-1">
                  <div className="font-semibold">{a.code}</div>
                  <div className="truncate font-mono text-[13px] text-muted">
                    {a.issuer}
                  </div>
                </div>
                <button
                  onClick={() => removeAsset(a.code, a.issuer)}
                  className="shrink-0 text-sm font-semibold text-muted hover:text-danger"
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        )}
        <div className="flex flex-col gap-2 sm:flex-row">
          <input
            className="field-input w-full sm:w-28"
            placeholder="Code (USDC)"
            value={newCode}
            onChange={(e) => setNewCode(e.target.value.toUpperCase())}
          />
          <input
            className="field-input min-w-0 flex-1 font-mono text-sm"
            placeholder="Issuer address (G…)"
            value={newIssuer}
            onChange={(e) => setNewIssuer(e.target.value)}
            spellCheck={false}
          />
          <button
            onClick={addAsset}
            className="bg-brand px-4 py-2.5 font-semibold text-white transition-colors hover:bg-brand-deep"
          >
            Add
          </button>
        </div>
      </section>

      <section className="surface mb-5">
        <h3 className="mb-[18px] text-[17px] font-bold">Preferences</h3>
        <Pref
          name="Hide balances by default"
          desc="Mask amounts until you reveal them."
          checked={hideBalances}
          onChange={toggleHide}
        />
        <Pref
          name="Use Stellar testnet"
          desc="This build runs on testnet (Horizon + Friendbot). Mainnet not enabled."
          checked={testnet}
          onChange={setTestnet}
          disabled
        />
        <Pref
          name="Transaction notifications"
          desc="Alert me when a signing request needs approval."
          checked={notifications}
          onChange={toggleNotif}
        />
      </section>

      <section className="surface mb-5 bg-[#fff7f7]">
        <h3 className="mb-[18px] text-[17px] font-bold">Danger zone</h3>
        <div className="flex flex-col items-start gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="min-w-0">
            <div className="font-semibold">Sign out of all devices</div>
            <div className="mt-0.5 text-sm text-muted">
              End every active session immediately.
            </div>
          </div>
          <button className="w-full shrink-0 bg-[#ffe5e5] px-4 py-2.5 font-semibold text-danger hover:bg-[#fdd6d6] sm:w-auto">
            Sign out all
          </button>
        </div>
      </section>
    </div>
  )
}
