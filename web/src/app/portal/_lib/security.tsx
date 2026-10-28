"use client";

import { useCallback, useEffect, useState } from "react";

import { CheckCircle2Icon, ShieldIcon } from "lucide-react";

import { portalApi } from "./portal";
import { Field, PrimaryButton } from "./ui";

// Per-user 2FA (TOTP) management. Every user manages their own second factor. The
// secret is shown for manual entry plus an otpauth:// link to open in the app —
// never sent to a third-party QR service, so the seed never leaves this origin.
export function SecurityPanel() {
  const [enabled, setEnabled] = useState<boolean | null>(null);
  const [setup, setSetup] = useState<{ secret: string; otpauth: string } | null>(null);
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [note, setNote] = useState("");

  const loadStatus = useCallback(async () => {
    try {
      const { enabled } = await portalApi.mfaStatus();
      setEnabled(enabled);
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "No se pudo cargar el estado de 2FA");
    }
  }, []);

  useEffect(() => {
    void loadStatus();
  }, [loadStatus]);

  async function startSetup() {
    setBusy(true);
    setErr("");
    setNote("");
    try {
      setSetup(await portalApi.mfaSetup());
      setCode("");
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "No se pudo iniciar la configuración");
    } finally {
      setBusy(false);
    }
  }

  async function enable() {
    setBusy(true);
    setErr("");
    try {
      await portalApi.mfaEnable(code.trim());
      setSetup(null);
      setCode("");
      setNote("2FA activado. Tu cuenta pedirá un código al iniciar sesión.");
      await loadStatus();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "Código inválido");
    } finally {
      setBusy(false);
    }
  }

  async function disable() {
    setBusy(true);
    setErr("");
    try {
      await portalApi.mfaDisable(code.trim());
      setCode("");
      setNote("2FA desactivado.");
      await loadStatus();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "Código inválido");
    } finally {
      setBusy(false);
    }
  }

  if (enabled === null) {
    return <p className="mt-4 text-sm text-slate-400">Cargando…</p>;
  }

  return (
    <div className="mt-4 max-w-lg">
      <section className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
        <h2 className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-cyan-300">
          <ShieldIcon className="size-4" /> Autenticación en dos pasos (2FA)
        </h2>

        {note ? <p className="mt-3 text-sm text-emerald-300">{note}</p> : null}
        {err ? <p className="mt-3 text-sm text-rose-400">{err}</p> : null}

        {enabled ? (
          <div className="mt-4 space-y-4">
            <p className="flex items-center gap-2 text-sm text-emerald-300">
              <CheckCircle2Icon className="size-4" /> 2FA activo en tu cuenta.
            </p>
            <p className="text-xs text-slate-400">
              Para desactivarlo, introduce un código actual de tu app de autenticación.
            </p>
            <Field
              label="Código actual"
              inputMode="numeric"
              placeholder="123456"
              value={code}
              onChange={(e) => setCode(e.target.value)}
            />
            <button
              type="button"
              onClick={() => void disable()}
              disabled={busy || code.trim().length < 6}
              className="w-full rounded-lg border border-rose-400/40 px-4 py-2.5 text-sm font-semibold text-rose-300 transition-colors hover:bg-rose-400/10 disabled:opacity-60"
            >
              {busy ? "Desactivando…" : "Desactivar 2FA"}
            </button>
          </div>
        ) : setup ? (
          <div className="mt-4 space-y-4">
            <p className="text-sm text-slate-300">
              1. Escanea o añade esta clave en tu app (Google Authenticator, Authy, 1Password…):
            </p>
            <code className="block break-all rounded-lg bg-[#0a1324] px-3 py-2.5 text-sm tracking-widest text-cyan-200">
              {setup.secret}
            </code>
            <a href={setup.otpauth} className="inline-block text-xs text-cyan-300 underline-offset-2 hover:underline">
              Abrir en la app (otpauth://…)
            </a>
            <p className="text-sm text-slate-300">2. Introduce el código de 6 dígitos que genera:</p>
            <Field
              label="Código de verificación"
              inputMode="numeric"
              placeholder="123456"
              value={code}
              onChange={(e) => setCode(e.target.value)}
            />
            <PrimaryButton type="button" onClick={() => void enable()} disabled={busy || code.trim().length < 6}>
              {busy ? "Activando…" : "Confirmar y activar"}
            </PrimaryButton>
          </div>
        ) : (
          <div className="mt-4 space-y-4">
            <p className="text-sm text-slate-400">
              Protege tu cuenta con un segundo factor. Recomendado para organizaciones con requisitos de seguridad.
            </p>
            <PrimaryButton type="button" onClick={() => void startSetup()} disabled={busy}>
              {busy ? "Preparando…" : "Activar 2FA"}
            </PrimaryButton>
          </div>
        )}
      </section>
    </div>
  );
}
