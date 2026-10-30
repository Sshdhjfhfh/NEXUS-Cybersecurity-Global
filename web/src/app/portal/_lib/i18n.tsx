"use client";

import { createContext, type ReactNode, useCallback, useContext, useEffect, useMemo, useState } from "react";

import en from "./locales/en";
import es, { type PortalDict } from "./locales/es";
import { SERVER_ERROR_PATTERNS_EN, SERVER_ERRORS_EN, SERVER_ERRORS_INTERNAL } from "./locales/server-errors";

// Client-portal i18n (ES/EN). Dictionaries live in ./locales; the English one is
// typed against the Spanish one, so a missing or mismatched translation is a
// compile error — the portal never ships half-translated. Dates and numbers go
// through Intl with the active locale.

export type { PortalDict };
export type PortalLang = "es" | "en";
const STORAGE_KEY = "nexus_portal_lang";

const DICTS: Record<PortalLang, PortalDict> = { es, en };
const LOCALE: Record<PortalLang, string> = { es: "es-ES", en: "en-US" };

// Module-level mirror of the active language, so non-React code (the API client)
// can localize server errors without threading the language through every call.
let activeLang: PortalLang = "es";

const SERVICE_UNAVAILABLE: Record<PortalLang, string> = {
  es: "El servicio no está disponible en este momento. Inténtalo de nuevo en unos minutos.",
  en: "The service is temporarily unavailable. Please try again in a few minutes.",
};

// localizeServerError turns a server error message into one fit for the user in
// the active language. Sentinels the UI branches on pass through untouched.
export function localizeServerError(msg: string): string {
  if (SERVER_ERRORS_INTERNAL.has(msg)) return SERVICE_UNAVAILABLE[activeLang];
  const text = activeLang === "en" ? (SERVER_ERRORS_EN[msg] ?? translatePattern(msg)) : msg;
  // Server messages are lowercase fragments; present them as sentences. Single-
  // token sentinels (mfa_required) are left alone; real sentences may contain "_"
  // in a value they quote (a DNS name).
  return text && (text.includes(" ") || !text.includes("_")) ? text.charAt(0).toUpperCase() + text.slice(1) : text;
}

function translatePattern(msg: string): string {
  for (const [re, en] of SERVER_ERROR_PATTERNS_EN) {
    if (re.test(msg)) return msg.replace(re, en);
  }
  return msg;
}

export function isPortalLang(v: unknown): v is PortalLang {
  return v === "es" || v === "en";
}

function detectLang(): PortalLang {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (isPortalLang(stored)) return stored;
  } catch {
    /* storage unavailable */
  }
  if (typeof navigator !== "undefined" && navigator.language?.toLowerCase().startsWith("en")) return "en";
  return "es";
}

type Ctx = {
  lang: PortalLang;
  t: PortalDict;
  setLang: (l: PortalLang) => void;
  fmtNumber: (n: number) => string;
  fmtDate: (iso: string) => string;
  fmtDay: (yyyyMmDd: string) => string;
  fmtDateTime: (iso: string) => string;
  fmtMonth: (yyyyMm: string) => string;
  fmtRelative: (iso: string) => string;
};

const I18nContext = createContext<Ctx | null>(null);

export function PortalI18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<PortalLang>("es");

  // Resolve the stored/browser language after mount (SSR always renders "es").
  useEffect(() => {
    setLangState(detectLang());
  }, []);

  useEffect(() => {
    activeLang = lang;
    document.documentElement.lang = lang;
  }, [lang]);

  const setLang = useCallback((l: PortalLang) => {
    activeLang = l;
    setLangState(l);
    try {
      localStorage.setItem(STORAGE_KEY, l);
    } catch {
      /* storage unavailable */
    }
  }, []);

  const value = useMemo<Ctx>(() => {
    const locale = LOCALE[lang];
    const num = new Intl.NumberFormat(locale);
    const date = new Intl.DateTimeFormat(locale, { dateStyle: "medium" });
    // Calendar days (YYYY-MM-DD, e.g. due dates) are zone-less: format them in UTC
    // so they never shift a day in timezones behind UTC.
    const day = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeZone: "UTC" });
    const dt = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" });
    const month = new Intl.DateTimeFormat(locale, { month: "short", year: "numeric", timeZone: "UTC" });
    const rel = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
    return {
      lang,
      t: DICTS[lang],
      setLang,
      fmtNumber: (n) => num.format(n),
      fmtDate: (iso) => date.format(new Date(iso)),
      fmtDay: (d) => day.format(new Date(`${d}T00:00:00Z`)),
      fmtDateTime: (iso) => dt.format(new Date(iso)),
      fmtMonth: (m) => {
        const [y, mo] = m.split("-").map(Number);
        return month.format(new Date(Date.UTC(y, mo - 1, 1)));
      },
      fmtRelative: (iso) => {
        const secs = Math.round((new Date(iso).getTime() - Date.now()) / 1000);
        const abs = Math.abs(secs);
        if (abs < 60) return rel.format(secs, "second");
        if (abs < 3600) return rel.format(Math.round(secs / 60), "minute");
        if (abs < 86400) return rel.format(Math.round(secs / 3600), "hour");
        return rel.format(Math.round(secs / 86400), "day");
      },
    };
  }, [lang, setLang]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function usePortalI18n(): Ctx {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("usePortalI18n must be used inside PortalI18nProvider");
  return ctx;
}

// LanguageSwitcher toggles ES/EN. onChange lets a signed-in page also persist the
// choice server-side so it follows the user to other devices.
export function LanguageSwitcher({ onChange }: { onChange?: (l: PortalLang) => void }) {
  const { lang, setLang, t } = usePortalI18n();
  return (
    <fieldset className="inline-flex rounded-lg border border-slate-200 bg-white p-0.5 text-xs">
      <legend className="sr-only">{t.languageLabel}</legend>
      {(["es", "en"] as PortalLang[]).map((l) => (
        <button
          key={l}
          type="button"
          aria-pressed={lang === l}
          lang={l}
          aria-label={DICTS[l].langName}
          onClick={() => {
            setLang(l);
            onChange?.(l);
          }}
          className={`rounded-md px-2 py-1 font-semibold uppercase transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-400 ${
            lang === l ? "bg-slate-900 text-white" : "text-slate-500 hover:text-slate-800"
          }`}
        >
          {l}
        </button>
      ))}
    </fieldset>
  );
}
