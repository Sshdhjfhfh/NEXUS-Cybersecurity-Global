"use client";

import { type FormEvent, Suspense, useState } from "react";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";

import { portalApi } from "../_lib/portal";
import { Field, PortalAuthShell, PrimaryButton } from "../_lib/ui";

function ResetInner() {
  const router = useRouter();
  const token = useSearchParams().get("token") ?? "";
  const [password, setPassword] = useState("");
  const [done, setDone] = useState(false);
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
      await portalApi.resetPassword(token, password);
      setDone(true);
      setTimeout(() => router.replace("/portal/login"), 1800);
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "No se pudo restablecer la contraseña");
    } finally {
      setLoading(false);
    }
  }

  if (!token) {
    return (
      <PortalAuthShell
        title="Enlace inválido"
        subtitle="Falta el token de restablecimiento."
        footer={
          <Link href="/portal/forgot" className="text-cyan-300 hover:text-cyan-200">
            Solicitar uno nuevo
          </Link>
        }
      >
        <p className="text-sm text-slate-400">Vuelve a solicitar el enlace de restablecimiento.</p>
      </PortalAuthShell>
    );
  }

  return (
    <PortalAuthShell
      title="Nueva contraseña"
      footer={
        <Link href="/portal/login" className="text-cyan-300 hover:text-cyan-200">
          Iniciar sesión
        </Link>
      }
    >
      {done ? (
        <p className="text-sm text-emerald-300">Contraseña actualizada. Redirigiendo al inicio de sesión…</p>
      ) : (
        <form onSubmit={submit} className="space-y-4">
          <Field
            label="Nueva contraseña (mín. 8)"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="new-password"
          />
          {err ? <p className="text-sm text-rose-400">{err}</p> : null}
          <PrimaryButton type="submit" disabled={loading || password.length < 8}>
            {loading ? "Guardando…" : "Guardar contraseña"}
          </PrimaryButton>
        </form>
      )}
    </PortalAuthShell>
  );
}

export default function PortalReset() {
  return (
    <Suspense
      fallback={
        <div className="grid min-h-screen place-items-center bg-[#060b18] text-sm text-slate-400">Cargando…</div>
      }
    >
      <ResetInner />
    </Suspense>
  );
}
