"use client";

import { useCallback, useEffect, useState } from "react";

import { usePortalI18n } from "./i18n";
import { type ClientFindingDetail, portalApi, type ResultsResponse, type Severity } from "./portal";

const SEV_TONE: Record<Severity, string> = {
  critical: "border-rose-200 bg-rose-50 text-rose-700",
  high: "border-orange-200 bg-orange-50 text-orange-700",
  medium: "border-amber-200 bg-amber-50 text-amber-800",
  low: "border-sky-200 bg-sky-50 text-sky-700",
};
const UNRATED_TONE = "border-slate-200 bg-slate-100 text-slate-600";

function SevBadge({ sev }: { sev: string }) {
  const { t } = usePortalI18n();
  const tone = SEV_TONE[sev as Severity] ?? UNRATED_TONE;
  const label = (t.sev as Record<string, string>)[sev] ?? t.sev.unrated;
  return (
    <span className={`rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${tone}`}>
      {label}
    </span>
  );
}

// ResultsPanel shows the released, curated results for one request. Everything it
// renders comes from the org-scoped, operator-released endpoints.
export function ResultsPanel({ requestId }: { requestId: number }) {
  const { t } = usePortalI18n();
  const [data, setData] = useState<ResultsResponse | null>(null);
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState<number | null>(null);
  const [detail, setDetail] = useState<Record<number, ClientFindingDetail>>({});

  const load = useCallback(async () => {
    setErr("");
    try {
      setData(await portalApi.requestResults(requestId));
    } catch (ex) {
      setErr(ex instanceof Error && ex.message ? ex.message : "error");
    } finally {
      setLoading(false);
    }
  }, [requestId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function toggle(fid: number) {
    if (open === fid) {
      setOpen(null);
      return;
    }
    setOpen(fid);
    if (!detail[fid]) {
      try {
        const { finding } = await portalApi.requestFinding(requestId, fid);
        setDetail((cur) => ({ ...cur, [fid]: finding }));
      } catch {
        /* leave the row collapsed if the detail fails */
      }
    }
  }

  if (loading) {
    return <p className="mt-3 text-xs text-slate-500">{t.results.loading}</p>;
  }
  if (err) {
    return (
      <p role="alert" className="mt-3 text-xs text-rose-600">
        {err === "error" ? t.results.loadError : err}
      </p>
    );
  }
  if (!data) {
    return null;
  }

  if (data.state === "not_started") {
    return <p className="mt-3 text-xs text-slate-500">{t.results.notStarted}</p>;
  }
  if (data.state === "in_progress") {
    return (
      <p className="mt-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-500">
        {t.results.inProgress}
      </p>
    );
  }

  const s = data.summary ?? { critical: 0, high: 0, medium: 0, low: 0 };
  return (
    <div className="mt-3 space-y-3">
      <div className="flex flex-wrap items-center gap-2 text-[11px]">
        <span className="text-slate-500">{t.results.summary}</span>
        {(["critical", "high", "medium", "low"] as Severity[]).map((sev) =>
          s[sev] > 0 ? (
            <span key={sev} className="inline-flex items-center gap-1 text-slate-600">
              <SevBadge sev={sev} /> {s[sev]}
            </span>
          ) : null,
        )}
        {data.total === 0 ? <span className="font-medium text-emerald-700">{t.results.none}</span> : null}
      </div>

      <ul className="space-y-2">
        {data.findings.map((f) => (
          <li key={f.id} className="overflow-hidden rounded-lg border border-slate-200 bg-slate-50/60">
            <button
              type="button"
              aria-expanded={open === f.id}
              onClick={() => toggle(f.id)}
              className="flex w-full flex-wrap items-center gap-2 px-3 py-2 text-left text-sm hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-cyan-400"
            >
              <SevBadge sev={f.severity} />
              <span className="font-medium text-slate-900">{f.name || f.vulnclass}</span>
              <span className="text-[11px] text-slate-500">{t.findingStatus[f.status] ?? f.status}</span>
            </button>
            {open === f.id ? (
              <div className="border-t border-slate-200 bg-white px-3 py-2.5 text-xs text-slate-600">
                {f.summary ? <p className="mb-2 text-slate-500">{f.summary}</p> : null}
                {detail[f.id] ? (
                  <pre className="whitespace-pre-wrap font-sans text-slate-700">{detail[f.id].report || "—"}</pre>
                ) : (
                  <p className="text-slate-500">{t.results.loadingDetail}</p>
                )}
              </div>
            ) : null}
          </li>
        ))}
      </ul>
    </div>
  );
}
