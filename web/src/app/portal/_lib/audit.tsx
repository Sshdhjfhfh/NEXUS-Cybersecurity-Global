"use client";

import { type FormEvent, useCallback, useEffect, useState } from "react";

import { DownloadIcon, SearchIcon } from "lucide-react";

import { usePortalI18n } from "./i18n";
import { type AuditCategory, type AuditEntry, type AuditFilter, downloadAuditCSV, portalApi } from "./portal";

const errText = (ex: unknown, fallback: string) => (ex instanceof Error && ex.message ? ex.message : fallback);

const CATEGORIES: AuditCategory[] = ["", "access", "team", "config", "work"];

// Read-only audit of this organization's own security events. Admin+ and
// auditors only (the server also enforces it). Gives enterprise/government
// clients self-service transparency into who did what in their account, with
// filters and a CSV export for their own records or an external auditor.
export function AuditPanel() {
  const { t, fmtDateTime } = usePortalI18n();
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [err, setErr] = useState("");
  // `draft` is what the form shows; `filter` is what the list was loaded with.
  const [draft, setDraft] = useState<AuditFilter>({ category: "", q: "", from: "", to: "" });
  const [filter, setFilter] = useState<AuditFilter>({});
  const [exporting, setExporting] = useState(false);
  const [exportMsg, setExportMsg] = useState("");

  const load = useCallback(
    async (f: AuditFilter) => {
      setErr("");
      setLoading(true);
      try {
        const res = await portalApi.listAudit(f);
        setEntries(res.entries ?? []);
        setHasMore(res.has_more);
      } catch (ex) {
        setErr(errText(ex, t.audit.loadError));
      } finally {
        setLoading(false);
      }
    },
    [t],
  );

  useEffect(() => {
    void load(filter);
  }, [load, filter]);

  async function more() {
    const last = entries[entries.length - 1];
    if (!last) return;
    setLoadingMore(true);
    try {
      const res = await portalApi.listAudit(filter, last.id);
      setEntries((prev) => [...prev, ...(res.entries ?? [])]);
      setHasMore(res.has_more);
    } catch (ex) {
      setErr(errText(ex, t.audit.loadError));
    } finally {
      setLoadingMore(false);
    }
  }

  function apply(e?: FormEvent) {
    e?.preventDefault();
    setExportMsg("");
    setFilter({ ...draft });
  }

  function clear() {
    const empty: AuditFilter = { category: "", q: "", from: "", to: "" };
    setDraft(empty);
    setExportMsg("");
    setFilter({});
  }

  async function exportCSV() {
    setExporting(true);
    setExportMsg("");
    try {
      const { truncated } = await downloadAuditCSV(filter);
      setExportMsg(truncated ? t.audit.exportTruncated : t.audit.exported);
    } catch (ex) {
      setExportMsg(errText(ex, t.audit.exportError));
    } finally {
      setExporting(false);
    }
  }

  const filtered = Boolean(filter.category || filter.q || filter.from || filter.to);
  const input =
    "rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-800 focus:border-cyan-500 focus:outline-none focus:ring-2 focus:ring-cyan-100";

  return (
    <div className="mt-4">
      <form
        onSubmit={apply}
        className="flex flex-wrap items-end gap-3 rounded-xl border border-slate-200 bg-white p-4 shadow-sm"
        aria-label={t.audit.filters}
      >
        <label className="text-xs font-medium text-slate-600">
          {t.audit.category}
          <select
            value={draft.category ?? ""}
            onChange={(e) => setDraft((d) => ({ ...d, category: e.target.value as AuditCategory }))}
            className={`mt-1 block ${input}`}
          >
            {CATEGORIES.map((c) => (
              <option key={c || "all"} value={c}>
                {t.audit.categories[c || "all"]}
              </option>
            ))}
          </select>
        </label>
        <label className="min-w-48 flex-1 text-xs font-medium text-slate-600">
          {t.audit.search}
          <input
            type="search"
            value={draft.q ?? ""}
            onChange={(e) => setDraft((d) => ({ ...d, q: e.target.value }))}
            placeholder={t.audit.searchPlaceholder}
            maxLength={200}
            className={`mt-1 block w-full ${input}`}
          />
        </label>
        <label className="text-xs font-medium text-slate-600">
          {t.audit.from}
          <input
            type="date"
            value={draft.from ?? ""}
            max={draft.to || undefined}
            onChange={(e) => setDraft((d) => ({ ...d, from: e.target.value }))}
            className={`mt-1 block ${input}`}
          />
        </label>
        <label className="text-xs font-medium text-slate-600">
          {t.audit.to}
          <input
            type="date"
            value={draft.to ?? ""}
            min={draft.from || undefined}
            onChange={(e) => setDraft((d) => ({ ...d, to: e.target.value }))}
            className={`mt-1 block ${input}`}
          />
        </label>
        <button
          type="submit"
          className="inline-flex items-center gap-1.5 rounded-lg bg-cyan-700 px-4 py-2 text-sm font-semibold text-white hover:bg-cyan-800"
        >
          <SearchIcon className="size-4" aria-hidden /> {t.audit.apply}
        </button>
        {filtered ? (
          <button type="button" onClick={clear} className="px-2 py-2 text-sm text-slate-600 hover:underline">
            {t.audit.clear}
          </button>
        ) : null}
      </form>

      <div className="mt-3 flex flex-wrap items-center gap-3">
        <p className="text-sm text-slate-500">
          {t.audit.datesUTC} {loading ? "" : t.audit.shown(entries.length, hasMore)}
        </p>
        <button
          type="button"
          onClick={() => void exportCSV()}
          disabled={exporting || loading}
          className="ml-auto inline-flex items-center gap-2 rounded-lg border border-slate-200 px-3 py-1.5 text-sm text-slate-600 transition-colors hover:border-cyan-400 hover:text-cyan-700 disabled:opacity-60"
        >
          <DownloadIcon className="size-4" aria-hidden /> {exporting ? t.audit.exporting : t.audit.export}
        </button>
      </div>
      {exportMsg ? (
        <p role="status" className="mt-2 text-xs text-slate-600">
          {exportMsg}
        </p>
      ) : null}

      {err ? (
        <p role="alert" className="mt-4 text-sm text-rose-600">
          {err}
        </p>
      ) : null}
      {loading ? <p className="mt-4 text-sm text-slate-500">{t.audit.loading}</p> : null}
      {!loading && !err && entries.length === 0 ? (
        <p className="mt-3 rounded-xl border border-dashed border-slate-300 bg-white p-5 text-sm text-slate-500">
          {filtered ? t.audit.noMatches : t.audit.empty}
        </p>
      ) : null}
      {!loading && entries.length > 0 ? (
        <ul className="mt-3 space-y-2">
          {entries.map((e) => (
            <li
              key={e.id}
              className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-xl border border-slate-200 bg-white p-3.5 text-sm shadow-sm"
            >
              <span className="font-medium text-slate-900">{t.audit.actions[e.action] ?? e.action}</span>
              <span className="text-slate-500">{e.actor}</span>
              {e.detail ? <span className="text-xs text-slate-500">{e.detail}</span> : null}
              <span className="ml-auto flex items-center gap-3 text-xs text-slate-500">
                {e.ip ? <span className="font-mono">{e.ip}</span> : null}
                <span>{fmtDateTime(e.created_at)}</span>
              </span>
            </li>
          ))}
        </ul>
      ) : null}
      {!loading && hasMore ? (
        <button
          type="button"
          onClick={() => void more()}
          disabled={loadingMore}
          className="mt-3 w-full rounded-xl border border-slate-200 bg-white py-2 text-sm text-slate-600 hover:border-cyan-400 hover:text-cyan-700 disabled:opacity-60"
        >
          {loadingMore ? t.audit.loading : t.audit.more}
        </button>
      ) : null}
    </div>
  );
}
