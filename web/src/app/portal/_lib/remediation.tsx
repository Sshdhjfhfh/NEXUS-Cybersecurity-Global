"use client";

import {
  type FormEvent,
  type KeyboardEvent,
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";

import {
  AlertTriangleIcon,
  CalendarClockIcon,
  ClipboardCheckIcon,
  MessageSquareIcon,
  RefreshCwIcon,
  SearchIcon,
  ShieldIcon,
  XIcon,
} from "lucide-react";

import { usePortalI18n } from "./i18n";
import { SevBadge } from "./overview";
import {
  type Me,
  type Member,
  portalApi,
  type RemediationBucket,
  type RemediationEvent,
  type RemediationItem,
  type RemediationState,
  type RemediationUpdate,
} from "./portal";

// Remediation workspace. The client organizes the fix (owners, target dates,
// progress, comments) and asks NEXUS to verify it; only a NEXUS retest can mark
// a finding fixed. Every permission shown here is enforced again server-side —
// hiding a control is a convenience, not the authorization.

const STATE_TONE: Record<RemediationState, string> = {
  open: "border-slate-200 bg-slate-100 text-slate-700",
  in_progress: "border-sky-200 bg-sky-50 text-sky-800",
  retest_requested: "border-violet-200 bg-violet-50 text-violet-800",
  retest_running: "border-violet-200 bg-violet-50 text-violet-800",
  still_vulnerable: "border-rose-200 bg-rose-50 text-rose-700",
  verified_fixed: "border-emerald-200 bg-emerald-50 text-emerald-800",
  risk_accepted: "border-amber-200 bg-amber-50 text-amber-800",
};

// States in which the team is still expected to act on the finding.
const WORKABLE: RemediationState[] = ["open", "in_progress", "still_vulnerable"];
const SEVERITIES = ["critical", "high", "medium", "low"] as const;
const BUCKETS = ["all", "open", "in_progress", "remediated", "risk_accepted"] as const;
type BucketFilter = (typeof BUCKETS)[number];

const card = "rounded-2xl border border-slate-200 bg-white p-5 shadow-sm";
const focusRing = "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400";
const input =
  "w-full rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-800 placeholder:text-slate-500 transition focus:border-cyan-400 focus:bg-white focus:outline-none focus:ring-4 focus:ring-cyan-100";
const primaryBtn = `inline-flex items-center justify-center gap-1.5 rounded-xl bg-gradient-to-r from-cyan-700 to-sky-700 px-3.5 py-2 text-sm font-semibold text-white shadow-sm transition hover:brightness-105 disabled:opacity-60 ${focusRing}`;
const secondaryBtn = `inline-flex items-center justify-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3.5 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-50 disabled:opacity-60 ${focusRing}`;

function isoDay(offsetDays: number): string {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() + offsetDays);
  return d.toISOString().slice(0, 10);
}

export function StateBadge({ state }: { state: RemediationState }) {
  const { t } = usePortalI18n();
  return (
    <span
      className={`inline-flex shrink-0 whitespace-nowrap rounded-full border px-2 py-0.5 text-[11px] font-semibold ${STATE_TONE[state] ?? STATE_TONE.open}`}
    >
      {t.rem.state[state] ?? state}
    </span>
  );
}

function DueCell({ item }: { item: RemediationItem }) {
  const { t, fmtDay } = usePortalI18n();
  if (!item.due_date || item.bucket === "remediated" || item.bucket === "risk_accepted") {
    return <span className="text-slate-500">—</span>;
  }
  return (
    <span className="inline-flex flex-wrap items-center gap-1.5">
      <span className={item.overdue ? "font-semibold text-rose-700" : "text-slate-700"}>{fmtDay(item.due_date)}</span>
      {item.overdue ? (
        <span className="rounded-full border border-rose-200 bg-rose-50 px-1.5 py-px text-[10px] font-semibold uppercase text-rose-700">
          {t.rem.overdue}
        </span>
      ) : (
        <span className="text-[11px] text-slate-500">
          {item.due_source === "custom" ? t.rem.customTag : t.rem.slaTag}
        </span>
      )}
    </span>
  );
}

