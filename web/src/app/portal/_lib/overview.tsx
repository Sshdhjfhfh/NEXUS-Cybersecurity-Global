"use client";

import { type ReactNode, useCallback, useEffect, useState } from "react";

import {
  ActivityIcon,
  ArrowRightIcon,
  ClipboardListIcon,
  InfoIcon,
  RefreshCwIcon,
  ShieldAlertIcon,
  ShieldCheckIcon,
  UsersIcon,
  WrenchIcon,
} from "lucide-react";

import { type PortalDict, usePortalI18n } from "./i18n";
import { type Me, type OverviewResponse, portalApi, type SeverityBucket, type ViewProfile } from "./portal";

// Client dashboard overview. Two perspectives on the SAME server-computed data:
// "executive" (exposure level, priorities, progress — business language) and
// "technical" (breakdown tables, trend, top open findings). Every figure comes
// from /api/portal/overview, which only counts operator-released, client-visible
// findings of this org; when there is nothing yet, an empty state explains why
// instead of showing a made-up score.

const SEVERITIES = ["critical", "high", "medium", "low", "unrated"] as const;
type Sev = (typeof SEVERITIES)[number];

const SEV_TONE: Record<Sev, string> = {
  critical: "border-rose-200 bg-rose-50 text-rose-700",
  high: "border-orange-200 bg-orange-50 text-orange-700",
  medium: "border-amber-200 bg-amber-50 text-amber-800",
  low: "border-sky-200 bg-sky-50 text-sky-700",
  unrated: "border-slate-200 bg-slate-100 text-slate-600",
};

const EXPOSURE_TONE: Record<Sev | "none", { card: string; icon: string }> = {
  critical: { card: "border-rose-200 bg-gradient-to-br from-rose-50 to-white", icon: "bg-rose-100 text-rose-700" },
  high: { card: "border-orange-200 bg-gradient-to-br from-orange-50 to-white", icon: "bg-orange-100 text-orange-700" },
  medium: { card: "border-amber-200 bg-gradient-to-br from-amber-50 to-white", icon: "bg-amber-100 text-amber-800" },
  low: { card: "border-sky-200 bg-gradient-to-br from-sky-50 to-white", icon: "bg-sky-100 text-sky-700" },
  unrated: { card: "border-slate-200 bg-white", icon: "bg-slate-100 text-slate-600" },
  none: {
    card: "border-emerald-200 bg-gradient-to-br from-emerald-50 to-white",
    icon: "bg-emerald-100 text-emerald-700",
  },
};

const card = "rounded-2xl border border-slate-200 bg-white p-5 shadow-sm";
const sectionTitle = "text-sm font-semibold text-slate-900";
const focusRing = "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400";

function sevLabel(t: PortalDict, sev: string): string {
  return (t.sev as Record<string, string>)[sev] ?? t.sev.unrated;
}

export function SevBadge({ sev }: { sev: string }) {
  const { t } = usePortalI18n();
  const key = (SEVERITIES as readonly string[]).includes(sev) ? (sev as Sev) : "unrated";
  return (
    <span
      className={`inline-flex shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${SEV_TONE[key]}`}
    >
      {sevLabel(t, key)}
    </span>
  );
}

function daysSince(iso: string): number {
  return Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000));
}

