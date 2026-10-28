"use client";

import { useCallback, useEffect, useState } from "react";

import { type AuditEntry, portalApi } from "./portal";

// Read-only audit of this organization's own security events. Admin+ only (the
// server also enforces it). Gives enterprise/government clients self-service
// transparency into who did what in their account.
const ACTION_LABEL: Record<string, string> = {
  "org.register": "Organización creada",
  "auth.login": "Inicio de sesión",
  "auth.login_failed": "Inicio de sesión fallido",
  "auth.login_blocked": "Acceso bloqueado (suspendida)",
  "invite.create": "Invitación creada",
  "invite.accept": "Invitación aceptada",
  "member.role_change": "Cambio de rol",
  "member.remove": "Miembro eliminado",
  "request.promote": "Solicitud promovida a evaluación",
  "request.status": "Estado de solicitud actualizado",
  "org.suspend": "Organización suspendida",
  "org.reactivate": "Organización reactivada",
};

export function AuditPanel() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    setErr("");
    try {
      const { entries } = await portalApi.listAudit();
      setEntries(entries ?? []);
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "No se pudo cargar la auditoría");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) {
    return <p className="mt-4 text-sm text-slate-400">Cargando auditoría…</p>;
  }
  if (err) {
    return <p className="mt-4 text-sm text-rose-400">{err}</p>;
  }

  return (
    <div className="mt-4">
      <p className="mb-3 text-sm text-slate-400">
        Registro de actividad de tu organización. {entries.length} evento(s).
      </p>
      {entries.length === 0 ? (
        <p className="rounded-xl border border-white/10 bg-white/[0.02] p-5 text-sm text-slate-400">
          Aún no hay eventos registrados.
        </p>
      ) : (
        <ul className="space-y-2">
          {entries.map((e) => (
            <li
              key={e.id}
              className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-xl border border-white/10 bg-white/[0.03] p-3.5 text-sm"
            >
              <span className="font-medium text-white">{ACTION_LABEL[e.action] ?? e.action}</span>
              <span className="text-slate-400">{e.actor}</span>
              {e.detail ? <span className="text-xs text-slate-500">{e.detail}</span> : null}
              <span className="ml-auto text-xs text-slate-500">{new Date(e.created_at).toLocaleString()}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
