"use client";

import { useEffect, useState } from "react";

import Link from "next/link";

import { useSiteI18n } from "./i18n";

// GDPR/ePrivacy-style cookie consent. Optional categories default to OFF (no
// pre-ticked boxes). The choice is stored locally with a version + timestamp so
// it can be re-asked when the policy changes. No non-essential scripts are
// loaded before consent (there are none yet; add them gated on these prefs).
const KEY = "nexus_cookie_consent";
const VERSION = 1;

export type CookiePrefs = { necessary: true; analytics: boolean; marketing: boolean };

export function openCookiePreferences() {
  try {
    window.dispatchEvent(new Event("nexus:open-cookies"));
  } catch {
    /* ignore */
  }
}

const T = {
  en: {
    title: "We value your privacy",
    body: "We use necessary cookies to run this site. With your consent we also use optional cookies for analytics and marketing. You can change your choice anytime.",
    necessary: "Necessary",
    necessaryD: "Required for the site to work. Always on.",
    analytics: "Analytics",
    analyticsD: "Help us understand how the site is used.",
    marketing: "Marketing",
    marketingD: "Used to measure and improve campaigns.",
    acceptAll: "Accept all",
    rejectAll: "Reject optional",
    save: "Save preferences",
    prefs: "Manage preferences",
    policy: "Cookie Policy",
    always: "Always active",
  },
  es: {
    title: "Valoramos tu privacidad",
    body: "Usamos cookies necesarias para que el sitio funcione. Con tu consentimiento también usamos cookies opcionales de analítica y marketing. Puedes cambiar tu elección cuando quieras.",
    necessary: "Necesarias",
    necessaryD: "Imprescindibles para que el sitio funcione. Siempre activas.",
    analytics: "Analítica",
    analyticsD: "Nos ayudan a entender cómo se usa el sitio.",
    marketing: "Marketing",
    marketingD: "Se usan para medir y mejorar campañas.",
    acceptAll: "Aceptar todo",
    rejectAll: "Rechazar opcionales",
    save: "Guardar preferencias",
    prefs: "Gestionar preferencias",
    policy: "Política de Cookies",
    always: "Siempre activas",
  },
};

export function CookieConsent() {
  const { lang } = useSiteI18n();
  const t = T[lang];
  const [open, setOpen] = useState(false);
  const [showPrefs, setShowPrefs] = useState(false);
  const [analytics, setAnalytics] = useState(false);
  const [marketing, setMarketing] = useState(false);

  useEffect(() => {
    let stored: string | null = null;
    try {
      stored = localStorage.getItem(KEY);
    } catch {
      /* ignore */
    }
    if (!stored) setOpen(true);
    const handler = () => {
      try {
        const cur = JSON.parse(localStorage.getItem(KEY) ?? "{}");
        setAnalytics(!!cur.analytics);
        setMarketing(!!cur.marketing);
      } catch {
        /* ignore */
      }
      setShowPrefs(true);
      setOpen(true);
    };
    window.addEventListener("nexus:open-cookies", handler);
    return () => window.removeEventListener("nexus:open-cookies", handler);
  }, []);

  function persist(prefs: CookiePrefs) {
    try {
      localStorage.setItem(KEY, JSON.stringify({ ...prefs, version: VERSION, date: new Date().toISOString() }));
    } catch {
      /* ignore */
    }
    setOpen(false);
    setShowPrefs(false);
  }

  if (!open) return null;

  return (
    <div className="fixed inset-x-0 bottom-0 z-[60] p-4">
      <div className="mx-auto max-w-3xl rounded-2xl border border-cyan-400/20 bg-[#0a1324]/95 p-5 shadow-2xl backdrop-blur-md">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-sm font-bold text-white">{t.title}</h2>
            <p className="mt-1.5 text-xs leading-relaxed text-slate-400">
              {t.body}{" "}
              <Link href="/legal/cookies" className="text-cyan-300 hover:text-cyan-200">
                {t.policy}
              </Link>
            </p>
          </div>
        </div>

        {showPrefs ? (
          <div className="mt-4 space-y-2.5">
            <Row title={t.necessary} desc={t.necessaryD} alwaysLabel={t.always} />
            <Toggle title={t.analytics} desc={t.analyticsD} checked={analytics} onChange={setAnalytics} />
            <Toggle title={t.marketing} desc={t.marketingD} checked={marketing} onChange={setMarketing} />
          </div>
        ) : null}

        <div className="mt-4 flex flex-wrap items-center justify-end gap-2.5">
          {!showPrefs ? (
            <button
              type="button"
              onClick={() => setShowPrefs(true)}
              className="mr-auto text-xs font-medium text-slate-300 underline-offset-2 hover:text-white hover:underline"
            >
              {t.prefs}
            </button>
          ) : null}
          <button
            type="button"
            onClick={() => persist({ necessary: true, analytics: false, marketing: false })}
            className="rounded-lg border border-white/15 px-4 py-2 text-xs font-semibold text-white transition-colors hover:border-cyan-400/50"
          >
            {t.rejectAll}
          </button>
          {showPrefs ? (
            <button
              type="button"
              onClick={() => persist({ necessary: true, analytics, marketing })}
              className="rounded-lg border border-white/15 px-4 py-2 text-xs font-semibold text-white transition-colors hover:border-cyan-400/50"
            >
              {t.save}
            </button>
          ) : null}
          <button
            type="button"
            onClick={() => persist({ necessary: true, analytics: true, marketing: true })}
            className="rounded-lg bg-gradient-to-r from-cyan-400 to-cyan-500 px-4 py-2 text-xs font-semibold text-[#060b18]"
          >
            {t.acceptAll}
          </button>
        </div>
      </div>
    </div>
  );
}

function Row({ title, desc, alwaysLabel }: { title: string; desc: string; alwaysLabel: string }) {
  return (
    <div className="flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-white/[0.02] px-4 py-3">
      <div>
        <div className="text-xs font-semibold text-white">{title}</div>
        <div className="text-[11px] text-slate-500">{desc}</div>
      </div>
      <span className="shrink-0 text-[10px] font-semibold uppercase tracking-wide text-cyan-300">{alwaysLabel}</span>
    </div>
  );
}

function Toggle({
  title,
  desc,
  checked,
  onChange,
}: {
  title: string;
  desc: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex cursor-pointer items-center justify-between gap-4 rounded-lg border border-white/10 bg-white/[0.02] px-4 py-3">
      <div>
        <div className="text-xs font-semibold text-white">{title}</div>
        <div className="text-[11px] text-slate-500">{desc}</div>
      </div>
      <span
        className={`relative inline-flex h-5 w-9 shrink-0 items-center rounded-full transition-colors ${
          checked ? "bg-cyan-400" : "bg-slate-600"
        }`}
      >
        <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} className="sr-only" />
        <span
          className={`inline-block size-4 rounded-full bg-white transition-transform ${checked ? "translate-x-4" : "translate-x-0.5"}`}
        />
      </span>
    </label>
  );
}
