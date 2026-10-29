import type { WorkspaceRole } from '../types.ts'

export const roles: WorkspaceRole[] = ['owner', 'operator', 'member']

export const roleLabels: Record<WorkspaceRole, string> = { owner: 'Owner', operator: 'Operator', member: 'Member' }
