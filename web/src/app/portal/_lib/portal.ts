// Client-side helper for the multi-tenant client portal. The portal token is
// separate from the console token and kept per-browser; every call hits
// /api/portal/* which scopes data to the caller's organization server-side.
const KEY = "nexus_portal_token";

export function getPortalToken(): string | null {
  try {
    return localStorage.getItem(KEY);
  } catch {
    return null;
  }
}
export function setPortalToken(token: string) {
  try {
    localStorage.setItem(KEY, token);
  } catch {
    /* ignore */
  }
}
export function clearPortalToken() {
  try {
    localStorage.removeItem(KEY);
  } catch {
    /* ignore */
  }
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getPortalToken();
  const res = await fetch(`/api/portal${path}`, {
    ...init,
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers as Record<string, string> | undefined),
    },
  });
  if (res.status === 401) {
    clearPortalToken();
    throw new Error("unauthorized");
  }
  if (!res.ok) {
    const data = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(data.error || `error ${res.status}`);
  }
  return res.status === 204 ? (undefined as T) : ((await res.json()) as T);
}

export type Org = { id: number; name: string; created_at: string };
export type PortalRole = "owner" | "admin" | "member";
export type PortalRequest = {
  id: number;
  title: string;
  scope: string;
  status: string;
  created_at: string;
};
export type Member = {
  id: number;
  email: string;
  role: PortalRole;
  created_at: string;
};
export type Invite = {
  id: number;
  email: string;
  role: PortalRole;
  created_at: string;
};
export type Me = { user: { id: number; role: PortalRole; email: string }; org: Org };

export const portalApi = {
  register: (org_name: string, email: string, password: string) =>
    req<{ token: string; org: Org }>("/register", {
      method: "POST",
      body: JSON.stringify({ org_name, email, password }),
    }),
  login: (email: string, password: string) =>
    req<{ token: string }>("/login", { method: "POST", body: JSON.stringify({ email, password }) }),
  me: () => req<Me>("/me"),
  listRequests: () => req<{ requests: PortalRequest[] }>("/requests"),
  createRequest: (title: string, scope: string) =>
    req<{ ok: boolean; id: number }>("/requests", { method: "POST", body: JSON.stringify({ title, scope }) }),

  // --- team management (admin+) ---
  listMembers: () => req<{ members: Member[] }>("/members"),
  updateMemberRole: (id: number, role: PortalRole) =>
    req<{ ok: boolean }>(`/members/${id}`, { method: "PATCH", body: JSON.stringify({ role }) }),
  removeMember: (id: number) => req<{ ok: boolean }>(`/members/${id}`, { method: "DELETE" }),
  listInvites: () => req<{ invites: Invite[] }>("/invites"),
  createInvite: (email: string, role: PortalRole) =>
    req<{ ok: boolean; id: number; token: string; email: string; role: PortalRole }>("/invites", {
      method: "POST",
      body: JSON.stringify({ email, role }),
    }),
  revokeInvite: (id: number) => req<{ ok: boolean }>(`/invites/${id}`, { method: "DELETE" }),

  // --- invitation acceptance (public, token-gated; no auth header needed) ---
  lookupInvite: (token: string) =>
    req<{ email: string; role: PortalRole; org: string }>(`/invites/lookup?token=${encodeURIComponent(token)}`),
  acceptInvite: (token: string, password: string) =>
    req<{ token: string }>("/invites/accept", {
      method: "POST",
      body: JSON.stringify({ token, password }),
    }),
};
