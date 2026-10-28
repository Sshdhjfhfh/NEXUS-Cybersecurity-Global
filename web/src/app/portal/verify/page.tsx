"use client";

import { Suspense, useEffect, useState } from "react";

import Link from "next/link";
import { useSearchParams } from "next/navigation";

import { portalApi } from "../_lib/portal";
import { PortalAuthShell } from "../_lib/ui";

function VerifyInner() {
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

  const subtitle =
    state === "checking"
      ? "Verificando…"
      : state === "ok"
        ? "Tu email ha sido verificado."
        : "El enlace no es válido o ya fue utilizado.";

  return (
    <PortalAuthShell
      title="Verificación de email"
      subtitle={subtitle}
      footer={
        <Link href="/portal/login" className="text-cyan-600 hover:text-cyan-700">
          Ir a iniciar sesión
        </Link>
      }
    >
      {state === "ok" ? (
        <p className="text-sm text-emerald-600">✓ Gracias, tu dirección de email está confirmada.</p>
      ) : state === "error" ? (
        <p className="text-sm text-slate-500">
          Inicia sesión y vuelve a enviar el correo de verificación desde la pestaña Seguridad.
        </p>
      ) : (
        <p className="text-sm text-slate-500">Un momento…</p>
      )}
    </PortalAuthShell>
  );
}

export default function PortalVerify() {
  return (
    <Suspense
      fallback={
        <div className="grid min-h-screen place-items-center bg-gradient-to-br from-slate-50 via-[#eef4fb] to-[#e6eefb] text-sm text-slate-500">
          Cargando…
        </div>
      }
    >
      <VerifyInner />
    </Suspense>
  );
}
