"use client";

import * as React from "react";

import Link from "next/link";

import { Building2Icon, RefreshCwIcon, RocketIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { http } from "@/lib/api";

const STATUSES = ["new", "scheduled", "in_progress", "done"] as const;

const STATUS_LABEL: Record<string, string> = {
  new: "Nueva",
  scheduled: "Programada",
  in_progress: "En curso",
  done: "Completada",
};

// Solicitudes de evaluación enviadas por clientes del portal multi-tenant.
// Solo visibles para el operador de la consola (requireAuth). El operador revisa
// el alcance autorizado y PROMUEVE la solicitud a una tarea del motor; promover
// NO lanza el motor autónomo: crea la tarea para que el operador confirme el
// alcance y la inicie con los controles habituales.
interface AdminRequest {
  id: number;
  org_id: number;
  org_name: string;
  title: string;
  scope: string;
  status: string;
  engine_task_id?: number;
  created_at: string;
}

export default function PortalRequestsPage() {
  const [requests, setRequests] = React.useState<AdminRequest[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [note, setNote] = React.useState<string | null>(null);
  const [promoting, setPromoting] = React.useState<number | null>(null);

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await http<{ requests: AdminRequest[] | null }>("/admin/portal/requests");
      setRequests(res.requests ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  async function promote(id: number) {
    setPromoting(id);
    setError(null);
    setNote(null);
    try {
      const res = await http<{ task_id: string; scope_added: number }>(`/admin/portal/requests/${id}/promote`, {
        method: "POST",
      });
      setNote(
        `Tarea #${res.task_id} creada (${res.scope_added} objetivo(s) de alcance añadidos). ` +
          `Revisa el alcance y arráncala desde Tareas.`,
      );
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setPromoting(null);
    }
  }

  async function updateStatus(id: number, status: string) {
    const prev = requests;
    setRequests((cur) => cur.map((r) => (r.id === id ? { ...r, status } : r)));
    try {
      await http(`/admin/portal/requests/${id}`, { method: "PATCH", body: JSON.stringify({ status }) });
    } catch (e) {
      setRequests(prev);
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="flex flex-col gap-4 p-4 md:p-6">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-xl font-semibold">
            <Building2Icon className="size-5" /> Solicitudes del portal
          </h1>
          <p className="text-sm text-muted-foreground">
            Evaluaciones solicitadas por clientes · {requests.length} en total
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void load()} disabled={loading}>
          <RefreshCwIcon className={loading ? "animate-spin" : ""} /> Actualizar
        </Button>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}
      {note ? (
        <p className="rounded-md border border-cyan-500/30 bg-cyan-500/10 px-3 py-2 text-sm text-cyan-700 dark:text-cyan-300">
          {note}
        </p>
      ) : null}

      {!loading && !error && requests.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            Aún no hay solicitudes de clientes del portal.
          </CardContent>
        </Card>
      ) : null}

      <div className="grid gap-3">
        {requests.map((r) => (
          <Card key={r.id}>
            <CardHeader className="pb-2">
              <CardTitle className="flex flex-wrap items-center gap-2 text-base">
                <span className="text-muted-foreground">#{r.id}</span>
                {r.title}
                <Badge variant="secondary">{r.org_name}</Badge>
                <select
                  value={r.status}
                  onChange={(e) => void updateStatus(r.id, e.target.value)}
                  className="rounded-md border bg-background px-2 py-0.5 text-xs"
                  aria-label="Estado de la solicitud"
                >
                  {STATUSES.map((st) => (
                    <option key={st} value={st}>
                      {STATUS_LABEL[st]}
                    </option>
                  ))}
                </select>
                <span className="ml-auto text-xs font-normal text-muted-foreground">
                  {new Date(r.created_at).toLocaleString()}
                </span>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm">
              {r.scope ? (
                <div>
                  <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    Alcance declarado
                  </span>
                  <p className="mt-0.5 whitespace-pre-wrap text-muted-foreground">{r.scope}</p>
                </div>
              ) : (
                <p className="text-xs text-muted-foreground">Sin alcance declarado.</p>
              )}
              <div className="flex items-center gap-3 pt-1">
                {r.engine_task_id ? (
                  <Link href="/function/tasks" className="inline-flex items-center gap-1 text-sm underline">
                    Tarea #{r.engine_task_id}
                  </Link>
                ) : (
                  <Button size="sm" onClick={() => void promote(r.id)} disabled={promoting === r.id}>
                    <RocketIcon className="size-4" />
                    {promoting === r.id ? "Promoviendo…" : "Promover a tarea"}
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
