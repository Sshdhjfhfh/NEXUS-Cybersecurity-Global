"use client";

import { useCallback, useEffect, useState } from "react";

import { LaptopIcon, SmartphoneIcon } from "lucide-react";

import { usePortalI18n } from "./i18n";
import { type PortalSession, portalApi } from "./portal";

const errText = (ex: unknown, fallback: string) => (ex instanceof Error && ex.message ? ex.message : fallback);

const isMobile = (s: PortalSession) => s.os === "iOS" || s.os === "Android";

// Where the current user's account is signed in. Every session is checked by
// the server on each request, so ending one here signs that device out at once.
export function SessionsPanel() {
  const { t, fmtDateTime, fmtRelative } = usePortalI18n();
  const [sessions, setSessions] = useState<PortalSession[] | null>(null);
  const [loadErr, setLoadErr] = useState("");
  const [busy, setBusy] = useState("");
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    setLoadErr("");
    try {
      setSessions((await portalApi.listSessions()).sessions);
    } catch (ex) {
      setLoadErr(errText(ex, t.sessions.loadError));
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  async function end(id: string) {
    setBusy(id);
    setMsg("");
    setErr("");
    try {
      await portalApi.revokeSession(id);
      setMsg(t.sessions.ended);
      await load();
    } catch (ex) {
      setErr(errText(ex, t.sessions.endError));
    } finally {
      setBusy("");
    }
  }

  async function endOthers() {
    setBusy("others");
    setMsg("");
    setErr("");
    try {
      const { revoked } = await portalApi.revokeOtherSessions();
      setMsg(t.sessions.othersEnded(revoked));
      await load();
    } catch (ex) {
      setErr(errText(ex, t.sessions.endError));
    } finally {
      setBusy("");
    }
  }

  const others = sessions?.filter((s) => !s.current).length ?? 0;

  return (
    <section
      className="mt-6 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"
      aria-labelledby="sessions-title"
    >
      <h2 id="sessions-title" className="text-sm font-semibold uppercase tracking-wide text-cyan-700">
        {t.sessions.title}
      </h2>
      <p className="mt-2 text-sm text-slate-600">{t.sessions.intro}</p>

      {sessions === null && loadErr ? (
        <p role="alert" className="mt-4 text-sm text-rose-600">
          {loadErr}{" "}
          <button type="button" onClick={() => void load()} className="font-semibold underline">
            {t.retry}
          </button>
        </p>
      ) : null}
      {sessions === null && !loadErr ? <p className="mt-4 text-sm text-slate-500">{t.sessions.loading}</p> : null}
      {sessions === null ? null : (
        <ul className="mt-4 divide-y divide-slate-100 rounded-xl border border-slate-200">
          {sessions.map((s) => {
            const Icon = isMobile(s) ? SmartphoneIcon : LaptopIcon;
            return (
              <li key={s.id} className="flex flex-wrap items-center gap-3 px-4 py-3">
                <Icon className="size-5 shrink-0 text-slate-400" aria-hidden />
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-slate-800">
                    {t.sessions.device(s.browser, s.os)}
                    {s.current ? (
                      <span className="ml-2 rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-semibold text-emerald-700">
                        {t.sessions.thisDevice}
                      </span>
                    ) : null}
                    {s.amr === "sso" ? (
                      <span className="ml-2 rounded-full bg-violet-50 px-2 py-0.5 text-xs font-semibold text-violet-700">
                        SSO
                      </span>
                    ) : null}
                  </p>
                  <p className="mt-0.5 text-xs text-slate-500">
                    {s.ip ? `${s.ip} · ` : ""}
                    <span title={fmtDateTime(s.last_seen_at)}>
                      {t.sessions.lastActive(fmtRelative(s.last_seen_at))}
                    </span>
                    {" · "}
                    {t.sessions.signedIn(fmtDateTime(s.created_at))}
                  </p>
                </div>
                {s.current ? null : (
                  <button
                    type="button"
                    onClick={() => void end(s.id)}
                    disabled={busy !== ""}
                    className="rounded-lg border border-slate-200 px-3 py-1.5 text-sm text-slate-600 transition-colors hover:border-rose-300 hover:text-rose-600 disabled:opacity-60"
                  >
                    {busy === s.id ? t.sessions.ending : t.sessions.end}
                  </button>
                )}
              </li>
            );
          })}
        </ul>
      )}

      {err ? (
        <p role="alert" className="mt-3 text-sm text-rose-600">
          {err}
        </p>
      ) : null}
      {msg ? (
        <p role="status" className="mt-3 text-sm text-emerald-700">
          {msg}
        </p>
      ) : null}
      {others > 0 ? (
        <button
          type="button"
          onClick={() => void endOthers()}
          disabled={busy !== ""}
          className="mt-4 rounded-lg border border-rose-300 px-4 py-2 text-sm font-semibold text-rose-600 transition-colors hover:bg-rose-50 disabled:opacity-60"
        >
          {busy === "others" ? t.sessions.ending : t.sessions.endOthers}
        </button>
      ) : null}
    </section>
  );
}
