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
  disabled: boolean;
  created_at: string;
};
export type Invite = {
  id: number;
  email: string;
  role: PortalRole;
  created_at: string;
};
export type ViewProfile = "" | "executive" | "technical";
export type Me = {
  user: {
    id: number;
    role: PortalRole;
    email: string;
    email_verified: boolean;
    view_profile: ViewProfile;
    lang: "" | "es" | "en";
  };
  org: Org;
};

export type SeverityBucket = { open: number; in_progress: number; remediated: number; risk_accepted: number };
export type OverviewFinding = {
  id: number;
  request_id: number;
  request_title: string;
  name: string;
  vulnclass: string;
  severity: string;
  status: string;
  created_at: string;
};
export type OverviewResponse = {
  overview: {
    requests: Record<string, number>;
    requests_total: number;
    released_requests: number;
    by_severity: Record<string, SeverityBucket>;
    top_open: OverviewFinding[];
    monthly: { month: string; count: number }[];
  };
  generated_at: string;
  // admin+ only
  account_security?: {
    active_members: number;
    mfa_enabled: number;
    unverified: number;
    disabled: number;
    pending_invites: number;
  };
  activity?: AuditEntry[];
};
export type AuditEntry = {
  id: number;
  actor: string;
  actor_type: string;
  action: string;
  target: string;
  detail?: string;
  ip: string;
  created_at: string;
};

export type Severity = "critical" | "high" | "medium" | "low";
export type ClientFinding = {
  id: number;
  name: string;
  vulnclass: string;
  severity: Severity;
  status: string;
  summary: string;
  created_at: string;
};
export type ClientFindingDetail = ClientFinding & { report: string };
export type ResultsResponse = {
  state: "not_started" | "in_progress" | "released";
  released?: boolean;
  findings: ClientFinding[];
  summary?: Record<Severity, number>;
  total?: number;
};

export const portalApi = {
  register: (org_name: string, email: string, password: string) =>
    req<{ token: string; org: Org }>("/register", {
      method: "POST",
      body: JSON.stringify({ org_name, email, password }),
    }),
  login: (email: string, password: string, code?: string) =>
    req<{ token: string }>("/login", { method: "POST", body: JSON.stringify({ email, password, code }) }),
  me: () => req<Me>("/me"),
  setPreferences: (prefs: { view_profile?: ViewProfile; lang?: "es" | "en" }) =>
    req<{ ok: boolean }>("/me/preferences", { method: "PATCH", body: JSON.stringify(prefs) }),
  overview: () => req<OverviewResponse>("/overview"),
  listRequests: () => req<{ requests: PortalRequest[] }>("/requests"),
  createRequest: (title: string, scope: string) =>
    req<{ ok: boolean; id: number }>("/requests", { method: "POST", body: JSON.stringify({ title, scope }) }),
  requestResults: (id: number) => req<ResultsResponse>(`/requests/${id}/results`),
  requestFinding: (id: number, fid: number) => req<{ finding: ClientFindingDetail }>(`/requests/${id}/findings/${fid}`),

  // --- team management (admin+) ---
  listMembers: () => req<{ members: Member[] }>("/members"),
  updateMemberRole: (id: number, role: PortalRole) =>
    req<{ ok: boolean }>(`/members/${id}`, { method: "PATCH", body: JSON.stringify({ role }) }),
  removeMember: (id: number) => req<{ ok: boolean }>(`/members/${id}`, { method: "DELETE" }),
  setMemberDisabled: (id: number, disabled: boolean) =>
    req<{ ok: boolean; disabled: boolean }>(`/members/${id}/disable`, {
      method: "POST",
      body: JSON.stringify({ disabled }),
    }),
  revokeMemberSessions: (id: number) =>
    req<{ ok: boolean }>(`/members/${id}/revoke-sessions`, { method: "POST", body: "{}" }),
  listInvites: () => req<{ invites: Invite[] }>("/invites"),
  createInvite: (email: string, role: PortalRole) =>
    req<{ ok: boolean; id: number; token: string; email: string; role: PortalRole }>("/invites", {
      method: "POST",
      body: JSON.stringify({ email, role }),
    }),
  revokeInvite: (id: number) => req<{ ok: boolean }>(`/invites/${id}`, { method: "DELETE" }),
  listAudit: () => req<{ entries: AuditEntry[] }>("/audit"),

  // --- per-user 2FA (TOTP) ---
  mfaStatus: () => req<{ enabled: boolean }>("/mfa/status"),
  mfaSetup: () => req<{ secret: string; otpauth: string }>("/mfa/setup", { method: "POST", body: "{}" }),
  mfaEnable: (code: string) => req<{ ok: boolean }>("/mfa/enable", { method: "POST", body: JSON.stringify({ code }) }),
  mfaDisable: (code: string) =>
    req<{ ok: boolean }>("/mfa/disable", { method: "POST", body: JSON.stringify({ code }) }),

  // --- account recovery + email verification ---
  forgotPassword: (email: string) =>
    req<{ ok: boolean }>("/password/forgot", { method: "POST", body: JSON.stringify({ email }) }),
  resetPassword: (token: string, password: string) =>
    req<{ ok: boolean }>("/password/reset", { method: "POST", body: JSON.stringify({ token, password }) }),
  changePassword: (old_password: string, new_password: string) =>
    req<{ ok: boolean }>("/password/change", {
      method: "POST",
      body: JSON.stringify({ old_password, new_password }),
    }),
  verifyEmail: (token: string) =>
    req<{ ok: boolean }>("/email/verify", { method: "POST", body: JSON.stringify({ token }) }),
  resendVerification: () =>
    req<{ ok: boolean; already_verified?: boolean }>("/email/resend", { method: "POST", body: "{}" }),

  // --- GDPR data rights ---
  deleteAccount: (password: string) =>
    req<{ ok: boolean }>("/account/delete", { method: "POST", body: JSON.stringify({ password }) }),

  // --- invitation acceptance (public, token-gated; no auth header needed) ---
  lookupInvite: (token: string) =>
    req<{ email: string; role: PortalRole; org: string }>(`/invites/lookup?token=${encodeURIComponent(token)}`),
  acceptInvite: (token: string, password: string) =>
    req<{ token: string }>("/invites/accept", {
      method: "POST",
      body: JSON.stringify({ token, password }),
    }),
};

// downloadExport fetches the org's data export and triggers a file download. It
// streams a blob (not JSON), so it bypasses the JSON `req` helper.
export async function downloadExport(): Promise<void> {
  const token = getPortalToken();
  const res = await fetch("/api/portal/export", {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) {
    throw new Error("No se pudo exportar");
  }
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = "nexus-export.json";
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
