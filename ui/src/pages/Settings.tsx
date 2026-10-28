import { readPreference, writePreference } from '../lib/preferences.ts'
import { useEffect, useState } from 'react'
import { CheckShieldIcon } from '../icons.tsx'
import { TokenLogo, ChainLogo } from '../logos.tsx'
import { api } from '../api.ts'
import { toast } from '../toast.tsx'
import LinkedAccounts from '../components/LinkedAccounts.tsx'
import WorkspaceMembers from '../components/workspace/WorkspaceMembers.tsx'
import ApprovalPolicyCard from '../components/approvals/ApprovalPolicyCard.tsx'
import type { ApprovalPolicySettings, CustomAsset, Workspace } from '../types.ts'

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
          'dot ' +
          (online === undefined
            ? 'bg-line-strong'
            : online
              ? 'bg-success'
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
  label,
  checked,
  onChange,
  disabled = false,
}: {
  label: string
  checked: boolean
  onChange: (v: boolean) => void
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-label={label}
      aria-checked={checked}
      disabled={disabled}
      onClick={() => !disabled && onChange(!checked)}
      className={
        'inline-flex h-7 w-12 shrink-0 items-center rounded-full transition-colors ' +
        (disabled ? 'cursor-not-allowed opacity-60 ' : '') +
        (checked ? 'bg-brand' : 'bg-line-strong')
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
      <Toggle label={name} checked={checked} onChange={onChange} disabled={disabled} />
    </div>
  )
}

