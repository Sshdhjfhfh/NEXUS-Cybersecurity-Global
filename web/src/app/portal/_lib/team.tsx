"use client";

import { type FormEvent, useCallback, useEffect, useState } from "react";

import { CopyIcon, Trash2Icon, UserPlusIcon } from "lucide-react";

import { type Invite, type Me, type Member, type PortalRole, portalApi } from "./portal";
import { Field, PrimaryButton } from "./ui";

const ROLE_LABEL: Record<PortalRole, string> = {
  owner: "Propietario",
  admin: "Administrador",
  member: "Miembro",
};

function RoleBadge({ role }: { role: PortalRole }) {
  const tone =
    role === "owner"
      ? "border-amber-400/30 bg-amber-400/10 text-amber-300"
      : role === "admin"
        ? "border-cyan-400/30 bg-cyan-400/10 text-cyan-300"
        : "border-white/15 bg-white/5 text-slate-300";
  return (
    <span className={`rounded-full border px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${tone}`}>
      {ROLE_LABEL[role]}
    </span>
  );
}

// TeamPanel is only mounted for admin/owner (the server also enforces this).
export function TeamPanel({ me }: { me: Me }) {
  const [members, setMembers] = useState<Member[]>([]);
  const [invites, setInvites] = useState<Invite[]>([]);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<PortalRole>("member");
  const [newLink, setNewLink] = useState("");
  const [copied, setCopied] = useState(false);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setErr("");
    try {
      const [{ members: m }, { invites: i }] = await Promise.all([portalApi.listMembers(), portalApi.listInvites()]);
      setMembers(m ?? []);
      setInvites(i ?? []);
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "No se pudo cargar el equipo");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function invite(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setErr("");
    setNewLink("");
    try {
      const res = await portalApi.createInvite(email.trim().toLowerCase(), role);
      const link = `${window.location.origin}/portal/accept?token=${encodeURIComponent(res.token)}`;
      setNewLink(link);
      setEmail("");
      setRole("member");
      await load();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "No se pudo crear la invitación");
    } finally {
      setBusy(false);
    }
  }

  async function changeRole(id: number, next: PortalRole) {
    setErr("");
    try {
      await portalApi.updateMemberRole(id, next);
      await load();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "No se pudo cambiar el rol");
    }
  }

  async function remove(id: number) {
    setErr("");
    try {
      await portalApi.removeMember(id);
      await load();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "No se pudo eliminar el miembro");
    }
  }

  async function revoke(id: number) {
    setErr("");
    try {
      await portalApi.revokeInvite(id);
      await load();
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "No se pudo revocar la invitación");
    }
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(newLink);
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    } catch {
      /* clipboard may be unavailable; the link is still visible to copy manually */
    }
  }

  const canManageOwners = me.user.role === "owner";

  if (loading) {
    return <p className="mt-4 text-sm text-slate-400">Cargando equipo…</p>;
  }

  return (
    <div className="mt-4 grid gap-8 lg:grid-cols-[1fr_1.4fr]">
      {/* Invite */}
      <section className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
        <h2 className="flex items-center gap-2 text-sm font-semibold uppercase tracking-wide text-cyan-300">
          <UserPlusIcon className="size-4" /> Invitar miembro
        </h2>
        <form onSubmit={invite} className="mt-4 space-y-4">
          <Field
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="colega@empresa.com"
            required
          />
          <label className="block text-xs font-medium text-slate-400">
            Rol
            <select
              value={role}
              onChange={(e) => setRole(e.target.value as PortalRole)}
              className="mt-1.5 w-full rounded-lg border border-white/15 bg-[#0a1324] px-3 py-2.5 text-sm text-white focus:border-cyan-400/60 focus:outline-none"
            >
              <option value="member">Miembro (crear y ver solicitudes)</option>
              <option value="admin">Administrador (gestiona el equipo)</option>
            </select>
          </label>
          {err ? <p className="text-sm text-rose-400">{err}</p> : null}
          <PrimaryButton type="submit" disabled={busy || !email.trim()}>
            {busy ? "Generando…" : "Generar invitación"}
          </PrimaryButton>
        </form>
        {newLink ? (
          <div className="mt-5 rounded-xl border border-cyan-400/25 bg-cyan-400/5 p-4">
            <p className="text-xs text-slate-300">
              Comparte este enlace con la persona invitada. Es de un solo uso y fija su email y rol.
            </p>
            <div className="mt-2 flex items-center gap-2">
              <code className="min-w-0 flex-1 truncate rounded-md bg-[#0a1324] px-2.5 py-1.5 text-[11px] text-cyan-200">
                {newLink}
              </code>
              <button
                type="button"
                onClick={copyLink}
                className="inline-flex shrink-0 items-center gap-1 rounded-md border border-white/15 px-2.5 py-1.5 text-[11px] text-slate-200 hover:border-cyan-400/50 hover:text-white"
              >
                <CopyIcon className="size-3.5" /> {copied ? "Copiado" : "Copiar"}
              </button>
            </div>
          </div>
        ) : null}
      </section>

      {/* Members + pending invites */}
      <section className="space-y-6">
        <div>
          <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-300">Miembros ({members.length})</h2>
          <ul className="mt-4 space-y-3">
            {members.map((m) => {
              const isSelf = m.id === me.user.id;
              const lockedOwner = m.role === "owner" && !canManageOwners;
              return (
                <li key={m.id} className="rounded-xl border border-white/10 bg-white/[0.03] p-4">
                  <div className="flex flex-wrap items-center justify-between gap-3">
                    <div className="min-w-0">
                      <span className="block truncate font-medium text-white">
                        {m.email}
                        {isSelf ? <span className="ml-1.5 text-[11px] text-slate-500">(tú)</span> : null}
                      </span>
                      <span className="text-[11px] text-slate-500">
                        Desde {new Date(m.created_at).toLocaleDateString()}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      {lockedOwner ? (
                        <RoleBadge role={m.role} />
                      ) : (
                        <select
                          value={m.role}
                          onChange={(e) => changeRole(m.id, e.target.value as PortalRole)}
                          className="rounded-md border border-white/15 bg-[#0a1324] px-2 py-1 text-xs text-white focus:border-cyan-400/60 focus:outline-none"
                        >
                          {canManageOwners ? <option value="owner">Propietario</option> : null}
                          <option value="admin">Administrador</option>
                          <option value="member">Miembro</option>
                        </select>
                      )}
                      {!isSelf && !lockedOwner ? (
                        <button
                          type="button"
                          onClick={() => remove(m.id)}
                          aria-label={`Eliminar ${m.email}`}
                          className="rounded-md border border-white/10 p-1.5 text-slate-400 hover:border-rose-400/40 hover:text-rose-300"
                        >
                          <Trash2Icon className="size-3.5" />
                        </button>
                      ) : null}
                    </div>
                  </div>
                </li>
              );
            })}
          </ul>
        </div>

        {invites.length > 0 ? (
          <div>
            <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-400">
              Invitaciones pendientes ({invites.length})
            </h2>
            <ul className="mt-4 space-y-2.5">
              {invites.map((inv) => (
                <li
                  key={inv.id}
                  className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-dashed border-white/15 bg-white/[0.02] p-3.5"
                >
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-slate-200">{inv.email}</span>
                    <RoleBadge role={inv.role} />
                  </div>
                  <button
                    type="button"
                    onClick={() => revoke(inv.id)}
                    className="text-[11px] text-slate-400 hover:text-rose-300"
                  >
                    Revocar
                  </button>
                </li>
              ))}
            </ul>
          </div>
        ) : null}
      </section>
    </div>
  );
}
