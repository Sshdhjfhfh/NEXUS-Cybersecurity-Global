"use client";

import { useCallback, useEffect, useState } from "react";

import { LockKeyholeIcon } from "lucide-react";

import { usePortalI18n } from "./i18n";
import { type Me, portalApi, type SecurityPolicyResponse } from "./portal";
import { PrimaryButton } from "./ui";

const errText = (ex: unknown, fallback: string) => (ex instanceof Error && ex.message ? ex.message : fallback);
const SESSION_CHOICES = [1, 4, 8, 12, 24, 72, 168];

// Org security policy: required 2FA, allowed networks, maximum session length.
// The owner edits it; admins and auditors see it read-only. The server enforces
// it on every request and refuses a change that would lock the owner out — the
// checks here only explain why the button is disabled.
export function OrgPolicyPanel({ me, ownMfaEnabled }: { me: Me; ownMfaEnabled: boolean }) {
  const { t, fmtDateTime } = usePortalI18n();
  const canEdit = me.user.role === "owner";
  const [data, setData] = useState<SecurityPolicyResponse | null>(null);
  const [loadErr, setLoadErr] = useState("");
  const [requireMfa, setRequireMfa] = useState(false);
  const [allowText, setAllowText] = useState("");
  const [hours, setHours] = useState(168);
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");

  const apply = useCallback((d: SecurityPolicyResponse) => {
    setData(d);
    setRequireMfa(d.policy.require_mfa);
    setAllowText(d.policy.ip_allowlist.join("\n"));
    setHours(d.policy.session_max_hours);
  }, []);

  const load = useCallback(async () => {
    setLoadErr("");
    try {
      apply(await portalApi.getSecurityPolicy());
    } catch (ex) {
      setLoadErr(errText(ex, t.policy.loadError));
    }
  }, [apply, t]);

  useEffect(() => {
    void load();
  }, [load]);

  async function save() {
    setBusy(true);
    setMsg("");
    setErr("");
    try {
      const ip_allowlist = allowText
        .split(/[\n,]/)
        .map((x) => x.trim())
        .filter(Boolean);
      apply(await portalApi.setSecurityPolicy({ require_mfa: requireMfa, ip_allowlist, session_max_hours: hours }));
      setMsg(t.policy.saved);
    } catch (ex) {
      setErr(errText(ex, t.policy.saveError));
    } finally {
      setBusy(false);
    }
  }

  function addMyIp() {
    if (!data) return;
    const lines = allowText.split("\n").map((x) => x.trim());
    if (!lines.includes(data.your_ip)) setAllowText([...lines.filter(Boolean), data.your_ip].join("\n"));
  }

  const box = "mt-6 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm";
  if (!data) {
    return (
      <section className={box}>
        {loadErr ? (
          <p role="alert" className="text-sm text-rose-600">
            {loadErr}{" "}
            <button type="button" onClick={() => void load()} className="font-semibold underline">
              {t.retry}
            </button>
          </p>
        ) : (
          <p className="text-sm text-slate-500">{t.policy.loading}</p>
        )}
      </section>
    );
  }

  const choices = SESSION_CHOICES.includes(hours) ? SESSION_CHOICES : [...SESSION_CHOICES, hours].sort((a, b) => a - b);
  const without = data.members_without_mfa;
  const mfaLocked = !canEdit || (!ownMfaEnabled && !requireMfa);

  return (
    <section className={box} aria-labelledby="policy-title">
      <h2
        id="policy-title"
        className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-cyan-700"
      >
        <LockKeyholeIcon className="size-4" aria-hidden /> {t.policy.title}
      </h2>
      <p className="mt-2 text-sm text-slate-600">{t.policy.intro}</p>
      {!canEdit ? <p className="mt-2 text-xs text-violet-700">{t.policy.readOnly}</p> : null}

      {/* Required 2FA */}
      <div className="mt-5 flex items-start justify-between gap-4">
        <div>
          <p id="pol-mfa" className="text-sm font-medium text-slate-800">
            {t.policy.requireMfa}
          </p>
          <p id="pol-mfa-hint" className="mt-1 text-xs text-slate-500">
            {canEdit && !ownMfaEnabled && !requireMfa ? t.policy.requireMfaNeedsOwn : t.policy.requireMfaHint}
          </p>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={requireMfa}
          aria-labelledby="pol-mfa"
          aria-describedby="pol-mfa-hint"
          disabled={mfaLocked || busy}
          onClick={() => setRequireMfa(!requireMfa)}
          className={`relative mt-0.5 inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400 focus-visible:ring-offset-2 disabled:opacity-60 ${requireMfa ? "bg-cyan-700" : "bg-slate-300"}`}
        >
          <span
            className={`inline-block size-5 rounded-full bg-white shadow transition-transform ${requireMfa ? "translate-x-5" : "translate-x-0.5"}`}
          />
        </button>
      </div>
      <div className="mt-2 text-xs text-slate-600">
        <p className={without.length ? "text-amber-700" : "text-emerald-700"}>{t.policy.withoutMfa(without.length)}</p>
        {without.length ? (
          <ul className="mt-1 list-inside list-disc text-slate-500">
            {without.slice(0, 20).map((e) => (
              <li key={e} className="truncate">
                {e}
              </li>
            ))}
            {without.length > 20 ? <li>…</li> : null}
          </ul>
        ) : null}
      </div>

      {/* Allowed networks */}
      <div className="mt-6">
        <label htmlFor="pol-allow" className="block text-sm font-medium text-slate-800">
          {t.policy.allowlist}
        </label>
        <p id="pol-allow-hint" className="mt-1 text-xs text-slate-500">
          {t.policy.allowlistHint}
        </p>
        <textarea
          id="pol-allow"
          aria-describedby="pol-allow-hint"
          rows={4}
          readOnly={!canEdit}
          value={allowText}
          onChange={(e) => setAllowText(e.target.value)}
          placeholder="203.0.113.0/24"
          spellCheck={false}
          className="mt-2 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 font-mono text-sm text-slate-800 outline-none focus:border-cyan-500 focus:ring-2 focus:ring-cyan-200 read-only:bg-slate-50"
        />
        <div className="mt-1 flex flex-wrap items-center gap-3 text-xs text-slate-500">
          <span>{t.policy.yourIp(data.your_ip)}</span>
          {canEdit ? (
            <button type="button" onClick={addMyIp} className="font-semibold text-cyan-700 hover:underline">
              {t.policy.addMyIp}
            </button>
          ) : null}
        </div>
      </div>

      {/* Session length */}
      <div className="mt-6">
        <label htmlFor="pol-hours" className="block text-sm font-medium text-slate-800">
          {t.policy.sessionMax}
        </label>
        <p id="pol-hours-hint" className="mt-1 text-xs text-slate-500">
          {t.policy.sessionMaxHint}
        </p>
        <select
          id="pol-hours"
          aria-describedby="pol-hours-hint"
          disabled={!canEdit}
          value={hours}
          onChange={(e) => setHours(Number(e.target.value))}
          className="mt-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-800 outline-none focus:border-cyan-500 focus:ring-2 focus:ring-cyan-200 disabled:bg-slate-50"
        >
          {choices
            .filter((h) => h <= data.max_session_hours)
            .map((h) => (
              <option key={h} value={h}>
                {t.policy.hours(h)}
              </option>
            ))}
        </select>
      </div>

      {data.policy.updated_at && data.policy.updated_by ? (
        <p className="mt-5 text-xs text-slate-500">
          {t.policy.updated(
            data.policy.updated_by === "operator" ? "NEXUS" : data.policy.updated_by,
            fmtDateTime(data.policy.updated_at),
          )}
        </p>
      ) : null}

      {canEdit ? (
        <div className="mt-5 space-y-3">
          <p className="text-xs text-slate-500">{t.policy.lockoutNote}</p>
          {err ? (
            <p role="alert" className="text-sm text-rose-600">
              {err}
            </p>
          ) : null}
          {msg ? (
            <p role="status" className="text-sm text-emerald-700">
              {msg}
            </p>
          ) : null}
          <PrimaryButton type="button" onClick={() => void save()} disabled={busy}>
            {busy ? t.policy.saving : t.policy.save}
          </PrimaryButton>
        </div>
      ) : null}
    </section>
  );
}
