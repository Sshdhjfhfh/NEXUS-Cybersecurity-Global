"use client";

import { type FormEvent, Suspense, useState } from "react";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";

import { usePortalI18n } from "../_lib/i18n";
import { portalApi } from "../_lib/portal";
import { authLink, Field, FormError, PortalAuthShell, PortalLoading, PrimaryButton } from "../_lib/ui";

function ResetInner() {
  const router = useRouter();
  const { t } = usePortalI18n();
  const token = useSearchParams().get("token") ?? "";
  const [password, setPassword] = useState("");
  const [done, setDone] = useState(false);
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
      await portalApi.resetPassword(token, password);
      setDone(true);
      setTimeout(() => router.replace("/portal/login"), 1800);
    } catch (ex) {
      setErr(ex instanceof Error && ex.message ? ex.message : t.auth.resetError);
    } finally {
      setLoading(false);
    }
  }

  if (!token) {
    return (
      <PortalAuthShell
        title={t.auth.resetInvalidTitle}
        subtitle={t.auth.resetMissing}
        footer={
          <Link href="/portal/forgot" className={authLink}>
            {t.auth.requestNew}
          </Link>
        }
      >
        <p className="text-sm text-slate-500">{t.auth.resetAgain}</p>
      </PortalAuthShell>
    );
  }

  return (
    <PortalAuthShell
      title={t.auth.resetTitle}
      footer={
        <Link href="/portal/login" className={authLink}>
          {t.auth.goSignIn}
        </Link>
      }
    >
      {done ? (
        <p role="status" className="text-sm text-emerald-700">
          {t.auth.resetDone}
        </p>
      ) : (
        <form onSubmit={submit} className="space-y-4">
          <Field
            label={t.sec.newPassword}
            hint={t.passwordRules}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="new-password"
          />
          {err ? <FormError>{err}</FormError> : null}
          <PrimaryButton type="submit" disabled={loading || password.length < 8}>
            {loading ? t.auth.saving : t.auth.savePassword}
          </PrimaryButton>
        </form>
      )}
    </PortalAuthShell>
  );
}

export default function PortalReset() {
  return (
    <Suspense fallback={<PortalLoading />}>
      <ResetInner />
    </Suspense>
  );
}
