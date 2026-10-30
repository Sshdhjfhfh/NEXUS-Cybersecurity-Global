"use client";

import * as React from "react";

import Link from "next/link";

import {
  Building2Icon,
  CheckCircle2Icon,
  FileSignatureIcon,
  FileTextIcon,
  RefreshCwIcon,
  RocketIcon,
  ShieldAlertIcon,
} from "lucide-react";

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

const TARGET_LABEL: Record<string, string> = {
  pending: "sin verificar",
  verified: "verificado",
  rejected: "rechazado",
  expired: "caducado",
};
const METHOD_LABEL: Record<string, string> = { dns: "DNS", http: "HTTP", manual: "aprobación manual" };

// Solicitudes de evaluación enviadas por clientes del portal multi-tenant.
// Solo visibles para el operador de la consola (requireAuth). Una solicitud solo
// puede PROMOVERSE a tarea del motor cuando todos sus objetivos están verificados
// y un administrador del cliente aceptó las reglas de enfrentamiento; el
// servidor lo impone (409 con los motivos). La excepción del operador exige un
// motivo escrito, queda auditada y nunca amplía a un dominio raíz. Promover NO
// lanza el motor: crea la tarea con el alcance exacto autorizado para que el
// operador la revise y la inicie con los controles habituales.
interface AuthTarget {
  id: number;
  kind: string;
  value: string;
  status: string;
  method?: string;
}
interface AdminRequest {
  id: number;
  org_id: number;
  org_name: string;
  title: string;
  scope: string;
  status: string;
  engine_task_id?: number;
  results_released: boolean;
  created_at: string;
  authorization?: {
    targets: AuthTarget[];
    ready: boolean;
    roe?: {
      signer_name: string;
      signer_title: string;
      accepted_email: string;
      window_start: string;
      window_end: string;
      emergency_contact: string;
      exclusions: string;
      roe_version: string;
      accepted_at: string;
    };
  };
  blockers?: string[];
}

const MIN_OVERRIDE = 20;

