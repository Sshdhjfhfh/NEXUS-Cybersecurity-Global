"use client";

import { type FormEvent, Suspense, useEffect, useState } from "react";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";

import { usePortalI18n } from "../_lib/i18n";
import { portalApi, setPortalToken } from "../_lib/portal";
import { authLink, Field, FormError, PortalAuthShell, PortalLoading, PrimaryButton } from "../_lib/ui";

function AcceptInner() {
  const router = useRouter();
  const { t } = usePortalI18n();
  const params = useSearchParams();
  const token = params.get("token") ?? "";

  const [info, setInfo] = useState<{ email: string; org: string; role: string } | null>(null);
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [fatal, setFatal] = useState<"" | "missing" | "invalid">("");
  const [loading, setLoading] = useState(false);
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    if (!token) {
      setFatal("missing");
      setChecking(false);
      return;
    }
    portalApi
      .lookupInvite(token)
      .then((i) => setInfo({ email: i.email, org: i.org, role: i.role }))
      .catch(() => setFatal("invalid"))
      .finally(() => setChecking(false));
  }, [token]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (password.length < 8) {
      setErr(t.passwordMin);
      return;
    }
    setLoading(true);
    setErr("");
    try {
      const { token: jwt } = await portalApi.acceptInvite(token, password);
      setPortalToken(jwt);
      router.replace("/portal");
    } catch (ex) {
      setErr(ex instanceof Error && ex.message !== "unauthorized" ? ex.message : t.auth.joinError);
    } finally {
      setLoading(false);
    }
  }

  if (checking) {
    return (
      <PortalAuthShell title={t.auth.inviteTitle} subtitle={t.auth.verifyChecking}>
        <p className="text-sm text-slate-500">{t.auth.oneMoment}</p>
      </PortalAuthShell>
    );
  }

  if (fatal) {
    return (
      <PortalAuthShell
        title={t.auth.inviteInvalidTitle}
        subtitle={fatal === "missing" ? t.auth.inviteMissing : t.auth.inviteInvalid}
        footer={
          <Link href="/portal/login" className={authLink}>
            {t.auth.goSignIn}
          </Link>
        }
      >
        <p className="text-sm text-slate-500">{t.auth.inviteAskNew}</p>
      </PortalAuthShell>
    );
  }

  return (
    <PortalAuthShell
      title={t.auth.joinTitle}
      subtitle={info ? `${info.org} · ${info.email}` : undefined}
      footer={
        <Link href="/portal/login" className={authLink}>
          {t.auth.goSignIn}
        </Link>
      }
    >
      <form onSubmit={submit} className="space-y-4">
        <Field label={t.email} type="email" value={info?.email ?? ""} readOnly disabled />
        <Field
          label={t.auth.createPassword}
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="new-password"
        />
        {err ? <FormError>{err}</FormError> : null}
        <PrimaryButton type="submit" disabled={loading || password.length < 8}>
          {loading ? t.auth.joining : t.auth.join}
        </PrimaryButton>
      </form>
    </PortalAuthShell>
  );
}

export default function PortalAccept() {
  return (
    <Suspense fallback={<PortalLoading />}>
      <AcceptInner />
    </Suspense>
  );
}
