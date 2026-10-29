"use client";

import { Suspense, useEffect, useState } from "react";

import Link from "next/link";
import { useSearchParams } from "next/navigation";

import { usePortalI18n } from "../_lib/i18n";
import { portalApi } from "../_lib/portal";
import { authLink, PortalAuthShell, PortalLoading } from "../_lib/ui";

function VerifyInner() {
  const { t } = usePortalI18n();
  const token = useSearchParams().get("token") ?? "";
  const [state, setState] = useState<"checking" | "ok" | "error">("checking");

  useEffect(() => {
    if (!token) {
      setState("error");
      return;
    }
    portalApi
      .verifyEmail(token)
      .then(() => setState("ok"))
      .catch(() => setState("error"));
  }, [token]);

  const subtitle = state === "checking" ? t.auth.verifyChecking : state === "ok" ? t.auth.verifyOk : t.auth.verifyBad;

  return (
    <PortalAuthShell
      title={t.auth.verifyTitle}
      subtitle={subtitle}
      footer={
        <Link href="/portal/login" className={authLink}>
          {t.auth.goSignIn}
        </Link>
      }
    >
      {state === "ok" ? (
        <p role="status" className="text-sm text-emerald-700">
          ✓ {t.auth.verifyThanks}
        </p>
      ) : state === "error" ? (
        <p className="text-sm text-slate-500">{t.auth.verifyResendHint}</p>
      ) : (
        <p className="text-sm text-slate-500">{t.auth.oneMoment}</p>
      )}
    </PortalAuthShell>
  );
}

export default function PortalVerify() {
  return (
    <Suspense fallback={<PortalLoading />}>
      <VerifyInner />
    </Suspense>
  );
}
