"use client";

import { type FormEvent, useEffect, useState } from "react";

import Link from "next/link";

import { usePortalI18n } from "./i18n";
import { type Me, portalApi, type TermsInfo } from "./portal";
import { authLink, FormError, PortalAuthShell, PrimaryButton } from "./ui";

// Consent to the Terms of Service and Privacy Policy. The server records which
// version each person accepted and refuses the portal until they accept the
// current one; these components only collect that consent.

// useTermsInfo loads the current version and where to read it (public).
export function useTermsInfo(): TermsInfo | null {
  const [info, setInfo] = useState<TermsInfo | null>(null);
  useEffect(() => {
    portalApi
      .terms()
      .then(setInfo)
      .catch(() => setInfo(null));
  }, []);
  return info;
}

// TermsCheckbox is the explicit opt-in on sign-up pages; the documents open in
// a new tab so the form keeps what was typed.
export function TermsCheckbox({
  info,
  checked,
  onChange,
}: {
  info: TermsInfo | null;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  const { t } = usePortalI18n();
  return (
    <div className="rounded-xl border border-slate-200 bg-slate-50 p-3.5">
      <label className="flex items-start gap-2.5 text-slate-700 text-sm">
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          className="mt-0.5 size-4 shrink-0 accent-cyan-700"
          required
        />
        <span>
          {t.terms.agreePrefix}{" "}
          <Link href={info?.terms ?? "/legal/terms"} target="_blank" rel="noopener" className={authLink}>
            {t.terms.termsLink}
          </Link>{" "}
          {t.terms.agreeJoin}{" "}
          <Link href={info?.privacy ?? "/legal/privacy"} target="_blank" rel="noopener" className={authLink}>
            {t.terms.privacyLink}
          </Link>
          .
        </span>
      </label>
      {info?.draft ? <p className="mt-2 pl-6 text-slate-500 text-xs">{t.terms.draftNote(info.version)}</p> : null}
    </div>
  );
}

// TermsGate replaces the portal for someone who has not accepted the current
// version: their first sign-in through SSO, an account older than the terms, or
// a new version.
export function TermsGate({ me, onAccepted, onLogout }: { me: Me; onAccepted: () => void; onLogout: () => void }) {
  const { t } = usePortalI18n();
  const info = me.terms ?? null;
  const updated = Boolean(me.user.terms_version);
  const [agreed, setAgreed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  async function accept(e: FormEvent) {
    e.preventDefault();
    if (!info) return;
    setBusy(true);
    setErr("");
    try {
      await portalApi.acceptTerms(info.version);
      onAccepted();
    } catch (ex) {
      setErr(ex instanceof Error && ex.message !== "unauthorized" ? ex.message : t.terms.acceptError);
      setBusy(false);
    }
  }

  return (
    <PortalAuthShell
      title={updated ? t.terms.updatedTitle : t.terms.title}
      subtitle={updated ? t.terms.updatedIntro : t.terms.intro(me.org.name)}
      footer={
        <button type="button" onClick={onLogout} className={authLink}>
          {t.logout}
        </button>
      }
    >
      <form onSubmit={accept} className="space-y-4">
        <p className="text-slate-600 text-sm">{t.terms.personal}</p>
        <TermsCheckbox info={info} checked={agreed} onChange={setAgreed} />
        {err ? <FormError>{err}</FormError> : null}
        <PrimaryButton type="submit" disabled={busy || !agreed || !info}>
          {busy ? t.terms.accepting : t.terms.accept}
        </PrimaryButton>
      </form>
    </PortalAuthShell>
  );
}
