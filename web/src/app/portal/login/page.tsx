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
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setErr("");
    try {
      const { token } = await portalApi.login(email, password);
      setPortalToken(token);
      router.replace("/portal");
    } catch (ex) {
      setErr(ex instanceof Error && ex.message !== "unauthorized" ? ex.message : "Email o contraseña incorrectos");
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
          ¿No tienes cuenta?{" "}
          <Link href="/portal/register" className="text-cyan-300 hover:text-cyan-200">
            Crear organización
          </Link>
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
        {err ? <p className="text-sm text-rose-400">{err}</p> : null}
        <PrimaryButton type="submit" disabled={loading || !email || !password}>
          {loading ? "Entrando…" : "Entrar"}
        </PrimaryButton>
      </form>
    </PortalAuthShell>
  );
}
