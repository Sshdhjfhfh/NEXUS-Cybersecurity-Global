"use client";

import { type FormEvent, useState } from "react";

import Link from "next/link";
import { useRouter } from "next/navigation";

import { usePortalI18n } from "../_lib/i18n";
import { portalApi, setPortalToken } from "../_lib/portal";
import { authLink, Field, FormError, PortalAuthShell, PrimaryButton } from "../_lib/ui";

export default function PortalLogin() {
  const router = useRouter();
  const { t } = usePortalI18n();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [needMfa, setNeedMfa] = useState(false);
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);
  const [ssoBusy, setSsoBusy] = useState(false);
  const [ssoRequired, setSsoRequired] = useState(false);

  // Single sign-on: the server finds the org that owns the email's domain and
  // returns its identity provider's sign-in URL.
  async function startSSO() {
    if (!email.trim()) {
      setErr(t.sso.emailFirst);
      return;
    }
    setSsoBusy(true);
    setErr("");
    try {
      const { url } = await portalApi.ssoStart(email.trim());
      window.location.assign(url);
    } catch (ex) {
      setErr(ex instanceof Error && ex.message ? ex.message : t.sso.startError);
      setSsoBusy(false);
    }
  }

  async function submit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setErr("");
    try {
      const { token } = await portalApi.login(email, password, needMfa ? code.trim() : undefined);
      setPortalToken(token);
      router.replace("/portal");
    } catch (ex) {
      const msg = ex instanceof Error ? ex.message : "";
      if (msg === "mfa_required") {
        // The account has 2FA: reveal the code field and ask for it.
        setNeedMfa(true);
        setErr(code ? t.auth.mfaInvalid : "");
      } else if (msg === "sso_required") {
        // The org signs its members in through its identity provider.
        setSsoRequired(true);
        setErr("");
      } else {
        setErr(msg && msg !== "unauthorized" ? msg : t.auth.badCredentials);
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <PortalAuthShell
      title={t.auth.portalTitle}
      subtitle={t.auth.loginSubtitle}
      footer={
        <>
          <Link href="/portal/forgot" className={authLink}>
            {t.auth.forgot}
          </Link>
          <span className="mt-2 block">
            {t.auth.noAccount}{" "}
            <Link href="/portal/register" className={authLink}>
              {t.auth.createOrg}
            </Link>
          </span>
        </>
      }
    >
      <form onSubmit={submit} className="space-y-4">
        <Field
          label={t.email}
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          autoComplete="email"
        />
        <Field
          label={t.auth.password}
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
        />
        {needMfa ? (
          <Field
            label={t.auth.mfaCode}
            inputMode="numeric"
            autoComplete="one-time-code"
            placeholder="123456"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            // biome-ignore lint/a11y/noAutofocus: focus the code field as soon as it appears
            autoFocus
          />
        ) : null}
        {ssoRequired ? (
          <p role="status" className="rounded-lg border border-cyan-200 bg-cyan-50 px-3 py-2 text-sm text-cyan-900">
            {t.sso.requiredByOrg}
          </p>
        ) : null}
        {err ? <FormError>{err}</FormError> : null}
        <PrimaryButton type="submit" disabled={loading || !email || !password || (needMfa && code.trim().length < 6)}>
          {loading ? t.auth.signingIn : needMfa ? t.auth.verify : t.auth.signIn}
        </PrimaryButton>
      </form>
      <div className="my-4 flex items-center gap-3 text-xs text-slate-400" aria-hidden>
        <span className="h-px flex-1 bg-slate-200" /> {t.sso.or} <span className="h-px flex-1 bg-slate-200" />
      </div>
      <button
        type="button"
        onClick={() => void startSSO()}
        disabled={ssoBusy}
        className={`w-full rounded-xl border px-4 py-2.5 text-sm font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400 disabled:opacity-60 ${
          ssoRequired
            ? "border-cyan-600 bg-cyan-700 text-white hover:bg-cyan-800"
            : "border-slate-300 text-slate-700 hover:border-cyan-400 hover:text-cyan-800"
        }`}
      >
        {ssoBusy ? t.sso.redirecting : t.sso.continue}
      </button>
      <p className="mt-2 text-center text-xs text-slate-500">{t.sso.hint}</p>
    </PortalAuthShell>
  );
}
