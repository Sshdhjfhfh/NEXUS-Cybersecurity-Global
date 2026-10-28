"use client";

import { type FormEvent, useState } from "react";

import Link from "next/link";
import { useRouter } from "next/navigation";

import { portalApi, setPortalToken } from "../_lib/portal";
import { Field, PortalAuthShell, PrimaryButton } from "../_lib/ui";

export default function PortalRegister() {
  const router = useRouter();
  const [orgName, setOrgName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (password.length < 8) {
      setErr("La contraseña debe tener al menos 8 caracteres.");
      return;
    }
    setLoading(true);
    setErr("");
    try {
      const { token } = await portalApi.register(orgName, email, password);
      setPortalToken(token);
      router.replace("/portal");
    } catch (ex) {
      setErr(ex instanceof Error && ex.message !== "unauthorized" ? ex.message : "No se pudo crear la cuenta");
    } finally {
      setLoading(false);
    }
  }

  return (
    <PortalAuthShell
      title="Crear organización"
      subtitle="Registra tu organización y su primer usuario."
      footer={
        <>
          ¿Ya tienes cuenta?{" "}
          <Link href="/portal/login" className="text-cyan-300 hover:text-cyan-200">
            Iniciar sesión
          </Link>
        </>
      }
    >
      <form onSubmit={submit} className="space-y-4">
        <Field label="Nombre de la organización" value={orgName} onChange={(e) => setOrgName(e.target.value)} />
        <Field
          label="Email"
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          autoComplete="email"
        />
        <Field
          label="Contraseña (mín. 8)"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="new-password"
        />
        {err ? <p className="text-sm text-rose-400">{err}</p> : null}
        <PrimaryButton type="submit" disabled={loading || !orgName || !email || password.length < 8}>
          {loading ? "Creando…" : "Crear cuenta"}
        </PrimaryButton>
      </form>
    </PortalAuthShell>
  );
}
