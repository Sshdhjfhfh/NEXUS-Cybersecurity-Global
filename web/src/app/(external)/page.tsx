"use client";

import { type FormEvent, useState } from "react";

import Link from "next/link";

import {
  ArrowRightIcon,
  BrainIcon,
  BugIcon,
  DatabaseIcon,
  FileTextIcon,
  FlaskConicalIcon,
  GlobeIcon,
  LayersIcon,
  LockIcon,
  PlayIcon,
  ShieldCheckIcon,
  ShieldIcon,
  SlidersHorizontalIcon,
  SwordsIcon,
  UserRoundIcon,
} from "lucide-react";

import { HeroVisual } from "./_site/hero-visual";
import { useSiteI18n } from "./_site/i18n";
import { Counter, Reveal } from "./_site/motion";
import { VideoBg } from "./_site/video-bg";

const COPY = {
  en: {
    eyebrow: "Global Threat Intelligence · AI-Driven Defense · A Safer Tomorrow",
    heroTitle: "Security intelligence engineered for",
    heroTitleAccent: "the real world.",
    heroBody:
      "NEXUS Cybersecurity Global delivers AI-powered threat intelligence, advanced security operations, and proactive defense for the world's most critical organizations.",
    requestDemo: "Request Demo",
    explore: "Explore Our Platform",
    stats: [
      { v: "Red · Blue · Purple", l: "Full-spectrum security" },
      { v: "Signed scope", l: "Authorization before capability" },
      { v: "AI research", l: "Detections that keep evolving" },
    ],
    capsKicker: "Capabilities",
    capsTitle: "A unified approach to a stronger tomorrow.",
    capsAside: "People. Intelligence. Technology. A more secure world.",
    caps: [
      { t: "Red Team", d: "Real-world adversary simulation to test and strengthen your defenses." },
      { t: "Blue Team", d: "24/7 threat monitoring, detection and response by expert operators." },
      { t: "Purple Team", d: "Bridging offense and defense for continuous improvement." },
      { t: "AI Research", d: "Pushing the boundaries of AI for a safer, more resilient world." },
    ],
    archKicker: "Platform Architecture",
    archTitle: "A seamless ecosystem built for global security.",
    archBody:
      "From client access to AI research, the NEXUS platform unifies people, intelligence and technology into a single, powerful ecosystem.",
    archCta: "Explore the Platform",
    nodes: [
      { t: "Website", d: "Public Information" },
      { t: "Client Portal", d: "Customer Access" },
      { t: "Security Engine", d: "AI-Powered Core" },
      { t: "Admin Console", d: "Operations & Management" },
      { t: "Evolution Lab", d: "Research & Innovation" },
    ],
    engineSteps: ["Detect", "Analyze", "Respond", "Evolve"],
    trustKicker: "Trust Center",
    trustTitle: "Built on transparency. Backed by responsibility.",
    trustBody: "Our policies, standards and practices ensure a safer, more open and more trustworthy digital world.",
    trust: [
      { t: "Terms of Service", d: "Our service terms and user obligations." },
      { t: "Privacy Policy", d: "How we protect and handle your data." },
      { t: "Acceptable Use Policy", d: "Guidelines for responsible and lawful use." },
      { t: "Security Policy", d: "Our security practices and commitments." },
      { t: "Vulnerability Disclosure", d: "Report security issues responsibly." },
    ],
    demoKicker: "Request a Demo",
    demoTitle: "See NEXUS in action.",
    demoBody: "Discover how NEXUS can help your organization prevent threats, reduce risk and stay ahead.",
    fFullName: "Full Name",
    fFullNamePh: "John Smith",
    fEmail: "Work Email",
    fEmailPh: "you@company.com",
    fCompany: "Company",
    fCompanyPh: "Your Organization",
    fIndustry: "Industry",
    fIndustryPh: "Select industry",
    industries: ["Finance", "Healthcare", "Government", "Technology", "Energy", "Retail", "Other"],
    consentNote: "By requesting a demo you agree that NEXUS may process this data to contact you.",
    sending: "Sending…",
    okTitle: "Request sent!",
    okBody: "Thank you — our team will be in touch shortly.",
    errReq: "Name, work email and company are required.",
    err: "Could not send your request. Please try again.",
    learnMore: "Learn more",
    tiKicker: "Threat Intelligence",
    tiTitle: "See every attack, everywhere, in real time.",
    tiBody:
      "Our global sensor network maps adversary infrastructure and live attack paths across continents, so your team acts before the breach — not after.",
    platKicker: "The Platform",
    platTitle: "One console for your entire security operation.",
    platBody:
      "Assets, assessments, findings, detections and reports in a single workspace — built so operators move fast and nothing slips through.",
    labKicker: "Evolution Lab",
    labTitle: "Detections that get smarter every generation.",
    labBody:
      "In our research lab, agent strategies evolve across generations under an independent evaluator — measured, verified gains, no guesswork.",
    indKicker: "Industries",
    indTitle: "Trusted where the stakes are highest.",
    indBody:
      "From finance to healthcare to government, NEXUS adapts to the threats and compliance demands of the world's most critical sectors.",
    industries2: ["Finance", "Healthcare", "Government", "Energy", "Technology", "Retail"],
    aboutKicker: "About NEXUS",
    aboutTitle: "People, intelligence and technology — united.",
    aboutBody: "A global team of operators and researchers building a safer digital world together, around the clock.",
  },
  es: {
    eyebrow: "Inteligencia global de amenazas · Defensa con IA · Un mañana más seguro",
    heroTitle: "Inteligencia de seguridad diseñada para",
    heroTitleAccent: "el mundo real.",
    heroBody:
      "NEXUS Cybersecurity Global ofrece inteligencia de amenazas con IA, operaciones de seguridad avanzadas y defensa proactiva para las organizaciones más críticas del mundo.",
    requestDemo: "Solicitar demo",
    explore: "Explorar la plataforma",
    stats: [
      { v: "Red · Blue · Purple", l: "Seguridad de espectro completo" },
      { v: "Alcance firmado", l: "Autorización antes que capacidad" },
      { v: "Investigación IA", l: "Detecciones que evolucionan" },
    ],
    capsKicker: "Capacidades",
    capsTitle: "Un enfoque unificado para un mañana más fuerte.",
    capsAside: "Personas. Inteligencia. Tecnología. Un mundo más seguro.",
    caps: [
      { t: "Red Team", d: "Simulación de adversarios reales para probar y reforzar tus defensas." },
      { t: "Blue Team", d: "Monitoreo, detección y respuesta de amenazas 24/7 por operadores expertos." },
      { t: "Purple Team", d: "Une ataque y defensa para una mejora continua." },
      { t: "Investigación IA", d: "Ampliando los límites de la IA por un mundo más seguro y resiliente." },
    ],
    archKicker: "Arquitectura de la plataforma",
    archTitle: "Un ecosistema integral para la seguridad global.",
    archBody:
      "Desde el acceso del cliente hasta la investigación en IA, la plataforma NEXUS unifica personas, inteligencia y tecnología en un único ecosistema potente.",
    archCta: "Explorar la plataforma",
    nodes: [
      { t: "Sitio web", d: "Información pública" },
      { t: "Portal del cliente", d: "Acceso de clientes" },
      { t: "Motor de seguridad", d: "Núcleo con IA" },
      { t: "Consola de admin", d: "Operación y gestión" },
      { t: "Evolution Lab", d: "Investigación e innovación" },
    ],
    engineSteps: ["Detectar", "Analizar", "Responder", "Evolucionar"],
    trustKicker: "Trust Center",
    trustTitle: "Construido sobre la transparencia. Respaldado por la responsabilidad.",
    trustBody:
      "Nuestras políticas, estándares y prácticas garantizan un mundo digital más seguro, abierto y confiable.",
    trust: [
      { t: "Términos del servicio", d: "Nuestros términos y las obligaciones del usuario." },
      { t: "Política de privacidad", d: "Cómo protegemos y tratamos tus datos." },
      { t: "Política de uso aceptable", d: "Pautas para un uso responsable y lícito." },
      { t: "Política de seguridad", d: "Nuestras prácticas y compromisos de seguridad." },
      { t: "Divulgación de vulnerabilidades", d: "Reporta problemas de seguridad de forma responsable." },
    ],
    demoKicker: "Solicitar una demo",
    demoTitle: "Ve NEXUS en acción.",
    demoBody: "Descubre cómo NEXUS ayuda a tu organización a prevenir amenazas, reducir el riesgo y adelantarse.",
    fFullName: "Nombre completo",
    fFullNamePh: "Juan Pérez",
    fEmail: "Email corporativo",
    fEmailPh: "tu@empresa.com",
    fCompany: "Empresa",
    fCompanyPh: "Tu organización",
    fIndustry: "Sector",
    fIndustryPh: "Selecciona un sector",
    industries: ["Finanzas", "Salud", "Gobierno", "Tecnología", "Energía", "Retail", "Otro"],
    consentNote: "Al solicitar una demo aceptas que NEXUS procese estos datos para contactarte.",
    sending: "Enviando…",
    okTitle: "¡Solicitud enviada!",
    okBody: "Gracias — nuestro equipo te contactará muy pronto.",
    errReq: "Nombre, email corporativo y empresa son obligatorios.",
    err: "No se pudo enviar tu solicitud. Inténtalo de nuevo.",
    learnMore: "Saber más",
    tiKicker: "Inteligencia de amenazas",
    tiTitle: "Ve cada ataque, en cualquier lugar, en tiempo real.",
    tiBody:
      "Nuestra red global de sensores mapea la infraestructura adversaria y las rutas de ataque en vivo en todos los continentes, para que tu equipo actúe antes de la brecha, no después.",
    platKicker: "La plataforma",
    platTitle: "Una consola para toda tu operación de seguridad.",
    platBody:
      "Activos, evaluaciones, hallazgos, detecciones e informes en un único espacio de trabajo — pensado para que los operadores vayan rápido y nada se escape.",
    labKicker: "Evolution Lab",
    labTitle: "Detecciones que mejoran en cada generación.",
    labBody:
      "En nuestro laboratorio, las estrategias de los agentes evolucionan por generaciones con un evaluador independiente — mejoras medidas y verificadas, sin conjeturas.",
    indKicker: "Industrias",
    indTitle: "Donde más está en juego, confían en nosotros.",
    indBody:
      "De las finanzas a la salud y el gobierno, NEXUS se adapta a las amenazas y exigencias de cumplimiento de los sectores más críticos del mundo.",
    industries2: ["Finanzas", "Salud", "Gobierno", "Energía", "Tecnología", "Retail"],
    aboutKicker: "Sobre NEXUS",
    aboutTitle: "Personas, inteligencia y tecnología — unidas.",
    aboutBody:
      "Un equipo global de operadores e investigadores construyendo juntos un mundo digital más seguro, las 24 horas.",
  },
};

