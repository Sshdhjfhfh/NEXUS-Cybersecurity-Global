"use client";

import { type FormEvent, useState } from "react";

import Link from "next/link";

import { portalApi } from "../_lib/portal";
import { Field, PortalAuthShell, PrimaryButton } from "../_lib/ui";

export default function PortalForgot() {
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
      title="Restablecer contraseña"
      subtitle="Te enviaremos un enlace para crear una nueva."
      footer={
        <Link href="/portal/login" className="text-cyan-300 hover:text-cyan-200">
          Volver a iniciar sesión
        </Link>
      }
    >
      {sent ? (
        <p className="text-sm text-slate-300">
          Si existe una cuenta con ese email, te hemos enviado un enlace para restablecer la contraseña. Revisa tu
          bandeja de entrada.
        </p>
      ) : (
        <form onSubmit={submit} className="space-y-4">
          <Field
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
          />
          <PrimaryButton type="submit" disabled={loading || !email}>
            {loading ? "Enviando…" : "Enviar enlace"}
          </PrimaryButton>
        </form>
      )}
    </PortalAuthShell>
  );
}
