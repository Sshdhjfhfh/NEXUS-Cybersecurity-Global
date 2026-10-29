"use client";

import { useCallback, useEffect, useState } from "react";

import { usePortalI18n } from "./i18n";
import { type AuditEntry, portalApi } from "./portal";

// Read-only audit of this organization's own security events. Admin+ only (the
// server also enforces it). Gives enterprise/government clients self-service
// transparency into who did what in their account.
export function AuditPanel() {
  const { t, fmtDateTime } = usePortalI18n();
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    setErr("");
    try {
      const { entries } = await portalApi.listAudit();
      setEntries(entries ?? []);
    } catch (ex) {
      setErr(ex instanceof Error && ex.message ? ex.message : t.audit.loadError);
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) {
    return <p className="mt-4 text-sm text-slate-500">{t.audit.loading}</p>;
  }
  if (err) {
    return (
      <p role="alert" className="mt-4 text-sm text-rose-600">
        {err}
      </p>
    );
  }

  return (
    <div className="mt-4">
      <p className="mb-3 text-sm text-slate-500">{t.audit.intro(entries.length)}</p>
      {entries.length === 0 ? (
        <p className="rounded-xl border border-dashed border-slate-300 bg-white p-5 text-sm text-slate-500">
          {t.audit.empty}
        </p>
      ) : (
        <ul className="space-y-2">
          {entries.map((e) => (
            <li
              key={e.id}
              className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-xl border border-slate-200 bg-white p-3.5 text-sm shadow-sm"
            >
              <span className="font-medium text-slate-900">{t.audit.actions[e.action] ?? e.action}</span>
              <span className="text-slate-500">{e.actor}</span>
              {e.detail ? <span className="text-xs text-slate-500">{e.detail}</span> : null}
              <span className="ml-auto text-xs text-slate-500">{fmtDateTime(e.created_at)}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
