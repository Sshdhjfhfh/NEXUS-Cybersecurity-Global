"use client";

import { type FormEvent, type KeyboardEvent, useCallback, useEffect, useRef, useState } from "react";

import {
  CheckCircle2Icon,
  CopyIcon,
  FileSignatureIcon,
  GlobeIcon,
  PlusIcon,
  RefreshCwIcon,
  ServerIcon,
  ShieldAlertIcon,
  ShieldCheckIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react";

import { localizeServerError, usePortalI18n } from "./i18n";
import {
  type AuthorizationResponse,
  type Me,
  type PortalRequest,
  type PortalTarget,
  portalApi,
  type TargetStatus,
  type TargetsResponse,
} from "./portal";
import { Field, PrimaryButton } from "./ui";

// Targets + Rules of Engagement. Nothing an organization registers is tested
// until it proves ownership (DNS TXT, an HTTP file, or a NEXUS approval on
// written evidence) and an admin signs the RoE for the request. The server
// enforces all of it; this UI only explains the state and collects the proof.

const focusRing = "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400";
const secondaryBtn = `inline-flex items-center justify-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 transition hover:bg-slate-50 disabled:opacity-60 ${focusRing}`;
const inputCls =
  "mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3.5 py-2.5 text-sm text-slate-800 placeholder:text-slate-400 transition focus:border-cyan-400 focus:bg-white focus:outline-none focus:ring-4 focus:ring-cyan-100";

const STATUS_TONE: Record<TargetStatus, string> = {
  pending: "border-amber-200 bg-amber-50 text-amber-700",
  verified: "border-emerald-200 bg-emerald-50 text-emerald-700",
  rejected: "border-rose-200 bg-rose-50 text-rose-700",
  expired: "border-slate-200 bg-slate-100 text-slate-600",
};

export function TargetStatusBadge({ status }: { status: TargetStatus }) {
  const { t } = usePortalI18n();
  return (
    <span
      className={`shrink-0 rounded-full border px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${STATUS_TONE[status]}`}
    >
      {t.tgt.status[status] ?? status}
    </span>
  );
}

function isAdmin(me: Me) {
  return me.user.role === "owner" || me.user.role === "admin";
}

function todayUTC(offsetDays = 0) {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() + offsetDays);
  return d.toISOString().slice(0, 10);
}

export function CopyValue({ value }: { value: string }) {
  const { t } = usePortalI18n();
  const [copied, setCopied] = useState(false);
  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard blocked: the value stays selectable */
    }
  }
  return (
    <div className="mt-1 flex items-center gap-2">
      <code className="min-w-0 flex-1 break-all rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-[11px] text-cyan-800">
        {value}
      </code>
      <button
        type="button"
        onClick={copy}
        className={`inline-flex shrink-0 items-center gap-1 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-[11px] text-slate-600 hover:border-cyan-400 hover:text-cyan-700 ${focusRing}`}
      >
        <CopyIcon className="size-3.5" aria-hidden /> {copied ? t.copied : t.copy}
      </button>
    </div>
  );
}

// ---------------------------------------------------------------- Targets tab

