"use client";

import { type FormEvent, useState } from "react";

import Link from "next/link";

import { usePortalI18n } from "../_lib/i18n";
import { portalApi } from "../_lib/portal";
import { authLink, Field, PortalAuthShell, PrimaryButton } from "../_lib/ui";

export default function PortalForgot() {
  const { t } = usePortalI18n();
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const [loading, setLoading] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    try {
      await portalApi.forgotPassword(email.trim().toLowerCase());
    } catch {
      /* intentionally ignored: never reveal whether the email exists */
    } finally {
      setSent(true);
      setLoading(false);
    }
  }

  return (
    <PortalAuthShell
      title={t.auth.forgotTitle}
      subtitle={t.auth.forgotSubtitle}
      footer={
        <Link href="/portal/login" className={authLink}>
          {t.auth.backToSignIn}
        </Link>
      }
    >
      {sent ? (
        <p role="status" className="text-sm text-slate-600">
          {t.auth.forgotSent}
        </p>
      ) : (
        <form onSubmit={submit} className="space-y-4">
          <Field
            label={t.email}
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
          />
          <PrimaryButton type="submit" disabled={loading || !email}>
            {loading ? t.auth.sending : t.auth.sendLink}
          </PrimaryButton>
        </form>
      )}
    </PortalAuthShell>
  );
}
