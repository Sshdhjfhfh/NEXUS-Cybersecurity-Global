"use client";

import * as React from "react";

import { KeyRoundIcon, ShieldCheckIcon } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api } from "@/lib/api";

// Two-factor authentication (TOTP) management for the console admin.
export default function MfaPage() {
  const [enabled, setEnabled] = React.useState<boolean | null>(null);
  const [secret, setSecret] = React.useState<string | null>(null);
  const [otpauth, setOtpauth] = React.useState<string | null>(null);
  const [code, setCode] = React.useState("");
  const [busy, setBusy] = React.useState(false);

  const refresh = React.useCallback(async () => {
    try {
      const { enabled } = await api.mfaStatus();
      setEnabled(enabled);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Error");
    }
  }, []);

  React.useEffect(() => {
    void refresh();
  }, [refresh]);

  async function startSetup() {
    setBusy(true);
    try {
      const res = await api.mfaSetup();
      setSecret(res.secret);
      setOtpauth(res.otpauth);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Error");
    } finally {
      setBusy(false);
    }
  }

  async function enable() {
    setBusy(true);
    try {
      await api.mfaEnable(code.trim());
      toast.success("2FA activado");
      setSecret(null);
      setOtpauth(null);
      setCode("");
      await refresh();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Código inválido");
    } finally {
      setBusy(false);
    }
  }

  async function disable() {
    setBusy(true);
    try {
      await api.mfaDisable(code.trim());
      toast.success("2FA desactivado");
      setCode("");
      await refresh();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Código inválido");
    } finally {
      setBusy(false);
    }
  }

  const onCode = (v: string) => setCode(v.replace(/\D/g, "").slice(0, 6));

  return (
    <div className="mx-auto max-w-2xl p-4 md:p-6">
      <h1 className="flex items-center gap-2 text-xl font-semibold">
        <ShieldCheckIcon className="size-5" /> Autenticación en dos pasos (2FA)
      </h1>
      <p className="mt-1 text-sm text-muted-foreground">
        Protege el acceso a la consola con un código temporal (TOTP) de una app como Google Authenticator, 1Password o
        Authy.
      </p>

      <Card className="mt-6">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-base">
            <KeyRoundIcon className="size-4" />
            Estado: {enabled === null ? "…" : enabled ? "Activado" : "Desactivado"}
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {enabled === false ? (
            secret ? (
              <>
                <div className="space-y-2">
                  <Label>1. Añade esta clave en tu app de autenticación</Label>
                  <div className="rounded-lg border bg-muted/40 p-3 font-mono text-sm break-all">{secret}</div>
                  {otpauth ? (
                    <a href={otpauth} className="text-xs text-primary underline-offset-4 hover:underline break-all">
                      Abrir en la app (otpauth://…)
                    </a>
                  ) : null}
                </div>
                <div className="space-y-2">
                  <Label htmlFor="enable-code">2. Introduce el código de 6 dígitos para activar</Label>
                  <Input
                    id="enable-code"
                    value={code}
                    onChange={(e) => onCode(e.target.value)}
                    placeholder="123456"
                    inputMode="numeric"
                  />
                </div>
                <Button onClick={enable} disabled={busy || code.length !== 6}>
                  Activar 2FA
                </Button>
              </>
            ) : (
              <Button onClick={startSetup} disabled={busy}>
                Configurar 2FA
              </Button>
            )
          ) : enabled === true ? (
            <>
              <p className="text-sm text-muted-foreground">
                La 2FA está activa. Para desactivarla, introduce un código válido.
              </p>
              <div className="space-y-2">
                <Label htmlFor="disable-code">Código actual</Label>
                <Input
                  id="disable-code"
                  value={code}
                  onChange={(e) => onCode(e.target.value)}
                  placeholder="123456"
                  inputMode="numeric"
                />
              </div>
              <Button variant="destructive" onClick={disable} disabled={busy || code.length !== 6}>
                Desactivar 2FA
              </Button>
            </>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
