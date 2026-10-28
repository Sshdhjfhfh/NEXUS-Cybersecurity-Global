"use client";

import { useCallback, useEffect, useState } from "react";

import { type ClientFindingDetail, portalApi, type ResultsResponse, type Severity } from "./portal";

const SEV_LABEL: Record<Severity, string> = {
  critical: "Crítica",
  high: "Alta",
  medium: "Media",
  low: "Baja",
};
const SEV_TONE: Record<Severity, string> = {
  critical: "border-rose-200 bg-rose-50 text-rose-700",
  high: "border-orange-200 bg-orange-50 text-orange-700",
  medium: "border-amber-200 bg-amber-50 text-amber-700",
  low: "border-sky-200 bg-sky-50 text-sky-700",
};
const STATUS_LABEL: Record<string, string> = {
  confirmed: "Confirmada",
  in_progress: "En análisis",
  resolved: "Resuelta",
  fixed: "Corregida",
  risk_accepted: "Riesgo aceptado",
};

function SevBadge({ sev }: { sev: Severity }) {
  return (
    <span
      className={`rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${SEV_TONE[sev]}`}
    >
      {SEV_LABEL[sev]}
    </span>
  );
}

// ResultsPanel shows the released, curated results for one request. Everything it
// renders comes from the org-scoped, operator-released endpoints.
export function ResultsPanel({ requestId }: { requestId: number }) {
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
      setErr(ex instanceof Error ? ex.message : "No se pudieron cargar los resultados");
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
    return <p className="mt-3 text-xs text-slate-500">Cargando resultados…</p>;
  }
  if (err) {
    return <p className="mt-3 text-xs text-rose-500">{err}</p>;
  }
  if (!data) {
    return null;
  }

  if (data.state === "not_started") {
    return <p className="mt-3 text-xs text-slate-400">La evaluación aún no ha comenzado.</p>;
  }
  if (data.state === "in_progress") {
    return (
      <p className="mt-3 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-500">
        Evaluación en curso. Los resultados aparecerán aquí cuando nuestro equipo los publique.
      </p>
    );
  }

  const s = data.summary ?? { critical: 0, high: 0, medium: 0, low: 0 };
  return (
    <div className="mt-3 space-y-3">
      <div className="flex flex-wrap items-center gap-2 text-[11px]">
        <span className="text-slate-500">Resumen:</span>
        {(["critical", "high", "medium", "low"] as Severity[]).map((sev) =>
          s[sev] > 0 ? (
            <span key={sev} className="inline-flex items-center gap-1 text-slate-600">
              <SevBadge sev={sev} /> {s[sev]}
            </span>
          ) : null,
        )}
        {data.total === 0 ? <span className="font-medium text-emerald-600">Sin hallazgos. 🎉</span> : null}
      </div>

      <ul className="space-y-2">
        {data.findings.map((f) => (
          <li key={f.id} className="overflow-hidden rounded-lg border border-slate-200 bg-slate-50/60">
            <button
              type="button"
              onClick={() => toggle(f.id)}
              className="flex w-full flex-wrap items-center gap-2 px-3 py-2 text-left text-sm hover:bg-slate-100"
            >
              <SevBadge sev={f.severity} />
              <span className="font-medium text-slate-900">{f.name || f.vulnclass}</span>
              <span className="text-[11px] text-slate-500">{STATUS_LABEL[f.status] ?? f.status}</span>
            </button>
            {open === f.id ? (
              <div className="border-t border-slate-200 bg-white px-3 py-2.5 text-xs text-slate-600">
                {f.summary ? <p className="mb-2 text-slate-500">{f.summary}</p> : null}
                {detail[f.id] ? (
                  <pre className="whitespace-pre-wrap font-sans text-slate-700">{detail[f.id].report || "—"}</pre>
                ) : (
                  <p className="text-slate-400">Cargando detalle…</p>
                )}
              </div>
            ) : null}
          </li>
        ))}
      </ul>
    </div>
  );
}
