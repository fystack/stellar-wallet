import { readFileSync, writeFileSync } from 'node:fs'
import { Asset, BASE_FEE, Horizon, Keypair, Networks, Operation, TransactionBuilder } from '@stellar/stellar-sdk'

export const HORIZON = process.env.E2E_HORIZON ?? 'https://horizon-testnet.stellar.org'
// A plain (non-MPC) testnet account that funds every e2e wallet, so tests don't depend on Friendbot.
export const FUND_WALLET_FILE = new URL('./.fund-wallet.json', import.meta.url)

export function fundKeypair(): Keypair {
  const secret = process.env.E2E_FUND_SECRET ?? JSON.parse(readFileSync(FUND_WALLET_FILE, 'utf8')).secret
  return Keypair.fromSecret(secret)
}

export function saveFundKeypair(keypair: Keypair) {
  writeFileSync(FUND_WALLET_FILE, JSON.stringify({ publicKey: keypair.publicKey(), secret: keypair.secret() }, null, 2) + '\n', { mode: 0o600 })
}

async function accountExists(server: Horizon.Server, address: string): Promise<boolean> {
  try {
    await server.loadAccount(address)
    return true
  } catch (error) {
    if ((error as { response?: { status?: number } }).response?.status === 404) return false
    throw error
  }
}

function resultCode(error: unknown): string {
  const data = (error as { response?: { data?: { extras?: { result_codes?: { transaction?: string } } } } }).response?.data
  return data?.extras?.result_codes?.transaction ?? ''
}

// Sends XLM from the fund wallet, creating the destination account when it doesn't exist yet.
// Parallel workers share the fund account, so a sequence clash (tx_bad_seq) is retried with a fresh sequence.
export async function fundAccount(destination: string, amount: string) {
  const server = new Horizon.Server(HORIZON)
  const source = fundKeypair()
  const operation = (await accountExists(server, destination))
    ? Operation.payment({ destination, asset: Asset.native(), amount })
    : Operation.createAccount({ destination, startingBalance: amount })
  for (let attempt = 1; ; attempt++) {
    const transaction = new TransactionBuilder(await server.loadAccount(source.publicKey()), { fee: BASE_FEE, networkPassphrase: Networks.TESTNET })
      .addOperation(operation)
      .setTimeout(60)
      .build()
    transaction.sign(source)
    try {
      await server.submitTransaction(transaction)
      return
    } catch (error) {
      if (resultCode(error) !== 'tx_bad_seq' || attempt >= 5) throw error
      await new Promise((resolve) => setTimeout(resolve, 500 * attempt + Math.random() * 500))
    }
  }
}

export async function nativeBalance(address: string): Promise<{ balance: number; subentries: number }> {
  const account = await new Horizon.Server(HORIZON).loadAccount(address)
  const native = account.balances.find((entry) => entry.asset_type === 'native')
  return { balance: Number(native?.balance ?? 0), subentries: account.subentry_count }
}
