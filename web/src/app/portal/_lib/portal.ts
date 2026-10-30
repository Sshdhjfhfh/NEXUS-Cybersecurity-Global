// Client-side helper for the multi-tenant client portal. The portal token is
// separate from the console token and kept per-browser; every call hits
// /api/portal/* which scopes data to the caller's organization server-side.
import { localizeServerError } from "./i18n";

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
    throw new Error(data.error ? localizeServerError(data.error) : `error ${res.status}`);
  }
  return res.status === 204 ? (undefined as T) : ((await res.json()) as T);
}

export type Org = { id: number; name: string; created_at: string };
// "auditor" is read-only: it sees what an admin sees and changes nothing.
export type PortalRole = "owner" | "admin" | "member" | "auditor";
export type PortalRequest = {
  id: number;
  title: string;
  scope: string;
  status: string;
  created_at: string;
  authorization?: RequestAuthorization;
};

// Targets the org has registered; only verified ones can be tested. "expired"
// is derived server-side once a verification is older than its validity.
export type TargetStatus = "pending" | "verified" | "rejected" | "expired";
export type PortalTarget = {
  id: number;
  kind: "domain" | "ip";
  value: string;
  token?: string; // proof token, only in the targets listing
  status: TargetStatus;
  method?: "dns" | "http" | "manual";
  verified_at?: string;
  expires_at?: string;
  note?: string;
  last_check_at?: string;
  last_error?: string;
  created_at: string;
};
export type TargetsResponse = {
  targets: PortalTarget[];
  validity_days: number;
  dns_prefix: string;
  http_path: string;
};
export type RoE = {
  request_id: number;
  accepted_email: string;
  signer_name: string;
  signer_title: string;
  window_start: string;
  window_end: string;
  emergency_contact: string;
  exclusions: string;
  roe_version: string;
  roe_hash: string;
  accepted_at: string;
};
export type RequestAuthorization = { targets: PortalTarget[]; roe?: RoE; ready: boolean };
export type RoETerms = { version: string; lang: "es" | "en"; text: string[]; hash: string };
export type AuthorizationResponse = {
  authorization: RequestAuthorization;
  promoted: boolean;
  terms: RoETerms;
  max_window_days: number;
};
export type RoEAcceptance = {
  signer_name: string;
  signer_title: string;
  window_start: string;
  window_end: string;
  emergency_contact: string;
  exclusions: string;
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
  expires_at: string;
  expired: boolean;
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
    notify_email?: boolean;
    mfa_enabled?: boolean;
    // The org requires 2FA and this user has not enrolled: the server refuses
    // everything except enrollment until they do.
    mfa_enrollment_required?: boolean;
    org_requires_mfa?: boolean;
    signed_in_with_sso?: boolean;
  };
  org: Org;
};

// Org security policy (owner edits; admins and auditors read).
export type SecurityPolicy = {
  require_mfa: boolean;
  ip_allowlist: string[];
  session_max_hours: number;
  updated_at?: string | null;
  updated_by?: string;
};
// Single sign-on (OIDC). The client secret is write-only: the API only says
// whether one is stored.
export type SSOConfig = {
  issuer: string;
  client_id: string;
  has_secret: boolean;
  email_domains: string[];
  enabled: boolean;
  enforce: boolean;
  auto_provision: boolean;
  default_role: "member" | "auditor";
  updated_at?: string | null;
  updated_by?: string;
};
export type SSOResponse = { sso: SSOConfig | null; redirect_uri: string; verified_domains: string[] };
export type SSOInput = Omit<SSOConfig, "has_secret" | "updated_at" | "updated_by"> & { client_secret: string };

// One signed-in device of the current user. browser/os are "" when unknown.
export type PortalSession = {
  id: string;
  amr: "pwd" | "sso";
  ip: string;
  browser: string;
  os: string;
  created_at: string;
  last_seen_at: string;
  expires_at: string;
  current: boolean;
};

export type SecurityPolicyResponse = {
  policy: SecurityPolicy;
  your_ip: string;
  max_session_hours: number;
  members_without_mfa: string[];
};

