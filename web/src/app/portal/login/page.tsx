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
        {err ? <FormError>{err}</FormError> : null}
        <PrimaryButton type="submit" disabled={loading || !email || !password || (needMfa && code.trim().length < 6)}>
          {loading ? t.auth.signingIn : needMfa ? t.auth.verify : t.auth.signIn}
        </PrimaryButton>
      </form>
    </PortalAuthShell>
  );
}
