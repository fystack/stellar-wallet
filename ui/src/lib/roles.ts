import type { WorkspaceRole } from '../types.ts'

export const roles: WorkspaceRole[] = ['owner', 'operator', 'member']

export const roleLabels: Record<WorkspaceRole, string> = { owner: 'Owner', operator: 'Operator', member: 'Member' }

export const roleHints: Record<WorkspaceRole, string> = {
  owner: 'Manages members, wallets and the approval policy',
  operator: 'Sends payments and approves others',
  member: 'Views wallets and history only',
}
