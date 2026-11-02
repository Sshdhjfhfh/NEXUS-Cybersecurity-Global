"use client";

import { type FormEvent, useCallback, useEffect, useState } from "react";

import { CopyIcon, LogOutIcon, SendIcon, Trash2Icon, UserPlusIcon } from "lucide-react";

import { usePortalI18n } from "./i18n";
import { type Invite, type IssuedInvite, type Me, type Member, type PortalRole, portalApi } from "./portal";
import { Field, PrimaryButton } from "./ui";

function RoleBadge({ role }: { role: PortalRole }) {
  const { t } = usePortalI18n();
  const tone =
    role === "owner"
      ? "border-amber-200 bg-amber-50 text-amber-800"
      : role === "admin"
        ? "border-cyan-200 bg-cyan-50 text-cyan-700"
        : role === "auditor"
          ? "border-violet-200 bg-violet-50 text-violet-700"
          : "border-slate-200 bg-slate-100 text-slate-600";
  return (
    <span className={`rounded-full border px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${tone}`}>
      {t.role[role] ?? role}
    </span>
  );
}

// TeamPanel is mounted for admin/owner, and read-only for an auditor (the
// server enforces both: an auditor's writes are refused).
export function TeamPanel({ me }: { me: Me }) {
  const { t, fmtDate } = usePortalI18n();
  const [members, setMembers] = useState<Member[]>([]);
  const [invites, setInvites] = useState<Invite[]>([]);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<PortalRole>("member");
  const [newLink, setNewLink] = useState("");
  const [sentNote, setSentNote] = useState("");
  const [resending, setResending] = useState(0);
  const [copied, setCopied] = useState(false);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);

  const errText = (ex: unknown, fallback: string) => (ex instanceof Error && ex.message ? ex.message : fallback);

  const load = useCallback(async () => {
    setErr("");
    try {
      const [{ members: m }, { invites: i }] = await Promise.all([portalApi.listMembers(), portalApi.listInvites()]);
      setMembers(m ?? []);
      setInvites(i ?? []);
    } catch (ex) {
      setErr(ex instanceof Error && ex.message ? ex.message : t.team.loadError);
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  async function invite(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr("");
    setNewLink("");
    try {
      const res = await portalApi.createInvite(email.trim().toLowerCase(), role);
      showIssued(res, t.team.sentTo(res.email));
      setEmail("");
      setRole("member");
      await load();
    } catch (ex) {
      setErr(errText(ex, t.team.inviteError));
    } finally {
      setBusy(false);
    }
  }

  async function changeRole(id: number, next: PortalRole) {
    setErr("");
    try {
      await portalApi.updateMemberRole(id, next);
      await load();
    } catch (ex) {
      setErr(errText(ex, t.team.roleError));
    }
  }

  async function remove(id: number, memberEmail: string) {
    setErr("");
    if (!window.confirm(t.team.confirmRemove(memberEmail))) {
      return;
    }
    try {
      await portalApi.removeMember(id);
      await load();
    } catch (ex) {
      setErr(errText(ex, t.team.removeError));
    }
  }

  function showIssued(res: IssuedInvite, note: string) {
    setNewLink(`${window.location.origin}/portal/accept?token=${encodeURIComponent(res.token)}`);
    setSentNote(res.emailed ? note : "");
    setCopied(false);
  }

  // Resending issues a new link (the previous one stops working) and restarts
  // the validity period, so it also renews an expired invitation.
  async function resend(id: number) {
    setErr("");
    setResending(id);
    try {
      const res = await portalApi.resendInvite(id);
      showIssued(res, t.team.resent(res.email));
      await load();
    } catch (ex) {
      setErr(errText(ex, t.team.resendError));
    } finally {
      setResending(0);
    }
  }

  async function revoke(id: number) {
    setErr("");
    try {
      await portalApi.revokeInvite(id);
      await load();
    } catch (ex) {
      setErr(errText(ex, t.team.revokeError));
    }
  }

  async function toggleDisabled(id: number, disabled: boolean) {
    setErr("");
    try {
      await portalApi.setMemberDisabled(id, disabled);
      await load();
    } catch (ex) {
      setErr(errText(ex, t.team.updateError));
    }
  }

  async function revokeSessions(id: number, memberEmail: string) {
    setErr("");
    if (!window.confirm(t.team.confirmEndSessions(memberEmail))) {
      return;
    }
    try {
      await portalApi.revokeMemberSessions(id);
    } catch (ex) {
      setErr(errText(ex, t.team.sessionsError));
    }
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(newLink);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      /* clipboard may be unavailable; the link is still visible to copy manually */
    }
  }

  const canManageOwners = me.user.role === "owner";
  const readOnly = me.user.role === "auditor";

  if (loading) {
    return <p className="mt-4 text-sm text-slate-500">{t.team.loading}</p>;
  }

  return (
    <div className={`mt-4 grid gap-8 ${readOnly ? "" : "lg:grid-cols-[1fr_1.4fr]"}`}>
      {/* Invite */}
      {readOnly ? null : (
        <section className="h-fit rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-cyan-700">
            <UserPlusIcon className="size-4" aria-hidden /> {t.team.invite}
          </h2>
          <form onSubmit={invite} className="mt-4 space-y-4">
            <Field
              label={t.email}
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t.team.emailPlaceholder}
              required
            />
            <label className="block text-xs font-medium text-slate-600">
              {t.team.role}
              <select
                value={role}
                onChange={(e) => setRole(e.target.value as PortalRole)}
                className="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm text-slate-800 focus:border-cyan-400 focus:bg-white focus:outline-none focus:ring-4 focus:ring-cyan-100"
              >
                <option value="member">{t.team.roleMemberOption}</option>
                <option value="admin">{t.team.roleAdminOption}</option>
                <option value="auditor">{t.team.roleAuditorOption}</option>
              </select>
            </label>
            {err ? (
              <p role="alert" className="text-sm text-rose-600">
                {err}
              </p>
            ) : null}
            <PrimaryButton type="submit" disabled={busy || !email.trim()}>
              {busy ? t.team.generating : t.team.generate}
            </PrimaryButton>
          </form>
          {newLink ? (
            <div className="mt-5 rounded-xl border border-cyan-200 bg-cyan-50 p-4" aria-live="polite">
              {sentNote ? (
                <p className="mb-2 flex items-center gap-1.5 font-medium text-cyan-800 text-sm">
                  <SendIcon className="size-3.5" aria-hidden /> {sentNote}
                </p>
              ) : null}
              <p className="text-xs text-slate-600">{t.team.shareLink}</p>
              <div className="mt-2 flex items-center gap-2">
                <code className="min-w-0 flex-1 truncate rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-[11px] text-cyan-700">
                  {newLink}
                </code>
                <button
                  type="button"
                  onClick={copyLink}
                  className="inline-flex shrink-0 items-center gap-1 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-[11px] text-slate-600 hover:border-cyan-400 hover:text-cyan-700"
                >
                  <CopyIcon className="size-3.5" aria-hidden /> {copied ? t.copied : t.copy}
                </button>
              </div>
            </div>
          ) : null}
        </section>
      )}

      {/* Members + pending invites */}
      <section className="space-y-6">
        <div>
          <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-700">
            {t.team.members(members.length)}
          </h2>
          {readOnly ? <p className="mt-1 text-xs text-slate-500">{t.team.readOnly}</p> : null}
          {readOnly && err ? (
            <p role="alert" className="mt-2 text-sm text-rose-600">
              {err}
            </p>
          ) : null}
          <ul className="mt-4 space-y-3">
            {members.map((m) => {
              const isSelf = m.id === me.user.id;
              // Read-only for an auditor; an owner row is locked for non-owners.
              const lockedOwner = readOnly || (m.role === "owner" && !canManageOwners);
              return (
                <li key={m.id} className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div className="min-w-0">
                      <span className="block truncate font-medium text-slate-900">
                        {m.email}
                        {isSelf ? <span className="ml-1.5 text-[11px] text-slate-500">{t.team.you}</span> : null}
                        {m.disabled ? (
                          <span className="ml-2 rounded-full border border-rose-200 bg-rose-50 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-rose-700">
                            {t.team.disabled}
                          </span>
                        ) : null}
                        {me.terms && m.terms_version !== me.terms.version ? (
                          <span className="ml-2 rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 font-semibold text-[10px] text-amber-800 uppercase tracking-wide">
                            {t.team.termsPending}
                          </span>
                        ) : null}
                      </span>
                      <span className="text-[11px] text-slate-500">{t.team.since(fmtDate(m.created_at))}</span>
                    </div>
                    <div className="flex items-center gap-2">
                      {lockedOwner ? (
                        <RoleBadge role={m.role} />
                      ) : (
                        <select
                          value={m.role}
                          aria-label={t.team.role}
                          onChange={(e) => changeRole(m.id, e.target.value as PortalRole)}
                          className="rounded-md border border-slate-200 bg-white px-2 py-1 text-xs text-slate-700 focus:border-cyan-400 focus:outline-none focus:ring-2 focus:ring-cyan-100"
                        >
                          {canManageOwners ? <option value="owner">{t.role.owner}</option> : null}
                          <option value="admin">{t.role.admin}</option>
                          <option value="member">{t.role.member}</option>
                          <option value="auditor">{t.role.auditor}</option>
                        </select>
                      )}
                      {!isSelf && !lockedOwner ? (
                        <button
                          type="button"
                          onClick={() => toggleDisabled(m.id, !m.disabled)}
                          className={`rounded-md border px-2 py-1 text-xs transition-colors ${
                            m.disabled
                              ? "border-emerald-300 text-emerald-700 hover:bg-emerald-50"
                              : "border-slate-200 text-slate-600 hover:border-amber-300 hover:text-amber-700"
                          }`}
                        >
                          {m.disabled ? t.team.reactivate : t.team.deactivate}
                        </button>
                      ) : null}
                      {!isSelf && !lockedOwner && !m.disabled ? (
                        <button
                          type="button"
                          onClick={() => revokeSessions(m.id, m.email)}
                          title={t.team.endSessions}
                          aria-label={t.team.endSessionsOf(m.email)}
                          className="rounded-md border border-slate-200 p-1.5 text-slate-500 hover:border-cyan-400 hover:text-cyan-700"
                        >
                          <LogOutIcon className="size-3.5" aria-hidden />
                        </button>
                      ) : null}
                      {!isSelf && !lockedOwner ? (
                        <button
                          type="button"
                          onClick={() => remove(m.id, m.email)}
                          title={t.team.remove(m.email)}
                          aria-label={t.team.remove(m.email)}
                          className="rounded-md border border-slate-200 p-1.5 text-slate-500 hover:border-rose-300 hover:text-rose-600"
                        >
                          <Trash2Icon className="size-3.5" aria-hidden />
                        </button>
                      ) : null}
                    </div>
                  </div>
                </li>
              );
            })}
          </ul>
        </div>

        {invites.length > 0 ? (
          <div>
            <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500">
              {t.team.pending(invites.length)}
            </h2>
            <ul className="mt-4 space-y-2.5">
              {invites.map((inv) => (
                <li
                  key={inv.id}
                  className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-dashed border-slate-300 bg-white p-3.5"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm text-slate-700">{inv.email}</span>
                    <RoleBadge role={inv.role} />
                    {inv.expired ? (
                      <span className="rounded-full bg-amber-50 px-2 py-0.5 text-[11px] font-semibold text-amber-800">
                        {t.team.inviteExpired}
                      </span>
                    ) : (
                      <span className="text-[11px] text-slate-500">
                        {t.team.inviteExpires(fmtDate(inv.expires_at))}
                      </span>
                    )}
                  </div>
                  {readOnly ? null : (
                    <div className="flex items-center gap-3">
                      <button
                        type="button"
                        onClick={() => void resend(inv.id)}
                        disabled={resending === inv.id}
                        className="font-medium text-[11px] text-cyan-700 hover:text-cyan-900 disabled:opacity-50"
                      >
                        {resending === inv.id ? t.team.resending : t.team.resend}
                      </button>
                      <button
                        type="button"
                        onClick={() => revoke(inv.id)}
                        className="text-[11px] text-slate-500 hover:text-rose-600"
                      >
                        {t.team.revoke}
                      </button>
                    </div>
                  )}
                </li>
              ))}
            </ul>
          </div>
        ) : null}
      </section>
    </div>
  );
}
