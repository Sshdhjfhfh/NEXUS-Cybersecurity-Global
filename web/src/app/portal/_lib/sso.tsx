"use client";

import { useCallback, useEffect, useState } from "react";

import { KeyRoundIcon } from "lucide-react";

import { usePortalI18n } from "./i18n";
import { type Me, portalApi, type SSOInput, type SSOResponse } from "./portal";
import { CopyValue } from "./targets";
import { Field, PrimaryButton } from "./ui";

const errText = (ex: unknown, fallback: string) => (ex instanceof Error && ex.message ? ex.message : fallback);

const EMPTY: SSOInput = {
  issuer: "",
  client_id: "",
  client_secret: "",
  email_domains: [],
  enabled: false,
  enforce: false,
  auto_provision: false,
  default_role: "member",
};

function Switch({
  id,
  checked,
  disabled,
  onChange,
  label,
  hint,
}: {
  id: string;
  checked: boolean;
  disabled: boolean;
  onChange: (v: boolean) => void;
  label: string;
  hint: string;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div>
        <p id={`${id}-l`} className="text-sm font-medium text-slate-800">
          {label}
        </p>
        <p id={`${id}-h`} className="mt-1 text-xs text-slate-500">
          {hint}
        </p>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-labelledby={`${id}-l`}
        aria-describedby={`${id}-h`}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={`relative mt-0.5 inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400 focus-visible:ring-offset-2 disabled:opacity-60 ${checked ? "bg-cyan-700" : "bg-slate-300"}`}
      >
        <span
          className={`inline-block size-5 rounded-full bg-white shadow transition-transform ${checked ? "translate-x-5" : "translate-x-0.5"}`}
        />
      </button>
    </div>
  );
}