type PanelProps = {
  me: Me;
  // Finding to open on arrival (e.g. from an overview priority).
  focusId: number | null;
  onFocusHandled: () => void;
};

export function RemediationPanel({ me, focusId, onFocusHandled }: PanelProps) {
  const { t } = usePortalI18n();
  const [items, setItems] = useState<RemediationItem[] | null>(null);
  const [sla, setSla] = useState<Record<string, number>>({});
  const [err, setErr] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [bucket, setBucket] = useState<BucketFilter>("all");
  const [sev, setSev] = useState("");
  const [mine, setMine] = useState(false);
  const [overdueOnly, setOverdueOnly] = useState(false);
  const [q, setQ] = useState("");
  const [selected, setSelected] = useState<number | null>(null);
  const [members, setMembers] = useState<Member[]>([]);
  const canAdmin = me.user.role === "owner" || me.user.role === "admin";

  const load = useCallback(async () => {
    setRefreshing(true);
    setErr(false);
    try {
      const data = await portalApi.listRemediation();
      setItems(data.items ?? []);
      setSla(data.sla_days ?? {});
    } catch {
      setErr(true);
    } finally {
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  // Assignable people: only admins assign, and only they can list members.
  useEffect(() => {
    if (!canAdmin) return;
    portalApi
      .listMembers()
      .then(({ members: m }) => setMembers((m ?? []).filter((x) => !x.disabled)))
      .catch(() => setMembers([]));
  }, [canAdmin]);

  useEffect(() => {
    if (focusId === null || items === null) return;
    if (items.some((i) => i.finding_id === focusId)) setSelected(focusId);
    onFocusHandled();
  }, [focusId, items, onFocusHandled]);

  const onItemChanged = useCallback((it: RemediationItem) => {
    setItems((cur) => (cur ? cur.map((x) => (x.finding_id === it.finding_id ? it : x)) : cur));
  }, []);

  const counts = useMemo(() => {
    const c: Record<BucketFilter, number> = { all: 0, open: 0, in_progress: 0, remediated: 0, risk_accepted: 0 };
    for (const i of items ?? []) {
      c.all++;
      c[i.bucket as RemediationBucket]++;
    }
    return c;
  }, [items]);
  const overdueCount = useMemo(() => (items ?? []).filter((i) => i.overdue).length, [items]);

  const filtered = useMemo(() => {
    const needle = q.trim().toLowerCase();
    return (items ?? []).filter(
      (i) =>
        (bucket === "all" || i.bucket === bucket) &&
        (!sev || i.severity === sev) &&
        (!mine || i.assignee_id === me.user.id) &&
        (!overdueOnly || i.overdue) &&
        (!needle || `${i.name} ${i.vulnclass} ${i.request_title}`.toLowerCase().includes(needle)),
    );
  }, [items, bucket, sev, mine, overdueOnly, q, me.user.id]);

  const filtersActive = bucket !== "all" || sev !== "" || mine || overdueOnly || q.trim() !== "";
  function clearFilters() {
    setBucket("all");
    setSev("");
    setMine(false);
    setOverdueOnly(false);
    setQ("");
  }

  if (err && !items) {
    return (
      <div role="alert" className="mt-6 rounded-2xl border border-rose-200 bg-rose-50 p-5 text-sm text-rose-700">
        {t.rem.loadError}{" "}
        <button type="button" onClick={() => void load()} className={`font-semibold underline ${focusRing}`}>
          {t.retry}
        </button>
      </div>
    );
  }
  if (!items) {
    return <p className="mt-6 text-sm text-slate-500">{t.rem.loading}</p>;
  }

  const th = "px-3 py-2 text-left text-[11px] font-semibold uppercase tracking-wide text-slate-500";
  const td = "px-3 py-3 align-top text-sm text-slate-700";

  return (
    <div className="mt-6 space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="max-w-3xl">
          <p className="text-sm text-slate-600">{t.rem.intro}</p>
          {sla.critical ? (
            <p className="mt-1 flex items-center gap-1.5 text-xs text-slate-500">
              <CalendarClockIcon className="size-3.5" aria-hidden />
              {t.rem.sla(sla.critical, sla.high, sla.medium, sla.low)}
            </p>
          ) : null}
        </div>
        <button type="button" onClick={() => void load()} disabled={refreshing} className={secondaryBtn}>
          <RefreshCwIcon
            className={`size-3.5 ${refreshing ? "animate-spin motion-reduce:animate-none" : ""}`}
            aria-hidden
          />
          {t.refresh}
        </button>
      </div>

      {items.length === 0 ? (
        <section className={`${card} flex flex-col items-center px-6 py-10 text-center`}>
          <span className="grid size-12 place-items-center rounded-2xl bg-cyan-50 text-cyan-700">
            <ClipboardCheckIcon className="size-6" aria-hidden />
          </span>
          <p className="mt-4 max-w-md text-sm text-slate-500">{t.rem.empty}</p>
        </section>
      ) : (
        <section className={card}>
          {/* Bucket filter */}
          <fieldset className="flex flex-wrap gap-2">
            <legend className="sr-only">{t.rem.col.state}</legend>
            {BUCKETS.map((b) => (
              <button
                key={b}
                type="button"
                aria-pressed={bucket === b}
                onClick={() => setBucket(b)}
                className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition ${focusRing} ${
                  bucket === b
                    ? "border-slate-900 bg-slate-900 text-white"
                    : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
                }`}
              >
                {t.rem.bucket[b]}
                <span className={`tabular-nums ${bucket === b ? "text-slate-300" : "text-slate-500"}`}>
                  {counts[b]}
                </span>
              </button>
            ))}
            {overdueCount > 0 ? (
              <button
                type="button"
                aria-pressed={overdueOnly}
                onClick={() => setOverdueOnly((v) => !v)}
                className={`inline-flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-semibold transition ${focusRing} ${
                  overdueOnly
                    ? "border-rose-700 bg-rose-700 text-white"
                    : "border-rose-200 bg-rose-50 text-rose-700 hover:bg-rose-100"
                }`}
              >
                <AlertTriangleIcon className="size-3.5" aria-hidden />
                {t.rem.overdueOnly}
                <span className="tabular-nums">{overdueCount}</span>
              </button>
            ) : null}
          </fieldset>

          {/* Secondary filters */}
          <div className="mt-4 flex flex-wrap items-center gap-3">
            <label className="relative min-w-[220px] flex-1">
              <span className="sr-only">{t.rem.search}</span>
              <SearchIcon
                className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-slate-500"
                aria-hidden
              />
              <input
                type="search"
                value={q}
                onChange={(e) => setQ(e.target.value)}
                placeholder={t.rem.search}
                className={`${input} pl-9`}
              />
            </label>
            <label className="text-xs text-slate-600">
              <span className="sr-only">{t.rem.filterSeverity}</span>
              <select value={sev} onChange={(e) => setSev(e.target.value)} className={`${input} w-auto pr-8`}>
                <option value="">{t.rem.allSeverities}</option>
                {SEVERITIES.map((s) => (
                  <option key={s} value={s}>
                    {t.sev[s]}
                  </option>
                ))}
              </select>
            </label>
            <label className="inline-flex cursor-pointer items-center gap-2 text-sm text-slate-700">
              <input
                type="checkbox"
                checked={mine}
                onChange={(e) => setMine(e.target.checked)}
                className="size-4 rounded border-slate-300 accent-cyan-700"
              />
              {t.rem.mine}
            </label>
          </div>

          <div className="mt-4 flex items-center justify-between gap-3 text-xs text-slate-500">
            <span aria-live="polite">{t.rem.count(filtered.length, items.length)}</span>
            {filtersActive ? (
              <button
                type="button"
                onClick={clearFilters}
                className={`rounded font-semibold text-cyan-700 hover:text-cyan-800 ${focusRing}`}
              >
                {t.rem.clearFilters}
              </button>
            ) : null}
          </div>

          {filtered.length === 0 ? (
            <p className="mt-3 rounded-xl border border-dashed border-slate-300 p-5 text-center text-sm text-slate-500">
              {t.rem.noMatch}
            </p>
          ) : (
            <>
              {/* Phones: one card per finding, every field visible without side-scrolling. */}
              <ul className="mt-2 divide-y divide-slate-100 md:hidden">
                {filtered.map((i) => (
                  <li key={i.finding_id} className={selected === i.finding_id ? "bg-cyan-50/60" : ""}>
                    <button
                      type="button"
                      onClick={() => setSelected(i.finding_id)}
                      aria-label={t.rem.openItem(i.name || i.vulnclass)}
                      className={`block w-full space-y-1.5 rounded-lg px-1 py-3 text-left hover:bg-slate-50 ${focusRing}`}
                    >
                      <span className="flex items-start gap-2">
                        <SevBadge sev={i.severity} />
                        <span className="min-w-0 font-medium text-slate-900">{i.name || i.vulnclass}</span>
                      </span>
                      <span className="block truncate text-xs text-slate-500">{i.request_title}</span>
                      <span className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-600">
                        <StateBadge state={i.state} />
                        <span className="break-all">{i.assignee_email || t.rem.unassigned}</span>
                        <DueCell item={i} />
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
              <div className="mt-2 hidden overflow-x-auto md:block">
                <table className="w-full min-w-[720px] border-collapse">
                  <thead className="border-b border-slate-200">
                    <tr>
                      <th scope="col" className={th}>
                        {t.rem.col.finding}
                      </th>
                      <th scope="col" className={th}>
                        {t.rem.col.state}
                      </th>
                      <th scope="col" className={th}>
                        {t.rem.col.owner}
                      </th>
                      <th scope="col" className={th}>
                        {t.rem.col.due}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {filtered.map((i) => (
                      <tr
                        key={i.finding_id}
                        className={`hover:bg-slate-50 ${selected === i.finding_id ? "bg-cyan-50/60" : ""}`}
                      >
                        <td className={td}>
                          <div className="flex items-start gap-2">
                            <SevBadge sev={i.severity} />
                            <div className="min-w-0">
                              <button
                                type="button"
                                onClick={() => setSelected(i.finding_id)}
                                aria-label={t.rem.openItem(i.name || i.vulnclass)}
                                className={`rounded text-left font-medium text-slate-900 hover:text-cyan-800 hover:underline ${focusRing}`}
                              >
                                {i.name || i.vulnclass}
                              </button>
                              <p className="mt-0.5 truncate text-xs text-slate-500">{i.request_title}</p>
                            </div>
                          </div>
                        </td>
                        <td className={td}>
                          <StateBadge state={i.state} />
                        </td>
                        <td className={td}>
                          {i.assignee_email ? (
                            <span className="break-all">{i.assignee_email}</span>
                          ) : (
                            <span className="text-slate-500">{t.rem.unassigned}</span>
                          )}
                        </td>
                        <td className={td}>
                          <DueCell item={i} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
        </section>
      )}

      {selected !== null ? (
        <RemediationDrawer
          fid={selected}
          me={me}
          members={members}
          sla={sla}
          onClose={() => setSelected(null)}
          onChanged={onItemChanged}
        />
      ) : null}
    </div>
  );
}

type DrawerProps = {
  fid: number;
  me: Me;
  members: Member[];
  sla: Record<string, number>;
  onClose: () => void;
  onChanged: (it: RemediationItem) => void;
};

function RemediationDrawer({ fid, me, members, sla, onClose, onChanged }: DrawerProps) {
  const { t, fmtDay, fmtDate } = usePortalI18n();
  const [item, setItem] = useState<RemediationItem | null>(null);
  const [report, setReport] = useState("");
  const [events, setEvents] = useState<RemediationEvent[]>([]);
  const [loadErr, setLoadErr] = useState(false);
  const [actionErr, setActionErr] = useState("");
  const [busy, setBusy] = useState("");
  const [assignee, setAssignee] = useState("");
  const [due, setDue] = useState("");
  const [note, setNote] = useState("");
  const [reason, setReason] = useState("");
  const [expires, setExpires] = useState(isoDay(90));
  const [comment, setComment] = useState("");
  const panelRef = useRef<HTMLDivElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);
  const canAdmin = me.user.role === "owner" || me.user.role === "admin";

  const load = useCallback(async () => {
    setLoadErr(false);
    try {
      const d = await portalApi.getRemediation(fid);
      setItem(d.item);
      setReport(d.report);
      setEvents(d.events ?? []);
      setAssignee(d.item.assignee_id ? String(d.item.assignee_id) : "");
      setDue(d.item.due_source === "custom" && d.item.due_date ? d.item.due_date : "");
      onChanged(d.item);
    } catch {
      setLoadErr(true);
    }
  }, [fid, onChanged]);

  useEffect(() => {
    setItem(null);
    void load();
  }, [load]);

  // Modal behaviour: focus inside on open, restore focus on close, lock page scroll.
  useEffect(() => {
    const prev = document.activeElement as HTMLElement | null;
    closeRef.current?.focus();
    const overflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = overflow;
      prev?.focus?.();
    };
  }, []);

  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if (e.key === "Escape") {
      e.stopPropagation();
      onClose();
      return;
    }
    if (e.key !== "Tab" || !panelRef.current) return;
    // Keep keyboard focus inside the dialog.
    const nodes = panelRef.current.querySelectorAll<HTMLElement>(
      'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), summary, [tabindex]:not([tabindex="-1"])',
    );
    if (nodes.length === 0) return;
    const first = nodes[0];
    const last = nodes[nodes.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }

  async function run(key: string, fn: () => Promise<unknown>) {
    setBusy(key);
    setActionErr("");
    try {
      await fn();
      await load();
      return true;
    } catch (ex) {
      setActionErr(ex instanceof Error && ex.message ? ex.message : t.rem.actionError);
      return false;
    } finally {
      setBusy("");
    }
  }

  function saveOwnership(e: FormEvent) {
    e.preventDefault();
    if (!item) return;
    const patch: RemediationUpdate = {};
    const newAssignee = assignee ? Number(assignee) : null;
    if (newAssignee !== item.assignee_id) patch.assignee_id = newAssignee;
    const curDue = item.due_source === "custom" ? (item.due_date ?? "") : "";
    if (due !== curDue) patch.due_date = due || null;
    if (Object.keys(patch).length === 0) return;
    void run("owner", () => portalApi.updateRemediation(fid, patch));
  }

  async function submitRetest(e: FormEvent) {
    e.preventDefault();
    if (await run("retest", () => portalApi.requestRetest(fid, note.trim()))) setNote("");
  }

  async function submitRisk(e: FormEvent) {
    e.preventDefault();
    if (await run("risk", () => portalApi.acceptRisk(fid, reason.trim(), expires))) setReason("");
  }

  function revoke() {
    if (!window.confirm(t.rem.confirmRevoke)) return;
    void run("revoke", () => portalApi.revokeRisk(fid));
  }

  async function submitComment(e: FormEvent) {
    e.preventDefault();
    if (await run("comment", () => portalApi.commentRemediation(fid, comment.trim()))) setComment("");
  }

  const workable = item ? WORKABLE.includes(item.state) : false;
  const title = item ? item.name || item.vulnclass : "";

  return (
    <div className="fixed inset-0 z-40 flex justify-end">
      <button
        type="button"
        aria-label={t.rem.close}
        tabIndex={-1}
        onClick={onClose}
        className="absolute inset-0 cursor-default bg-slate-900/40 backdrop-blur-[1px]"
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="rem-drawer-title"
        onKeyDown={onKeyDown}
        className="relative flex h-full w-full max-w-2xl flex-col overflow-hidden bg-slate-50 shadow-2xl"
      >
        <header className="flex items-start gap-3 border-b border-slate-200 bg-white px-5 py-4">
          <div className="min-w-0 flex-1">
            {item ? (
              <div className="flex flex-wrap items-center gap-2">
                <SevBadge sev={item.severity} />
                <StateBadge state={item.state} />
              </div>
            ) : null}
            <h2 id="rem-drawer-title" className="mt-1.5 text-lg font-semibold text-slate-900">
              {title || t.loading}
            </h2>
            {item ? (
              <p className="mt-0.5 text-xs text-slate-500">
                {t.rem.inEvaluation(item.request_title)} · {t.rem.detected(fmtDate(item.detected_at))}
              </p>
            ) : null}
          </div>
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            aria-label={t.rem.close}
            className={`rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800 ${focusRing}`}
          >
            <XIcon className="size-5" aria-hidden />
          </button>
        </header>

        <div className="flex-1 space-y-4 overflow-y-auto px-5 py-5">
          {loadErr ? (
            <div role="alert" className="rounded-xl border border-rose-200 bg-rose-50 p-4 text-sm text-rose-700">
              {t.rem.loadError}{" "}
              <button type="button" onClick={() => void load()} className={`font-semibold underline ${focusRing}`}>
                {t.retry}
              </button>
            </div>
          ) : !item ? (
            <p className="text-sm text-slate-500">{t.loading}</p>
          ) : (
            <>
              <p className={`rounded-xl border px-4 py-3 text-sm ${STATE_TONE[item.state]}`}>
                {t.rem.stateHint[item.state]}
              </p>
              {item.risk_expired && item.risk_expires_at ? (
                <p className="flex items-start gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
                  <AlertTriangleIcon className="mt-0.5 size-4 shrink-0" aria-hidden />
                  {t.rem.riskExpired(fmtDay(item.risk_expires_at))}
                </p>
              ) : null}
              {actionErr ? (
                <p
                  role="alert"
                  className="rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700"
                >
                  {actionErr}
                </p>
              ) : null}

              {/* Ownership + target date */}
              <Section title={`${t.rem.owner} · ${t.rem.targetDate}`}>
                {canAdmin && item.state !== "verified_fixed" ? (
                  <form onSubmit={saveOwnership} className="grid gap-3 sm:grid-cols-2">
                    <label className="text-xs font-medium text-slate-600">
                      {t.rem.owner}
                      <select
                        value={assignee}
                        onChange={(e) => setAssignee(e.target.value)}
                        className={`${input} mt-1.5`}
                      >
                        <option value="">{t.rem.unassigned}</option>
                        {members.map((m) => (
                          <option key={m.id} value={m.id}>
                            {m.email}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="text-xs font-medium text-slate-600">
                      {t.rem.targetDate}
                      <input
                        type="date"
                        value={due}
                        min={isoDay(-365)}
                        max={isoDay(730)}
                        onChange={(e) => setDue(e.target.value)}
                        className={`${input} mt-1.5`}
                      />
                    </label>
                    <p className="text-xs text-slate-500 sm:col-span-2">
                      {item.due_date ? (
                        <>
                          <DueCell item={item} />{" "}
                        </>
                      ) : null}
                      {sla[item.severity] ? t.rem.targetHint(sla[item.severity]) : null}
                    </p>
                    <div className="flex flex-wrap gap-2 sm:col-span-2">
                      <button type="submit" disabled={busy !== ""} className={primaryBtn}>
                        {busy === "owner" ? t.rem.saving : t.rem.save}
                      </button>
                      {due ? (
                        <button type="button" onClick={() => setDue("")} className={secondaryBtn}>
                          {t.rem.useSla}
                        </button>
                      ) : null}
                    </div>
                  </form>
                ) : (
                  <dl className="grid gap-3 text-sm sm:grid-cols-2">
                    <div>
                      <dt className="text-xs text-slate-500">{t.rem.owner}</dt>
                      <dd className="mt-0.5 break-all text-slate-800">{item.assignee_email || t.rem.unassigned}</dd>
                    </div>
                    <div>
                      <dt className="text-xs text-slate-500">{t.rem.targetDate}</dt>
                      <dd className="mt-0.5">
                        <DueCell item={item} />
                      </dd>
                    </div>
                    {!canAdmin ? <p className="text-xs text-slate-500 sm:col-span-2">{t.rem.adminOnly}</p> : null}
                  </dl>
                )}
              </Section>

              {/* Team progress */}
              {workable ? (
                <Section title={t.rem.progress}>
                  <div className="flex flex-wrap gap-2">
                    {item.state === "in_progress" ? (
                      <button
                        type="button"
                        disabled={busy !== ""}
                        onClick={() => void run("state", () => portalApi.updateRemediation(fid, { state: "open" }))}
                        className={secondaryBtn}
                      >
                        {t.rem.markOpen}
                      </button>
                    ) : (
                      <button
                        type="button"
                        disabled={busy !== ""}
                        onClick={() =>
                          void run("state", () => portalApi.updateRemediation(fid, { state: "in_progress" }))
                        }
                        className={secondaryBtn}
                      >
                        {t.rem.markInProgress}
                      </button>
                    )}
                  </div>
                </Section>
              ) : null}

              {/* NEXUS verification */}
              {item.state !== "risk_accepted" ? (
                <Section title={t.rem.retestTitle} icon={<ShieldIcon className="size-4" aria-hidden />}>
                  <p className="text-xs text-slate-500">{t.rem.retestExplain}</p>
                  {item.state === "retest_requested" && item.retest_requested_at ? (
                    <div className="mt-3 rounded-xl border border-violet-200 bg-violet-50 px-4 py-3 text-sm text-violet-900">
                      <p className="font-medium">{t.rem.requestedOn(fmtDate(item.retest_requested_at))}</p>
                      {item.retest_note ? <p className="mt-1 whitespace-pre-wrap">{item.retest_note}</p> : null}
                    </div>
                  ) : null}
                  {item.last_retest ? (
                    <p className="mt-3 text-sm text-slate-700">
                      <span className="font-medium">{t.rem.lastRetest}:</span>{" "}
                      {item.last_retest.status === "completed"
                        ? t.rem.verdict[item.last_retest.verdict]
                        : t.rem.retestStatus[item.last_retest.status]}{" "}
                      <span className="text-xs text-slate-500">
                        · {fmtDate(item.last_retest.finished_at ?? item.last_retest.created_at)}
                      </span>
                    </p>
                  ) : null}
                  {workable ? (
                    <form onSubmit={submitRetest} className="mt-3 space-y-2">
                      <label className="block text-xs font-medium text-slate-600">
                        {t.rem.retestNote}
                        <textarea
                          value={note}
                          onChange={(e) => setNote(e.target.value)}
                          maxLength={1000}
                          rows={2}
                          placeholder={t.rem.retestNotePlaceholder}
                          className={`${input} mt-1.5`}
                        />
                      </label>
                      <button type="submit" disabled={busy !== ""} className={primaryBtn}>
                        {busy === "retest" ? t.rem.requesting : t.rem.requestRetest}
                      </button>
                    </form>
                  ) : null}
                </Section>
              ) : null}

              {/* Risk acceptance */}
              {item.state === "risk_accepted" || (canAdmin && workable) ? (
                <Section title={t.rem.riskTitle} icon={<AlertTriangleIcon className="size-4" aria-hidden />}>
                  {item.state === "risk_accepted" ? (
                    <div className="space-y-2 text-sm">
                      {item.risk_expires_at ? (
                        <p className="text-slate-700">
                          {t.rem.acceptedBy(item.risk_accepted_by || "—", fmtDay(item.risk_expires_at))}
                        </p>
                      ) : null}
                      {item.risk_reason ? (
                        <p className="whitespace-pre-wrap rounded-xl bg-slate-50 px-3 py-2 text-slate-700">
                          {item.risk_reason}
                        </p>
                      ) : null}
                      {canAdmin && item.risk_expires_at ? (
                        <button type="button" disabled={busy !== ""} onClick={revoke} className={secondaryBtn}>
                          {t.rem.revokeRisk}
                        </button>
                      ) : !canAdmin ? (
                        <p className="text-xs text-slate-500">{t.rem.riskAdminOnly}</p>
                      ) : null}
                    </div>
                  ) : (
                    <details className="group">
                      <summary className={`cursor-pointer rounded text-sm font-medium text-cyan-700 ${focusRing}`}>
                        {t.rem.acceptRisk}…
                      </summary>
                      <form onSubmit={submitRisk} className="mt-3 space-y-3">
                        <p className="text-xs text-slate-500">{t.rem.riskExplain}</p>
                        <label className="block text-xs font-medium text-slate-600">
                          {t.rem.reason}
                          <textarea
                            value={reason}
                            onChange={(e) => setReason(e.target.value)}
                            required
                            minLength={10}
                            maxLength={2000}
                            rows={3}
                            className={`${input} mt-1.5`}
                          />
                        </label>
                        <label className="block text-xs font-medium text-slate-600">
                          {t.rem.expires}
                          <input
                            type="date"
                            value={expires}
                            required
                            min={isoDay(1)}
                            max={isoDay(365)}
                            onChange={(e) => setExpires(e.target.value)}
                            className={`${input} mt-1.5 sm:w-56`}
                          />
                        </label>
                        <button
                          type="submit"
                          disabled={busy !== "" || reason.trim().length < 10 || !expires}
                          className={primaryBtn}
                        >
                          {busy === "risk" ? t.rem.accepting : t.rem.acceptRisk}
                        </button>
                      </form>
                    </details>
                  )}
                </Section>
              ) : null}

              {/* Report */}
              <Section title={t.rem.report}>
                {item.summary ? <p className="mb-2 text-sm text-slate-600">{item.summary}</p> : null}
                <details>
                  <summary className={`cursor-pointer rounded text-sm font-medium text-cyan-700 ${focusRing}`}>
                    {t.rem.showReport}
                  </summary>
                  <pre className="mt-2 max-h-96 overflow-auto whitespace-pre-wrap rounded-xl bg-slate-50 p-3 font-sans text-xs leading-relaxed text-slate-700">
                    {report || t.rem.noReport}
                  </pre>
                </details>
              </Section>

              {/* Timeline + comments */}
              <Section title={t.rem.timeline} icon={<MessageSquareIcon className="size-4" aria-hidden />}>
                {events.length === 0 ? (
                  <p className="text-sm text-slate-500">{t.rem.noEvents}</p>
                ) : (
                  <ol className="space-y-3">
                    {events.map((e) => (
                      <TimelineEntry key={e.id} e={e} />
                    ))}
                  </ol>
                )}
                <form onSubmit={submitComment} className="mt-4 space-y-2">
                  <label className="block text-xs font-medium text-slate-600">
                    {t.rem.comment}
                    <textarea
                      value={comment}
                      onChange={(e) => setComment(e.target.value)}
                      maxLength={2000}
                      rows={2}
                      placeholder={t.rem.commentPlaceholder}
                      className={`${input} mt-1.5`}
                    />
                  </label>
                  <button type="submit" disabled={busy !== "" || !comment.trim()} className={primaryBtn}>
                    {t.rem.send}
                  </button>
                </form>
              </Section>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function Section({ title, icon, children }: { title: string; icon?: ReactNode; children: ReactNode }) {
  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
      <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold text-slate-900">
        {icon !== undefined ? (
          <span className="text-cyan-700" aria-hidden>
            {icon}
          </span>
        ) : null}
        {title}
      </h3>
      {children}
    </section>
  );
}

function TimelineEntry({ e }: { e: RemediationEvent }) {
  const { t, fmtDay, fmtDateTime, fmtRelative } = usePortalI18n();
  let summary = e.detail;
  let body = "";
  switch (e.kind) {
    case "state":
      summary = t.rem.state[e.detail] ?? e.detail;
      break;
    case "due_set":
      summary = fmtDay(e.detail);
      break;
    case "risk_accepted":
      // "<YYYY-MM-DD> <reason>"
      summary = fmtDay(e.detail.slice(0, 10));
      body = e.detail.slice(11);
      break;
    case "comment":
    case "retest_requested":
      body = e.detail;
      break;
  }
  return (
    <li className="flex gap-3">
      <span
        className={`mt-1.5 size-2 shrink-0 rounded-full ${e.kind === "comment" ? "bg-cyan-600" : "bg-slate-300"}`}
        aria-hidden
      />
      <div className="min-w-0 flex-1 text-sm">
        <p className="text-slate-700">
          <span className="break-all font-medium text-slate-900">{e.actor || "—"}</span> {t.rem.event(e.kind, summary)}{" "}
          <time dateTime={e.created_at} title={fmtDateTime(e.created_at)} className="text-xs text-slate-500">
            · {fmtRelative(e.created_at)}
          </time>
        </p>
        {body ? (
          <p className="mt-1 whitespace-pre-wrap rounded-xl bg-slate-50 px-3 py-2 text-slate-700">{body}</p>
        ) : null}
      </div>
    </li>
  );
}
