"use client";

import Link from "next/link";

import { CheckIcon } from "lucide-react";

import { useSiteI18n } from "../_site/i18n";
import { CtaBand, PageHeader, Section } from "../_site/sections";

const COPY = {
  en: {
    kicker: "Pricing",
    title: "Pricing built around your scope.",
    body: "NEXUS is sold to organizations, tailored to the systems you protect and the depth of engagement. We publish no fixed list price — you get a clear quote after a short scoping call.",
    plans: [
      {
        name: "Platform",
        for: "Security teams running their own assessments.",
        points: ["Assets & authorization", "Authorized assessments", "Findings & detections", "Reports & audit trail"],
      },
      {
        name: "Services",
        for: "Expert-led red / blue / purple engagements.",
        points: [
          "Scoped engagements",
          "Detection engineering",
          "Remediation-ready findings",
          "Executive & technical reports",
        ],
      },
      {
        name: "Platform + Services",
        for: "Both, under one signed scope of work.",
        points: ["Everything in Platform", "Everything in Services", "Priority support", "Shared roadmap"],
        featured: true,
      },
    ],
    quote: "Request a quote",
    note: "No invented prices or tiers: final pricing depends on scope, environment and engagement depth, and is confirmed in writing.",
    ctaTitle: "Get a tailored quote.",
    ctaBody: "A 20-minute scoping call is all we need to send you a proposal.",
    cta: "Request a demo",
  },
  es: {
    kicker: "Precios",
    title: "Precios a la medida de tu alcance.",
    body: "NEXUS se vende a organizaciones, adaptado a los sistemas que proteges y a la profundidad de la evaluación. No publicamos un precio de lista fijo — recibes una cotización clara tras una breve llamada de alcance.",
    plans: [
      {
        name: "Plataforma",
        for: "Equipos que ejecutan sus propias evaluaciones.",
        points: [
          "Activos y autorización",
          "Evaluaciones autorizadas",
          "Hallazgos y detecciones",
          "Informes y auditoría",
        ],
      },
      {
        name: "Servicios",
        for: "Evaluaciones red / blue / purple dirigidas por expertos.",
        points: [
          "Evaluaciones con alcance",
          "Ingeniería de detección",
          "Hallazgos listos para remediar",
          "Informes ejecutivos y técnicos",
        ],
      },
      {
        name: "Plataforma + Servicios",
        for: "Ambos, bajo un mismo alcance firmado.",
        points: ["Todo lo de Plataforma", "Todo lo de Servicios", "Soporte prioritario", "Hoja de ruta compartida"],
        featured: true,
      },
    ],
    quote: "Solicitar cotización",
    note: "Sin precios ni planes inventados: el precio final depende del alcance, el entorno y la profundidad de la evaluación, y se confirma por escrito.",
    ctaTitle: "Obtén una cotización a medida.",
    ctaBody: "Una llamada de alcance de 20 minutos basta para enviarte una propuesta.",
    cta: "Solicitar demo",
  },
};

export default function Pricing() {
  const { lang } = useSiteI18n();
  const c = COPY[lang];
  return (
    <>
      <PageHeader kicker={c.kicker} title={c.title} body={c.body} />
      <Section>
        <div className="grid gap-5 lg:grid-cols-3">
          {c.plans.map((p) => (
            <div
              key={p.name}
              className={`flex flex-col rounded-2xl border p-7 ${
                p.featured
                  ? "border-cyan-400/50 bg-cyan-400/5 shadow-[0_0_30px_rgba(34,211,238,0.15)]"
                  : "border-white/10 bg-white/[0.03]"
              }`}
            >
              <h3 className="text-lg font-bold text-white">{p.name}</h3>
              <p className="mt-1 text-sm text-slate-400">{p.for}</p>
              <ul className="mt-5 flex-1 space-y-2.5">
                {p.points.map((pt) => (
                  <li key={pt} className="flex items-start gap-2 text-sm text-slate-300">
                    <CheckIcon className="mt-0.5 size-4 shrink-0 text-cyan-300" />
                    {pt}
                  </li>
                ))}
              </ul>
              <Link
                href="/request-demo"
                className={`mt-6 rounded-lg px-4 py-2.5 text-center text-sm font-semibold transition-colors ${
                  p.featured
                    ? "bg-gradient-to-r from-cyan-400 to-cyan-500 text-[#060b18]"
                    : "border border-white/15 text-white hover:border-cyan-400/50"
                }`}
              >
                {c.quote}
              </Link>
            </div>
          ))}
        </div>
        <p className="mt-8 max-w-3xl text-xs leading-relaxed text-slate-500">{c.note}</p>
      </Section>
      <CtaBand title={c.ctaTitle} body={c.ctaBody} cta={c.cta} href="/request-demo" />
    </>
  );
}
