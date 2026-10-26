"use client";

import * as React from "react";

import { InboxIcon, RefreshCwIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { http } from "@/lib/api";

// Solicitudes comerciales captadas por la web pública (POST /api/public/leads).
// Lectura solo para usuarios autenticados (GET /api/leads).
interface Lead {
  id: number;
  organization: string;
  contact_name: string;
  email: string;
  country: string;
  service: string;
  message: string;
  status: string;
  consent: boolean;
  source_ip: string;
  created_at: string;
}

export default function LeadsPage() {
  const [leads, setLeads] = React.useState<Lead[]>([]);
  const [total, setTotal] = React.useState(0);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await http<{ leads: Lead[] | null; total: number }>("/leads?limit=200");
      setLeads(res.leads ?? []);
      setTotal(res.total ?? 0);
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
            <InboxIcon className="size-5" /> Leads
          </h1>
          <p className="text-sm text-muted-foreground">Solicitudes recibidas desde la web pública · {total} en total</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void load()} disabled={loading}>
          <RefreshCwIcon className={loading ? "animate-spin" : ""} /> Actualizar
        </Button>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {!loading && !error && leads.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            Aún no hay solicitudes. Envía una desde /request-demo.
          </CardContent>
        </Card>
      ) : null}

      <div className="grid gap-3">
        {leads.map((l) => (
          <Card key={l.id}>
            <CardHeader className="pb-2">
              <CardTitle className="flex flex-wrap items-center gap-2 text-base">
                <span className="text-muted-foreground">#{l.id}</span>
                {l.organization}
                {l.service ? <Badge variant="secondary">{l.service}</Badge> : null}
                <Badge variant="outline">{l.status}</Badge>
                <span className="ml-auto text-xs font-normal text-muted-foreground">
                  {new Date(l.created_at).toLocaleString()}
                </span>
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-1 text-sm">
              <div>
                {l.contact_name} ·{" "}
                <a className="underline" href={`mailto:${l.email}`}>
                  {l.email}
                </a>
                {l.country ? ` · ${l.country}` : ""}
              </div>
              {l.message ? <p className="whitespace-pre-wrap text-muted-foreground">{l.message}</p> : null}
              <div className="text-xs text-muted-foreground">
                IP {l.source_ip} · consentimiento {l.consent ? "sí" : "no"}
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
