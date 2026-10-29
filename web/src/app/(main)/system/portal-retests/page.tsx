"use client";

import * as React from "react";

import Link from "next/link";

import { ClipboardCheckIcon, ExternalLinkIcon, RefreshCwIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { http } from "@/lib/api";

// Verificaciones solicitadas por clientes del portal: el cliente indica que ya
// corrigió un hallazgo y pide a NEXUS que lo compruebe. El operador lanza el
// retest desde el detalle del hallazgo (flujo de retest existente, con
// supervisión humana). Un veredicto "corregido" cierra el hallazgo; uno
// "reproducido" lo devuelve al cliente como "sigue vulnerable" y sale de esta
// cola. Los no concluyentes o fallidos siguen aquí.
interface RetestRequest {
  org_id: number;
  org_name: string;
  finding_id: number;
  finding_name: string;
  severity: string;
  requested_at: string;
  requested_by: string;
  note: string;
  latest_retest?: { status: string; verdict: string; created_at: string; finished_at?: string };
}

const SEV_LABEL: Record<string, string> = { critical: "Crítica", high: "Alta", medium: "Media", low: "Baja" };
const SEV_CLASS: Record<string, string> = {
  critical: "border-rose-500/40 text-rose-600 dark:text-rose-400",
  high: "border-orange-500/40 text-orange-600 dark:text-orange-400",
  medium: "border-amber-500/40 text-amber-700 dark:text-amber-400",
  low: "border-sky-500/40 text-sky-700 dark:text-sky-400",
};
const RETEST_LABEL: Record<string, string> = {
  pending: "pendiente",
  running: "en curso",
  completed: "completado",
  failed: "fallido",
  stopped: "detenido",
};
const VERDICT_LABEL: Record<string, string> = {
  reproduced: "reproducido",
  fixed: "corregido",
  inconclusive: "no concluyente",
};

function retestText(r: NonNullable<RetestRequest["latest_retest"]>, requestedAt: string): string {
  const after = new Date(r.created_at) >= new Date(requestedAt);
  const state =
    r.status === "completed" && r.verdict ? (VERDICT_LABEL[r.verdict] ?? r.verdict) : RETEST_LABEL[r.status];
  return `${after ? "Retest tras la solicitud" : "Último retest (anterior a la solicitud)"}: ${state ?? r.status}`;
}

export default function PortalRetestsPage() {
  const [items, setItems] = React.useState<RetestRequest[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await http<{ requests: RetestRequest[] | null }>("/admin/portal/retest-requests");
      setItems(res.requests ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="flex flex-col gap-4 p-4 md:p-6">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-xl font-semibold">
            <ClipboardCheckIcon className="size-5" /> Verificaciones solicitadas
          </h1>
          <p className="text-sm text-muted-foreground">
            Hallazgos que un cliente marcó como corregidos y espera que NEXUS verifique · {items.length}{" "}
            {items.length === 1 ? "pendiente" : "pendientes"}
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void load()} disabled={loading}>
          <RefreshCwIcon className={loading ? "animate-spin" : ""} /> Actualizar
        </Button>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {!loading && !error && items.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            No hay verificaciones pendientes.
          </CardContent>
        </Card>
      ) : null}

      <div className="grid gap-3">
        {items.map((r) => (
          <Card key={`${r.org_id}-${r.finding_id}`}>
            <CardHeader className="pb-2">
              <CardTitle className="flex flex-wrap items-center gap-2 text-base">
                <span className="text-muted-foreground">#{r.finding_id}</span>
                {r.finding_name}
                <Badge variant="outline" className={SEV_CLASS[r.severity]}>
                  {SEV_LABEL[r.severity] ?? "Sin clasificar"}
                </Badge>
                <Badge variant="secondary">{r.org_name}</Badge>
                <span className="ml-auto text-xs font-normal text-muted-foreground">
                  {new Date(r.requested_at).toLocaleString()}
                </span>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              <p className="text-xs text-muted-foreground">Solicitado por {r.requested_by || "—"}</p>
              {r.note ? (
                <div>
                  <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    Nota del cliente
                  </span>
                  <p className="mt-0.5 whitespace-pre-wrap">{r.note}</p>
                </div>
              ) : null}
              {r.latest_retest ? (
                <p className="text-xs text-muted-foreground">{retestText(r.latest_retest, r.requested_at)}</p>
              ) : null}
              <div className="pt-1">
                <Button asChild size="sm">
                  <Link href={`/function/findings/detail?id=${r.finding_id}`}>
                    <ExternalLinkIcon className="size-4" /> Abrir hallazgo y lanzar retest
                  </Link>
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
