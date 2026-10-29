"use client";

import { type FormEvent, useCallback, useEffect, useState } from "react";

import Link from "next/link";
import { useRouter } from "next/navigation";

import { ArrowRightIcon, LogOutIcon } from "lucide-react";

import { AuditPanel } from "./_lib/audit";
import { isPortalLang, LanguageSwitcher, type PortalLang, usePortalI18n } from "./_lib/i18n";
import { OverviewPanel } from "./_lib/overview";
import {
  clearPortalToken,
  getPortalToken,
  type Me,
  type PortalRequest,
  portalApi,
  type ViewProfile,
} from "./_lib/portal";
import { ResultsPanel } from "./_lib/results";
import { SecurityPanel } from "./_lib/security";
import { TeamPanel } from "./_lib/team";
import { Field, PrimaryButton } from "./_lib/ui";

const STATUS_TONE: Record<string, string> = {
  new: "bg-slate-100 text-slate-600 border-slate-200",
  scheduled: "bg-sky-50 text-sky-700 border-sky-200",
  in_progress: "bg-amber-50 text-amber-700 border-amber-200",
  done: "bg-emerald-50 text-emerald-700 border-emerald-200",
};

type Tab = "overview" | "requests" | "team" | "audit" | "security";
type View = Exclude<ViewProfile, "">;

// Default perspective when the user has not chosen one: owners usually answer to
// the business (executive); everyone else is typically hands-on (technical).
function defaultView(me: Me): View {
  if (me.user.view_profile) return me.user.view_profile;
  return me.user.role === "owner" ? "executive" : "technical";
}

// Saving a UI preference is best-effort: the choice still applies in this session.
function ignorePrefError() {
  return undefined;
}

