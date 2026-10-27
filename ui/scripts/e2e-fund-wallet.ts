// Manages the e2e fund wallet: `create` makes and Friendbot-funds it, `sweep` returns XLM from local test wallets to it.
import { execFileSync } from 'node:child_process'
import { existsSync } from 'node:fs'
import { Keypair } from '@stellar/stellar-sdk'
import { FUND_WALLET_FILE, HORIZON, fundKeypair, nativeBalance, saveFundKeypair } from '../e2e/fund.ts'
import { API, DB, PASSWORD } from '../e2e/helpers.ts'
const HEADERS = { 'Content-Type': 'application/json', 'X-CSRF-Protection': '1', Origin: 'http://localhost:5173' }

async function create() {
  if (existsSync(FUND_WALLET_FILE)) throw new Error(`${FUND_WALLET_FILE.pathname} already exists`)
  const keypair = Keypair.random()
  const response = await fetch(`https://friendbot.stellar.org/?addr=${keypair.publicKey()}`)
  if (!response.ok) throw new Error(`friendbot: ${response.status} ${await response.text()}`)
  saveFundKeypair(keypair)
  console.log(`fund wallet ${keypair.publicKey()} created with 10,000 XLM`)
}

async function call(path: string, cookie: string, init: RequestInit = {}) {
  const response = await fetch(API + '/api/v1' + path, { ...init, headers: { ...HEADERS, Cookie: cookie } })
  const body = response.status === 204 ? null : await response.json()
  if (!response.ok) throw new Error(`${init.method ?? 'GET'} ${path}: ${response.status} ${JSON.stringify(body)}`)
  return body?.data
}

async function login(email: string) {
  const response = await fetch(API + '/api/v1/auth/login', { method: 'POST', headers: HEADERS, body: JSON.stringify({ email, password: PASSWORD }) })
  if (!response.ok) throw new Error(`login ${email}: ${response.status}`)
  return (response.headers.get('set-cookie') ?? '').split(';')[0]
}

// Only wallets made by the current MPC cluster can sign, so pass the time it started: `sweep 2026-10-27T06:39`.
async function sweep(since: string) {
  const fund = fundKeypair().publicKey()
  const rows = execFileSync('sqlite3', ['-separator', '|', DB,
    `SELECT w.id, w.address, w.workspace_id, u.email FROM wallets w JOIN users u ON u.id = w.user_id
     WHERE w.status = 'ready' AND w.pubkey != '' AND u.email LIKE 'e2e_%' AND w.created_at >= '${since}' ORDER BY w.created_at`,
  ], { encoding: 'utf8' }).trim().split('\n').filter(Boolean)
  let total = 0
  for (const row of rows) {
    const [walletId, address, workspaceId, email] = row.split('|')
    let account
    try {
      account = await nativeBalance(address)
    } catch {
      continue
    }
    const spendable = account.balance - (1 + 0.5 * account.subentries) - 0.01
    if (spendable < 1) continue
    const amount = spendable.toFixed(7)
    try {
      const cookie = await login(email)
      if (!workspaceId.startsWith('personal-')) await call(`/workspaces/${workspaceId}/approval-policy`, cookie, { method: 'DELETE' })
      const result = await call('/transactions', cookie, { method: 'POST', body: JSON.stringify({ wallet_id: walletId, to: fund, amount }) })
      let status = result.transaction.status
      for (let attempt = 0; attempt < 30 && (status === 'signing' || status === 'broadcast'); attempt++) {
        await new Promise((resolve) => setTimeout(resolve, 2000))
        status = (await call(`/transactions/${result.transaction.id}`, cookie)).status
      }
      console.log(`${address} → ${amount} XLM: ${status}`)
      if (status === 'confirmed') total += Number(amount)
    } catch (error) {
      console.log(`${address}: skipped (${(error as Error).message})`)
    }
  }
  console.log(`swept ${total.toFixed(2)} XLM into ${fund}; balance ${(await nativeBalance(fund)).balance} XLM (${HORIZON})`)
}

const [command, argument] = process.argv.slice(2)
if (command === 'create') await create()
else if (command === 'sweep' && argument) await sweep(argument)
else if (command === 'balance') console.log((await nativeBalance(fundKeypair().publicKey())).balance, 'XLM')
else throw new Error('usage: e2e-fund-wallet.ts create | sweep <since ISO time> | balance')
