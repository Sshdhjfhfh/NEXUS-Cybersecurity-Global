"use client";

import { createContext, type ReactNode, useCallback, useContext, useEffect, useState } from "react";

export type SiteLang = "es" | "en";

// Self-contained dictionary for the public marketing site, independent of the
// console's i18n catalog. Keeps the site shippable without touching the large
// shared catalogs.
const DICT: Record<SiteLang, Record<string, string>> = {
  es: {
    "nav.platform": "Plataforma",
    "nav.solutions": "Soluciones",
    "nav.services": "Servicios",
    "nav.research": "Investigación",
    "nav.demo": "Solicitar demo",
    "nav.console": "Consola",

    "hero.badge": "Plataforma de ciberseguridad con agentes de IA",
    "hero.title": "Seguridad ofensiva y defensiva, orquestada por agentes autónomos y gobernada por humanos.",
    "hero.subtitle":
      "NEXUS combina reconocimiento, detección de vulnerabilidades y detección defensiva (Sigma/ATT&CK) dentro de límites de autorización verificables. Cada acción pasa por un control de alcance firmado.",
    "hero.cta.demo": "Solicitar una demo",
    "hero.cta.solutions": "Ver soluciones",
    "hero.note": "Funciones de investigación marcadas como Research/Beta. Sin clientes ni cifras inventadas.",

    "pillars.title": "Un sistema, tres frentes",
    "pillars.red.title": "Ofensiva autorizada",
    "pillars.red.body":
      "Recon (subfinder/katana) → escaneo web (httpx/nuclei) → SCA de dependencias y secretos, todo dentro del alcance firmado. Detecta y reporta; no explota.",
    "pillars.blue.title": "Defensiva",
    "pillars.blue.body":
      "Detección pasiva con reglas Sigma y etiquetado MITRE ATT&CK, correlación por ventanas y evolución verificada de detecciones.",
    "pillars.research.title": "Investigación en IA",
    "pillars.research.body":
      "Laboratorio de evolución de agentes por generaciones, con evaluador independiente. Beta — resultados reproducibles, sin promesas.",

    "trust.title": "Autorización antes que capacidad",
    "trust.body":
      "El alcance de cada evaluación se aplica FUERA del modelo. Un objetivo o ruta fuera del alcance firmado aborta la ejecución. Ningún agente amplía su propio permiso.",
    "trust.badge1": "Alcance firmado",
    "trust.badge2": "Auditoría de acciones",
    "trust.badge3": "Sin propagación autónoma",

    "cta.title": "¿Listo para una evaluación controlada?",
    "cta.body": "Cuéntanos tu organización y el servicio de interés. Te contactamos para un piloto.",
    "cta.button": "Solicitar demo",

    "footer.tagline": "Ciberseguridad con agentes de IA, gobernada por humanos.",
    "footer.rights": "Todos los derechos reservados.",
    "footer.note":
      "NEXUS es una plataforma en desarrollo. Las funciones experimentales se identifican como Research/Beta.",

    "solutions.title": "Soluciones",
    "solutions.subtitle": "Capacidades reales, dentro de límites verificables.",
    "sol.offensive.t": "Seguridad ofensiva",
    "sol.offensive.b":
      "Descubrimiento de superficie, evaluación web, SCA de dependencias y detección de secretos. Hallazgos listos para remediar, con alcance autorizado.",
    "sol.defensive.t": "Seguridad defensiva",
    "sol.defensive.b":
      "Detección Sigma con ATT&CK sobre telemetría, correlación y reducción de ruido. Pensado para ingeniería de detección.",
    "sol.purple.t": "Purple team",
    "sol.purple.b":
      "Ciclo de simulación y validación de detecciones en laboratorio autorizado. En construcción (Beta).",
    "sol.research.t": "Investigación en IA",
    "sol.research.b": "Evolución de estrategias de agentes por generaciones con evaluación independiente. Research.",

    "form.title": "Solicitar una demo",
    "form.subtitle": "Completa el formulario y te contactaremos.",
    "form.org": "Organización",
    "form.name": "Nombre de contacto",
    "form.email": "Email corporativo",
    "form.country": "País",
    "form.service": "Servicio de interés",
    "form.message": "Mensaje",
    "form.consent": "Acepto que NEXUS procese estos datos para responder a mi solicitud.",
    "form.submit": "Enviar solicitud",
    "form.sending": "Enviando…",
    "form.ok": "¡Solicitud enviada! Te contactaremos pronto.",
    "form.err": "No se pudo enviar. Revisa los campos e inténtalo de nuevo.",
    "form.required": "Organización, nombre y email son obligatorios.",
    "svc.offensive": "Seguridad ofensiva",
    "svc.defensive": "Seguridad defensiva",
    "svc.purple": "Purple team",
    "svc.ai-research": "Investigación en IA",
    "svc.managed": "Servicio gestionado",
    "svc.other": "Otro",
    "svc.choose": "Selecciona…",
  },
  en: {
    "nav.platform": "Platform",
    "nav.solutions": "Solutions",
    "nav.services": "Services",
    "nav.research": "Research",
    "nav.demo": "Request demo",
    "nav.console": "Console",

    "hero.badge": "AI-agent cybersecurity platform",
    "hero.title": "Offensive and defensive security, orchestrated by autonomous agents and governed by humans.",
    "hero.subtitle":
      "NEXUS combines recon, vulnerability detection and defensive detection (Sigma/ATT&CK) within verifiable authorization boundaries. Every action passes a signed scope check.",
    "hero.cta.demo": "Request a demo",
    "hero.cta.solutions": "Explore solutions",
    "hero.note": "Research/Beta features are labeled as such. No invented customers or figures.",

    "pillars.title": "One system, three fronts",
    "pillars.red.title": "Authorized offense",
    "pillars.red.body":
      "Recon (subfinder/katana) → web scanning (httpx/nuclei) → dependency SCA and secret detection, all within signed scope. Detects and reports; never exploits.",
    "pillars.blue.title": "Defense",
    "pillars.blue.body":
      "Passive detection with Sigma rules and MITRE ATT&CK tagging, windowed correlation and verified detection evolution.",
    "pillars.research.title": "AI research",
    "pillars.research.body":
      "Generational agent-evolution lab with an independent evaluator. Beta — reproducible results, no promises.",

    "trust.title": "Authorization before capability",
    "trust.body":
      "Each engagement's scope is enforced OUTSIDE the model. A target or path outside the signed scope aborts execution. No agent widens its own permission.",
    "trust.badge1": "Signed scope",
    "trust.badge2": "Action audit",
    "trust.badge3": "No autonomous spread",

    "cta.title": "Ready for a controlled assessment?",
    "cta.body": "Tell us your organization and the service you're interested in. We'll reach out for a pilot.",
    "cta.button": "Request demo",

    "footer.tagline": "AI-agent cybersecurity, governed by humans.",
    "footer.rights": "All rights reserved.",
    "footer.note": "NEXUS is a platform under development. Experimental features are labeled Research/Beta.",

    "solutions.title": "Solutions",
    "solutions.subtitle": "Real capabilities, within verifiable boundaries.",
    "sol.offensive.t": "Offensive security",
    "sol.offensive.b":
      "Surface discovery, web assessment, dependency SCA and secret detection. Remediation-ready findings, within authorized scope.",
    "sol.defensive.t": "Defensive security",
    "sol.defensive.b":
      "Sigma detection with ATT&CK over telemetry, correlation and noise reduction. Built for detection engineering.",
    "sol.purple.t": "Purple team",
    "sol.purple.b": "Simulation and detection-validation loop in an authorized lab. Under construction (Beta).",
    "sol.research.t": "AI research",
    "sol.research.b": "Generational agent-strategy evolution with independent evaluation. Research.",

    "form.title": "Request a demo",
    "form.subtitle": "Fill out the form and we'll get in touch.",
    "form.org": "Organization",
    "form.name": "Contact name",
    "form.email": "Work email",
    "form.country": "Country",
    "form.service": "Service of interest",
    "form.message": "Message",
    "form.consent": "I agree that NEXUS may process this data to respond to my request.",
    "form.submit": "Send request",
    "form.sending": "Sending…",
    "form.ok": "Request sent! We'll be in touch soon.",
    "form.err": "Could not send. Check the fields and try again.",
    "form.required": "Organization, name and email are required.",
    "svc.offensive": "Offensive security",
    "svc.defensive": "Defensive security",
    "svc.purple": "Purple team",
    "svc.ai-research": "AI research",
    "svc.managed": "Managed service",
    "svc.other": "Other",
    "svc.choose": "Select…",
  },
};

type Ctx = { lang: SiteLang; setLang: (l: SiteLang) => void; t: (k: string) => string };
const SiteI18nCtx = createContext<Ctx | null>(null);

export function SiteI18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<SiteLang>("es");
  useEffect(() => {
    try {
      const saved = localStorage.getItem("nexus_site_lang");
      if (saved === "en" || saved === "es") setLangState(saved);
    } catch {
      /* ignore */
    }
  }, []);
  const setLang = useCallback((l: SiteLang) => {
    setLangState(l);
    try {
      localStorage.setItem("nexus_site_lang", l);
    } catch {
      /* ignore */
    }
  }, []);
  const t = useCallback((k: string) => DICT[lang][k] ?? DICT.en[k] ?? k, [lang]);
  return <SiteI18nCtx.Provider value={{ lang, setLang, t }}>{children}</SiteI18nCtx.Provider>;
}

export function useSiteI18n(): Ctx {
  const ctx = useContext(SiteI18nCtx);
  if (!ctx) throw new Error("useSiteI18n must be used within SiteI18nProvider");
  return ctx;
}
