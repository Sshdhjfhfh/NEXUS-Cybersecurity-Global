"use client";

import { createContext, type ReactNode, useCallback, useContext, useEffect, useMemo, useState } from "react";

// Client-portal i18n (ES/EN). The English dictionary is typed as `typeof es`, so
// a missing or mismatched translation is a compile error — the portal never ships
// half-translated. Dates and numbers go through Intl with the active locale.

export type PortalLang = "es" | "en";
const STORAGE_KEY = "nexus_portal_lang";

const es = {
  langName: "Español",
  loading: "Cargando…",
  retry: "Reintentar",
  refresh: "Actualizar",
  logout: "Salir",
  hello: (org: string) => `Hola, ${org}`,
  subtitle: "Tu postura de seguridad y tus evaluaciones, en un solo lugar.",
  emailUnverified: "Tu email aún no está verificado.",
  verifyNow: "Verificar ahora",
  contactPrompt: "¿Prefieres hablar con el equipo?",
  contact: "Contacto",
  languageLabel: "Idioma",

  tabs: {
    overview: "Resumen",
    requests: "Evaluaciones",
    team: "Equipo",
    audit: "Auditoría",
    security: "Seguridad",
  },

  view: {
    label: "Vista",
    executive: "Ejecutiva",
    technical: "Técnica",
    executiveHint: "Riesgo y prioridades en lenguaje de negocio",
    technicalHint: "Detalle por severidad, estado y tendencia",
  },

  sev: { critical: "Crítica", high: "Alta", medium: "Media", low: "Baja", unrated: "Sin clasificar" },
  findingStatus: {
    confirmed: "Abierta",
    in_progress: "En corrección",
    resolved: "Resuelta",
    fixed: "Corregida",
    risk_accepted: "Riesgo aceptado",
  } as Record<string, string>,
  requestStatus: {
    new: "Recibida",
    scheduled: "Programada",
    in_progress: "En curso",
    done: "Completada",
  } as Record<string, string>,

  req: {
    newRequest: "Nueva solicitud de evaluación",
    titleLabel: "Título",
    titlePlaceholder: "Evaluación web Q4",
    scopeLabel: "Alcance (opcional)",
    scopePlaceholder: "Dominios / sistemas autorizados…",
    createError: "No se pudo crear la solicitud",
    sending: "Enviando…",
    create: "Crear solicitud",
    yours: "Tus evaluaciones",
    none: "Aún no tienes solicitudes. Crea la primera aquí.",
    showResults: "Ver resultados",
    hideResults: "Ocultar resultados",
  },

  ov: {
    updated: (when: string) => `Actualizado ${when}`,
    loadError: "No se pudo cargar el resumen.",
    exposureTitle: "Nivel de exposición actual",
    exposure: {
      critical: "Crítico",
      high: "Alto",
      medium: "Moderado",
      low: "Bajo",
      none: "Sin riesgos abiertos",
    } as Record<string, string>,
    exposureLine: (parts: string) => `Tienes ${parts} pendientes de corregir.`,
    exposureNone: "No hay hallazgos abiertos en tus evaluaciones publicadas.",
    countBySev: (n: number, sev: string, unrated: boolean) =>
      unrated
        ? `${n} ${n === 1 ? "hallazgo sin clasificar" : "hallazgos sin clasificar"}`
        : `${n} ${n === 1 ? "hallazgo" : "hallazgos"} de severidad ${sev}`,
    howCalculated: "¿Cómo se calcula?",
    methodology:
      "El nivel corresponde a la severidad más alta entre los hallazgos abiertos (abiertos o en corrección) de las evaluaciones cuyos resultados ya publicamos. No es una puntuación estimada: solo cuenta hallazgos verificados por nuestro equipo.",
    openFindings: "Hallazgos abiertos",
    openFindingsHint: (inProgress: number) => `${inProgress} en corrección`,
    remediation: "Avance de remediación",
    remediationOf: (done: number, total: number) => `${done} de ${total} corregidos`,
    remediationMethod:
      "Corregidos ÷ (abiertos + en corrección + corregidos). Los riesgos aceptados se excluyen del cálculo.",
    evaluations: "Evaluaciones",
    evaluationsHint: (active: number, done: number) => `${active} activas · ${done} completadas`,
    priorities: "Prioridades",
    prioritiesHint: "Lo más importante que corregir primero",
    noPriorities: "No hay hallazgos abiertos. Buen trabajo.",
    viewDetail: "Ver detalle",
    inEvaluation: (title: string) => `Evaluación: ${title}`,
    openSince: (days: number) => (days <= 0 ? "abierta hoy" : `abierta hace ${days} ${days === 1 ? "día" : "días"}`),
    accountSecurity: "Seguridad de las cuentas",
    mfaCoverage: (on: number, total: number) => `${on} de ${total} usuarios con 2FA activada`,
    unverified: (n: number) => `${n} ${n === 1 ? "email sin verificar" : "emails sin verificar"}`,
    pendingInvites: (n: number) => `${n} ${n === 1 ? "invitación pendiente" : "invitaciones pendientes"}`,
    allGood: "Todo en orden",
    manageTeam: "Gestionar equipo",
    emptyTitle: "Aún no hay resultados publicados",
    emptyBody:
      "Cuando una evaluación termine y nuestro equipo publique sus resultados, aquí verás tu nivel de exposición, las prioridades y el avance de la remediación.",
    emptyCta: "Solicitar una evaluación",
    bySeverity: "Hallazgos por severidad y estado",
    col: {
      severity: "Severidad",
      open: "Abiertas",
      inProgress: "En corrección",
      remediated: "Corregidas",
      riskAccepted: "Riesgo aceptado",
      total: "Total",
      finding: "Hallazgo",
      type: "Tipo",
      status: "Estado",
      evaluation: "Evaluación",
      age: "Antigüedad",
      month: "Mes",
      count: "Hallazgos",
    },
    trend: "Hallazgos publicados por mes",
    trendHint: "Según la fecha de detección (últimos 6 meses, UTC)",
    topOpen: "Hallazgos abiertos prioritarios",
    days: (n: number) => `${n} d`,
    evaluationsByStatus: "Evaluaciones por estado",
    activity: "Actividad reciente",
    noActivity: "Sin actividad registrada.",
    seeAudit: "Ver auditoría completa",
  },
};

