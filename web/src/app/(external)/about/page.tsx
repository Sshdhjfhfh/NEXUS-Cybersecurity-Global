"use client";

import { useSiteI18n } from "../_site/i18n";
import { Band, CtaBand, InfoCard, PageHeader, Section, SectionHeading } from "../_site/sections";

const COPY = {
  en: {
    kicker: "About NEXUS",
    title: "People, intelligence and technology — united for a safer digital world.",
    body: "NEXUS Cybersecurity Global builds AI-assisted offensive and defensive security, governed by humans and bounded by verifiable authorization.",
    values: [
      {
        t: "Authorization before capability",
        d: "Every assessment runs within a signed scope enforced outside the model. No system is tested without verified authority.",
      },
      {
        t: "Honesty over hype",
        d: "We label experimental work as Research or Beta, and never claim certifications or results we don't have.",
      },
      {
        t: "Engineering first",
        d: "Real detections, reproducible benchmarks and verified improvements — not slideware.",
      },
    ],
    b1k: "Our approach",
    b1t: "Offense and defense, as one system.",
    b1b: "Red, blue and purple teaming feed each other: what the offense finds sharpens the defense, and what the defense learns guides the next test.",
    legalNote:
      "NEXUS is an early-stage company. Legal entity details (registered name, jurisdiction and registration number) are being finalized and will be published here once confirmed — we don't publish placeholder corporate facts.",
    ctaTitle: "Work with us.",
    ctaBody: "Partner on an engagement, or join the mission.",
    cta: "Contact us",
  },
  es: {
    kicker: "Sobre NEXUS",
    title: "Personas, inteligencia y tecnología — unidas por un mundo digital más seguro.",
    body: "NEXUS Cybersecurity Global construye seguridad ofensiva y defensiva asistida por IA, gobernada por humanos y limitada por una autorización verificable.",
    values: [
      {
        t: "Autorización antes que capacidad",
        d: "Cada evaluación se ejecuta dentro de un alcance firmado que se aplica fuera del modelo. Ningún sistema se prueba sin autoridad verificada.",
      },
      {
        t: "Honestidad antes que humo",
        d: "Etiquetamos lo experimental como Research o Beta, y nunca afirmamos certificaciones o resultados que no tenemos.",
      },
      {
        t: "Ingeniería primero",
        d: "Detecciones reales, benchmarks reproducibles y mejoras verificadas — no presentaciones.",
      },
    ],
    b1k: "Nuestro enfoque",
    b1t: "Ataque y defensa, como un solo sistema.",
    b1b: "Red, blue y purple team se retroalimentan: lo que encuentra el ataque afina la defensa, y lo que aprende la defensa guía la siguiente prueba.",
    legalNote:
      "NEXUS es una empresa en etapa temprana. Los datos de la entidad legal (razón social, jurisdicción y número de registro) se están finalizando y se publicarán aquí una vez confirmados — no publicamos datos corporativos de relleno.",
    ctaTitle: "Trabaja con nosotros.",
    ctaBody: "Colabora en una evaluación, o súmate a la misión.",
    cta: "Contáctanos",
  },
};

export default function About() {
  const { lang } = useSiteI18n();
  const c = COPY[lang];
  return (
    <>
      <PageHeader kicker={c.kicker} title={c.title} body={c.body} img="/site/about-team.webp" />
      <Section>
        <SectionHeading title={c.b1k === "Our approach" ? "What we stand for" : "En qué creemos"} />
        <div className="mt-8 grid gap-5 md:grid-cols-3">
          {c.values.map((v) => (
            <InfoCard key={v.t} title={v.t} body={v.d} />
          ))}
        </div>
      </Section>
      <Band kicker={c.b1k} title={c.b1t} body={c.b1b} img="/site/building.webp" reverse />
      <Section>
        <p className="max-w-3xl rounded-xl border border-white/10 bg-white/[0.02] p-5 text-xs leading-relaxed text-slate-400">
          {c.legalNote}
        </p>
      </Section>
      <CtaBand title={c.ctaTitle} body={c.ctaBody} cta={c.cta} href="/contact" />
    </>
  );
}