export default function Settings({
  workspace,
  userEmail,
  onCreateWorkspace,
  isOwner,
  approvalPolicy,
  approvalPolicyFailed,
  onApprovalPolicyChange,
  onMembersChange,
  onSignOutEverywhere,
}: {
  workspace: Workspace | null
  userEmail: string
  onCreateWorkspace: (name: string) => Promise<void>
  isOwner: boolean
  approvalPolicy: ApprovalPolicySettings | null
  approvalPolicyFailed: boolean
  onApprovalPolicyChange: (settings: ApprovalPolicySettings) => void
  onMembersChange: () => void
  onSignOutEverywhere: () => Promise<void>
}) {
  const [signingOutEverywhere, setSigningOutEverywhere] = useState(false)
  const [hideBalances, setHideBalances] = useState(
    () => readPreference('pref_hide') === '1',
  )
  const [testnet, setTestnet] = useState(true)
  const [notifications, setNotifications] = useState(
    () => readPreference('pref_notif') !== '0',
  )
  const toggleHide = (v: boolean) => {
    setHideBalances(v)
    writePreference('pref_hide', v ? '1' : '0')
  }
  const toggleNotif = (v: boolean) => {
    setNotifications(v)
    writePreference('pref_notif', v ? '1' : '0')
  }
  const [nodes, setNodes] = useState<NodeStatus[]>([])
  const [threshold, setThreshold] = useState('2-of-3')

  const [horizonUrl, setHorizonUrl] = useState('')
  const [registry, setRegistry] = useState<{ workspaceId: string; assets: CustomAsset[] | null }>({ workspaceId: '', assets: null })
  const [assetsReload, setAssetsReload] = useState(0)
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
      .then((r) => setHorizonUrl(r.horizonUrl))
      .catch(() => {})
  }, [])

  const workspaceId = workspace?.id ?? ''
  useEffect(() => {
    if (!workspaceId) return
    let current = true
    api
      .listAssets(workspaceId)
      .then((r) => current && setRegistry({ workspaceId, assets: r.assets }))
      .catch(() => current && setRegistry({ workspaceId, assets: null }))
    return () => {
      current = false
    }
  }, [workspaceId, assetsReload])
  const assetsLoaded = registry.workspaceId === workspaceId
  const assets = assetsLoaded ? registry.assets : null


  async function addAsset() {
    if (!newCode.trim() || !newIssuer.trim()) return
    try {
      const r = await api.addAsset(workspaceId, newCode.trim(), newIssuer.trim())
      setRegistry({ workspaceId, assets: r.assets })
      setNewCode('')
      setNewIssuer('')
      toast.success('Asset added')
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Could not add asset')
    }
  }

  async function removeAsset(code: string, issuer: string) {
    try {
      const r = await api.removeAsset(workspaceId, code, issuer)
      setRegistry({ workspaceId, assets: r.assets })
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Could not remove asset')
    }
  }

  return (
    <div className="mx-auto mt-2.5 max-w-[760px]">
      <header className="mb-6">
        <h1 className="mb-2 text-2xl font-bold sm:text-3xl">Settings</h1>
        <p className="text-base leading-relaxed text-muted">
          Manage your MPC cluster, security, and preferences.
        </p>
      </header>
      <WorkspaceMembers workspace={workspace} userEmail={userEmail} onCreateWorkspace={onCreateWorkspace} onChange={onMembersChange} />
      {workspace && !workspace.personal && approvalPolicyFailed && (
        <section role="alert" className="surface mb-5">
          <h3 className="mb-1 text-[17px] font-bold">Payment approvals</h3>
          <p className="mb-4 text-sm text-muted">Unable to load the approval policy.</p>
          <button className="btn btn-primary btn-lg" onClick={onMembersChange}>Try again</button>
        </section>
      )}
      {workspace && !workspace.personal && approvalPolicy && (
        <ApprovalPolicyCard
          key={workspace.id}
          workspaceId={workspace.id}
          isOwner={isOwner}
          settings={approvalPolicy}
          onChange={onApprovalPolicyChange}
        />
      )}
      <LinkedAccounts />

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
                  'dot ' +
                  (n.online ? 'bg-success' : 'bg-line-strong')
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
            <div className="field-input w-full break-all font-mono text-sm text-ink-soft">
              {horizonUrl}
            </div>
          </div>
        </div>
        <div className="text-sm text-muted">
          The Horizon endpoint is set by the server operator in config.yaml.
        </div>
      </section>

      <section className="surface mb-5">
        <h3 className="mb-[18px] text-[17px] font-bold">Custom assets</h3>
        {!assetsLoaded ? null : !assets ? (
          <div role="alert" className="mb-4">
            <p className="mb-3 text-sm text-muted">Unable to load this workspace's assets.</p>
            <button className="btn btn-secondary" onClick={() => setAssetsReload((count) => count + 1)}>Try again</button>
          </div>
        ) : assets.length === 0 ? (
          <div className="mb-4 text-sm text-muted">
            No custom assets. Add a Stellar asset (code + issuer) to make it
            available to this workspace's wallets.
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
                {isOwner && (
                  <button
                    onClick={() => removeAsset(a.code, a.issuer)}
                    aria-label={`Remove ${a.code}`}
                    className="btn btn-ghost-danger shrink-0 text-sm"
                  >
                    Remove
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
        {!isOwner && <p className="text-sm text-muted">Only workspace owners can add or remove assets.</p>}
        {isOwner && (
          <div className="flex flex-col gap-2 sm:flex-row">
            <input
              className="field-input w-full sm:w-28"
              aria-label="Asset code"
              placeholder="Code (USDC)"
              value={newCode}
              onChange={(e) => setNewCode(e.target.value.toUpperCase())}
            />
            <input
              className="field-input min-w-0 flex-1 font-mono text-sm"
              aria-label="Asset issuer"
              placeholder="Issuer address (G…)"
              value={newIssuer}
              onChange={(e) => setNewIssuer(e.target.value)}
              spellCheck={false}
            />
            <button
              onClick={addAsset}
              className="btn btn-primary"
            >
              Add
            </button>
          </div>
        )}
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

      <section className="surface mb-5 bg-danger-soft">
        <h3 className="mb-[18px] text-[17px] font-bold">Danger zone</h3>
        <div className="flex flex-col items-start gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="min-w-0">
            <div className="font-semibold">Sign out of all devices</div>
            <div className="mt-0.5 text-sm text-muted">
              End every active session immediately.
            </div>
          </div>
          <button
            className="btn btn-danger-soft w-full shrink-0 sm:w-auto"
            disabled={signingOutEverywhere}
            onClick={() => {
              setSigningOutEverywhere(true)
              void onSignOutEverywhere().finally(() => setSigningOutEverywhere(false))
            }}
          >
            {signingOutEverywhere ? 'Signing out…' : 'Sign out all'}
          </button>
        </div>
      </section>
    </div>
  )
}
