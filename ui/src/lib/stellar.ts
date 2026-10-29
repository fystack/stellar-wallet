// Stellar base fee: 100 stroops per operation.
export const BASE_FEE_XLM = '0.00001'
const BASE_RESERVE_XLM = 1

export const TRUSTLINE_MEMO = 'Add trustline'

// XLM must keep the base reserve plus the fee; other assets can be sent in full.
export function maxSendable(balance: number, isNative: boolean): number {
  return isNative ? Math.max(0, balance - Number(BASE_FEE_XLM) - BASE_RESERVE_XLM) : balance
}

export function exceedsMax(amount: number, max: number): boolean {
  return amount > max + 1e-9
}

// "XLM" or "CODE:ISSUER" → the asset code shown to users.
export function assetLabel(value: string): string {
  return value === 'XLM' ? 'XLM' : value.split(':')[0]
}
