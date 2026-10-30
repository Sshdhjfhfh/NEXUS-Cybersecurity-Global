"use client";

import * as React from "react";

import { CheckIcon, GlobeIcon, RefreshCwIcon, ServerIcon, ShieldCheckIcon, XIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { http } from "@/lib/api";

// Objetivos que los clientes del portal registran para ser evaluados. Lo normal
// es que el cliente demuestre la titularidad por DNS (TXT) o HTTP (archivo en
// /.well-known) desde el portal. Para sistemas internos, IP privadas o cuando
// el cliente aporta una autorización escrita, el operador aprueba o rechaza
// aquí. La nota es obligatoria: es el registro de POR QUÉ NEXUS consideró al
// cliente autorizado, y queda en la auditoría de la organización.
interface Target {
  id: number;
  org_id: number;
  org_name?: string;
  kind: "domain" | "ip";
  value: string;
  status: "pending" | "verified" | "rejected" | "expired";
  method?: string;
  verified_at?: string;
  expires_at?: string;
  verified_by?: string;
  note?: string;
  last_check_at?: string;
  last_error?: string;
  created_at: string;
}

const FILTERS = [
  { id: "pending", label: "Pendientes" },
  { id: "verified", label: "Verificados" },
  { id: "rejected", label: "Rechazados" },
  { id: "", label: "Todos" },
] as const;
type Filter = (typeof FILTERS)[number]["id"];

const STATUS_LABEL: Record<Target["status"], string> = {
  pending: "Pendiente",
  verified: "Verificado",
  rejected: "Rechazado",
  expired: "Caducado",
};
const STATUS_CLASS: Record<Target["status"], string> = {
  pending: "border-amber-500/40 text-amber-700 dark:text-amber-400",
  verified: "border-emerald-500/40 text-emerald-700 dark:text-emerald-400",
  rejected: "border-rose-500/40 text-rose-600 dark:text-rose-400",
  expired: "border-slate-500/40 text-muted-foreground",
};
const METHOD_LABEL: Record<string, string> = { dns: "DNS", http: "HTTP", manual: "aprobación manual" };
const MIN_NOTE = 10;

export default function PortalTargetsPage() {
  const [filter, setFilter] = React.useState<Filter>("pending");
  const [targets, setTargets] = React.useState<Target[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [notes, setNotes] = React.useState<Record<number, string>>({});
  const [busy, setBusy] = React.useState<number | null>(null);

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const q = filter ? `?status=${filter}` : "";
      const res = await http<{ targets: Target[] | null }>(`/admin/portal/targets${q}`);
      setTargets(res.targets ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [filter]);

  React.useEffect(() => {
    void load();
  }, [load]);

  async function decide(t: Target, approve: boolean) {
    const note = (notes[t.id] ?? "").trim();
    if (
      !approve &&
      !window.confirm(`¿Rechazar ${t.value} de ${t.org_name ?? "la organización"}? El cliente no podrá usarlo.`)
    ) {
      return;
    }
    setBusy(t.id);
    setError(null);
    try {
      await http(`/admin/portal/targets/${t.id}/decision`, {
        method: "POST",
        body: JSON.stringify({ approve, note }),
      });
      setNotes((cur) => ({ ...cur, [t.id]: "" }));
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="flex flex-col gap-4 p-4 md:p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-xl font-semibold">
            <ShieldCheckIcon className="size-5" /> Objetivos de clientes
          </h1>
          <p className="text-sm text-muted-foreground">
            Titularidad de dominios e IP · solo los verificados pueden entrar en una evaluación
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void load()} disabled={loading}>
          <RefreshCwIcon className={loading ? "animate-spin" : ""} /> Actualizar
        </Button>
      </div>

      <div className="flex flex-wrap gap-1.5" role="tablist" aria-label="Filtrar por estado">
        {FILTERS.map((f) => (
          <Button
            key={f.id || "all"}
            size="sm"
            role="tab"
            aria-selected={filter === f.id}
            variant={filter === f.id ? "default" : "outline"}
            onClick={() => setFilter(f.id)}
          >
            {f.label}
          </Button>
        ))}
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {!loading && !error && targets.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            {filter === "pending" ? "No hay objetivos pendientes de revisión." : "No hay objetivos."}
          </CardContent>
        </Card>
      ) : null}

      <div className="grid gap-3">
        {targets.map((t) => {
          const note = notes[t.id] ?? "";
          const Icon = t.kind === "ip" ? ServerIcon : GlobeIcon;
          return (
            <Card key={t.id}>
              <CardHeader className="pb-2">
                <CardTitle className="flex flex-wrap items-center gap-2 text-base">
                  <Icon className="size-4 text-muted-foreground" />
                  <span className="font-mono">{t.value}</span>
                  <Badge variant="outline" className={STATUS_CLASS[t.status]}>
                    {STATUS_LABEL[t.status]}
                  </Badge>
                  <Badge variant="secondary">{t.org_name ?? `org ${t.org_id}`}</Badge>
                  <span className="ml-auto text-xs font-normal text-muted-foreground">
                    {new Date(t.created_at).toLocaleString()}
                  </span>
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-2 text-sm">
                {t.status === "verified" && t.method ? (
                  <p className="text-xs text-muted-foreground">
                    Verificado por {METHOD_LABEL[t.method] ?? t.method}
                    {t.verified_at ? ` el ${new Date(t.verified_at).toLocaleDateString()}` : ""}
                    {t.expires_at ? ` · caduca ${new Date(t.expires_at).toLocaleDateString()}` : ""}
                  </p>
                ) : null}
                {t.note ? <p className="whitespace-pre-wrap text-xs text-muted-foreground">Nota: {t.note}</p> : null}
                {t.last_error && t.status !== "verified" ? (
                  <p className="text-xs text-muted-foreground">
                    Última comprobación del cliente
                    {t.last_check_at ? ` (${new Date(t.last_check_at).toLocaleString()})` : ""}: {t.last_error}
                  </p>
                ) : null}
                <div className="space-y-2 pt-1">
                  <label htmlFor={`note-${t.id}`} className="block text-xs font-medium">
                    Evidencia / motivo (mín. {MIN_NOTE} caracteres; visible en la auditoría del cliente)
                  </label>
                  <textarea
                    id={`note-${t.id}`}
                    value={note}
                    onChange={(e) => setNotes((cur) => ({ ...cur, [t.id]: e.target.value }))}
                    rows={2}
                    maxLength={1000}
                    placeholder="p. ej. Carta de autorización firmada por el CISO, ref. …"
                    className="w-full rounded-md border bg-background px-2 py-1.5 text-sm"
                  />
                  <div className="flex flex-wrap gap-2">
                    <Button
                      size="sm"
                      onClick={() => void decide(t, true)}
                      disabled={busy === t.id || note.trim().length < MIN_NOTE}
                    >
                      <CheckIcon className="size-4" /> {t.status === "verified" ? "Renovar aprobación" : "Aprobar"}
                    </Button>
                    {t.status !== "rejected" ? (
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => void decide(t, false)}
                        disabled={busy === t.id || note.trim().length < MIN_NOTE}
                      >
                        <XIcon className="size-4" /> Rechazar
                      </Button>
                    ) : null}
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