function prefersReducedMotion(): boolean {
  return typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

export default function PortalDashboard() {
  const router = useRouter();
  const { t, setLang, fmtDateTime } = usePortalI18n();
  const [me, setMe] = useState<Me | null>(null);
  const [view, setView] = useState<View>("executive");
  const [reqs, setReqs] = useState<PortalRequest[]>([]);
  const [tab, setTab] = useState<Tab>("overview");
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
      setView(defaultView(meData));
      // A language saved on the account wins over this browser's default.
      if (isPortalLang(meData.user.lang)) setLang(meData.user.lang);
      const { requests } = await portalApi.listRequests();
      setReqs(requests ?? []);
    } catch {
      router.replace("/portal/login");
    } finally {
      setLoading(false);
    }
  }, [router, setLang]);

  useEffect(() => {
    if (!getPortalToken()) {
      router.replace("/portal/login");
      return;
    }
    void load();
  }, [load, router]);

  // When a request is opened from elsewhere (e.g. an overview priority), bring it
  // into view once the requests tab has rendered.
  useEffect(() => {
    if (tab !== "requests" || openResults === null) return;
    document
      .getElementById(`request-${openResults}`)
      ?.scrollIntoView({ behavior: prefersReducedMotion() ? "auto" : "smooth", block: "start" });
  }, [tab, openResults]);

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
      setErr(ex instanceof Error ? ex.message : t.req.createError);
    } finally {
      setCreating(false);
    }
  }

  function changeView(v: View) {
    setView(v);
    // Persist so the choice follows the user; a failure only loses the preference.
    portalApi.setPreferences({ view_profile: v }).catch(ignorePrefError);
  }

  function changeLang(l: PortalLang) {
    portalApi.setPreferences({ lang: l }).catch(ignorePrefError);
  }

  function openRequest(id: number) {
    setOpenResults(id);
    setTab("requests");
  }

  function logout() {
    clearPortalToken();
    router.replace("/portal/login");
  }

  if (loading || !me) {
    return (
      <div className="grid min-h-screen place-items-center bg-gradient-to-b from-slate-50 to-[#eef3fa] text-sm text-slate-500">
        {t.loading}
      </div>
    );
  }

  const canManageTeam = me.user.role === "owner" || me.user.role === "admin";
  const initial = (me.org.name || "?").trim().charAt(0).toUpperCase();

  const tabs: { id: Tab; label: string; show: boolean }[] = [
    { id: "overview", label: t.tabs.overview, show: true },
    { id: "requests", label: t.tabs.requests, show: true },
    { id: "team", label: t.tabs.team, show: canManageTeam },
    { id: "audit", label: t.tabs.audit, show: canManageTeam },
    { id: "security", label: t.tabs.security, show: true },
  ];

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-50 to-[#eef3fa] text-slate-800">
      <header className="sticky top-0 z-10 border-b border-slate-200/70 bg-white/85 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-3 px-4 sm:px-5">
          <Link href="/" className="flex shrink-0 items-center" aria-label="NEXUS">
            <span className="rounded-lg bg-gradient-to-r from-[#0b1b3a] to-[#0a2540] px-3 py-1.5">
              {/* biome-ignore lint/performance/noImgElement: brand asset */}
              <img src="/brand/logo.webp" alt="NEXUS" className="h-5 w-auto" />
            </span>
          </Link>
          <div className="flex min-w-0 items-center gap-2 text-sm sm:gap-3">
            <LanguageSwitcher onChange={changeLang} />
            <span className="flex min-w-0 items-center gap-2 rounded-full border border-slate-200 bg-white px-2 py-1.5 sm:px-3">
              <span className="grid size-6 shrink-0 place-items-center rounded-full bg-gradient-to-br from-cyan-700 to-sky-700 text-xs font-bold text-white">
                {initial}
              </span>
              <span className="hidden truncate font-medium text-slate-700 sm:inline">{me.org.name}</span>
            </span>
            <button
              type="button"
              onClick={logout}
              className="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400"
            >
              <LogOutIcon className="size-4" aria-hidden /> <span className="hidden sm:inline">{t.logout}</span>
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-8 sm:px-5 sm:py-10">
        <h1 className="text-2xl font-bold text-slate-900">{t.hello(me.org.name)} 👋</h1>
        <p className="mt-1 text-sm text-slate-500">{t.subtitle}</p>

        {!me.user.email_verified ? (
          <div className="mt-4 flex flex-wrap items-center gap-2 rounded-xl border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm text-amber-800">
            {t.emailUnverified}
            <button
              type="button"
              onClick={() => setTab("security")}
              className="font-semibold text-amber-900 underline underline-offset-2 hover:text-amber-950"
            >
              {t.verifyNow}
            </button>
          </div>
        ) : null}

        <div
          role="tablist"
          className="mt-6 flex gap-1 overflow-x-auto overflow-y-hidden border-b border-slate-200 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
        >
          {tabs
            .filter((x) => x.show)
            .map((x) => (
              <button
                key={x.id}
                type="button"
                role="tab"
                aria-selected={tab === x.id}
                onClick={() => setTab(x.id)}
                className={`-mb-px shrink-0 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-cyan-400 ${
                  tab === x.id
                    ? "border-cyan-500 text-slate-900"
                    : "border-transparent text-slate-500 hover:text-slate-700"
                }`}
              >
                {x.label}
              </button>
            ))}
        </div>

        {tab === "overview" ? (
          <OverviewPanel
            me={me}
            view={view}
            onViewChange={changeView}
            onOpenRequest={openRequest}
            onGoTab={(x) => setTab(x)}
          />
        ) : null}
        {tab === "team" && canManageTeam ? <TeamPanel me={me} /> : null}
        {tab === "audit" && canManageTeam ? <AuditPanel /> : null}
        {tab === "security" ? <SecurityPanel me={me} /> : null}

        {tab === "requests" ? (
          <div className="mt-6 grid gap-8 lg:grid-cols-[1fr_1.4fr]">
            <section className="h-fit rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
              <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-700">{t.req.newRequest}</h2>
              <form onSubmit={create} className="mt-4 space-y-4">
                <Field
                  label={t.req.titleLabel}
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder={t.req.titlePlaceholder}
                />
                <label className="block text-xs font-medium text-slate-600">
                  {t.req.scopeLabel}
                  <textarea
                    value={scope}
                    onChange={(e) => setScope(e.target.value)}
                    rows={3}
                    placeholder={t.req.scopePlaceholder}
                    className="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3.5 py-2.5 text-sm text-slate-800 placeholder:text-slate-400 transition focus:border-cyan-400 focus:bg-white focus:outline-none focus:ring-4 focus:ring-cyan-100"
                  />
                </label>
                {err ? (
                  <p role="alert" className="text-sm text-rose-600">
                    {err}
                  </p>
                ) : null}
                <PrimaryButton type="submit" disabled={creating || !title.trim()}>
                  {creating ? t.req.sending : t.req.create}
                </PrimaryButton>
              </form>
            </section>

            <section>
              <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-700">{t.req.yours}</h2>
              {reqs.length === 0 ? (
                <p className="mt-4 rounded-xl border border-dashed border-slate-300 bg-white p-5 text-sm text-slate-500">
                  {t.req.none}
                </p>
              ) : (
                <ul className="mt-4 space-y-3">
                  {reqs.map((r) => (
                    <li
                      key={r.id}
                      id={`request-${r.id}`}
                      className={`scroll-mt-24 rounded-xl border bg-white p-4 shadow-sm ${
                        openResults === r.id ? "border-cyan-300 ring-4 ring-cyan-50" : "border-slate-200"
                      }`}
                    >
                      <div className="flex items-center justify-between gap-3">
                        <span className="font-semibold text-slate-900">{r.title}</span>
                        <span
                          className={`shrink-0 rounded-full border px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${
                            STATUS_TONE[r.status] ?? "border-slate-200 bg-slate-100 text-slate-600"
                          }`}
                        >
                          {t.requestStatus[r.status] ?? r.status}
                        </span>
                      </div>
                      {r.scope ? <p className="mt-1 text-xs text-slate-500">{r.scope}</p> : null}
                      <div className="mt-1 flex items-center justify-between gap-2">
                        <p className="text-[11px] text-slate-500">
                          #{r.id} · {fmtDateTime(r.created_at)}
                        </p>
                        <button
                          type="button"
                          aria-expanded={openResults === r.id}
                          onClick={() => setOpenResults((cur) => (cur === r.id ? null : r.id))}
                          className="rounded text-[11px] font-medium text-cyan-700 hover:text-cyan-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400"
                        >
                          {openResults === r.id ? t.req.hideResults : t.req.showResults}
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
          {t.contactPrompt}{" "}
          <Link href="/contact" className="inline-flex items-center gap-1 text-cyan-700 hover:text-cyan-800">
            {t.contact} <ArrowRightIcon className="size-3" aria-hidden />
          </Link>
        </p>
      </main>
    </div>
  );
}
