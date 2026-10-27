"use client";

import { ActivityIcon, BrainIcon, RadarIcon, ShieldCheckIcon, ShieldIcon, SwordsIcon } from "lucide-react";

import { useSiteI18n } from "../_site/i18n";
import { CtaBand, InfoCard, PageHeader, Section, SectionHeading } from "../_site/sections";

const COPY = {
  en: {
    kicker: "Services",
    title: "Security that spans offense, defense and assurance.",
    body: "NEXUS is both a security platform and a professional-services team. You can run the platform yourself, engage our operators for a project, or combine both under one signed scope of work.",
    model1Title: "NEXUS Platform (SaaS)",
    model1Body:
      "A single console for assets, authorized assessments, findings, detections and reports — built for security teams that move fast.",
    model2Title: "Professional Services",
    model2Body:
      "Expert-led offensive and defensive engagements — red team, detection engineering and advisory — delivered within a documented authorization.",
    svcKicker: "Capabilities",
    svcTitle: "What we deliver.",
    cards: [
      {
        t: "Red Team",
        d: "Authorized adversary simulation to test and strengthen your defenses. Detects and reports — never exploits beyond the agreed scope.",
      },
      {
        t: "Blue Team",
        d: "Detection engineering with Sigma and MITRE ATT&CK over your telemetry — correlation, tuning and noise reduction.",
      },
      {
        t: "Purple Team",
        d: "A simulation-and-validation loop that proves your detections against real techniques.",
        badge: "Beta",
      },
      {
        t: "Vulnerability Management",
        d: "Dependency SCA and secret detection with remediation-ready findings, inside authorized source scope.",
      },
      {
        t: "Threat Detection",
        d: "Passive, windowed detection with ATT&CK tagging — built for engineering, not alert fatigue.",
      },
      {
        t: "AI Research",
        d: "Generational agent-strategy evolution with an independent evaluator. Experimental — results, not promises.",
        badge: "Research",
      },
    ],
    note: "Experimental capabilities are labeled Research/Beta and are not sold as generally available. Every engagement runs within a signed authorization and scope — see our Acceptable Use and Authorized Testing terms.",
    ctaTitle: "Scope an engagement with us.",
    ctaBody: "Tell us your goals and environment; we'll propose a plan and authorization.",
    cta: "Request a demo",
  },
  es: {
    kicker: "Servicios",
    title: "Seguridad que abarca ataque, defensa y aseguramiento.",
    body: "NEXUS es a la vez una plataforma de seguridad y un equipo de servicios profesionales. Puedes usar la plataforma por tu cuenta, contratar a nuestros operadores para un proyecto, o combinar ambos bajo un mismo alcance firmado.",
    model1Title: "Plataforma NEXUS (SaaS)",
    model1Body:
      "Una consola única para activos, evaluaciones autorizadas, hallazgos, detecciones e informes — pensada para equipos que van rápido.",
    model2Title: "Servicios profesionales",
    model2Body:
      "Evaluaciones ofensivas y defensivas dirigidas por expertos — red team, ingeniería de detección y consultoría — dentro de una autorización documentada.",
    svcKicker: "Capacidades",
    svcTitle: "Lo que entregamos.",
    cards: [
      {
        t: "Red Team",
        d: "Simulación de adversarios autorizada para probar y reforzar tus defensas. Detecta y reporta — no explota más allá del alcance acordado.",
      },
      {
        t: "Blue Team",
        d: "Ingeniería de detección con Sigma y MITRE ATT&CK sobre tu telemetría — correlación, ajuste y reducción de ruido.",
      },
      {
        t: "Purple Team",
        d: "Un ciclo de simulación y validación que prueba tus detecciones contra técnicas reales.",
        badge: "Beta",
      },
      {
        t: "Gestión de vulnerabilidades",
        d: "SCA de dependencias y detección de secretos con hallazgos listos para remediar, dentro del alcance de código autorizado.",
      },
      {
        t: "Detección de amenazas",
        d: "Detección pasiva por ventanas con etiquetado ATT&CK — pensada para ingeniería, no para fatiga de alertas.",
      },
      {
        t: "Investigación IA",
        d: "Evolución de estrategias de agentes por generaciones con evaluador independiente. Experimental — resultados, no promesas.",
        badge: "Research",
      },
    ],
    note: "Las capacidades experimentales se etiquetan como Research/Beta y no se venden como disponibles de forma general. Cada evaluación se ejecuta dentro de una autorización y alcance firmados — ver nuestros términos de Uso Aceptable y Pruebas Autorizadas.",
    ctaTitle: "Definamos una evaluación contigo.",
    ctaBody: "Cuéntanos tus objetivos y entorno; te proponemos un plan y la autorización.",
    cta: "Solicitar demo",
  },
};

const ICONS = [SwordsIcon, ShieldIcon, ShieldCheckIcon, RadarIcon, ActivityIcon, BrainIcon];

export default function Services() {
  const { lang } = useSiteI18n();
  const c = COPY[lang];
  return (
    <>
      <PageHeader kicker={c.kicker} title={c.title} body={c.body} img="/site/blue-team.webp" />
      <Section className="grid gap-5 md:grid-cols-2">
        <InfoCard title={c.model1Title} body={c.model1Body} />
        <InfoCard title={c.model2Title} body={c.model2Body} />
      </Section>
      <Section className="pt-0">
        <SectionHeading kicker={c.svcKicker} title={c.svcTitle} />
        <div className="mt-8 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {c.cards.map((card, i) => {
            const Icon = ICONS[i];
            return (
              <InfoCard
                key={card.t}
                icon={<Icon className="size-6" />}
                title={card.t}
                body={card.d}
                badge={card.badge}
              />
            );
          })}
        </div>
        <p className="mt-8 max-w-3xl text-xs leading-relaxed text-slate-500">{c.note}</p>
      </Section>
      <CtaBand title={c.ctaTitle} body={c.ctaBody} cta={c.cta} href="/request-demo" />
    </>
  );
}