// Derived figures shared by both views. Kept in one place so the executive and
// technical views can never disagree about a number.
function derive(data: OverviewResponse) {
  const bs = data.overview.by_severity;
  const zero: SeverityBucket = { open: 0, in_progress: 0, remediated: 0, risk_accepted: 0 };
  const sevs = SEVERITIES.filter((s) => s !== "unrated" || bs.unrated);
  const bucket = (s: Sev) => bs[s] ?? zero;
  const openOf = (s: Sev) => bucket(s).open + bucket(s).in_progress;
  let open = 0;
  let inProgress = 0;
  let remediated = 0;
  let riskAccepted = 0;
  for (const s of sevs) {
    open += openOf(s);
    inProgress += bucket(s).in_progress;
    remediated += bucket(s).remediated;
    riskAccepted += bucket(s).risk_accepted;
  }
  const exposure: Sev | "none" = sevs.find((s) => openOf(s) > 0) ?? "none";
  const denom = open + remediated;
  const pct = denom > 0 ? Math.round((remediated / denom) * 100) : null;
  const r = data.overview.requests;
  return {
    sevs,
    bucket,
    openOf,
    open,
    inProgress,
    remediated,
    riskAccepted,
    exposure,
    denom,
    pct,
    hasResults: data.overview.released_requests > 0,
    activeEvals: (r.new ?? 0) + (r.scheduled ?? 0) + (r.in_progress ?? 0),
    doneEvals: r.done ?? 0,
  };
}

type Props = {
  me: Me;
  view: Exclude<ViewProfile, "">;
  onViewChange: (v: Exclude<ViewProfile, "">) => void;
  onOpenRequest: (requestId: number) => void;
  onOpenFinding: (findingId: number) => void;
  onGoTab: (tab: "remediation" | "requests" | "team" | "audit" | "security") => void;
};

