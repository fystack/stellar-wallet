import type { PaymentRequest } from '../types.ts'

export function approvalCount(request: PaymentRequest) {
  return request.votes.filter((vote) => vote.decision === 'approve' && vote.counted).length
}

export function sameEmail(a: string, b: string) {
  return a.toLowerCase() === b.toLowerCase()
}

export function isInitiator(request: PaymentRequest, email: string) {
  return sameEmail(request.initiatorEmail, email)
}

export function hasVoted(request: PaymentRequest, email: string) {
  return request.votes.some((vote) => sameEmail(vote.email, email))
}
