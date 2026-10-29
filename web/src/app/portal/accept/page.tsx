"use client";

import { type FormEvent, Suspense, useEffect, useState } from "react";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";

import { portalApi, setPortalToken } from "../_lib/portal";
import { Field, PortalAuthShell, PrimaryButton } from "../_lib/ui";

function AcceptInner() {
  const router = useRouter();
  const params = useSearchParams();
  const token = params.get("token") ?? "";

  const [info, setInfo] = useState<{ email: string; org: string; role: string } | null>(null);
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [fatal, setFatal] = useState("");
  const [loading, setLoading] = useState(false);
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    if (!token) {
      setFatal("Falta el token de invitación.");
      setChecking(false);
      return;
    }
    portalApi
      .lookupInvite(token)
      .then((i) => setInfo({ email: i.email, org: i.org, role: i.role }))
      .catch(() => setFatal("Esta invitación no es válida o ya fue utilizada."))
      .finally(() => setChecking(false));
  }, [token]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (password.length < 8) {
      setErr("La contraseña debe tener al menos 8 caracteres.");
      return;
    }
    setLoading(true);
    setErr("");
    try {
      const { token: jwt } = await portalApi.acceptInvite(token, password);
      setPortalToken(jwt);
      router.replace("/portal");
    } catch (ex) {
      setErr(ex instanceof Error && ex.message !== "unauthorized" ? ex.message : "No se pudo aceptar la invitación");
    } finally {
      setLoading(false);
    }
  }

  if (checking) {
    return (
      <PortalAuthShell title="Invitación" subtitle="Verificando…">
        <p className="text-sm text-slate-500">Un momento…</p>
      </PortalAuthShell>
    );
  }

  if (fatal) {
    return (
      <PortalAuthShell
        title="Invitación no válida"
        subtitle={fatal}
        footer={
          <Link href="/portal/login" className="text-cyan-600 hover:text-cyan-700">
            Ir a iniciar sesión
          </Link>
        }
      >
        <p className="text-sm text-slate-500">Pide a quien te invitó que genere un nuevo enlace de invitación.</p>
      </PortalAuthShell>
    );
  }

  return (
    <PortalAuthShell
      title="Unirte a tu equipo"
      subtitle={info ? `${info.org} · ${info.email}` : undefined}
      footer={
        <Link href="/portal/login" className="text-cyan-600 hover:text-cyan-700">
          Iniciar sesión
        </Link>
      }
    >
      <form onSubmit={submit} className="space-y-4">
        <Field label="Email" type="email" value={info?.email ?? ""} readOnly disabled />
        <Field
          label="Crea tu contraseña (mín. 8)"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="new-password"
        />
        {err ? <p className="text-sm text-rose-500">{err}</p> : null}
        <PrimaryButton type="submit" disabled={loading || password.length < 8}>
          {loading ? "Creando cuenta…" : "Unirme"}
        </PrimaryButton>
      </form>
    </PortalAuthShell>
  );
}

export default function PortalAccept() {
  return (
    <Suspense
      fallback={
        <div className="grid min-h-screen place-items-center bg-gradient-to-br from-slate-50 via-[#eef4fb] to-[#e6eefb] text-sm text-slate-500">
          Cargando…
        </div>
      }
    >
      <AcceptInner />
    </Suspense>
  );
}
