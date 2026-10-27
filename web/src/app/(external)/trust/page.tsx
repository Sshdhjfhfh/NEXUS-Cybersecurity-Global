"use client";

import Link from "next/link";

import { ArrowRightIcon } from "lucide-react";

import { useSiteI18n } from "../_site/i18n";
import { CtaBand, PageHeader, Section } from "../_site/sections";

type Status = "done" | "dev" | "planned" | "audit" | "pending";

const STATUS_STYLE: Record<Status, string> = {
  done: "border-emerald-400/40 bg-emerald-400/10 text-emerald-300",
  dev: "border-cyan-400/40 bg-cyan-400/10 text-cyan-300",
  planned: "border-slate-400/30 bg-slate-400/10 text-slate-300",
  audit: "border-violet-400/40 bg-violet-400/10 text-violet-200",
  pending: "border-amber-400/40 bg-amber-400/10 text-amber-200",
};

const STATUS_LABEL = {
  en: {
    done: "Implemented",
    dev: "In development",
    planned: "Planned",
    audit: "Pending external audit",
    pending: "Pending",
  },
  es: {
    done: "Implementado",
    dev: "En desarrollo",
    planned: "Planificado",
    audit: "Pendiente de auditoría externa",
    pending: "Pendiente",
  },
};

type Control = { t: string; s: Status };
type Group = { title: string; controls: Control[] };

const COPY: Record<
  "en" | "es",
  {
    kicker: string;
    title: string;
    body: string;
    note: string;
    docsTitle: string;
    groups: Group[];
    ctaTitle: string;
    ctaBody: string;
    cta: string;
  }
> = {
  en: {
    kicker: "Trust Center",
    title: "Built on transparency. Backed by responsibility.",
    body: "We classify every control honestly. Nothing here is presented as a certification we have not earned.",
    note: "We do not claim SOC 2, ISO 27001 or any certification we have not obtained and had audited.",
    docsTitle: "Legal & policies",
    groups: [
      {
        title: "Authorization & scope",
        controls: [
          { t: "Engagement scope enforced outside the model (signed scope gate)", s: "done" },
          { t: "Verification of authority over target assets before testing", s: "dev" },
          { t: "Audit trail over security-relevant actions", s: "dev" },
        ],
      },
      {
        title: "Authentication & access",
        controls: [
          { t: "Console login with hashed credentials (bcrypt)", s: "done" },
          { t: "Rate limiting & bot filtering on public forms", s: "done" },
          { t: "Multi-factor authentication (MFA)", s: "planned" },
        ],
      },
      {
        title: "Application security",
        controls: [
          { t: "Baseline security headers (nosniff, frame-options, referrer, permissions, HSTS)", s: "done" },
          { t: "Content Security Policy — enforced & nonce-based (report-only today)", s: "dev" },
          { t: "Dependency SCA & secret scanning", s: "dev" },
        ],
      },
      {
        title: "Data protection",
        controls: [
          { t: "Consent capture + cookie preference manager", s: "done" },
          { t: "Data minimization & retention limits", s: "dev" },
          { t: "Personal-data bank registration (Perú, Ley 29733)", s: "pending" },
        ],
      },
      {
        title: "Transparency",
        controls: [
          { t: "Published legal documents (draft, pending legal review)", s: "done" },
          { t: "Responsible vulnerability disclosure policy", s: "done" },
        ],
      },
      {
        title: "External assurance",
        controls: [
          { t: "SOC 2 / ISO 27001 certifications", s: "audit" },
          { t: "Independent security audit", s: "planned" },
        ],
      },
    ],
    ctaTitle: "Questions about our security?",
    ctaBody: "Reach our security team or request the latest security overview.",
    cta: "Contact us",
  },
  es: {
    kicker: "Trust Center",
    title: "Construido sobre la transparencia. Respaldado por la responsabilidad.",
    body: "Clasificamos cada control con honestidad. Nada aquí se presenta como una certificación que no hayamos obtenido.",
    note: "No afirmamos SOC 2, ISO 27001 ni ninguna certificación que no hayamos obtenido y auditado.",
    docsTitle: "Legal y políticas",
    groups: [
      {
        title: "Autorización y alcance",
        controls: [
          { t: "Alcance de la evaluación aplicado fuera del modelo (control de alcance firmado)", s: "done" },
          { t: "Verificación de autoridad sobre los activos objetivo antes de probar", s: "dev" },
          { t: "Traza de auditoría de acciones relevantes para la seguridad", s: "dev" },
        ],
      },
      {
        title: "Autenticación y acceso",
        controls: [
          { t: "Login de consola con credenciales con hash (bcrypt)", s: "done" },
          { t: "Límite de tasa y filtrado de bots en formularios públicos", s: "done" },
          { t: "Autenticación multifactor (MFA)", s: "planned" },
        ],
      },
      {
        title: "Seguridad de la aplicación",
        controls: [
          { t: "Cabeceras de seguridad base (nosniff, frame-options, referrer, permissions, HSTS)", s: "done" },
          { t: "Content Security Policy — enforced y con nonce (hoy en report-only)", s: "dev" },
          { t: "SCA de dependencias y detección de secretos", s: "dev" },
        ],
      },
      {
        title: "Protección de datos",
        controls: [
          { t: "Captura de consentimiento + gestor de preferencias de cookies", s: "done" },
          { t: "Minimización de datos y límites de conservación", s: "dev" },
          { t: "Registro de banco de datos personales (Perú, Ley 29733)", s: "pending" },
        ],
      },
      {
        title: "Transparencia",
        controls: [
          { t: "Documentos legales publicados (borrador, pendiente de revisión legal)", s: "done" },
          { t: "Política de divulgación responsable de vulnerabilidades", s: "done" },
        ],
      },
      {
        title: "Aseguramiento externo",
        controls: [
          { t: "Certificaciones SOC 2 / ISO 27001", s: "audit" },
          { t: "Auditoría de seguridad independiente", s: "planned" },
        ],
      },
    ],
    ctaTitle: "¿Dudas sobre nuestra seguridad?",
    ctaBody: "Contacta a nuestro equipo de seguridad o solicita el resumen de seguridad más reciente.",
    cta: "Contáctanos",
  },
};