export type SeverityBucket = { open: number; in_progress: number; remediated: number; risk_accepted: number };
export type OverviewFinding = {
  id: number;
  request_id: number;
  request_title: string;
  name: string;
  vulnclass: string;
  severity: string;
  status: RemediationState;
  created_at: string;
  due_date?: string;
  overdue: boolean;
};
export type OverviewResponse = {
  overview: {
    requests: Record<string, number>;
    requests_total: number;
    released_requests: number;
    by_severity: Record<string, SeverityBucket>;
    top_open: OverviewFinding[];
    overdue: number;
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

// Filters for the audit trail; dates are YYYY-MM-DD (UTC), both inclusive.
export type AuditCategory = "" | "access" | "team" | "config" | "work";
export type AuditFilter = { category?: AuditCategory; q?: string; from?: string; to?: string };

function auditQuery(f: AuditFilter, before?: number): string {
  const p = new URLSearchParams();
  if (f.category) p.set("category", f.category);
  if (f.q?.trim()) p.set("q", f.q.trim());
  if (f.from) p.set("from", f.from);
  if (f.to) p.set("to", f.to);
  if (before) p.set("before", String(before));
  return p.toString();
}

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

// Effective remediation state, computed server-side from the client's progress,
// NEXUS retests and risk acceptances. Only a NEXUS retest yields verified_fixed.
export type RemediationState =
  | "open"
  | "in_progress"
  | "retest_requested"
  | "retest_running"
  | "still_vulnerable"
  | "verified_fixed"
  | "risk_accepted";
export type RemediationBucket = "open" | "in_progress" | "remediated" | "risk_accepted";
export type RemediationItem = {
  finding_id: number;
  request_id: number;
  request_title: string;
  name: string;
  vulnclass: string;
  severity: string;
  summary: string;
  detected_at: string;
  state: RemediationState;
  bucket: RemediationBucket;
  assignee_id: number | null;
  assignee_email?: string;
  due_date?: string; // YYYY-MM-DD
  due_source?: "custom" | "sla";
  sla_days?: number;
  overdue: boolean;
  risk_reason?: string;
  risk_expires_at?: string;
  risk_expired?: boolean;
  risk_accepted_by?: string;
  retest_requested_at?: string;
  retest_note?: string;
  last_retest?: {
    status: "pending" | "running" | "completed" | "failed" | "stopped";
    verdict: "" | "reproduced" | "fixed" | "inconclusive";
    created_at: string;
    finished_at?: string;
  };
};
export type RemediationEvent = { id: number; actor: string; kind: string; detail: string; created_at: string };
export type RemediationUpdate = {
  assignee_id?: number | null;
  due_date?: string | null;
  state?: "open" | "in_progress";
};
type ItemResponse = { ok: boolean; item: RemediationItem };

export const portalApi = {
  register: (org_name: string, email: string, password: string) =>
    req<{ token: string; org: Org }>("/register", {
      method: "POST",
      body: JSON.stringify({ org_name, email, password }),
    }),
  login: (email: string, password: string, code?: string) =>
    req<{ token: string }>("/login", { method: "POST", body: JSON.stringify({ email, password, code }) }),
  me: () => req<Me>("/me"),
  setPreferences: (prefs: { view_profile?: ViewProfile; lang?: "es" | "en"; notify_email?: boolean }) =>
    req<{ ok: boolean }>("/me/preferences", { method: "PATCH", body: JSON.stringify(prefs) }),
  overview: () => req<OverviewResponse>("/overview"),
  listRequests: () => req<{ requests: PortalRequest[] }>("/requests"),
  createRequest: (title: string, scope: string, target_ids: number[]) =>
    req<{ ok: boolean; id: number }>("/requests", {
      method: "POST",
      body: JSON.stringify({ title, scope, target_ids }),
    }),
  requestAuthorization: (id: number, lang: "es" | "en") =>
    req<AuthorizationResponse>(`/requests/${id}/authorization?lang=${lang}`),
  acceptRoE: (id: number, terms: RoETerms, a: RoEAcceptance) =>
    req<{ ok: boolean; authorization: RequestAuthorization }>(`/requests/${id}/roe`, {
      method: "POST",
      body: JSON.stringify({ accept: true, lang: terms.lang, version: terms.version, hash: terms.hash, ...a }),
    }),

  // --- targets: ownership verification ---
  listTargets: () => req<TargetsResponse>("/targets"),
  addTarget: (value: string) =>
    req<{ ok: boolean; target: PortalTarget }>("/targets", { method: "POST", body: JSON.stringify({ value }) }),
  verifyTarget: (id: number, method: "dns" | "http") =>
    req<{ ok: boolean; target: PortalTarget; error: string }>(`/targets/${id}/verify`, {
      method: "POST",
      body: JSON.stringify({ method }),
    }),
  deleteTarget: (id: number) => req<{ ok: boolean }>(`/targets/${id}`, { method: "DELETE" }),
  requestResults: (id: number) => req<ResultsResponse>(`/requests/${id}/results`),
  requestFinding: (id: number, fid: number) => req<{ finding: ClientFindingDetail }>(`/requests/${id}/findings/${fid}`),

  // --- remediation workspace ---
  listRemediation: () => req<{ items: RemediationItem[]; sla_days: Record<string, number> }>("/remediation"),
  getRemediation: (fid: number) =>
    req<{ item: RemediationItem; report: string; events: RemediationEvent[] }>(`/remediation/${fid}`),
  updateRemediation: (fid: number, patch: RemediationUpdate) =>
    req<ItemResponse>(`/remediation/${fid}`, { method: "PATCH", body: JSON.stringify(patch) }),
  commentRemediation: (fid: number, text: string) =>
    req<{ ok: boolean }>(`/remediation/${fid}/comments`, { method: "POST", body: JSON.stringify({ text }) }),
  requestRetest: (fid: number, note: string) =>
    req<ItemResponse>(`/remediation/${fid}/retest`, { method: "POST", body: JSON.stringify({ note }) }),
  acceptRisk: (fid: number, reason: string, expires_at: string) =>
    req<ItemResponse>(`/remediation/${fid}/accept-risk`, {
      method: "POST",
      body: JSON.stringify({ reason, expires_at }),
    }),
  revokeRisk: (fid: number) => req<ItemResponse>(`/remediation/${fid}/revoke-risk`, { method: "POST", body: "{}" }),

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
  listAudit: (f: AuditFilter = {}, before?: number) =>
    req<{ entries: AuditEntry[]; has_more: boolean }>(`/audit?${auditQuery(f, before)}`),

  // --- per-user 2FA (TOTP) ---
  mfaStatus: () => req<{ enabled: boolean }>("/mfa/status"),
  mfaSetup: () => req<{ secret: string; otpauth: string }>("/mfa/setup", { method: "POST", body: "{}" }),
  mfaEnable: (code: string) => req<{ ok: boolean }>("/mfa/enable", { method: "POST", body: JSON.stringify({ code }) }),
  mfaDisable: (code: string) =>
    req<{ ok: boolean }>("/mfa/disable", { method: "POST", body: JSON.stringify({ code }) }),

  // --- single sign-on ---
  ssoStart: (email: string) => req<{ url: string }>("/sso/start", { method: "POST", body: JSON.stringify({ email }) }),
  ssoCallback: (code: string, state: string) =>
    req<{ token: string }>("/sso/callback", { method: "POST", body: JSON.stringify({ code, state }) }),
  getSSO: () => req<SSOResponse>("/sso"),
  setSSO: (c: SSOInput) => req<SSOResponse>("/sso", { method: "PUT", body: JSON.stringify(c) }),
  deleteSSO: () => req<{ ok: boolean }>("/sso", { method: "DELETE" }),

  // --- the caller's own sessions ---
  logout: () => req<{ ok: boolean }>("/logout", { method: "POST", body: "{}" }),
  listSessions: () => req<{ sessions: PortalSession[] }>("/sessions"),
  revokeSession: (id: string) => req<{ ok: boolean }>(`/sessions/${encodeURIComponent(id)}`, { method: "DELETE" }),
  revokeOtherSessions: () => req<{ revoked: number }>("/sessions/revoke-others", { method: "POST", body: "{}" }),

  // --- org security policy ---
  getSecurityPolicy: () => req<SecurityPolicyResponse>("/security-policy"),
  setSecurityPolicy: (p: { require_mfa: boolean; ip_allowlist: string[]; session_max_hours: number }) =>
    req<SecurityPolicyResponse>("/security-policy", { method: "PUT", body: JSON.stringify(p) }),

  // --- account recovery + email verification ---
  forgotPassword: (email: string) =>
    req<{ ok: boolean }>("/password/forgot", { method: "POST", body: JSON.stringify({ email }) }),
  resetPassword: (token: string, password: string) =>
    req<{ ok: boolean }>("/password/reset", { method: "POST", body: JSON.stringify({ token, password }) }),
  changePassword: (old_password: string, new_password: string) =>
    req<{ ok: boolean; other_sessions_ended?: number }>("/password/change", {
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
    throw new Error(localizeServerError("no se pudo exportar"));
  }
  saveBlob(await res.blob(), "nexus-export.json");
}

// downloadAuditCSV saves the filtered audit trail as CSV (the export is itself
// recorded in the trail).
export async function downloadAuditCSV(f: AuditFilter): Promise<{ truncated: boolean }> {
  const token = getPortalToken();
  const res = await fetch(`/api/portal/audit/export?${auditQuery(f)}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (res.status === 401) {
    clearPortalToken();
    throw new Error("unauthorized");
  }
  if (!res.ok) {
    const data = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(localizeServerError(data.error ?? "no se pudo leer la auditoría"));
  }
  const name = /filename="([^"]+)"/.exec(res.headers.get("Content-Disposition") ?? "")?.[1];
  saveBlob(await res.blob(), name ?? "nexus-audit.csv");
  return { truncated: res.headers.get("X-Nexus-Truncated") === "true" };
}

export type ReportKind = "executive" | "technical";

// downloadReport fetches a branded PDF report of a released evaluation, in the
// UI language, and saves it under the file name the server chose.
export async function downloadReport(requestId: number, kind: ReportKind, lang: string): Promise<void> {
  const token = getPortalToken();
  const res = await fetch(`/api/portal/requests/${requestId}/report?kind=${kind}&lang=${encodeURIComponent(lang)}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (res.status === 401) {
    clearPortalToken();
    throw new Error("unauthorized");
  }
  if (!res.ok) {
    const data = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(localizeServerError(data.error ?? "no se pudo generar el informe"));
  }
  const name = /filename="([^"]+)"/.exec(res.headers.get("Content-Disposition") ?? "")?.[1];
  saveBlob(await res.blob(), name ?? `NEXUS-${kind}-report.pdf`);
}

function saveBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}
