import type { PaymentRequest } from '../types.ts'

export function approvalCount(request: PaymentRequest) {
  return request.votes.filter((vote) => vote.decision === 'approve' && vote.counted).length
}
