"use client";

import { Suspense, useEffect, useRef, useState } from "react";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";

import { usePortalI18n } from "../../_lib/i18n";
import { portalApi, setPortalToken } from "../../_lib/portal";
import { authLink, FormError, PortalAuthShell, PortalLoading } from "../../_lib/ui";

// The identity provider sends the browser back here with ?code&state (or
// ?error). The code is exchanged server-side; the state is single-use, so the
// exchange runs exactly once even under React's development double-mount.
function CallbackInner() {
  const { t } = usePortalI18n();
  const router = useRouter();
  const params = useSearchParams();
  const [err, setErr] = useState("");
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    const code = params.get("code") ?? "";
    const state = params.get("state") ?? "";
    if (params.get("error") || !code || !state) {
      setErr(params.get("error") === "access_denied" ? t.sso.cancelled : t.sso.failed);
      return;
    }
    portalApi
      .ssoCallback(code, state)
      .then(({ token }) => {
        setPortalToken(token);
        router.replace("/portal");
      })
      .catch((ex: unknown) => {
        const msg = ex instanceof Error ? ex.message : "";
        setErr(msg && msg !== "unauthorized" ? msg : t.sso.failed);
      });
  }, [params, router, t]);

  return (
    <PortalAuthShell
      title={t.sso.title}
      subtitle={err ? t.sso.failedTitle : t.sso.checking}
      footer={
        <Link href="/portal/login" className={authLink}>
          {t.auth.backToSignIn}
        </Link>
      }
    >
      {err ? <FormError>{err}</FormError> : <p className="text-sm text-slate-500">{t.auth.oneMoment}</p>}
    </PortalAuthShell>
  );
}

export default function PortalSSOCallback() {
  return (
    <Suspense fallback={<PortalLoading />}>
      <CallbackInner />
    </Suspense>
  );
}