export function OverviewPanel({ me, view, onViewChange, onOpenRequest, onOpenFinding, onGoTab }: Props) {
  const { t, fmtRelative } = usePortalI18n();
  const [data, setData] = useState<OverviewResponse | null>(null);
  const [err, setErr] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    setRefreshing(true);
    setErr(false);
    try {
      setData(await portalApi.overview());
    } catch {
      setErr(true);
    } finally {
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const canAdmin = me.user.role === "owner" || me.user.role === "admin";

  return (
    <div className="mt-6 space-y-5">
      {/* Toolbar: perspective switch + freshness */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <fieldset className="inline-flex rounded-xl border border-slate-200 bg-white p-1 shadow-sm">
          <legend className="sr-only">{t.view.label}</legend>
          {(["executive", "technical"] as const).map((v) => (
            <button
              key={v}
              type="button"
              aria-pressed={view === v}
              title={v === "executive" ? t.view.executiveHint : t.view.technicalHint}
              onClick={() => onViewChange(v)}
              className={`rounded-lg px-3.5 py-1.5 text-sm font-medium transition ${focusRing} ${
                view === v ? "bg-slate-900 text-white shadow-sm" : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              {t.view[v]}
            </button>
          ))}
        </fieldset>
        <div className="flex items-center gap-2 text-xs text-slate-500">
          {data ? <span aria-live="polite">{t.ov.updated(fmtRelative(data.generated_at))}</span> : null}
          <button
            type="button"
            onClick={() => void load()}
            disabled={refreshing}
            className={`inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 font-medium text-slate-600 transition hover:bg-slate-50 disabled:opacity-60 ${focusRing}`}
          >
            <RefreshCwIcon className={`size-3.5 ${refreshing ? "animate-spin motion-reduce:animate-none" : ""}`} />
            {t.refresh}
          </button>
        </div>
      </div>

      {err && !data ? (
        <div role="alert" className="rounded-2xl border border-rose-200 bg-rose-50 p-5 text-sm text-rose-700">
          {t.ov.loadError}{" "}
          <button type="button" onClick={() => void load()} className={`font-semibold underline ${focusRing}`}>
            {t.retry}
          </button>
        </div>
      ) : !data ? (
        <OverviewSkeleton />
      ) : view === "executive" ? (
        <ExecutiveView
          data={data}
          canAdmin={canAdmin}
          onOpenRequest={onOpenRequest}
          onOpenFinding={onOpenFinding}
          onGoTab={onGoTab}
        />
      ) : (
        <TechnicalView
          data={data}
          canAdmin={canAdmin}
          onOpenRequest={onOpenRequest}
          onOpenFinding={onOpenFinding}
          onGoTab={onGoTab}
        />
      )}
    </div>
  );
}

function OverviewSkeleton() {
  return (
    <div aria-hidden className="space-y-5 motion-safe:animate-pulse">
      <div className="h-36 rounded-2xl bg-slate-200/60" />
      <div className="grid gap-4 sm:grid-cols-3">
        <div className="h-28 rounded-2xl bg-slate-200/60" />
        <div className="h-28 rounded-2xl bg-slate-200/60" />
        <div className="h-28 rounded-2xl bg-slate-200/60" />
      </div>
      <div className="h-48 rounded-2xl bg-slate-200/60" />
    </div>
  );
}

type ViewProps = {
  data: OverviewResponse;
  canAdmin: boolean;
  onOpenRequest: (id: number) => void;
  onOpenFinding: (id: number) => void;
  onGoTab: Props["onGoTab"];
};

function OverdueTag() {
  const { t } = usePortalI18n();
  return (
    <span className="rounded-full border border-rose-200 bg-rose-50 px-1.5 py-px text-[10px] font-semibold uppercase text-rose-700">
      {t.rem.overdue}
    </span>
  );
}

function EmptyResults({ onGoTab }: { onGoTab: Props["onGoTab"] }) {
  const { t } = usePortalI18n();
  return (
    <section className={`${card} flex flex-col items-center px-6 py-10 text-center`}>
      <span className="grid size-12 place-items-center rounded-2xl bg-cyan-50 text-cyan-700">
        <ClipboardListIcon className="size-6" />
      </span>
      <h2 className="mt-4 text-base font-semibold text-slate-900">{t.ov.emptyTitle}</h2>
      <p className="mt-1.5 max-w-md text-sm text-slate-500">{t.ov.emptyBody}</p>
      <button
        type="button"
        onClick={() => onGoTab("requests")}
        className={`mt-5 inline-flex items-center gap-1.5 rounded-xl bg-gradient-to-r from-cyan-700 to-sky-700 px-4 py-2 text-sm font-semibold text-white shadow-sm transition hover:brightness-105 ${focusRing}`}
      >
        {t.ov.emptyCta} <ArrowRightIcon className="size-4" />
      </button>
    </section>
  );
}

function ExecutiveView({ data, canAdmin, onOpenFinding, onGoTab }: ViewProps) {
  const { t, lang, fmtNumber } = usePortalI18n();
  const d = derive(data);
  const tone = EXPOSURE_TONE[d.exposure];
  const parts = d.sevs
    .filter((s) => d.openOf(s) > 0)
    .map((s) => t.ov.countBySev(d.openOf(s), sevLabel(t, s).toLowerCase(), s === "unrated"));
  const listFmt = new Intl.ListFormat(lang, { type: "conjunction" });

  return (
    <div className="space-y-5">
      {d.hasResults ? (
        <section className={`rounded-2xl border p-6 shadow-sm ${tone.card}`} aria-labelledby="exposure-title">
          <div className="flex flex-wrap items-start gap-4">
            <span className={`grid size-12 shrink-0 place-items-center rounded-2xl ${tone.icon}`}>
              {d.exposure === "none" ? (
                <ShieldCheckIcon className="size-6" aria-hidden />
              ) : (
                <ShieldAlertIcon className="size-6" aria-hidden />
              )}
            </span>
            <div className="min-w-0 flex-1">
              <p id="exposure-title" className="text-xs font-semibold uppercase tracking-wide text-slate-500">
                {t.ov.exposureTitle}
              </p>
              <p className="mt-0.5 text-2xl font-bold text-slate-900">
                {d.exposure === "unrated" ? t.sev.unrated : t.ov.exposure[d.exposure]}
              </p>
              <p className="mt-1 text-sm text-slate-600">
                {parts.length > 0 ? t.ov.exposureLine(listFmt.format(parts)) : t.ov.exposureNone}
              </p>
            </div>
          </div>
          <details className="group mt-4 text-sm">
            <summary
              className={`inline-flex cursor-pointer list-none items-center gap-1.5 rounded text-xs font-medium text-slate-500 hover:text-slate-700 ${focusRing}`}
            >
              <InfoIcon className="size-3.5" aria-hidden /> {t.ov.howCalculated}
            </summary>
            <p className="mt-2 max-w-2xl text-xs leading-relaxed text-slate-500">{t.ov.methodology}</p>
          </details>
        </section>
      ) : (
        <EmptyResults onGoTab={onGoTab} />
      )}

      <div className="grid gap-4 sm:grid-cols-3">
        {d.hasResults ? (
          <>
            <Kpi
              icon={<ShieldAlertIcon className="size-4" />}
              label={t.ov.openFindings}
              value={fmtNumber(d.open)}
              hint={t.ov.openFindingsHint(d.inProgress, data.overview.overdue ?? 0)}
              action={{ label: t.tabs.remediation, onClick: () => onGoTab("remediation") }}
            />
            <div className={card}>
              <p className="flex items-center gap-2 text-xs font-medium text-slate-500">
                <WrenchIcon className="size-4 text-cyan-700" aria-hidden /> {t.ov.remediation}
              </p>
              <p className="mt-2 text-2xl font-bold text-slate-900">{d.pct === null ? "—" : `${d.pct}%`}</p>
              <div
                role="progressbar"
                aria-label={t.ov.remediation}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={d.pct ?? 0}
                className="mt-2 h-2 overflow-hidden rounded-full bg-slate-100"
              >
                <div
                  className="h-full rounded-full bg-gradient-to-r from-emerald-400 to-emerald-500"
                  style={{ width: `${d.pct ?? 0}%` }}
                />
              </div>
              <p className="mt-1.5 text-xs text-slate-500" title={t.ov.remediationMethod}>
                {t.ov.remediationOf(d.remediated, d.denom)}
              </p>
            </div>
          </>
        ) : null}
        <Kpi
          icon={<ClipboardListIcon className="size-4" />}
          label={t.ov.evaluations}
          value={fmtNumber(data.overview.requests_total)}
          hint={t.ov.evaluationsHint(d.activeEvals, d.doneEvals)}
          action={{ label: t.tabs.requests, onClick: () => onGoTab("requests") }}
        />
      </div>

      {d.hasResults ? (
        <section className={card} aria-labelledby="prio-title">
          <div className="flex items-baseline justify-between gap-3">
            <h2 id="prio-title" className={sectionTitle}>
              {t.ov.priorities}
            </h2>
            <span className="text-xs text-slate-500">{t.ov.prioritiesHint}</span>
          </div>
          {data.overview.top_open.length === 0 ? (
            <p className="mt-4 rounded-xl bg-emerald-50 px-4 py-3 text-sm text-emerald-700">{t.ov.noPriorities}</p>
          ) : (
            <ol className="mt-3 divide-y divide-slate-100">
              {data.overview.top_open.slice(0, 3).map((f, i) => (
                <li key={f.id} className="flex flex-wrap items-center gap-3 py-3">
                  <span className="grid size-7 shrink-0 place-items-center rounded-full bg-slate-100 text-xs font-bold text-slate-600">
                    {i + 1}
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <SevBadge sev={f.severity} />
                      <span className="truncate font-medium text-slate-900">{f.name || f.vulnclass}</span>
                      {f.overdue ? <OverdueTag /> : null}
                    </div>
                    <p className="mt-0.5 text-xs text-slate-500">
                      {t.ov.inEvaluation(f.request_title)} · {t.ov.openSince(daysSince(f.created_at))} ·{" "}
                      {t.rem.state[f.status] ?? f.status}
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => onOpenFinding(f.id)}
                    className={`inline-flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-semibold text-cyan-700 hover:bg-cyan-50 ${focusRing}`}
                  >
                    {t.ov.viewDetail} <ArrowRightIcon className="size-3.5" aria-hidden />
                  </button>
                </li>
              ))}
            </ol>
          )}
        </section>
      ) : null}

      {canAdmin && data.account_security ? (
        <AccountSecurityCard acct={data.account_security} onGoTab={onGoTab} />
      ) : null}
    </div>
  );
}

function Kpi({
  icon,
  label,
  value,
  hint,
  action,
}: {
  icon: ReactNode;
  label: string;
  value: string;
  hint: string;
  action?: { label: string; onClick: () => void };
}) {
  return (
    <div className={card}>
      <p className="flex items-center gap-2 text-xs font-medium text-slate-500">
        <span className="text-cyan-700" aria-hidden>
          {icon}
        </span>
        {label}
      </p>
      <p className="mt-2 text-2xl font-bold text-slate-900">{value}</p>
      <div className="mt-1.5 flex items-center justify-between gap-2">
        <p className="text-xs text-slate-500">{hint}</p>
        {action ? (
          <button
            type="button"
            onClick={action.onClick}
            className={`inline-flex items-center gap-1 rounded text-xs font-semibold text-cyan-700 hover:text-cyan-800 ${focusRing}`}
          >
            {action.label} <ArrowRightIcon className="size-3" aria-hidden />
          </button>
        ) : null}
      </div>
    </div>
  );
}

function AccountSecurityCard({
  acct,
  onGoTab,
}: {
  acct: NonNullable<OverviewResponse["account_security"]>;
  onGoTab: Props["onGoTab"];
}) {
  const { t } = usePortalI18n();
  const mfaComplete = acct.mfa_enabled >= acct.active_members;
  const issues: string[] = [];
  if (acct.unverified > 0) issues.push(t.ov.unverified(acct.unverified));
  if (acct.pending_invites > 0) issues.push(t.ov.pendingInvites(acct.pending_invites));
  const allGood = mfaComplete && issues.length === 0;
  return (
    <section className={card} aria-labelledby="acct-title">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id="acct-title" className={`flex items-center gap-2 ${sectionTitle}`}>
          <UsersIcon className="size-4 text-cyan-700" aria-hidden /> {t.ov.accountSecurity}
        </h2>
        <button
          type="button"
          onClick={() => onGoTab("team")}
          className={`inline-flex items-center gap-1 rounded text-xs font-semibold text-cyan-700 hover:text-cyan-800 ${focusRing}`}
        >
          {t.ov.manageTeam} <ArrowRightIcon className="size-3" aria-hidden />
        </button>
      </div>
      <p
        className={`mt-2 inline-flex items-center gap-1.5 text-sm ${mfaComplete ? "text-emerald-700" : "text-amber-800"}`}
      >
        {mfaComplete ? (
          <ShieldCheckIcon className="size-4" aria-hidden />
        ) : (
          <ShieldAlertIcon className="size-4" aria-hidden />
        )}
        {t.ov.mfaCoverage(acct.mfa_enabled, acct.active_members)}
      </p>
      {allGood ? (
        <p className="mt-1 text-sm font-medium text-emerald-700">{t.ov.allGood}</p>
      ) : issues.length > 0 ? (
        <ul className="mt-2 flex flex-wrap gap-2">
          {issues.map((i) => (
            <li key={i} className="rounded-full border border-amber-200 bg-amber-50 px-2.5 py-1 text-xs text-amber-800">
              {i}
            </li>
          ))}
        </ul>
      ) : null}
    </section>
  );
}

function TechnicalView({ data, canAdmin, onOpenRequest, onOpenFinding, onGoTab }: ViewProps) {
  const { t, fmtNumber, fmtMonth, fmtRelative } = usePortalI18n();
  const d = derive(data);
  const maxMonth = Math.max(1, ...data.overview.monthly.map((m) => m.count));
  const th = "px-3 py-2 text-left text-[11px] font-semibold uppercase tracking-wide text-slate-500";
  const td = "px-3 py-2.5 text-sm text-slate-700";

  return (
    <div className="space-y-5">
      {!d.hasResults ? <EmptyResults onGoTab={onGoTab} /> : null}

      {d.hasResults ? (
        <section className={card} aria-labelledby="sev-title">
          <h2 id="sev-title" className={sectionTitle}>
            {t.ov.bySeverity}
          </h2>
          <div className="mt-3 overflow-x-auto">
            <table className="w-full min-w-[520px] border-collapse">
              <thead className="border-b border-slate-200">
                <tr>
                  <th scope="col" className={th}>
                    {t.ov.col.severity}
                  </th>
                  <th scope="col" className={`${th} text-right`}>
                    {t.ov.col.open}
                  </th>
                  <th scope="col" className={`${th} text-right`}>
                    {t.ov.col.inProgress}
                  </th>
                  <th scope="col" className={`${th} text-right`}>
                    {t.ov.col.remediated}
                  </th>
                  <th scope="col" className={`${th} text-right`}>
                    {t.ov.col.riskAccepted}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {d.sevs.map((s) => {
                  const b = d.bucket(s);
                  return (
                    <tr key={s}>
                      <th scope="row" className={`${td} text-left font-normal`}>
                        <SevBadge sev={s} />
                      </th>
                      <td className={`${td} text-right font-semibold tabular-nums`}>{fmtNumber(b.open)}</td>
                      <td className={`${td} text-right tabular-nums`}>{fmtNumber(b.in_progress)}</td>
                      <td className={`${td} text-right tabular-nums`}>{fmtNumber(b.remediated)}</td>
                      <td className={`${td} text-right tabular-nums`}>{fmtNumber(b.risk_accepted)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}

      <div className="grid gap-5 lg:grid-cols-[1.4fr_1fr]">
        <section className={card} aria-labelledby="trend-title">
          <h2 id="trend-title" className={sectionTitle}>
            {t.ov.trend}
          </h2>
          <p className="text-xs text-slate-500">{t.ov.trendHint}</p>
          {/* Visual bars (decorative) + an equivalent data table for screen readers. */}
          <div aria-hidden className="mt-4 flex h-36 items-end gap-3">
            {data.overview.monthly.map((m) => (
              <div key={m.month} className="flex h-full flex-1 flex-col items-center justify-end gap-1.5">
                <span className="text-[11px] font-semibold tabular-nums text-slate-600">{m.count}</span>
                <div
                  className="w-full max-w-10 rounded-t-md bg-gradient-to-t from-cyan-500 to-sky-400"
                  style={{ height: `${m.count === 0 ? 2 : Math.max(6, (m.count / maxMonth) * 100)}%` }}
                />
                <span className="text-[10px] text-slate-500">{fmtMonth(m.month)}</span>
              </div>
            ))}
          </div>
          <table className="sr-only">
            <caption>{t.ov.trend}</caption>
            <thead>
              <tr>
                <th scope="col">{t.ov.col.month}</th>
                <th scope="col">{t.ov.col.count}</th>
              </tr>
            </thead>
            <tbody>
              {data.overview.monthly.map((m) => (
                <tr key={m.month}>
                  <td>{fmtMonth(m.month)}</td>
                  <td>{m.count}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>

        <section className={card} aria-labelledby="evals-title">
          <h2 id="evals-title" className={sectionTitle}>
            {t.ov.evaluationsByStatus}
          </h2>
          <ul className="mt-3 space-y-2">
            {(["new", "scheduled", "in_progress", "done"] as const).map((st) => (
              <li key={st} className="flex items-center justify-between text-sm">
                <span className="text-slate-600">{t.requestStatus[st]}</span>
                <span className="font-semibold tabular-nums text-slate-900">
                  {fmtNumber(data.overview.requests[st] ?? 0)}
                </span>
              </li>
            ))}
          </ul>
          <button
            type="button"
            onClick={() => onGoTab("requests")}
            className={`mt-4 inline-flex items-center gap-1 rounded text-xs font-semibold text-cyan-700 hover:text-cyan-800 ${focusRing}`}
          >
            {t.tabs.requests} <ArrowRightIcon className="size-3" aria-hidden />
          </button>
        </section>
      </div>

      {d.hasResults ? (
        <section className={card} aria-labelledby="top-title">
          <h2 id="top-title" className={sectionTitle}>
            {t.ov.topOpen}
          </h2>
          {data.overview.top_open.length === 0 ? (
            <p className="mt-3 rounded-xl bg-emerald-50 px-4 py-3 text-sm text-emerald-700">{t.ov.noPriorities}</p>
          ) : (
            <div className="mt-3 overflow-x-auto">
              <table className="w-full min-w-[640px] border-collapse">
                <thead className="border-b border-slate-200">
                  <tr>
                    <th scope="col" className={th}>
                      {t.ov.col.severity}
                    </th>
                    <th scope="col" className={th}>
                      {t.ov.col.finding}
                    </th>
                    <th scope="col" className={th}>
                      {t.ov.col.status}
                    </th>
                    <th scope="col" className={th}>
                      {t.ov.col.evaluation}
                    </th>
                    <th scope="col" className={`${th} text-right`}>
                      {t.ov.col.age}
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {data.overview.top_open.map((f) => (
                    <tr key={f.id} className="hover:bg-slate-50">
                      <td className={td}>
                        <SevBadge sev={f.severity} />
                      </td>
                      <td className={td}>
                        <button
                          type="button"
                          onClick={() => onOpenFinding(f.id)}
                          className={`rounded text-left font-medium text-slate-900 hover:text-cyan-800 hover:underline ${focusRing}`}
                        >
                          {f.name || f.vulnclass}
                        </button>
                        {f.name && f.vulnclass ? (
                          <span className="block text-xs text-slate-500">{f.vulnclass}</span>
                        ) : null}
                      </td>
                      <td className={td}>
                        <span className="inline-flex flex-wrap items-center gap-1.5">
                          {t.rem.state[f.status] ?? f.status}
                          {f.overdue ? <OverdueTag /> : null}
                        </span>
                      </td>
                      <td className={td}>
                        <button
                          type="button"
                          onClick={() => onOpenRequest(f.request_id)}
                          className={`rounded text-left font-medium text-cyan-700 hover:underline ${focusRing}`}
                        >
                          {f.request_title}
                        </button>
                      </td>
                      <td className={`${td} text-right tabular-nums`}>{t.ov.days(daysSince(f.created_at))}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      ) : null}

      {canAdmin && data.account_security ? (
        <AccountSecurityCard acct={data.account_security} onGoTab={onGoTab} />
      ) : null}

      {canAdmin && data.activity ? (
        <section className={card} aria-labelledby="act-title">
          <div className="flex items-center justify-between gap-3">
            <h2 id="act-title" className={`flex items-center gap-2 ${sectionTitle}`}>
              <ActivityIcon className="size-4 text-cyan-700" aria-hidden /> {t.ov.activity}
            </h2>
            <button
              type="button"
              onClick={() => onGoTab("audit")}
              className={`inline-flex items-center gap-1 rounded text-xs font-semibold text-cyan-700 hover:text-cyan-800 ${focusRing}`}
            >
              {t.ov.seeAudit} <ArrowRightIcon className="size-3" aria-hidden />
            </button>
          </div>
          {data.activity.length === 0 ? (
            <p className="mt-3 text-sm text-slate-500">{t.ov.noActivity}</p>
          ) : (
            <ul className="mt-3 divide-y divide-slate-100">
              {data.activity.map((e) => (
                <li key={e.id} className="flex flex-wrap items-center gap-x-3 gap-y-0.5 py-2 text-sm">
                  <span className="font-medium text-slate-800">{t.audit.actions[e.action] ?? e.action}</span>
                  <span className="text-slate-600">{e.actor}</span>
                  <span className="ml-auto text-xs text-slate-500">{fmtRelative(e.created_at)}</span>
                </li>
              ))}
            </ul>
          )}
        </section>
      ) : null}
    </div>
  );
}