export default function PortalRequestsPage() {
  const [requests, setRequests] = React.useState<AdminRequest[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [note, setNote] = React.useState<string | null>(null);
  const [promoting, setPromoting] = React.useState<number | null>(null);
  const [overrideFor, setOverrideFor] = React.useState<number | null>(null);
  const [overrideReason, setOverrideReason] = React.useState("");

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

  async function promote(id: number, override_reason?: string) {
    setPromoting(id);
    setError(null);
    setNote(null);
    try {
      const res = await http<{ task_id: string; scope_added: number }>(`/admin/portal/requests/${id}/promote`, {
        method: "POST",
        body: JSON.stringify(override_reason ? { override_reason } : {}),
      });
      setNote(
        `Tarea #${res.task_id} creada (${res.scope_added} objetivo(s) de alcance añadidos). ` +
          `Revisa el alcance y arráncala desde Tareas.`,
      );
      setOverrideFor(null);
      setOverrideReason("");
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setPromoting(null);
    }
  }

  async function toggleRelease(id: number, released: boolean) {
    setError(null);
    const prev = requests;
    setRequests((cur) => cur.map((r) => (r.id === id ? { ...r, results_released: released } : r)));
    try {
      await http(`/admin/portal/requests/${id}/release`, { method: "POST", body: JSON.stringify({ released }) });
    } catch (e) {
      setRequests(prev);
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  // previewReport opens the branded PDF the client would get. Before release the
  // server stamps it as a draft ("BORRADOR"). The tab is opened synchronously so
  // the browser does not treat it as an unsolicited popup.
  async function previewReport(id: number, kind: "executive" | "technical") {
    setError(null);
    const win = window.open("", "_blank");
    try {
      const token = localStorage.getItem("nexus_ia_token");
      const res = await fetch(`/api/admin/portal/requests/${id}/report?kind=${kind}&lang=es`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      });
      if (!res.ok) {
        const data = (await res.json().catch(() => ({}))) as { error?: string };
        throw new Error(data.error ?? `error ${res.status}`);
      }
      const url = URL.createObjectURL(await res.blob());
      if (win) {
        win.location.href = url;
      } else {
        window.location.assign(url);
      }
      setTimeout(() => URL.revokeObjectURL(url), 60_000);
    } catch (e) {
      win?.close();
      setError(e instanceof Error ? e.message : String(e));
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
              <AuthorizationSummary r={r} />
              {r.scope ? (
                <div>
                  <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    Notas del cliente (no amplían el alcance)
                  </span>
                  <p className="mt-0.5 whitespace-pre-wrap text-muted-foreground">{r.scope}</p>
                </div>
              ) : null}
              <div className="flex flex-wrap items-center gap-3 pt-1">
                {r.engine_task_id ? (
                  <>
                    <Link href="/function/tasks" className="inline-flex items-center gap-1 text-sm underline">
                      Tarea #{r.engine_task_id}
                    </Link>
                    <Button
                      size="sm"
                      variant={r.results_released ? "outline" : "default"}
                      onClick={() => void toggleRelease(r.id, !r.results_released)}
                    >
                      {r.results_released ? "Retirar resultados" : "Liberar resultados al cliente"}
                    </Button>
                    {r.results_released ? <Badge variant="secondary">Resultados visibles para el cliente</Badge> : null}
                    <Button size="sm" variant="ghost" onClick={() => void previewReport(r.id, "executive")}>
                      <FileTextIcon className="size-4" /> Informe ejecutivo
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => void previewReport(r.id, "technical")}>
                      <FileTextIcon className="size-4" /> Informe técnico
                    </Button>
                    {r.results_released ? null : (
                      <span className="text-xs text-muted-foreground">Vista previa como borrador</span>
                    )}
                  </>
                ) : r.blockers && r.blockers.length > 0 ? (
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => {
                      setOverrideFor(overrideFor === r.id ? null : r.id);
                      setOverrideReason("");
                    }}
                  >
                    <ShieldAlertIcon className="size-4" /> Promover por excepción…
                  </Button>
                ) : (
                  <Button size="sm" onClick={() => void promote(r.id)} disabled={promoting === r.id}>
                    <RocketIcon className="size-4" />
                    {promoting === r.id ? "Promoviendo…" : "Promover a tarea"}
                  </Button>
                )}
              </div>
              {overrideFor === r.id && !r.engine_task_id ? (
                <div className="space-y-2 rounded-md border border-amber-500/40 bg-amber-500/5 p-3">
                  <label htmlFor={`override-${r.id}`} className="block text-xs font-medium">
                    Motivo de la excepción (mín. {MIN_OVERRIDE} caracteres). Queda en la auditoría del cliente; el
                    alcance se limita a los hosts exactos.
                  </label>
                  <textarea
                    id={`override-${r.id}`}
                    value={overrideReason}
                    onChange={(e) => setOverrideReason(e.target.value)}
                    rows={2}
                    maxLength={1000}
                    placeholder="p. ej. Contrato firmado fuera del portal, ref. …"
                    className="w-full rounded-md border bg-background px-2 py-1.5 text-sm"
                  />
                  <Button
                    size="sm"
                    onClick={() => void promote(r.id, overrideReason.trim())}
                    disabled={promoting === r.id || overrideReason.trim().length < MIN_OVERRIDE}
                  >
                    <RocketIcon className="size-4" />
                    {promoting === r.id ? "Promoviendo…" : "Promover con excepción"}
                  </Button>
                </div>
              ) : null}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}

function AuthorizationSummary({ r }: { r: AdminRequest }) {
  const a = r.authorization;
  if (!a) return null;
  const roe = a.roe;
  return (
    <div className="space-y-1.5 rounded-md border bg-muted/30 p-3">
      <div className="flex flex-wrap items-center gap-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
        Autorización
        {r.engine_task_id ? null : a.ready ? (
          <Badge variant="outline" className="border-emerald-500/40 text-emerald-700 dark:text-emerald-400">
            <CheckCircle2Icon className="size-3" /> lista
          </Badge>
        ) : (
          <Badge variant="outline" className="border-amber-500/40 text-amber-700 dark:text-amber-400">
            <ShieldAlertIcon className="size-3" /> bloqueada
          </Badge>
        )}
      </div>
      {a.targets.length > 0 ? (
        <ul className="flex flex-wrap gap-1.5">
          {a.targets.map((t) => (
            <li key={t.id} className="rounded border bg-background px-2 py-0.5 font-mono text-xs">
              {t.value}{" "}
              <span className={t.status === "verified" ? "text-emerald-600" : "text-amber-600"}>
                · {TARGET_LABEL[t.status] ?? t.status}
                {t.status === "verified" && t.method ? ` (${METHOD_LABEL[t.method] ?? t.method})` : ""}
              </span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-xs text-muted-foreground">
          Sin objetivos registrados (solicitud anterior a la verificación).
        </p>
      )}
      {roe ? (
        <p className="flex items-start gap-1.5 text-xs text-muted-foreground">
          <FileSignatureIcon className="mt-0.5 size-3.5 shrink-0" />
          <span>
            RoE {roe.roe_version} firmada por {roe.signer_name} ({roe.signer_title}, {roe.accepted_email}) · ventana{" "}
            {roe.window_start} → {roe.window_end} · emergencia: {roe.emergency_contact}
            {roe.exclusions ? ` · exclusiones: ${roe.exclusions}` : ""}
          </span>
        </p>
      ) : null}
      {!r.engine_task_id && r.blockers && r.blockers.length > 0 ? (
        <ul className="list-disc pl-5 text-xs text-amber-700 dark:text-amber-400">
          {r.blockers.map((b) => (
            <li key={b}>{b}</li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
