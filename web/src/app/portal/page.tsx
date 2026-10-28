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

const STATUS_TONE: Record<string, string> = {
  new: "bg-slate-100 text-slate-600 border-slate-200",
  scheduled: "bg-sky-50 text-sky-700 border-sky-200",
  in_progress: "bg-amber-50 text-amber-700 border-amber-200",
  done: "bg-emerald-50 text-emerald-700 border-emerald-200",
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
    return (
      <div className="grid min-h-screen place-items-center bg-gradient-to-b from-slate-50 to-[#eef3fa] text-sm text-slate-500">
        Cargando…
      </div>
    );
  }

  const canManageTeam = me.user.role === "owner" || me.user.role === "admin";
  const initial = (me.org.name || "?").trim().charAt(0).toUpperCase();

  const tabClass = (active: boolean) =>
    `-mb-px border-b-2 px-4 py-2.5 text-sm font-medium transition-colors ${
      active ? "border-cyan-500 text-slate-900" : "border-transparent text-slate-500 hover:text-slate-700"
    }`;

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-50 to-[#eef3fa] text-slate-800">
      <header className="sticky top-0 z-10 border-b border-slate-200/70 bg-white/85 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-5xl items-center justify-between px-5">
          <Link href="/" className="flex items-center" aria-label="NEXUS">
            <span className="rounded-lg bg-gradient-to-r from-[#0b1b3a] to-[#0a2540] px-3 py-1.5">
              {/* biome-ignore lint/performance/noImgElement: brand asset */}
              <img src="/brand/logo.webp" alt="NEXUS" className="h-5 w-auto" />
            </span>
          </Link>
          <div className="flex items-center gap-3 text-sm">
            <span className="flex items-center gap-2 rounded-full border border-slate-200 bg-white px-3 py-1.5">
              <span className="grid size-6 place-items-center rounded-full bg-gradient-to-br from-cyan-500 to-sky-500 text-xs font-bold text-white">
                {initial}
              </span>
              <span className="font-medium text-slate-700">{me.org.name}</span>
            </span>
            <button
              type="button"
              onClick={logout}
              className="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-800"
            >
              <LogOutIcon className="size-4" /> Salir
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-5xl px-5 py-10">
        <h1 className="text-2xl font-bold text-slate-900">Hola, {me.org.name} 👋</h1>
        <p className="mt-1 text-sm text-slate-500">Gestiona tus solicitudes de evaluación de seguridad.</p>

        {!me.user.email_verified ? (
          <div className="mt-4 flex flex-wrap items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm text-amber-800">
            Tu email aún no está verificado.
            <button
              type="button"
              onClick={() => setTab("security")}
              className="font-semibold text-amber-900 underline underline-offset-2 hover:text-amber-950"
            >
              Verificar ahora
            </button>
          </div>
        ) : null}

        {/* Tabs */}
        <div className="mt-6 flex gap-1 border-b border-slate-200">
          <button type="button" onClick={() => setTab("requests")} className={tabClass(tab === "requests")}>
            Solicitudes
          </button>
          {canManageTeam ? (
            <button type="button" onClick={() => setTab("team")} className={tabClass(tab === "team")}>
              Equipo
            </button>
          ) : null}
          {canManageTeam ? (
            <button type="button" onClick={() => setTab("audit")} className={tabClass(tab === "audit")}>
              Auditoría
            </button>
          ) : null}
          <button type="button" onClick={() => setTab("security")} className={tabClass(tab === "security")}>
            Seguridad
          </button>
        </div>

        {tab === "team" && canManageTeam ? <TeamPanel me={me} /> : null}
        {tab === "audit" && canManageTeam ? <AuditPanel /> : null}
        {tab === "security" ? <SecurityPanel me={me} /> : null}

        {tab === "requests" ? (
          <div className="mt-6 grid gap-8 lg:grid-cols-[1fr_1.4fr]">
            {/* New request */}
            <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
              <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-700">Nueva solicitud</h2>
              <form onSubmit={create} className="mt-4 space-y-4">
                <Field
                  label="Título"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Evaluación web Q4"
                />
                <label className="block text-xs font-medium text-slate-600">
                  Alcance (opcional)
                  <textarea
                    value={scope}
                    onChange={(e) => setScope(e.target.value)}
                    rows={3}
                    placeholder="Dominios / sistemas autorizados…"
                    className="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3.5 py-2.5 text-sm text-slate-800 placeholder:text-slate-400 transition focus:border-cyan-400 focus:bg-white focus:outline-none focus:ring-4 focus:ring-cyan-100"
                  />
                </label>
                {err ? <p className="text-sm text-rose-500">{err}</p> : null}
                <PrimaryButton type="submit" disabled={creating || !title.trim()}>
                  {creating ? "Enviando…" : "Crear solicitud"}
                </PrimaryButton>
              </form>
            </section>

            {/* Requests list */}
            <section>
              <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-700">Tus solicitudes</h2>
              {reqs.length === 0 ? (
                <p className="mt-4 rounded-xl border border-dashed border-slate-300 bg-white p-5 text-sm text-slate-500">
                  Aún no tienes solicitudes. Crea la primera a la izquierda.
                </p>
              ) : (
                <ul className="mt-4 space-y-3">
                  {reqs.map((r) => (
                    <li key={r.id} className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
                      <div className="flex items-center justify-between gap-3">
                        <span className="font-semibold text-slate-900">{r.title}</span>
                        <span
                          className={`rounded-full border px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${
                            STATUS_TONE[r.status] ?? "border-slate-200 bg-slate-100 text-slate-600"
                          }`}
                        >
                          {STATUS_LABEL[r.status] ?? r.status}
                        </span>
                      </div>
                      {r.scope ? <p className="mt-1 text-xs text-slate-500">{r.scope}</p> : null}
                      <div className="mt-1 flex items-center justify-between gap-2">
                        <p className="text-[11px] text-slate-400">
                          #{r.id} · {new Date(r.created_at).toLocaleString()}
                        </p>
                        <button
                          type="button"
                          onClick={() => setOpenResults((cur) => (cur === r.id ? null : r.id))}
                          className="text-[11px] font-medium text-cyan-600 hover:text-cyan-700"
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
          <Link href="/contact" className="inline-flex items-center gap-1 text-cyan-600 hover:text-cyan-700">
            Contacto <ArrowRightIcon className="size-3" />
          </Link>
        </p>
      </main>
    </div>
  );
}