const CAP_ICONS = [SwordsIcon, ShieldIcon, LayersIcon, BrainIcon];
const CAP_THEMES = [
  "from-rose-500/20 to-rose-900/5 border-rose-500/30 text-rose-400",
  "from-sky-500/20 to-sky-900/5 border-sky-500/30 text-sky-400",
  "from-purple-500/20 to-purple-900/5 border-purple-500/30 text-purple-400",
  "from-teal-500/20 to-teal-900/5 border-teal-500/30 text-teal-400",
];
const CAP_IMAGES = ["/site/cap-red.webp", "/site/cap-blue.webp", "/site/cap-purple.webp", "/site/cap-ai.webp"];
const NODE_ICONS = [GlobeIcon, UserRoundIcon, LayersIcon, SlidersHorizontalIcon, FlaskConicalIcon];
const TRUST_ICONS = [FileTextIcon, LockIcon, ShieldCheckIcon, DatabaseIcon, BugIcon];

type Status = { kind: "idle" } | { kind: "sending" } | { kind: "ok" } | { kind: "err"; msg: string };

export default function Home() {
  const { lang } = useSiteI18n();
  const c = COPY[lang];
  const [status, setStatus] = useState<Status>({ kind: "idle" });

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const get = (k: string) => String(fd.get(k) ?? "").trim();
    const organization = get("company");
    const contact_name = get("full_name");
    const email = get("email");
    const industry = get("industry");
    if (!organization || !contact_name || !email) {
      setStatus({ kind: "err", msg: c.errReq });
      return;
    }
    setStatus({ kind: "sending" });
    try {
      const res = await fetch("/api/public/leads", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          organization,
          contact_name,
          email,
          service: "",
          message: industry ? `Industry: ${industry}` : "",
          consent: true,
          website: get("website"),
        }),
      });
      if (!res.ok) {
        setStatus({ kind: "err", msg: c.err });
        return;
      }
      setStatus({ kind: "ok" });
    } catch {
      setStatus({ kind: "err", msg: c.err });
    }
  }

  return (
    <>
      {/* ============ HERO ============ */}
      <section className="relative overflow-hidden border-b border-white/5">
        {/* Operations-center looping video as the hero backdrop (poster fallback). */}
        <VideoBg
          src="/site/hero-loop.mp4"
          poster="/site/hero-soc.webp"
          className="pointer-events-none absolute inset-0 size-full object-cover opacity-70"
        />
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,#060b18_2%,rgba(6,11,24,0.88)_34%,rgba(6,11,24,0.35)_68%,rgba(6,11,24,0.55)_100%)]" />
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(90%_70%_at_78%_0%,rgba(14,116,214,0.18),transparent_60%)]" />
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_bottom,transparent,rgba(6,11,24,0.25),#060b18)]" />
        <div className="relative mx-auto grid max-w-[1400px] items-center gap-10 px-6 py-20 lg:grid-cols-[1.05fr_1fr] lg:py-24">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{c.eyebrow}</p>
            <h1 className="mt-6 max-w-xl text-4xl font-bold leading-[1.08] tracking-tight text-white sm:text-5xl lg:text-6xl">
              {c.heroTitle}{" "}
              <span className="bg-gradient-to-r from-cyan-300 to-sky-400 bg-clip-text text-transparent">
                {c.heroTitleAccent}
              </span>
            </h1>
            <p className="mt-6 max-w-lg text-base leading-relaxed text-slate-300">{c.heroBody}</p>
            <div className="mt-8 flex flex-wrap items-center gap-3">
              <Link
                href="/request-demo"
                className="inline-flex items-center gap-2 rounded-lg bg-gradient-to-r from-cyan-400 to-cyan-500 px-6 py-3 text-sm font-semibold text-[#060b18] shadow-[0_0_28px_rgba(34,211,238,0.4)] transition-all hover:shadow-[0_0_40px_rgba(34,211,238,0.6)]"
              >
                {c.requestDemo} <ArrowRightIcon className="size-4" />
              </Link>
              <Link
                href="/solutions"
                className="inline-flex items-center gap-2 rounded-lg border border-white/15 px-6 py-3 text-sm font-semibold text-white transition-colors hover:border-cyan-400/50"
              >
                {c.explore} <PlayIcon className="size-3.5 fill-current" />
              </Link>
            </div>
            <div className="mt-12 flex flex-wrap gap-x-10 gap-y-6">
              {c.stats.map((s) => (
                <div key={s.l} className="border-l border-cyan-400/30 pl-3">
                  <div className="text-lg font-bold text-cyan-300">
                    <Counter value={s.v} />
                  </div>
                  <div className="mt-1 max-w-[11rem] text-[11px] font-medium uppercase tracking-wide text-slate-400">
                    {s.l}
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className="relative">
            <HeroVisual />
          </div>
        </div>
      </section>

      {/* ============ CAPABILITIES ============ */}
      <section id="ai-research" className="mx-auto max-w-[1400px] px-6 py-20">
        <div className="flex flex-col justify-between gap-6 md:flex-row md:items-end">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{c.capsKicker}</p>
            <h2 className="mt-3 max-w-md text-3xl font-bold tracking-tight text-white sm:text-4xl">{c.capsTitle}</h2>
          </div>
          <Link
            href="/solutions"
            className="group inline-flex items-center gap-2 text-sm text-slate-400 transition-colors hover:text-white md:max-w-[14rem] md:text-right"
          >
            {c.capsAside} <ArrowRightIcon className="size-4 transition-transform group-hover:translate-x-1" />
          </Link>
        </div>
        <div className="mt-10 grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {c.caps.map((cap, i) => {
            const Icon = CAP_ICONS[i];
            return (
              <Link
                key={cap.t}
                href="/solutions"
                className={`group relative flex h-72 flex-col justify-end overflow-hidden rounded-2xl border p-6 transition-transform hover:-translate-y-1 ${CAP_THEMES[i]}`}
              >
                {/* biome-ignore lint/performance/noImgElement: static marketing asset */}
                <img
                  src={CAP_IMAGES[i]}
                  alt=""
                  aria-hidden="true"
                  className="absolute inset-0 size-full object-cover opacity-30 transition-opacity duration-300 group-hover:opacity-45"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-[#060b18] via-[#060b18]/75 to-[#060b18]/20" />
                <Icon className="absolute left-6 top-6 size-7" />
                <div className="relative">
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-semibold text-white">{cap.t}</h3>
                    <ArrowRightIcon className="size-4 text-white/60 transition-transform group-hover:translate-x-1" />
                  </div>
                  <p className="mt-2 text-sm leading-relaxed text-slate-200/85">{cap.d}</p>
                </div>
              </Link>
            );
          })}
        </div>
      </section>

      {/* ============ THREAT INTELLIGENCE ============ */}
      <section className="relative overflow-hidden border-y border-white/5 bg-[#070d1c]">
        {/* biome-ignore lint/performance/noImgElement: static marketing asset */}
        <img
          src="/site/section-particles.webp"
          alt=""
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 size-full object-cover opacity-10"
        />
        <div className="relative mx-auto grid max-w-[1400px] items-center gap-10 px-6 py-20 lg:grid-cols-2">
          <Reveal>
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{c.tiKicker}</p>
            <h2 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">{c.tiTitle}</h2>
            <p className="mt-4 max-w-md text-sm leading-relaxed text-slate-400">{c.tiBody}</p>
            <Link
              href="/solutions"
              className="mt-6 inline-flex items-center gap-2 text-sm font-semibold text-cyan-300 transition-colors hover:text-cyan-200"
            >
              {c.learnMore} <ArrowRightIcon className="size-4" />
            </Link>
          </Reveal>
          <Reveal delay={120}>
            <div className="overflow-hidden rounded-2xl border border-cyan-400/20 shadow-2xl">
              <VideoBg
                src="/site/threatmap-loop.mp4"
                poster="/site/threat-map.webp"
                className="aspect-video w-full object-cover"
              />
            </div>
          </Reveal>
        </div>
      </section>

      {/* ============ PLATFORM ARCHITECTURE ============ */}
      <section id="platform" className="border-y border-white/5 bg-[#070d1c]">
        <div className="mx-auto max-w-[1400px] px-6 py-20">
          <div className="grid gap-12 lg:grid-cols-[0.8fr_1.6fr] lg:items-center">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{c.archKicker}</p>
              <h2 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">{c.archTitle}</h2>
              <p className="mt-4 text-sm leading-relaxed text-slate-400">{c.archBody}</p>
              <Link
                href="/solutions"
                className="mt-6 inline-flex items-center gap-2 rounded-lg border border-cyan-400/40 px-5 py-2.5 text-sm font-semibold text-cyan-300 transition-colors hover:bg-cyan-400/10"
              >
                {c.archCta} <ArrowRightIcon className="size-4" />
              </Link>
            </div>
            <div className="grid grid-cols-2 items-stretch gap-3 sm:grid-cols-3 lg:grid-cols-5">
              {c.nodes.map((n, i) => {
                const Icon = NODE_ICONS[i];
                const center = i === 2;
                return (
                  <div
                    key={n.t}
                    className={`relative flex flex-col rounded-xl border p-4 ${
                      center
                        ? "border-cyan-400/50 bg-gradient-to-b from-cyan-500/15 to-[#070d1c] shadow-[0_0_30px_rgba(34,211,238,0.25)] sm:-translate-y-2"
                        : "border-white/10 bg-white/[0.03]"
                    }`}
                  >
                    <Icon className={`size-5 ${center ? "text-cyan-300" : "text-slate-300"}`} />
                    <div className="mt-6 text-sm font-semibold text-white">{n.t}</div>
                    <div className="mt-0.5 text-[11px] text-slate-400">{n.d}</div>
                    {center ? (
                      <div className="mt-4 grid place-items-center">
                        <VideoBg
                          src="/site/engine-loop.mp4"
                          poster="/site/security-engine.webp"
                          className="h-24 w-full rounded-lg object-cover"
                        />
                        <div className="mt-3 flex flex-wrap justify-center gap-x-2 gap-y-1 text-[9px] font-semibold uppercase tracking-wide text-cyan-200/80">
                          {c.engineSteps.map((s) => (
                            <span key={s}>{s}</span>
                          ))}
                        </div>
                      </div>
                    ) : null}
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </section>

      {/* ============ PLATFORM SHOWCASE ============ */}
      <section className="mx-auto max-w-[1400px] px-6 py-20">
        <Reveal className="mx-auto max-w-2xl text-center">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{c.platKicker}</p>
          <h2 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">{c.platTitle}</h2>
          <p className="mt-4 text-sm leading-relaxed text-slate-400">{c.platBody}</p>
        </Reveal>
        <Reveal delay={120} className="mt-10">
          <div className="overflow-hidden rounded-xl border border-white/10 bg-[#0a1324] shadow-[0_30px_80px_-20px_rgba(34,211,238,0.25)]">
            <div className="flex items-center gap-1.5 border-b border-white/10 px-4 py-2.5">
              <span className="size-2.5 rounded-full bg-rose-400/70" />
              <span className="size-2.5 rounded-full bg-amber-400/70" />
              <span className="size-2.5 rounded-full bg-emerald-400/70" />
              <span className="ml-3 text-[11px] text-slate-500">console.nexus-security.io</span>
            </div>
            {/* biome-ignore lint/performance/noImgElement: static marketing asset */}
            <img src="/site/platform-dashboard.webp" alt="NEXUS security console" className="w-full object-cover" />
          </div>
        </Reveal>
      </section>

      {/* ============ TRUST CENTER ============ */}
      <section id="trust" className="mx-auto max-w-[1400px] px-6 py-20">
        <div className="max-w-xl">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{c.trustKicker}</p>
          <h2 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">{c.trustTitle}</h2>
          <p className="mt-4 text-sm leading-relaxed text-slate-400">{c.trustBody}</p>
        </div>
        <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
          {c.trust.map((item, i) => {
            const Icon = TRUST_ICONS[i];
            return (
              <Link
                key={item.t}
                href="/request-demo"
                className="group rounded-xl border border-white/10 bg-white/[0.03] p-5 transition-colors hover:border-cyan-400/40"
              >
                <Icon className="size-6 text-cyan-300" />
                <h3 className="mt-4 text-sm font-semibold text-white">{item.t}</h3>
                <p className="mt-1.5 text-[12px] leading-relaxed text-slate-400">{item.d}</p>
              </Link>
            );
          })}
        </div>
      </section>

      {/* ============ EVOLUTION LAB ============ */}
      <section className="border-y border-white/5 bg-[#070d1c]">
        <div className="mx-auto grid max-w-[1400px] items-center gap-10 px-6 py-20 lg:grid-cols-2">
          <Reveal className="lg:order-2">
            <span className="inline-block rounded-full border border-violet-400/40 bg-violet-400/10 px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-violet-200">
              Research
            </span>
            <p className="mt-3 text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{c.labKicker}</p>
            <h2 className="mt-2 text-3xl font-bold tracking-tight text-white sm:text-4xl">{c.labTitle}</h2>
            <p className="mt-4 max-w-md text-sm leading-relaxed text-slate-400">{c.labBody}</p>
          </Reveal>
          <Reveal delay={120} className="lg:order-1">
            {/* biome-ignore lint/performance/noImgElement: static marketing asset */}
            <img
              src="/site/evolution-lab.webp"
              alt="AI agent evolution lab"
              className="aspect-video w-full rounded-2xl border border-violet-400/20 object-cover shadow-2xl"
            />
          </Reveal>
        </div>
      </section>

      {/* ============ INDUSTRIES ============ */}
      <section className="mx-auto grid max-w-[1400px] items-center gap-10 px-6 py-20 lg:grid-cols-2">
        <Reveal>
          {/* biome-ignore lint/performance/noImgElement: static marketing asset */}
          <img
            src="/site/industries.webp"
            alt="Industries NEXUS protects"
            className="aspect-video w-full rounded-2xl border border-white/10 object-cover shadow-2xl"
          />
        </Reveal>
        <Reveal delay={120}>
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{c.indKicker}</p>
          <h2 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">{c.indTitle}</h2>
          <p className="mt-4 max-w-md text-sm leading-relaxed text-slate-400">{c.indBody}</p>
          <div className="mt-6 flex flex-wrap gap-2">
            {c.industries2.map((ind) => (
              <span
                key={ind}
                className="rounded-full border border-white/15 bg-white/[0.03] px-3.5 py-1.5 text-xs font-medium text-slate-200"
              >
                {ind}
              </span>
            ))}
          </div>
        </Reveal>
      </section>

      {/* ============ ABOUT ============ */}
      <section className="relative overflow-hidden border-y border-white/5">
        {/* biome-ignore lint/performance/noImgElement: static marketing asset */}
        <img
          src="/site/about-team.webp"
          alt=""
          aria-hidden="true"
          className="pointer-events-none absolute inset-0 size-full object-cover opacity-30"
        />
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,#060b18_10%,rgba(6,11,24,0.6)_60%,rgba(6,11,24,0.85)_100%)]" />
        <div id="about" className="relative mx-auto max-w-[1400px] px-6 py-24">
          <Reveal className="max-w-xl">
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{c.aboutKicker}</p>
            <h2 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">{c.aboutTitle}</h2>
            <p className="mt-4 text-sm leading-relaxed text-slate-300">{c.aboutBody}</p>
          </Reveal>
        </div>
      </section>

      {/* ============ REQUEST A DEMO ============ */}
      <section className="border-t border-white/5 bg-[#070d1c]">
        <div className="mx-auto max-w-[1400px] px-6 py-16">
          <div className="grid gap-10 lg:grid-cols-[0.8fr_1.6fr] lg:items-center">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{c.demoKicker}</p>
              <h2 className="mt-3 text-3xl font-bold tracking-tight text-white sm:text-4xl">{c.demoTitle}</h2>
              <p className="mt-4 max-w-sm text-sm leading-relaxed text-slate-400">{c.demoBody}</p>
            </div>
            {status.kind === "ok" ? (
              <div className="rounded-2xl border border-cyan-400/30 bg-cyan-400/5 p-8 text-center">
                <div className="mx-auto grid size-11 place-items-center rounded-full bg-cyan-400/15 text-xl text-cyan-300">
                  ✓
                </div>
                <h3 className="mt-4 text-lg font-bold text-white">{c.okTitle}</h3>
                <p className="mt-1 text-sm text-slate-400">{c.okBody}</p>
              </div>
            ) : (
              <form onSubmit={onSubmit} noValidate className="grid items-end gap-4 md:grid-cols-2 xl:grid-cols-5">
                <Field name="full_name" label={c.fFullName} ph={c.fFullNamePh} />
                <Field name="email" label={c.fEmail} ph={c.fEmailPh} type="email" />
                <Field name="company" label={c.fCompany} ph={c.fCompanyPh} />
                <label className="block text-xs font-medium text-slate-400">
                  {c.fIndustry}
                  <select
                    name="industry"
                    defaultValue=""
                    className="mt-1.5 w-full rounded-lg border border-white/15 bg-[#0a1324] px-3 py-2.5 text-sm text-white focus:border-cyan-400/60 focus:outline-none"
                  >
                    <option value="" className="bg-[#0a1324]">
                      {c.fIndustryPh}
                    </option>
                    {c.industries.map((ind) => (
                      <option key={ind} value={ind} className="bg-[#0a1324]">
                        {ind}
                      </option>
                    ))}
                  </select>
                </label>
                {/* Honeypot */}
                <input
                  name="website"
                  tabIndex={-1}
                  autoComplete="off"
                  aria-hidden="true"
                  className="absolute left-[-9999px] h-0 w-0 opacity-0"
                />
                <button
                  type="submit"
                  disabled={status.kind === "sending"}
                  className="inline-flex h-[42px] items-center justify-center gap-2 self-end rounded-lg bg-gradient-to-r from-cyan-400 to-cyan-500 px-6 text-sm font-semibold text-[#060b18] shadow-[0_0_24px_rgba(34,211,238,0.4)] transition-all hover:shadow-[0_0_36px_rgba(34,211,238,0.6)] disabled:opacity-60"
                >
                  {status.kind === "sending" ? c.sending : c.requestDemo} <ArrowRightIcon className="size-4" />
                </button>
                <p className="text-[11px] text-slate-500 md:col-span-2 xl:col-span-5">{c.consentNote}</p>
                {status.kind === "err" ? (
                  <p className="text-sm text-rose-400 md:col-span-2 xl:col-span-5">{status.msg}</p>
                ) : null}
              </form>
            )}
          </div>
        </div>
      </section>
    </>
  );
}

function Field({ name, label, ph, type = "text" }: { name: string; label: string; ph: string; type?: string }) {
  return (
    <label className="block text-xs font-medium text-slate-400">
      {label}
      <input
        name={name}
        type={type}
        placeholder={ph}
        className="mt-1.5 w-full rounded-lg border border-white/15 bg-[#0a1324] px-3 py-2.5 text-sm text-white placeholder:text-slate-600 focus:border-cyan-400/60 focus:outline-none"
      />
    </label>
  );
}