export type PortalDict = typeof es;

const en: PortalDict = {
  langName: "English",
  loading: "Loading…",
  retry: "Retry",
  refresh: "Refresh",
  logout: "Sign out",
  hello: (org: string) => `Hello, ${org}`,
  subtitle: "Your security posture and assessments, in one place.",
  emailUnverified: "Your email address is not verified yet.",
  verifyNow: "Verify now",
  contactPrompt: "Prefer to talk to the team?",
  contact: "Contact",
  languageLabel: "Language",

  tabs: {
    overview: "Overview",
    requests: "Assessments",
    team: "Team",
    audit: "Audit log",
    security: "Security",
  },

  view: {
    label: "View",
    executive: "Executive",
    technical: "Technical",
    executiveHint: "Risk and priorities in business terms",
    technicalHint: "Breakdown by severity, status and trend",
  },

  sev: { critical: "Critical", high: "High", medium: "Medium", low: "Low", unrated: "Unrated" },
  findingStatus: {
    confirmed: "Open",
    in_progress: "Being fixed",
    resolved: "Resolved",
    fixed: "Fixed",
    risk_accepted: "Risk accepted",
  },
  requestStatus: {
    new: "Received",
    scheduled: "Scheduled",
    in_progress: "In progress",
    done: "Completed",
  },

  req: {
    newRequest: "New assessment request",
    titleLabel: "Title",
    titlePlaceholder: "Q4 web assessment",
    scopeLabel: "Scope (optional)",
    scopePlaceholder: "Authorized domains / systems…",
    createError: "The request could not be created",
    sending: "Sending…",
    create: "Create request",
    yours: "Your assessments",
    none: "You have no requests yet. Create the first one here.",
    showResults: "View results",
    hideResults: "Hide results",
  },

  ov: {
    updated: (when: string) => `Updated ${when}`,
    loadError: "The overview could not be loaded.",
    exposureTitle: "Current exposure level",
    exposure: {
      critical: "Critical",
      high: "High",
      medium: "Moderate",
      low: "Low",
      none: "No open risks",
    },
    exposureLine: (parts: string) => `You have ${parts} still to fix.`,
    exposureNone: "There are no open findings in your published assessments.",
    countBySev: (n: number, sev: string, unrated: boolean) =>
      unrated
        ? `${n} unrated ${n === 1 ? "finding" : "findings"}`
        : `${n} ${sev.toLowerCase()}-severity ${n === 1 ? "finding" : "findings"}`,
    howCalculated: "How is this calculated?",
    methodology:
      "The level is the highest severity among open findings (open or being fixed) in the assessments whose results we have published. It is not an estimated score: only findings verified by our team are counted.",
    openFindings: "Open findings",
    openFindingsHint: (inProgress: number) => `${inProgress} being fixed`,
    remediation: "Remediation progress",
    remediationOf: (done: number, total: number) => `${done} of ${total} fixed`,
    remediationMethod: "Fixed ÷ (open + being fixed + fixed). Accepted risks are excluded from the calculation.",
    evaluations: "Assessments",
    evaluationsHint: (active: number, done: number) => `${active} active · ${done} completed`,
    priorities: "Priorities",
    prioritiesHint: "What to fix first",
    noPriorities: "No open findings. Well done.",
    viewDetail: "View details",
    inEvaluation: (title: string) => `Assessment: ${title}`,
    openSince: (days: number) => (days <= 0 ? "opened today" : `open for ${days} ${days === 1 ? "day" : "days"}`),
    accountSecurity: "Account security",
    mfaCoverage: (on: number, total: number) => `${on} of ${total} users have 2FA enabled`,
    unverified: (n: number) => `${n} unverified ${n === 1 ? "email" : "emails"}`,
    pendingInvites: (n: number) => `${n} pending ${n === 1 ? "invitation" : "invitations"}`,
    allGood: "All good",
    manageTeam: "Manage team",
    emptyTitle: "No published results yet",
    emptyBody:
      "Once an assessment finishes and our team publishes its results, you will see your exposure level, priorities and remediation progress here.",
    emptyCta: "Request an assessment",
    bySeverity: "Findings by severity and status",
    col: {
      severity: "Severity",
      open: "Open",
      inProgress: "Being fixed",
      remediated: "Fixed",
      riskAccepted: "Risk accepted",
      total: "Total",
      finding: "Finding",
      type: "Type",
      status: "Status",
      evaluation: "Assessment",
      age: "Age",
      month: "Month",
      count: "Findings",
    },
    trend: "Published findings per month",
    trendHint: "By detection date (last 6 months, UTC)",
    topOpen: "Top open findings",
    days: (n: number) => `${n} d`,
    evaluationsByStatus: "Assessments by status",
    activity: "Recent activity",
    noActivity: "No activity recorded.",
    seeAudit: "See full audit log",
  },
};

const DICTS: Record<PortalLang, PortalDict> = { es, en };
const LOCALE: Record<PortalLang, string> = { es: "es-ES", en: "en-US" };

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
    document.documentElement.lang = lang;
  }, [lang]);

  const setLang = useCallback((l: PortalLang) => {
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
    const dt = new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" });
    const month = new Intl.DateTimeFormat(locale, { month: "short", year: "numeric", timeZone: "UTC" });
    const rel = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
    return {
      lang,
      t: DICTS[lang],
      setLang,
      fmtNumber: (n) => num.format(n),
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
