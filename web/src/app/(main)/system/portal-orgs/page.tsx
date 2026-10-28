"use client";

import * as React from "react";

import { BuildingIcon, ChevronDownIcon, ChevronRightIcon, RefreshCwIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { http } from "@/lib/api";

// Consola de plataforma: todas las organizaciones (tenants) del portal. Solo la
// ve el operador de la consola (requireAuth). Suspender bloquea al tenant (login
// y toda petición autenticada) hasta reactivarlo; la acción se aplica en el
// servidor, no solo en la UI.
interface OrgSummary {
  id: number;
  name: string;
  suspended: boolean;
  created_at: string;
  member_count: number;
  request_count: number;
}

interface Member {
  id: number;
  email: string;
  role: string;
  created_at: string;
}

const ROLE_LABEL: Record<string, string> = {
  owner: "Propietario",
  admin: "Administrador",
  member: "Miembro",
};

export default function PortalOrgsPage() {
  const [orgs, setOrgs] = React.useState<OrgSummary[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [busy, setBusy] = React.useState<number | null>(null);
  const [expanded, setExpanded] = React.useState<number | null>(null);
  const [members, setMembers] = React.useState<Record<number, Member[]>>({});

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await http<{ orgs: OrgSummary[] | null }>("/admin/portal/orgs");
      setOrgs(res.orgs ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  async function toggleSuspend(org: OrgSummary) {
    const next = !org.suspended;
    if (next && !window.confirm(`¿Suspender «${org.name}»? Sus usuarios no podrán acceder hasta reactivarla.`)) {
      return;
    }
    setBusy(org.id);
    setError(null);
    const prev = orgs;
    setOrgs((cur) => cur.map((o) => (o.id === org.id ? { ...o, suspended: next } : o)));
    try {
      await http(`/admin/portal/orgs/${org.id}`, { method: "PATCH", body: JSON.stringify({ suspended: next }) });
    } catch (e) {
      setOrgs(prev);
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(null);
    }
  }

  async function toggleExpand(id: number) {
    if (expanded === id) {
      setExpanded(null);
      return;
    }
    setExpanded(id);
    if (!members[id]) {
      try {
        const res = await http<{ members: Member[] | null }>(`/admin/portal/orgs/${id}/members`);
        setMembers((cur) => ({ ...cur, [id]: res.members ?? [] }));
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      }
    }
  }

  return (
    <div className="flex flex-col gap-4 p-4 md:p-6">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-xl font-semibold">
            <BuildingIcon className="size-5" /> Organizaciones
          </h1>
          <p className="text-sm text-muted-foreground">Clientes del portal · {orgs.length} en total</p>
        </div>
        <Button variant="outline" size="sm" onClick={() => void load()} disabled={loading}>
          <RefreshCwIcon className={loading ? "animate-spin" : ""} /> Actualizar
        </Button>
      </div>

      {error ? <p className="text-sm text-destructive">{error}</p> : null}

      {!loading && !error && orgs.length === 0 ? (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            Aún no hay organizaciones registradas en el portal.
          </CardContent>
        </Card>
      ) : null}

      <div className="grid gap-3">
        {orgs.map((o) => {
          const isOpen = expanded === o.id;
          return (
            <Card key={o.id}>
              <CardHeader className="pb-2">
                <CardTitle className="flex flex-wrap items-center gap-2 text-base">
                  <button
                    type="button"
                    onClick={() => void toggleExpand(o.id)}
                    className="inline-flex items-center gap-1.5 hover:text-primary"
                    aria-expanded={isOpen}
                  >
                    {isOpen ? <ChevronDownIcon className="size-4" /> : <ChevronRightIcon className="size-4" />}
                    {o.name}
                  </button>
                  {o.suspended ? <Badge variant="destructive">Suspendida</Badge> : null}
                  <Badge variant="secondary">{o.member_count} miembro(s)</Badge>
                  <Badge variant="secondary">{o.request_count} solicitud(es)</Badge>
                  <span className="ml-auto text-xs font-normal text-muted-foreground">
                    {new Date(o.created_at).toLocaleDateString()}
                  </span>
                  <Button
                    variant={o.suspended ? "outline" : "destructive"}
                    size="sm"
                    onClick={() => void toggleSuspend(o)}
                    disabled={busy === o.id}
                  >
                    {o.suspended ? "Reactivar" : "Suspender"}
                  </Button>
                </CardTitle>
              </CardHeader>
              {isOpen ? (
                <CardContent className="text-sm">
                  {members[o.id] === undefined ? (
                    <p className="text-muted-foreground">Cargando miembros…</p>
                  ) : members[o.id].length === 0 ? (
                    <p className="text-muted-foreground">Sin miembros.</p>
                  ) : (
                    <ul className="divide-y divide-border">
                      {members[o.id].map((m) => (
                        <li key={m.id} className="flex items-center justify-between gap-3 py-2">
                          <span>{m.email}</span>
                          <Badge variant="outline">{ROLE_LABEL[m.role] ?? m.role}</Badge>
                        </li>
                      ))}
                    </ul>
                  )}
                </CardContent>
              ) : null}
            </Card>
          );
        })}
      </div>
    </div>
  );
}