const LEGAL_LINKS = [
  { k: "terms", en: "Terms of Service", es: "Términos del Servicio" },
  { k: "privacy", en: "Privacy Policy", es: "Política de Privacidad" },
  { k: "cookies", en: "Cookie Policy", es: "Política de Cookies" },
  { k: "aup", en: "Acceptable Use", es: "Uso Aceptable" },
  { k: "security", en: "Security Overview", es: "Resumen de Seguridad" },
  { k: "vdp", en: "Vulnerability Disclosure", es: "Divulgación de Vulnerabilidades" },
];

export default function Trust() {
  const { lang } = useSiteI18n();
  const c = COPY[lang];
  const labels = STATUS_LABEL[lang];
  return (
    <>
      <PageHeader kicker={c.kicker} title={c.title} body={c.body} />
      <Section>
        <div className="grid gap-6 lg:grid-cols-2">
          {c.groups.map((g) => (
            <div key={g.title} className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
              <h2 className="text-sm font-semibold uppercase tracking-wide text-cyan-300">{g.title}</h2>
              <ul className="mt-4 space-y-3">
                {g.controls.map((ctrl) => (
                  <li key={ctrl.t} className="flex items-start justify-between gap-3">
                    <span className="text-sm text-slate-300">{ctrl.t}</span>
                    <span
                      className={`shrink-0 whitespace-nowrap rounded-full border px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${STATUS_STYLE[ctrl.s]}`}
                    >
                      {labels[ctrl.s]}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
        <p className="mt-8 max-w-3xl rounded-xl border border-amber-400/20 bg-amber-400/5 p-4 text-xs leading-relaxed text-amber-100/80">
          {c.note}
        </p>

        <h2 className="mt-14 text-lg font-bold text-white">{c.docsTitle}</h2>
        <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {LEGAL_LINKS.map((d) => (
            <Link
              key={d.k}
              href={`/legal/${d.k}`}
              className="group flex items-center justify-between gap-2 rounded-xl border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-slate-300 transition-colors hover:border-cyan-400/40 hover:text-white"
            >
              {lang === "en" ? d.en : d.es}
              <ArrowRightIcon className="size-4 text-slate-500 transition-transform group-hover:translate-x-1" />
            </Link>
          ))}
        </div>
      </Section>
      <CtaBand title={c.ctaTitle} body={c.ctaBody} cta={c.cta} href="/contact" />
    </>
  );
}