export function TargetsPanel({ me }: { me: Me }) {
  const { t } = usePortalI18n();
  const admin = isAdmin(me);
  const [data, setData] = useState<TargetsResponse | null>(null);
  const [loadErr, setLoadErr] = useState("");
  const [value, setValue] = useState("");
  const [adding, setAdding] = useState(false);
  const [addErr, setAddErr] = useState("");

  const load = useCallback(async () => {
    try {
      setData(await portalApi.listTargets());
      setLoadErr("");
    } catch (ex) {
      setLoadErr(ex instanceof Error && ex.message ? ex.message : t.tgt.loadError);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  async function add(e: FormEvent) {
    e.preventDefault();
    setAdding(true);
    setAddErr("");
    try {
      await portalApi.addTarget(value.trim());
      setValue("");
      await load();
    } catch (ex) {
      setAddErr(ex instanceof Error && ex.message ? ex.message : t.tgt.addError);
    } finally {
      setAdding(false);
    }
  }

  if (!data && !loadErr) return <p className="mt-6 text-sm text-slate-500">{t.tgt.loading}</p>;

  return (
    <div className="mt-6 grid gap-8 lg:grid-cols-[1fr_1.6fr]">
      <section className="h-fit space-y-4">
        <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-cyan-700">
            <ShieldCheckIcon className="size-4" aria-hidden /> {t.tabs.targets}
          </h2>
          <p className="mt-2 text-sm text-slate-600">{t.tgt.intro}</p>
          {data ? <p className="mt-2 text-xs text-slate-500">{t.tgt.validity(data.validity_days)}</p> : null}
          {admin ? (
            <form onSubmit={add} className="mt-5 space-y-3">
              <Field
                label={t.tgt.valueLabel}
                value={value}
                onChange={(e) => setValue(e.target.value)}
                placeholder={t.tgt.valuePlaceholder}
                autoComplete="off"
                spellCheck={false}
              />
              {addErr ? (
                <p role="alert" className="text-sm text-rose-600">
                  {addErr}
                </p>
              ) : null}
              <PrimaryButton type="submit" disabled={adding || !value.trim()}>
                <span className="inline-flex items-center gap-1.5">
                  <PlusIcon className="size-4" aria-hidden /> {adding ? t.tgt.adding : t.tgt.add}
                </span>
              </PrimaryButton>
            </form>
          ) : (
            <p className="mt-4 rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-600">
              {t.tgt.memberHint}
            </p>
          )}
        </div>
        <p className="flex gap-2 rounded-xl border border-sky-200 bg-sky-50 px-4 py-3 text-xs text-sky-800">
          <ShieldAlertIcon className="mt-0.5 size-4 shrink-0" aria-hidden /> {t.tgt.manualHint}
        </p>
      </section>

      <section>
        <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-700">
          {t.tgt.list(data?.targets.length ?? 0)}
        </h2>
        {loadErr ? (
          <p role="alert" className="mt-4 text-sm text-rose-600">
            {loadErr}
          </p>
        ) : null}
        {data && data.targets.length === 0 ? (
          <p className="mt-4 rounded-xl border border-dashed border-slate-300 bg-white p-5 text-sm text-slate-500">
            {t.tgt.empty}
          </p>
        ) : null}
        {data && data.targets.length > 0 ? (
          <ul className="mt-4 space-y-3">
            {data.targets.map((x) => (
              <TargetRow key={x.id} target={x} info={data} admin={admin} onChanged={load} />
            ))}
          </ul>
        ) : null}
      </section>
    </div>
  );
}

function TargetRow({
  target,
  info,
  admin,
  onChanged,
}: {
  target: PortalTarget;
  info: TargetsResponse;
  admin: boolean;
  onChanged: () => Promise<void>;
}) {
  const { t, fmtDate } = usePortalI18n();
  const [busy, setBusy] = useState<"" | "dns" | "http" | "delete">("");
  const [msg, setMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const actionable = target.status === "pending" || target.status === "expired";
  const [open, setOpen] = useState(actionable && admin);
  const proofURL = `https://${target.value}${info.http_path}`;

  async function check(method: "dns" | "http") {
    setBusy(method);
    setMsg(null);
    try {
      const r = await portalApi.verifyTarget(target.id, method);
      setMsg(r.ok ? { ok: true, text: t.tgt.checkOk } : { ok: false, text: localizeServerError(r.error) });
      await onChanged();
    } catch (ex) {
      setMsg({ ok: false, text: ex instanceof Error && ex.message ? ex.message : t.genericError });
    } finally {
      setBusy("");
    }
  }

  async function remove() {
    if (!window.confirm(t.tgt.confirmRemove(target.value))) return;
    setBusy("delete");
    setMsg(null);
    try {
      await portalApi.deleteTarget(target.id);
      await onChanged();
    } catch (ex) {
      setMsg({ ok: false, text: ex instanceof Error && ex.message ? ex.message : t.tgt.removeError });
      setBusy("");
    }
  }

  const Icon = target.kind === "ip" ? ServerIcon : GlobeIcon;

  return (
    <li className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
      <div className="flex flex-wrap items-center gap-2">
        <Icon className="size-4 shrink-0 text-slate-400" aria-hidden />
        <span className="min-w-0 flex-1 truncate font-mono text-sm font-medium text-slate-900">{target.value}</span>
        <TargetStatusBadge status={target.status} />
        {admin ? (
          <button
            type="button"
            onClick={remove}
            disabled={busy !== ""}
            aria-label={t.tgt.remove(target.value)}
            title={t.tgt.remove(target.value)}
            className={`rounded-lg p-1.5 text-slate-400 hover:bg-rose-50 hover:text-rose-600 disabled:opacity-50 ${focusRing}`}
          >
            <Trash2Icon className="size-4" aria-hidden />
          </button>
        ) : null}
      </div>

      {target.status === "verified" && target.method && target.expires_at ? (
        <p className="mt-1.5 text-xs text-emerald-700">
          {t.tgt.verifiedUntil(t.tgt.method[target.method] ?? target.method, fmtDate(target.expires_at))}
        </p>
      ) : null}
      {target.status === "expired" ? <p className="mt-1.5 text-xs text-slate-600">{t.tgt.expiredHint}</p> : null}
      {target.status === "rejected" && target.note ? (
        <p className="mt-1.5 text-xs text-rose-700">{t.tgt.rejected(target.note)}</p>
      ) : null}
      {actionable && target.last_error && !msg ? (
        <p className="mt-1.5 text-xs text-slate-500">{t.tgt.lastError(localizeServerError(target.last_error))}</p>
      ) : null}
      {msg ? (
        <p role="status" className={`mt-1.5 text-xs ${msg.ok ? "text-emerald-700" : "text-rose-600"}`}>
          {msg.text}
        </p>
      ) : null}

      {actionable && admin && target.token ? (
        <div className="mt-3">
          <button
            type="button"
            aria-expanded={open}
            onClick={() => setOpen((v) => !v)}
            className={`rounded text-xs font-medium text-cyan-700 hover:text-cyan-800 ${focusRing}`}
          >
            {t.tgt.howTo}
          </button>
          {open ? (
            <div className="mt-2 space-y-3 rounded-xl border border-slate-200 bg-slate-50 p-3 text-xs text-slate-600">
              {target.kind === "domain" ? (
                <>
                  <div>
                    <p>{t.tgt.dnsStep(`${info.dns_prefix}${target.value}`)}</p>
                    <CopyValue value={target.token} />
                  </div>
                  <div>
                    <p>{t.tgt.httpStep(proofURL)}</p>
                    <CopyValue value={target.token} />
                  </div>
                </>
              ) : (
                <div>
                  <p>{t.tgt.ipStep(proofURL)}</p>
                  <CopyValue value={target.token} />
                </div>
              )}
              <div className="flex flex-wrap gap-2 pt-1">
                {target.kind === "domain" ? (
                  <button type="button" onClick={() => check("dns")} disabled={busy !== ""} className={secondaryBtn}>
                    <RefreshCwIcon className={`size-3.5 ${busy === "dns" ? "animate-spin" : ""}`} aria-hidden />
                    {busy === "dns" ? t.tgt.checking : t.tgt.checkDns}
                  </button>
                ) : null}
                <button type="button" onClick={() => check("http")} disabled={busy !== ""} className={secondaryBtn}>
                  <RefreshCwIcon className={`size-3.5 ${busy === "http" ? "animate-spin" : ""}`} aria-hidden />
                  {busy === "http" ? t.tgt.checking : t.tgt.checkHttp}
                </button>
              </div>
            </div>
          ) : null}
        </div>
      ) : null}
    </li>
  );
}

// ------------------------------------------------- picker in the request form

export function TargetPicker({
  selected,
  onChange,
  onGoTargets,
}: {
  selected: number[];
  onChange: (ids: number[]) => void;
  onGoTargets: () => void;
}) {
  const { t } = usePortalI18n();
  const [targets, setTargets] = useState<PortalTarget[] | null>(null);

  useEffect(() => {
    portalApi
      .listTargets()
      .then((r) => setTargets(r.targets.filter((x) => x.status !== "rejected")))
      .catch(() => setTargets([]));
  }, []);

  function toggle(id: number) {
    onChange(selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id]);
  }

  return (
    <fieldset className="block">
      <legend className="text-xs font-medium text-slate-600">{t.req.targetsLabel}</legend>
      <p className="mt-0.5 text-[11px] text-slate-500">{t.req.targetsHint}</p>
      {targets === null ? (
        <p className="mt-2 text-xs text-slate-500">{t.tgt.loading}</p>
      ) : targets.length === 0 ? (
        <div className="mt-2 rounded-xl border border-dashed border-slate-300 bg-slate-50 p-3 text-xs text-slate-500">
          {t.req.noTargets}{" "}
          <button
            type="button"
            onClick={onGoTargets}
            className={`rounded font-semibold text-cyan-700 hover:text-cyan-800 ${focusRing}`}
          >
            {t.req.goTargets}
          </button>
        </div>
      ) : (
        <ul className="mt-2 max-h-56 space-y-1.5 overflow-y-auto pr-1">
          {targets.map((x) => (
            <li key={x.id}>
              <label className="flex cursor-pointer items-center gap-2.5 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm hover:border-cyan-300 has-[:checked]:border-cyan-400 has-[:checked]:bg-cyan-50/60">
                <input
                  type="checkbox"
                  checked={selected.includes(x.id)}
                  onChange={() => toggle(x.id)}
                  className="size-4 accent-cyan-700"
                />
                <span className="min-w-0 flex-1 truncate font-mono text-xs text-slate-800">{x.value}</span>
                <TargetStatusBadge status={x.status} />
              </label>
            </li>
          ))}
        </ul>
      )}
    </fieldset>
  );
}

// ------------------------------------------ authorization state per request

export function RequestAuthorizationBlock({
  request,
  me,
  onChanged,
}: {
  request: PortalRequest;
  me: Me;
  onChanged: () => void;
}) {
  const { t, fmtDay } = usePortalI18n();
  const [dialog, setDialog] = useState(false);
  const a = request.authorization;
  // Requests already turned into an assessment, or legacy ones without targets
  // that NEXUS handled, need no client action.
  if (!a || request.status !== "new") return null;

  const total = a.targets.length;
  const verified = a.targets.filter((x) => x.status === "verified").length;
  const windowEnded = a.roe ? a.roe.window_end < todayUTC() : false;

  return (
    <div
      className={`mt-3 rounded-xl border px-3 py-2.5 text-xs ${
        a.ready ? "border-emerald-200 bg-emerald-50/60" : "border-amber-200 bg-amber-50/60"
      }`}
    >
      <p className={`flex items-center gap-1.5 font-semibold ${a.ready ? "text-emerald-800" : "text-amber-800"}`}>
        {a.ready ? (
          <CheckCircle2Icon className="size-3.5" aria-hidden />
        ) : (
          <ShieldAlertIcon className="size-3.5" aria-hidden />
        )}
        {a.ready ? t.authz.ready : t.authz.notReady}
      </p>
      {total === 0 ? (
        <p className="mt-1 text-slate-600">{t.authz.noTargets}</p>
      ) : (
        <ul className="mt-1.5 space-y-1 text-slate-600">
          <li>
            {t.authz.targetsVerified(verified, total)}
            {verified < total ? <span className="text-slate-500"> — {t.authz.unverified}</span> : null}
          </li>
          <li>
            {a.roe
              ? t.authz.roeAccepted(a.roe.signer_name, fmtDay(a.roe.window_start), fmtDay(a.roe.window_end))
              : t.authz.roePending}
            {windowEnded ? <span className="text-rose-700"> — {t.authz.windowEnded}</span> : null}
          </li>
        </ul>
      )}
      {total > 0 && !a.roe ? (
        isAdmin(me) ? (
          <button
            type="button"
            onClick={() => setDialog(true)}
            className={`mt-2 inline-flex items-center gap-1.5 rounded-lg bg-gradient-to-r from-cyan-700 to-sky-700 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:brightness-105 ${focusRing}`}
          >
            <FileSignatureIcon className="size-3.5" aria-hidden /> {t.authz.accept}
          </button>
        ) : (
          <p className="mt-1.5 text-slate-500">{t.authz.adminOnly}</p>
        )
      ) : null}
      {dialog ? (
        <RoEDialog
          requestId={request.id}
          onClose={() => setDialog(false)}
          onAccepted={() => {
            setDialog(false);
            onChanged();
          }}
        />
      ) : null}
    </div>
  );
}

function RoEDialog({
  requestId,
  onClose,
  onAccepted,
}: {
  requestId: number;
  onClose: () => void;
  onAccepted: () => void;
}) {
  const { t, lang } = usePortalI18n();
  const panelRef = useRef<HTMLDivElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);
  const [data, setData] = useState<AuthorizationResponse | null>(null);
  const [loadErr, setLoadErr] = useState("");
  const [signerName, setSignerName] = useState("");
  const [signerTitle, setSignerTitle] = useState("");
  const [start, setStart] = useState(todayUTC());
  const [end, setEnd] = useState(todayUTC(14));
  const [contact, setContact] = useState("");
  const [exclusions, setExclusions] = useState("");
  const [agree, setAgree] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  useEffect(() => {
    portalApi
      .requestAuthorization(requestId, lang)
      .then(setData)
      .catch((ex) => setLoadErr(ex instanceof Error && ex.message ? ex.message : t.roe.loadError));
  }, [requestId, lang, t]);

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
    const nodes = panelRef.current.querySelectorAll<HTMLElement>(
      'button:not([disabled]), [href], input:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
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

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!data) return;
    setBusy(true);
    setErr("");
    try {
      await portalApi.acceptRoE(requestId, data.terms, {
        signer_name: signerName.trim(),
        signer_title: signerTitle.trim(),
        window_start: start,
        window_end: end,
        emergency_contact: contact.trim(),
        exclusions: exclusions.trim(),
      });
      onAccepted();
    } catch (ex) {
      setErr(ex instanceof Error && ex.message ? ex.message : t.roe.error);
    } finally {
      setBusy(false);
    }
  }

  const maxEnd = data ? addDays(start, data.max_window_days) : undefined;
  const complete =
    agree && signerName.trim().length >= 3 && signerTitle.trim().length >= 2 && contact.trim().length >= 5;

  return (
    <div className="fixed inset-0 z-40 grid place-items-center p-3 sm:p-6">
      <button
        type="button"
        aria-label={t.roe.close}
        tabIndex={-1}
        onClick={onClose}
        className="absolute inset-0 cursor-default bg-slate-900/40 backdrop-blur-[1px]"
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="roe-title"
        onKeyDown={onKeyDown}
        className="relative flex max-h-full w-full max-w-2xl flex-col overflow-hidden rounded-2xl bg-white shadow-2xl"
      >
        <header className="flex items-start gap-3 border-b border-slate-200 px-5 py-4">
          <div className="min-w-0 flex-1">
            <h2 id="roe-title" className="flex items-center gap-2 text-lg font-semibold text-slate-900">
              <FileSignatureIcon className="size-5 text-cyan-700" aria-hidden /> {t.roe.title}
            </h2>
            <p className="mt-0.5 text-xs text-slate-500">{t.roe.intro}</p>
          </div>
          <button
            ref={closeRef}
            type="button"
            onClick={onClose}
            aria-label={t.roe.close}
            className={`rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 hover:text-slate-800 ${focusRing}`}
          >
            <XIcon className="size-5" aria-hidden />
          </button>
        </header>

        <div className="overflow-y-auto px-5 py-4">
          {loadErr ? (
            <p role="alert" className="text-sm text-rose-600">
              {loadErr}
            </p>
          ) : null}
          {!data && !loadErr ? <p className="text-sm text-slate-500">{t.loading}</p> : null}
          {data ? (
            <form onSubmit={submit} className="space-y-4">
              <div>
                <p className="text-xs font-medium text-slate-600">{t.roe.targets}</p>
                <ul className="mt-1 flex flex-wrap gap-1.5">
                  {data.authorization.targets.map((x) => (
                    <li
                      key={x.id}
                      className="flex items-center gap-1.5 rounded-md border border-slate-200 bg-slate-50 px-2 py-1"
                    >
                      <span className="font-mono text-[11px] text-slate-800">{x.value}</span>
                      <TargetStatusBadge status={x.status} />
                    </li>
                  ))}
                </ul>
              </div>

              <ol className="list-decimal space-y-2 rounded-xl border border-slate-200 bg-slate-50 py-3 pl-8 pr-4 text-xs leading-relaxed text-slate-700">
                {data.terms.text.map((clause) => (
                  <li key={clause}>{clause}</li>
                ))}
              </ol>
              <p className="break-all text-[10px] text-slate-500">
                {t.roe.version(data.terms.version)} · {t.roe.fingerprint}: <code>{data.terms.hash}</code>
              </p>

              <div className="grid gap-3 sm:grid-cols-2">
                <Field
                  label={t.roe.signerName}
                  value={signerName}
                  onChange={(e) => setSignerName(e.target.value)}
                  maxLength={120}
                  autoComplete="name"
                  required
                />
                <Field
                  label={t.roe.signerTitle}
                  value={signerTitle}
                  onChange={(e) => setSignerTitle(e.target.value)}
                  maxLength={120}
                  autoComplete="organization-title"
                  required
                />
                <Field
                  label={t.roe.windowStart}
                  type="date"
                  value={start}
                  min={todayUTC()}
                  onChange={(e) => setStart(e.target.value)}
                  required
                />
                <Field
                  label={t.roe.windowEnd}
                  type="date"
                  value={end}
                  min={start}
                  max={maxEnd}
                  onChange={(e) => setEnd(e.target.value)}
                  required
                />
              </div>
              <p className="-mt-2 text-[11px] text-slate-500">{t.roe.windowHint(data.max_window_days)}</p>
              <Field
                label={t.roe.emergency}
                value={contact}
                onChange={(e) => setContact(e.target.value)}
                maxLength={200}
                required
              />
              <label className="block text-xs font-medium text-slate-600">
                {t.roe.exclusions}
                <textarea
                  value={exclusions}
                  onChange={(e) => setExclusions(e.target.value)}
                  rows={3}
                  maxLength={2000}
                  placeholder={t.roe.exclusionsPlaceholder}
                  className={inputCls}
                />
              </label>
              <label className="flex items-start gap-2.5 rounded-xl border border-cyan-200 bg-cyan-50/60 p-3 text-xs text-slate-700">
                <input
                  type="checkbox"
                  checked={agree}
                  onChange={(e) => setAgree(e.target.checked)}
                  className="mt-0.5 size-4 shrink-0 accent-cyan-700"
                />
                {t.roe.confirm}
              </label>
              {err ? (
                <p role="alert" className="text-sm text-rose-600">
                  {err}
                </p>
              ) : null}
              <PrimaryButton type="submit" disabled={busy || !complete}>
                {busy ? t.roe.submitting : t.roe.submit}
              </PrimaryButton>
            </form>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function addDays(yyyyMmDd: string, n: number) {
  const d = new Date(`${yyyyMmDd}T00:00:00Z`);
  if (Number.isNaN(d.getTime())) return undefined;
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}
