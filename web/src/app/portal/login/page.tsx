"use client";

import { type FormEvent, useState } from "react";

import Link from "next/link";
import { useRouter } from "next/navigation";

import { portalApi, setPortalToken } from "../_lib/portal";
import { Field, PortalAuthShell, PrimaryButton } from "../_lib/ui";

export default function PortalLogin() {
  const router = useRouter();
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
        setErr(code ? "Código inválido. Inténtalo de nuevo." : "");
      } else {
        setErr(msg && msg !== "unauthorized" ? msg : "Email o contraseña incorrectos");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <PortalAuthShell
      title="Portal de clientes"
      subtitle="Inicia sesión en tu organización."
      footer={
        <>
          <Link href="/portal/forgot" className="text-cyan-600 hover:text-cyan-700">
            ¿Olvidaste tu contraseña?
          </Link>
          <span className="mt-2 block">
            ¿No tienes cuenta?{" "}
            <Link href="/portal/register" className="text-cyan-600 hover:text-cyan-700">
              Crear organización
            </Link>
          </span>
        </>
      }
    >
      <form onSubmit={submit} className="space-y-4">
        <Field
          label="Email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          autoComplete="email"
        />
        <Field
          label="Contraseña"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
        />
        {needMfa ? (
          <Field
            label="Código de verificación (2FA)"
            inputMode="numeric"
            autoComplete="one-time-code"
            placeholder="123456"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            // biome-ignore lint/a11y/noAutofocus: focus the code field as soon as it appears
            autoFocus
          />
        ) : null}
        {err ? <p className="text-sm text-rose-500">{err}</p> : null}
        <PrimaryButton type="submit" disabled={loading || !email || !password || (needMfa && code.trim().length < 6)}>
          {loading ? "Entrando…" : needMfa ? "Verificar" : "Entrar"}
        </PrimaryButton>
      </form>
    </PortalAuthShell>
  );
}
