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
export type PortalRequest = {
  id: number;
  title: string;
  scope: string;
  status: string;
  created_at: string;
};

export const portalApi = {
  register: (org_name: string, email: string, password: string) =>
    req<{ token: string; org: Org }>("/register", {
      method: "POST",
      body: JSON.stringify({ org_name, email, password }),
    }),
  login: (email: string, password: string) =>
    req<{ token: string }>("/login", { method: "POST", body: JSON.stringify({ email, password }) }),
  me: () => req<{ user: { id: number }; org: Org }>("/me"),
  listRequests: () => req<{ requests: PortalRequest[] }>("/requests"),
  createRequest: (title: string, scope: string) =>
    req<{ ok: boolean; id: number }>("/requests", { method: "POST", body: JSON.stringify({ title, scope }) }),
};
