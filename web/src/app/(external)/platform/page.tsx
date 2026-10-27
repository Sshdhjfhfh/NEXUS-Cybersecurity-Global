"use client";

import { useSiteI18n } from "../_site/i18n";
import { Band, CtaBand, InfoCard, PageHeader, Section, SectionHeading } from "../_site/sections";

const COPY = {
  en: {
    kicker: "Platform",
    title: "One console for your entire security operation.",
    body: "Assets, authorized assessments, findings, detections and reports in a single workspace — so operators move fast and nothing slips through.",
    modules: [
      { t: "Assets & scope", d: "Register the systems you own and the authorization that governs every test." },
      { t: "Assessments", d: "Launch authorized offensive and defensive assessments within a signed scope." },
      { t: "Findings", d: "Triage, retest and track remediation with evidence attached." },
      { t: "Detections", d: "Sigma + ATT&CK detection engineering over your telemetry." },
      { t: "Reports", d: "Generate clear, shareable reports for engineers and stakeholders." },
      { t: "Admin & audit", d: "Role-based access and an audit trail over every action." },
    ],
    b1k: "Security Engine",
    b1t: "An AI-powered core, governed by humans.",
    b1b: "The engine orchestrates recon, scanning and detection — but every action passes a scope check enforced outside the model. No agent widens its own permission.",
    ctaTitle: "See the platform in action.",
    ctaBody: "Request a guided walkthrough with our team.",
    cta: "Request a demo",
  },
  es: {
    kicker: "Plataforma",
    title: "Una consola para toda tu operación de seguridad.",
    body: "Activos, evaluaciones autorizadas, hallazgos, detecciones e informes en un único espacio de trabajo — para que los operadores vayan rápido y nada se escape.",
    modules: [
      { t: "Activos y alcance", d: "Registra los sistemas que posees y la autorización que rige cada prueba." },
      { t: "Evaluaciones", d: "Lanza evaluaciones ofensivas y defensivas autorizadas dentro de un alcance firmado." },
      { t: "Hallazgos", d: "Prioriza, re-testea y sigue la remediación con evidencia adjunta." },
      { t: "Detecciones", d: "Ingeniería de detección Sigma + ATT&CK sobre tu telemetría." },
      { t: "Informes", d: "Genera informes claros y compartibles para ingeniería y dirección." },
      { t: "Admin y auditoría", d: "Acceso por roles y traza de auditoría sobre cada acción." },
    ],
    b1k: "Motor de Seguridad",
    b1t: "Un núcleo con IA, gobernado por humanos.",
    b1b: "El motor orquesta reconocimiento, escaneo y detección — pero cada acción pasa un control de alcance que se aplica fuera del modelo. Ningún agente amplía su propio permiso.",
    ctaTitle: "Ve la plataforma en acción.",
    ctaBody: "Solicita un recorrido guiado con nuestro equipo.",
    cta: "Solicitar demo",
  },
};

export default function Platform() {
  const { lang } = useSiteI18n();
  const c = COPY[lang];
  return (
    <>
      <PageHeader kicker={c.kicker} title={c.title} body={c.body} img="/site/platform-dashboard.webp" />
      <Section>
        <SectionHeading title={c.title} />
        <div className="mt-8 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {c.modules.map((m) => (
            <InfoCard key={m.t} title={m.t} body={m.d} />
          ))}
        </div>
      </Section>
      <Band kicker={c.b1k} title={c.b1t} body={c.b1b} img="/site/security-engine.webp" reverse />
      <CtaBand title={c.ctaTitle} body={c.ctaBody} cta={c.cta} href="/request-demo" />
    </>
  );
}
