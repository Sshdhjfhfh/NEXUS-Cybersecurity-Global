"use client";

import { type FormEvent, useCallback, useEffect, useState } from "react";

import Link from "next/link";
import { useRouter } from "next/navigation";

import { ArrowRightIcon, LogOutIcon } from "lucide-react";

import { AuditPanel } from "./_lib/audit";
import { clearPortalToken, getPortalToken, type Me, type PortalRequest, portalApi } from "./_lib/portal";
import { ResultsPanel } from "./_lib/results";
import { SecurityPanel } from "./_lib/security";
import { TeamPanel } from "./_lib/team";
import { Field, PrimaryButton } from "./_lib/ui";

const STATUS_LABEL: Record<string, string> = {
  new: "Nueva",
  scheduled: "Programada",
  in_progress: "En curso",
  done: "Completada",
};

type Tab = "requests" | "team" | "audit" | "security";

export default function PortalDashboard() {
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [reqs, setReqs] = useState<PortalRequest[]>([]);
  const [tab, setTab] = useState<Tab>("requests");
  const [openResults, setOpenResults] = useState<number | null>(null);
  const [title, setTitle] = useState("");
  const [scope, setScope] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(true);
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    try {
      const meData = await portalApi.me();
      setMe(meData);
      const { requests } = await portalApi.listRequests();
      setReqs(requests ?? []);
    } catch {
      router.replace("/portal/login");
    } finally {
      setLoading(false);
    }
  }, [router]);

  useEffect(() => {
    if (!getPortalToken()) {
      router.replace("/portal/login");
      return;
    }
    void load();
  }, [load, router]);

  async function create(e: FormEvent) {
    e.preventDefault();
    setCreating(true);
    setErr("");
    try {
      await portalApi.createRequest(title.trim(), scope.trim());
      setTitle("");
      setScope("");
      const { requests } = await portalApi.listRequests();
      setReqs(requests ?? []);
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "No se pudo crear la solicitud");
    } finally {
      setCreating(false);
    }
  }

  function logout() {
    clearPortalToken();
    router.replace("/portal/login");
  }

  if (loading || !me) {
    return <div className="grid min-h-screen place-items-center bg-[#060b18] text-sm text-slate-400">Cargando…</div>;
  }

  const canManageTeam = me.user.role === "owner" || me.user.role === "admin";

  return (
    <div className="min-h-screen bg-[#060b18] text-white">
      <header className="border-b border-white/10">
        <div className="mx-auto flex h-16 max-w-5xl items-center justify-between px-5">
          <Link href="/" className="flex items-center gap-2">
            {/* biome-ignore lint/performance/noImgElement: brand asset */}
            <img src="/brand/logo.webp" alt="NEXUS" className="h-8 w-auto" />
          </Link>
          <div className="flex items-center gap-4 text-sm">
            <span className="text-slate-400">{me.org.name}</span>
            <button
              type="button"
              onClick={logout}
              className="inline-flex items-center gap-1.5 text-slate-300 hover:text-white"
            >
              <LogOutIcon className="size-4" /> Salir
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-5 py-10">
        <h1 className="text-2xl font-bold">Hola, {me.org.name}</h1>
        <p className="mt-1 text-sm text-slate-400">Gestiona tus solicitudes de evaluación de seguridad.</p>

        {!me.user.email_verified ? (
          <div className="mt-4 flex flex-wrap items-center gap-2 rounded-xl border border-amber-400/30 bg-amber-400/10 px-4 py-2.5 text-sm text-amber-200">
            Tu email aún no está verificado.
            <button
              type="button"
              onClick={() => setTab("security")}
              className="font-semibold underline underline-offset-2 hover:text-amber-100"
            >
              Verificar ahora
            </button>
          </div>
        ) : null}

        {/* Tabs */}
        <div className="mt-6 flex gap-1 border-b border-white/10">
          <button
            type="button"
            onClick={() => setTab("requests")}
            className={`-mb-px border-b-2 px-4 py-2.5 text-sm font-medium transition-colors ${
              tab === "requests"
                ? "border-cyan-400 text-white"
                : "border-transparent text-slate-400 hover:text-slate-200"
            }`}
          >
            Solicitudes
          </button>
          {canManageTeam ? (
            <button
              type="button"
              onClick={() => setTab("team")}
              className={`-mb-px border-b-2 px-4 py-2.5 text-sm font-medium transition-colors ${
                tab === "team" ? "border-cyan-400 text-white" : "border-transparent text-slate-400 hover:text-slate-200"
              }`}
            >
              Equipo
            </button>
          ) : null}
          {canManageTeam ? (
            <button
              type="button"
              onClick={() => setTab("audit")}
              className={`-mb-px border-b-2 px-4 py-2.5 text-sm font-medium transition-colors ${
                tab === "audit"
                  ? "border-cyan-400 text-white"
                  : "border-transparent text-slate-400 hover:text-slate-200"
              }`}
            >
              Auditoría
            </button>
          ) : null}
          <button
            type="button"
            onClick={() => setTab("security")}
            className={`-mb-px border-b-2 px-4 py-2.5 text-sm font-medium transition-colors ${
              tab === "security"
                ? "border-cyan-400 text-white"
                : "border-transparent text-slate-400 hover:text-slate-200"
            }`}
          >
            Seguridad
          </button>
        </div>

        {tab === "team" && canManageTeam ? <TeamPanel me={me} /> : null}
        {tab === "audit" && canManageTeam ? <AuditPanel /> : null}
        {tab === "security" ? <SecurityPanel me={me} /> : null}

        {tab === "requests" ? (
          <div className="mt-6 grid gap-8 lg:grid-cols-[1fr_1.4fr]">
            {/* New request */}
            <section className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
              <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-300">Nueva solicitud</h2>
              <form onSubmit={create} className="mt-4 space-y-4">
                <Field
                  label="Título"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Evaluación web Q4"
                />
                <label className="block text-xs font-medium text-slate-400">
                  Alcance (opcional)
                  <textarea
                    value={scope}
                    onChange={(e) => setScope(e.target.value)}
                    rows={3}
                    placeholder="Dominios / sistemas autorizados…"
                    className="mt-1.5 w-full rounded-lg border border-white/15 bg-[#0a1324] px-3 py-2.5 text-sm text-white placeholder:text-slate-600 focus:border-cyan-400/60 focus:outline-none"
                  />
                </label>
                {err ? <p className="text-sm text-rose-400">{err}</p> : null}
                <PrimaryButton type="submit" disabled={creating || !title.trim()}>
                  {creating ? "Enviando…" : "Crear solicitud"}
                </PrimaryButton>
              </form>
            </section>

            {/* Requests list */}
            <section>
              <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-300">Tus solicitudes</h2>
              {reqs.length === 0 ? (
                <p className="mt-4 rounded-xl border border-white/10 bg-white/[0.02] p-5 text-sm text-slate-400">
                  Aún no tienes solicitudes. Crea la primera a la izquierda.
                </p>
              ) : (
                <ul className="mt-4 space-y-3">
                  {reqs.map((r) => (
                    <li key={r.id} className="rounded-xl border border-white/10 bg-white/[0.03] p-4">
                      <div className="flex items-center justify-between gap-3">
                        <span className="font-semibold text-white">{r.title}</span>
                        <span className="rounded-full border border-cyan-400/30 bg-cyan-400/10 px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-cyan-300">
                          {STATUS_LABEL[r.status] ?? r.status}
                        </span>
                      </div>
                      {r.scope ? <p className="mt-1 text-xs text-slate-400">{r.scope}</p> : null}
                      <div className="mt-1 flex items-center justify-between gap-2">
                        <p className="text-[11px] text-slate-500">
                          #{r.id} · {new Date(r.created_at).toLocaleString()}
                        </p>
                        <button
                          type="button"
                          onClick={() => setOpenResults((cur) => (cur === r.id ? null : r.id))}
                          className="text-[11px] font-medium text-cyan-300 hover:text-cyan-200"
                        >
                          {openResults === r.id ? "Ocultar resultados" : "Ver resultados"}
                        </button>
                      </div>
                      {openResults === r.id ? <ResultsPanel requestId={r.id} /> : null}
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </div>
        ) : null}

        <p className="mt-10 text-xs text-slate-500">
          ¿Prefieres hablar con el equipo?{" "}
          <Link href="/contact" className="inline-flex items-center gap-1 text-cyan-300 hover:text-cyan-200">
            Contacto <ArrowRightIcon className="size-3" />
          </Link>
        </p>
      </main>
    </div>
  );
}
