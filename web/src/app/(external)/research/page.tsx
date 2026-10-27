"use client";

import { useSiteI18n } from "../_site/i18n";
import { Band, CtaBand, InfoCard, PageHeader, Section, SectionHeading } from "../_site/sections";

const COPY = {
  en: {
    kicker: "AI Research",
    title: "Pushing the boundaries of autonomous defense.",
    body: "Our research explores how AI agents can discover, detect and defend — safely, within verifiable authorization. These are experimental efforts, shared honestly as Research or Beta.",
    cards: [
      {
        t: "Agent evolution",
        d: "Strategies that improve across generations, measured by an independent evaluator.",
        badge: "Research",
      },
      {
        t: "Detection benchmarking",
        d: "Reproducible benchmarks that separate real detection gains from noise.",
        badge: "Research",
      },
      {
        t: "Verified promotion",
        d: "Only changes that beat a held-out evaluation are promoted — regressions are rejected.",
        badge: "Beta",
      },
    ],
    b1k: "Evolution Lab",
    b1t: "Detections that get smarter every generation.",
    b1b: "In the lab, agent strategies evolve across generations under an independent evaluator — measured, verified gains, with no fabricated results.",
    note: "Research and Beta features are experimental and not sold as generally available products.",
    ctaTitle: "Collaborate on research.",
    ctaBody: "Interested in our methodology or results? Get in touch.",
    cta: "Contact us",
  },
  es: {
    kicker: "Investigación IA",
    title: "Ampliando los límites de la defensa autónoma.",
    body: "Nuestra investigación explora cómo los agentes de IA pueden descubrir, detectar y defender — de forma segura, dentro de una autorización verificable. Son esfuerzos experimentales, compartidos con honestidad como Research o Beta.",
    cards: [
      {
        t: "Evolución de agentes",
        d: "Estrategias que mejoran por generaciones, medidas por un evaluador independiente.",
        badge: "Research",
      },
      {
        t: "Benchmarking de detección",
        d: "Benchmarks reproducibles que separan las mejoras reales del ruido.",
        badge: "Research",
      },
      {
        t: "Promoción verificada",
        d: "Solo se promueven cambios que superan una evaluación reservada — las regresiones se rechazan.",
        badge: "Beta",
      },
    ],
    b1k: "Evolution Lab",
    b1t: "Detecciones que mejoran en cada generación.",
    b1b: "En el laboratorio, las estrategias de los agentes evolucionan por generaciones con un evaluador independiente — mejoras medidas y verificadas, sin resultados inventados.",
    note: "Las funciones Research y Beta son experimentales y no se venden como productos disponibles de forma general.",
    ctaTitle: "Colabora en investigación.",
    ctaBody: "¿Te interesa nuestra metodología o resultados? Escríbenos.",
    cta: "Contáctanos",
  },
};

export default function Research() {
  const { lang } = useSiteI18n();
  const c = COPY[lang];
  return (
    <>
      <PageHeader kicker={c.kicker} title={c.title} body={c.body} img="/site/ai-research.webp" />
      <Section>
        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {c.cards.map((card) => (
            <InfoCard key={card.t} title={card.t} body={card.d} badge={card.badge} />
          ))}
        </div>
        <p className="mt-8 max-w-3xl text-xs leading-relaxed text-slate-500">{c.note}</p>
      </Section>
      <Band kicker={c.b1k} title={c.b1t} body={c.b1b} img="/site/evolution-lab.webp" />
      <CtaBand title={c.ctaTitle} body={c.ctaBody} cta={c.cta} href="/contact" />
    </>
  );
}
