"use client";

import * as React from "react";

import { RefreshCwIcon, ShieldCheckIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { http } from "@/lib/api";

// Registro de auditoría a prueba de manipulaciones (cadena de hash). Solo visible
// para el operador de la consola (requireAuth). "Verificar integridad" recorre
// toda la cadena en el servidor y confirma que ninguna entrada fue alterada,
// borrada o reordenada.
interface AuditEntry {
  id: number;
  org_id?: number;
  actor: string;
  actor_type: string;
  action: string;
  target: string;
  detail?: string;
  ip: string;
  created_at: string;
}

interface Verification {
  ok: boolean;
  count: number;
  broken_at?: number;
  reason?: string;
}

const ACTION_TONE: Record<string, string> = {
  "auth.login_failed": "destructive",
  "auth.login_blocked": "destructive",
  "org.suspend": "destructive",
  "member.remove": "destructive",
};

export default function PortalAuditPage() {
  const [entries, setEntries] = React.useState<AuditEntry[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [verif, setVerif] = React.useState<Verification | null>(null);
  const [verifying, setVerifying] = React.useState(false);

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await http<{ entries: AuditEntry[] | null }>("/admin/portal/audit?limit=500");
      setEntries(res.entries ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  async function verify() {
    setVerifying(true);
    setError(null);
    try {
      setVerif(await http<Verification>("/admin/portal/audit/verify"));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setVerifying(false);
    }
  }

  return (
    <div className="flex flex-col gap-4 p-4 md:p-6">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-xl font-semibold">
            <ShieldCheckIcon className="size-5" /> Auditoría
          </h1>
          <p className="text-sm text-muted-foreground">Registro inalterable de acciones · {entries.length} entradas</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => void verify()} disabled={verifying}>
            <ShieldCheckIcon className={verifying ? "animate-pulse" : ""} /> Verificar integridad
          </Button>
          <Button variant="outline" size="sm" onClick={() => void load()} disabled={loading}>
            <RefreshCwIcon className={loading ? "animate-spin" : ""} /> Actualizar
          </Button>
        </div>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {verif ? (
        verif.ok ? (
          <p className="rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700 dark:text-emerald-300">
            ✓ Cadena íntegra — {verif.count} entradas verificadas. Ninguna fue alterada ni eliminada.
          </p>
        ) : (
          <p className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            ✗ Integridad comprometida en la entrada #{verif.broken_at}
            {verif.reason ? ` — ${verif.reason}` : ""}.
          </p>
        )
      ) : null}

      {!loading && !error && entries.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            Aún no hay entradas de auditoría.
          </CardContent>
        </Card>
      ) : null}

      {entries.length > 0 ? (
        <Card>
          <CardContent className="overflow-x-auto p-0">
            <table className="w-full text-sm">
              <thead className="border-b text-left text-xs uppercase tracking-wide text-muted-foreground">
                <tr>
                  <th className="px-3 py-2 font-medium">Fecha</th>
                  <th className="px-3 py-2 font-medium">Actor</th>
                  <th className="px-3 py-2 font-medium">Acción</th>
                  <th className="px-3 py-2 font-medium">Objetivo</th>
                  <th className="px-3 py-2 font-medium">Detalle</th>
                  <th className="px-3 py-2 font-medium">Org</th>
                  <th className="px-3 py-2 font-medium">IP</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((e) => (
                  <tr key={e.id} className="border-b last:border-0">
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-muted-foreground">
                      {new Date(e.created_at).toLocaleString()}
                    </td>
                    <td className="px-3 py-2">
                      {e.actor}
                      <span className="ml-1 text-[10px] text-muted-foreground">({e.actor_type})</span>
                    </td>
                    <td className="px-3 py-2">
                      <Badge variant={(ACTION_TONE[e.action] as "destructive" | undefined) ?? "secondary"}>
                        {e.action}
                      </Badge>
                    </td>
                    <td className="px-3 py-2 text-xs text-muted-foreground">{e.target || "—"}</td>
                    <td className="px-3 py-2 text-xs text-muted-foreground">{e.detail || "—"}</td>
                    <td className="px-3 py-2 text-xs text-muted-foreground">{e.org_id ?? "—"}</td>
                    <td className="px-3 py-2 text-xs text-muted-foreground">{e.ip || "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}
