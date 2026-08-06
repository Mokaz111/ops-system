import type { User } from '../types/api';

export function isPlatformAdmin(user: User | null | undefined): boolean {
  return user?.role === 'admin';
}

export function getMembershipRole(user: User | null | undefined, workspaceId?: string): string | null {
  if (!user) return null;
  if (isPlatformAdmin(user)) return 'admin';
  if (!user.memberships?.length) return null;
  if (workspaceId) {
    return user.memberships.find((m) => m.workspace_id === workspaceId)?.role ?? null;
  }
  const order = ['admin', 'member', 'viewer'];
  for (const role of order) {
    if (user.memberships.some((m) => m.role === role)) return role;
  }
  return user.memberships[0]?.role ?? null;
}

export function isWorkspaceAdmin(user: User | null | undefined, workspaceId?: string): boolean {
  return isPlatformAdmin(user) || getMembershipRole(user, workspaceId) === 'admin';
}

export function getPrimaryWorkspaceId(user: User | null | undefined): string | null {
  if (!user?.memberships?.length) return null;
  const adminMembership = user.memberships.find((m) => m.role === 'admin');
  return (adminMembership ?? user.memberships[0]).workspace_id;
}

export function formatMemberships(user: User | null | undefined): string {
  if (!user?.memberships?.length) return '-';
  return user.memberships.map((m) => `${m.workspace_name} (${m.role})`).join(', ');
}
