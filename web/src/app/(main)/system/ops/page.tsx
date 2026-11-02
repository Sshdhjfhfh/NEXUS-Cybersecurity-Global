"use client";

import * as React from "react";

import { HeartPulseIcon, RefreshCwIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { http } from "@/lib/api";

// Estado del servicio: la vigilancia de operación del backend (base de datos,
// copias de seguridad, disco, errores 5xx, correo saliente). Cuando algo empieza
// a fallar avisa por email a NEXUS_OPS_EMAIL; aquí se ve el estado actual y se
// puede comprobar al momento. Que el backend entero esté caído no se puede ver
// desde aquí: para eso hace falta un monitor externo sobre /api/ready.
interface OpsCheck {
  name: string;
  title: string;
  status: "ok" | "fail" | "off";
  detail: string;
  since: string;
}

interface OpsStatus {
  checks: OpsCheck[] | null;
  checked_at: string | null;
  interval: string;
  ops_email_set: boolean;
  smtp: boolean;
}

const STATUS_LABEL: Record<OpsCheck["status"], string> = { ok: "Bien", fail: "Falla", off: "Sin configurar" };
const STATUS_CLASS: Record<OpsCheck["status"], string> = {
  ok: "border-emerald-500/40 text-emerald-700 dark:text-emerald-400",
  fail: "border-rose-500/40 text-rose-600 dark:text-rose-400",
  off: "border-slate-500/40 text-muted-foreground",
};

export default function OpsStatusPage() {
  const [data, setData] = React.useState<OpsStatus | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  const load = React.useCallback(async (now: boolean) => {
    setLoading(true);
    setError(null);
    try {
      setData(await http<OpsStatus>(now ? "/admin/ops/check" : "/admin/ops/status", now ? { method: "POST" } : {}));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load(false);
  }, [load]);

  const checks = data?.checks ?? [];
  const failing = checks.filter((c) => c.status === "fail").length;

  return (
    <div className="flex flex-col gap-4 p-4 md:p-6">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 font-semibold text-xl">
            <HeartPulseIcon className="size-5" /> Estado del servicio
          </h1>
          <p className="text-muted-foreground text-sm">
            {data?.checked_at
              ? `Última comprobación: ${new Date(data.checked_at).toLocaleString()} · cada ${data.interval}`
              : "Sin comprobaciones todavía"}
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void load(true)} disabled={loading}>
          <RefreshCwIcon className={loading ? "animate-spin" : ""} /> Comprobar ahora
        </Button>
      </div>

      {error ? <p className="text-destructive text-sm">{error}</p> : null}

      {data && !error ? (
        <p className={`text-sm ${failing > 0 ? "text-rose-600 dark:text-rose-400" : "text-muted-foreground"}`}>
          {failing > 0
            ? `${failing} ${failing === 1 ? "comprobación falla" : "comprobaciones fallan"}.`
            : "Todo lo que se vigila funciona."}{" "}
          {data.ops_email_set
            ? "Los avisos se envían por email al equipo de operación."
            : "Sin NEXUS_OPS_EMAIL los avisos solo quedan en el log del backend."}
        </p>
      ) : null}

      <div className="grid gap-3">
        {checks.map((c) => (
          <Card key={c.name}>
            <CardContent className="flex flex-col gap-1 py-4 sm:flex-row sm:items-start sm:gap-4">
              <div className="flex min-w-48 items-center gap-2">
                <Badge variant="outline" className={STATUS_CLASS[c.status]}>
                  {STATUS_LABEL[c.status]}
                </Badge>
                <span className="font-medium">{c.title}</span>
              </div>
              <div className="flex-1 text-sm">
                <p className="break-words">{c.detail}</p>
                <p className="text-muted-foreground text-xs">Desde {new Date(c.since).toLocaleString()}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <p className="text-muted-foreground text-xs">
        Si el backend entero se cae, esta página tampoco carga: configura un monitor externo que consulte{" "}
        <code>/api/ready</code> (ver docs/engineering/MONITORING.md).
      </p>
    </div>
  );
}
