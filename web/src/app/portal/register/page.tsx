"use client";

import { type FormEvent, useState } from "react";

import Link from "next/link";
import { useRouter } from "next/navigation";

import { usePortalI18n } from "../_lib/i18n";
import { portalApi, setPortalToken } from "../_lib/portal";
import { authLink, Field, FormError, PortalAuthShell, PrimaryButton } from "../_lib/ui";

export default function PortalRegister() {
  const router = useRouter();
  const { t } = usePortalI18n();
  const [orgName, setOrgName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (password.length < 8) {
      setErr(t.passwordMin);
      return;
    }
    setLoading(true);
    setErr("");
    try {
      const { token } = await portalApi.register(orgName, email, password);
      setPortalToken(token);
      router.replace("/portal");
    } catch (ex) {
      setErr(ex instanceof Error && ex.message !== "unauthorized" ? ex.message : t.auth.registerError);
    } finally {
      setLoading(false);
    }
  }

  return (
    <PortalAuthShell
      title={t.auth.createOrg}
      subtitle={t.auth.registerSubtitle}
      footer={
        <>
          {t.auth.haveAccount}{" "}
          <Link href="/portal/login" className={authLink}>
            {t.auth.goSignIn}
          </Link>
        </>
      }
    >
      <form onSubmit={submit} className="space-y-4">
        <Field
          label={t.auth.orgName}
          value={orgName}
          onChange={(e) => setOrgName(e.target.value)}
          autoComplete="organization"
        />
        <Field
          label={t.email}
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          autoComplete="email"
        />
        <Field
          label={t.auth.passwordNew}
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="new-password"
        />
        {err ? <FormError>{err}</FormError> : null}
        <PrimaryButton type="submit" disabled={loading || !orgName || !email || password.length < 8}>
          {loading ? t.auth.creating : t.auth.createAccount}
        </PrimaryButton>
      </form>
    </PortalAuthShell>
  );
}