// Single sign-on (OpenID Connect) with the organization's identity provider.
// The owner configures it; admins and auditors see it read-only. Only domains
// whose ownership was proven by DNS (Targets tab) can be used.
export function SSOPanel({ me }: { me: Me }) {
  const { t, fmtDateTime } = usePortalI18n();
  const canEdit = me.user.role === "owner";
  const [data, setData] = useState<SSOResponse | null>(null);
  const [form, setForm] = useState<SSOInput>(EMPTY);
  const [loadErr, setLoadErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState("");
  const [err, setErr] = useState("");
  const [confirmDelete, setConfirmDelete] = useState(false);

  const apply = useCallback((d: SSOResponse) => {
    setData(d);
    const c = d.sso;
    setForm(
      c
        ? {
            issuer: c.issuer,
            client_id: c.client_id,
            client_secret: "",
            email_domains: c.email_domains,
            enabled: c.enabled,
            enforce: c.enforce,
            auto_provision: c.auto_provision,
            default_role: c.default_role,
          }
        : EMPTY,
    );
  }, []);

  const load = useCallback(async () => {
    setLoadErr("");
    try {
      apply(await portalApi.getSSO());
    } catch (ex) {
      setLoadErr(errText(ex, t.sso.loadError));
    }
  }, [apply, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const set = <K extends keyof SSOInput>(k: K, v: SSOInput[K]) => setForm((f) => ({ ...f, [k]: v }));

  async function save() {
    setBusy(true);
    setMsg("");
    setErr("");
    try {
      apply(await portalApi.setSSO({ ...form, enforce: form.enabled && form.enforce }));
      setMsg(t.sso.saved);
    } catch (ex) {
      setErr(errText(ex, t.sso.saveError));
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    setBusy(true);
    setErr("");
    try {
      await portalApi.deleteSSO();
      setConfirmDelete(false);
      await load();
      setMsg(t.sso.removed);
    } catch (ex) {
      setErr(errText(ex, t.sso.saveError));
    } finally {
      setBusy(false);
    }
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
          <p className="text-sm text-slate-500">{t.sso.loading}</p>
        )}
      </section>
    );
  }

  const domains = Array.from(new Set([...data.verified_domains, ...form.email_domains])).sort();
  const ro = !canEdit || busy;

  return (
    <section className={box} aria-labelledby="sso-title">
      <h2
        id="sso-title"
        className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-cyan-700"
      >
        <KeyRoundIcon className="size-4" aria-hidden /> {t.sso.panelTitle}
      </h2>
      <p className="mt-2 text-sm text-slate-600">{t.sso.intro}</p>
      {!canEdit ? <p className="mt-2 text-xs text-violet-700">{t.sso.readOnly}</p> : null}
      <p className="mt-3 text-xs">
        <span
          className={`rounded-full px-2 py-0.5 font-semibold ${data.sso?.enabled ? "bg-emerald-50 text-emerald-700" : "bg-slate-100 text-slate-600"}`}
        >
          {data.sso?.enabled ? (data.sso.enforce ? t.sso.statusEnforced : t.sso.statusOn) : t.sso.statusOff}
        </span>
      </p>

      <div className="mt-5 space-y-1">
        <p className="text-xs font-medium text-slate-600">{t.sso.redirectUri}</p>
        <div className="flex items-center gap-2">
          <code className="min-w-0 flex-1 truncate rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-xs text-slate-700">
            {data.redirect_uri}
          </code>
          <CopyValue value={data.redirect_uri} />
        </div>
        <p className="text-xs text-slate-500">{t.sso.redirectHint}</p>
      </div>

      <div className="mt-5 space-y-4">
        <Field
          label={t.sso.issuer}
          placeholder="https://login.microsoftonline.com/<tenant>/v2.0"
          value={form.issuer}
          readOnly={!canEdit}
          onChange={(e) => set("issuer", e.target.value)}
          spellCheck={false}
        />
        <Field
          label={t.sso.clientId}
          value={form.client_id}
          readOnly={!canEdit}
          onChange={(e) => set("client_id", e.target.value)}
          spellCheck={false}
        />
        {canEdit ? (
          <Field
            label={t.sso.clientSecret}
            type="password"
            autoComplete="off"
            placeholder={data.sso?.has_secret ? t.sso.secretKept : ""}
            value={form.client_secret}
            onChange={(e) => set("client_secret", e.target.value)}
          />
        ) : null}
      </div>

      <fieldset className="mt-5">
        <legend className="text-sm font-medium text-slate-800">{t.sso.domains}</legend>
        <p className="mt-1 text-xs text-slate-500">{t.sso.domainsHint}</p>
        {domains.length === 0 ? (
          <p className="mt-2 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-800">{t.sso.noVerifiedDomains}</p>
        ) : (
          <div className="mt-2 flex flex-wrap gap-2">
            {domains.map((d) => {
              const on = form.email_domains.includes(d);
              return (
                <label
                  key={d}
                  className={`inline-flex items-center gap-2 rounded-lg border px-3 py-1.5 text-sm ${on ? "border-cyan-500 bg-cyan-50 text-cyan-900" : "border-slate-200 text-slate-600"}`}
                >
                  <input
                    type="checkbox"
                    checked={on}
                    disabled={ro}
                    onChange={() =>
                      set("email_domains", on ? form.email_domains.filter((x) => x !== d) : [...form.email_domains, d])
                    }
                  />
                  @{d}
                </label>
              );
            })}
          </div>
        )}
      </fieldset>

      <div className="mt-6 space-y-5">
        <Switch
          id="sso-enabled"
          checked={form.enabled}
          disabled={ro}
          onChange={(v) => set("enabled", v)}
          label={t.sso.enabled}
          hint={t.sso.enabledHint}
        />
        <Switch
          id="sso-enforce"
          checked={form.enabled && form.enforce}
          disabled={ro || !form.enabled}
          onChange={(v) => set("enforce", v)}
          label={t.sso.enforce}
          hint={t.sso.enforceHint}
        />
        <Switch
          id="sso-jit"
          checked={form.auto_provision}
          disabled={ro}
          onChange={(v) => set("auto_provision", v)}
          label={t.sso.autoProvision}
          hint={t.sso.autoProvisionHint}
        />
        {form.auto_provision ? (
          <label className="block text-xs font-medium text-slate-600">
            {t.sso.defaultRole}
            <select
              value={form.default_role}
              disabled={ro}
              onChange={(e) => set("default_role", e.target.value as SSOInput["default_role"])}
              className="mt-1 block rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-800 disabled:bg-slate-50"
            >
              <option value="member">{t.role.member}</option>
              <option value="auditor">{t.role.auditor}</option>
            </select>
          </label>
        ) : null}
      </div>

      {data.sso?.updated_at && data.sso.updated_by ? (
        <p className="mt-5 text-xs text-slate-500">
          {t.policy.updated(data.sso.updated_by, fmtDateTime(data.sso.updated_at))}
        </p>
      ) : null}

      {canEdit ? (
        <div className="mt-5 space-y-3">
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
            {busy ? t.sso.saving : t.sso.save}
          </PrimaryButton>
          {data.sso ? (
            confirmDelete ? (
              <div className="flex flex-wrap items-center gap-2 text-sm">
                <span className="text-slate-600">{t.sso.confirmRemove}</span>
                <button
                  type="button"
                  onClick={() => void remove()}
                  disabled={busy}
                  className="rounded-lg bg-rose-600 px-3 py-1.5 font-semibold text-white hover:bg-rose-700 disabled:opacity-60"
                >
                  {t.sso.remove}
                </button>
                <button
                  type="button"
                  onClick={() => setConfirmDelete(false)}
                  className="rounded-lg border border-slate-200 px-3 py-1.5 text-slate-600"
                >
                  {t.cancel}
                </button>
              </div>
            ) : (
              <button
                type="button"
                onClick={() => setConfirmDelete(true)}
                className="text-sm font-semibold text-rose-600 hover:underline"
              >
                {t.sso.remove}
              </button>
            )
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
