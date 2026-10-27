"use client";

import { useState } from "react";

import Link from "next/link";

import { ChevronDownIcon, GlobeIcon, MenuIcon, SearchIcon, XIcon } from "lucide-react";

import { useSiteI18n } from "./i18n";

// Public marketing site chrome, matching the NEXUS Cybersecurity Global design:
// a slim dark header with a mega-nav and a rich multi-column footer.

// Brand marks as inline SVG (lucide no longer ships brand icons).
type IconProps = { className?: string };
function LinkedinIcon({ className }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} role="img" aria-label="LinkedIn">
      <title>LinkedIn</title>
      <path d="M20.45 20.45h-3.56v-5.57c0-1.33-.02-3.04-1.85-3.04-1.85 0-2.14 1.45-2.14 2.94v5.67H9.35V9h3.41v1.56h.05c.48-.9 1.64-1.85 3.37-1.85 3.6 0 4.27 2.37 4.27 5.46v6.28zM5.34 7.43a2.06 2.06 0 1 1 0-4.13 2.06 2.06 0 0 1 0 4.13zM7.12 20.45H3.56V9h3.56v11.45zM22.22 0H1.77C.79 0 0 .77 0 1.73v20.54C0 23.22.79 24 1.77 24h20.45c.98 0 1.78-.78 1.78-1.73V1.73C24 .77 23.2 0 22.22 0z" />
    </svg>
  );
}
function TwitterIcon({ className }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} role="img" aria-label="X">
      <title>X</title>
      <path d="M18.24 2.25h3.31l-7.23 8.26 8.5 11.24h-6.65l-5.2-6.82-5.96 6.82H1.69l7.73-8.84L1.26 2.25h6.82l4.71 6.23 5.45-6.23zm-1.16 17.52h1.84L7.01 4.13H5.03l12.05 15.64z" />
    </svg>
  );
}
function YoutubeIcon({ className }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} role="img" aria-label="YouTube">
      <title>YouTube</title>
      <path d="M23.5 6.2a3.02 3.02 0 0 0-2.12-2.14C19.5 3.55 12 3.55 12 3.55s-7.5 0-9.38.51A3.02 3.02 0 0 0 .5 6.2 31.6 31.6 0 0 0 0 12a31.6 31.6 0 0 0 .5 5.8 3.02 3.02 0 0 0 2.12 2.14c1.88.51 9.38.51 9.38.51s7.5 0 9.38-.51a3.02 3.02 0 0 0 2.12-2.14A31.6 31.6 0 0 0 24 12a31.6 31.6 0 0 0-.5-5.8zM9.55 15.57V8.43L15.82 12l-6.27 3.57z" />
    </svg>
  );
}
function GithubIcon({ className }: IconProps) {
  return (
    <svg viewBox="0 0 24 24" fill="currentColor" className={className} role="img" aria-label="GitHub">
      <title>GitHub</title>
      <path d="M12 .5A11.5 11.5 0 0 0 .5 12a11.5 11.5 0 0 0 7.86 10.92c.58.1.79-.25.79-.56v-2c-3.2.7-3.88-1.37-3.88-1.37-.53-1.34-1.3-1.7-1.3-1.7-1.06-.72.08-.71.08-.71 1.17.08 1.79 1.2 1.79 1.2 1.04 1.79 2.73 1.27 3.4.97.1-.76.4-1.27.73-1.56-2.55-.29-5.24-1.28-5.24-5.69 0-1.26.45-2.29 1.19-3.1-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.18 1.18a11 11 0 0 1 5.8 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.59.23 2.76.11 3.05.74.81 1.19 1.84 1.19 3.1 0 4.42-2.69 5.39-5.25 5.68.41.36.78 1.06.78 2.14v3.17c0 .31.21.67.8.56A11.5 11.5 0 0 0 23.5 12 11.5 11.5 0 0 0 12 .5z" />
    </svg>
  );
}

const NAV = {
  es: [
    { label: "Plataforma", href: "/solutions", caret: true },
    { label: "Soluciones", href: "/solutions", caret: true },
    { label: "Investigación IA", href: "/#ai-research", caret: false },
    { label: "Servicios", href: "/solutions", caret: true },
    { label: "Trust Center", href: "/#trust", caret: true },
    { label: "Nosotros", href: "/#about", caret: true },
    { label: "Contacto", href: "/request-demo", caret: false },
  ],
  en: [
    { label: "Platform", href: "/solutions", caret: true },
    { label: "Solutions", href: "/solutions", caret: true },
    { label: "AI Research", href: "/#ai-research", caret: false },
    { label: "Services", href: "/solutions", caret: true },
    { label: "Trust Center", href: "/#trust", caret: true },
    { label: "About", href: "/#about", caret: true },
    { label: "Contact", href: "/request-demo", caret: false },
  ],
};

function LangToggle() {
  const { lang, setLang } = useSiteI18n();
  return (
    <button
      type="button"
      onClick={() => setLang(lang === "en" ? "es" : "en")}
      className="inline-flex items-center gap-1.5 rounded-md border border-white/15 px-2.5 py-1.5 text-xs font-medium text-slate-200 transition-colors hover:border-cyan-400/50 hover:text-white"
      aria-label="Change language"
    >
      <GlobeIcon className="size-3.5" />
      {lang.toUpperCase()}
      <ChevronDownIcon className="size-3 opacity-60" />
    </button>
  );
}

function Wordmark() {
  return (
    <Link href="/" className="flex items-center gap-2.5">
      <span className="grid size-9 place-items-center rounded-lg bg-gradient-to-br from-cyan-400 to-blue-600 text-[15px] font-black text-[#060b18] shadow-[0_0_20px_rgba(34,211,238,0.35)]">
        N
      </span>
      <span className="leading-none">
        <span className="block text-base font-bold tracking-tight text-white">NEXUS</span>
        <span className="block text-[9px] font-medium uppercase tracking-[0.25em] text-cyan-300/70">
          Cybersecurity Global
        </span>
      </span>
    </Link>
  );
}

export function SiteHeader() {
  const { lang } = useSiteI18n();
  const nav = NAV[lang];
  const [open, setOpen] = useState(false);
  return (
    <header className="sticky top-0 z-50 border-b border-white/10 bg-[#060b18]/85 backdrop-blur-md">
      <div className="mx-auto flex h-[68px] max-w-[1400px] items-center justify-between gap-6 px-6">
        <Wordmark />
        <nav className="hidden items-center gap-6 text-[13px] font-medium text-slate-300 lg:flex">
          {nav.map((item) => (
            <Link
              key={item.label}
              href={item.href}
              className="inline-flex items-center gap-1 transition-colors hover:text-white"
            >
              {item.label}
              {item.caret ? <ChevronDownIcon className="size-3.5 opacity-50" /> : null}
            </Link>
          ))}
        </nav>
        <div className="flex items-center gap-3">
          <button
            type="button"
            aria-label="Search"
            className="hidden text-slate-300 transition-colors hover:text-white sm:block"
          >
            <SearchIcon className="size-[18px]" />
          </button>
          <Link
            href="/request-demo"
            className="hidden items-center gap-1.5 rounded-lg bg-gradient-to-r from-cyan-400 to-cyan-500 px-4 py-2 text-[13px] font-semibold text-[#060b18] shadow-[0_0_20px_rgba(34,211,238,0.35)] transition-all hover:shadow-[0_0_28px_rgba(34,211,238,0.5)] sm:inline-flex"
          >
            {lang === "en" ? "Request Demo" : "Solicitar demo"} <span aria-hidden>→</span>
          </Link>
          <LangToggle />
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            aria-label={open ? "Close menu" : "Open menu"}
            aria-expanded={open}
            className="text-slate-200 transition-colors hover:text-white lg:hidden"
          >
            {open ? <XIcon className="size-6" /> : <MenuIcon className="size-6" />}
          </button>
        </div>
      </div>

      {/* Mobile menu */}
      {open ? (
        <nav className="border-t border-white/10 bg-[#060b18] px-6 py-4 lg:hidden">
          <ul className="flex flex-col divide-y divide-white/5">
            {nav.map((item) => (
              <li key={item.label}>
                <Link
                  href={item.href}
                  onClick={() => setOpen(false)}
                  className="block py-3 text-sm font-medium text-slate-200 transition-colors hover:text-cyan-300"
                >
                  {item.label}
                </Link>
              </li>
            ))}
          </ul>
          <Link
            href="/request-demo"
            onClick={() => setOpen(false)}
            className="mt-4 flex items-center justify-center gap-1.5 rounded-lg bg-gradient-to-r from-cyan-400 to-cyan-500 px-4 py-2.5 text-sm font-semibold text-[#060b18]"
          >
            {lang === "en" ? "Request Demo" : "Solicitar demo"} <span aria-hidden>→</span>
          </Link>
        </nav>
      ) : null}
    </header>
  );
}

const FOOTER_COLS = {
  en: [
    { title: "Platform", links: ["Overview", "Features", "Integrations", "Roadmap"] },
    { title: "Solutions", links: ["Red Team", "Blue Team", "Purple Team", "AI Research"] },
    { title: "Services", links: ["Advisory", "Managed Services", "Incident Response", "Training"] },
    { title: "Company", links: ["About Us", "Careers", "News", "Contact"] },
    { title: "Resources", links: ["Trust Center", "Blog", "Events", "Support"] },
  ],
  es: [
    { title: "Plataforma", links: ["Visión general", "Funciones", "Integraciones", "Hoja de ruta"] },
    { title: "Soluciones", links: ["Red Team", "Blue Team", "Purple Team", "Investigación IA"] },
    { title: "Servicios", links: ["Consultoría", "Servicios gestionados", "Respuesta a incidentes", "Formación"] },
    { title: "Empresa", links: ["Nosotros", "Empleo", "Noticias", "Contacto"] },
    { title: "Recursos", links: ["Trust Center", "Blog", "Eventos", "Soporte"] },
  ],
};

const LEGAL = {
  en: ["Terms of Service", "Privacy Policy", "Acceptable Use Policy", "Security Policy", "Vulnerability Disclosure"],
  es: [
    "Términos del servicio",
    "Privacidad",
    "Política de uso aceptable",
    "Política de seguridad",
    "Divulgación de vulnerabilidades",
  ],
};

export function SiteFooter() {
  const { lang } = useSiteI18n();
  const cols = FOOTER_COLS[lang];
  return (
    <footer className="relative overflow-hidden border-t border-white/10 bg-[#060b18] text-slate-400">
      <div className="pointer-events-none absolute -bottom-40 -right-32 size-96 rounded-full bg-[radial-gradient(circle,rgba(37,99,235,0.18),transparent_70%)]" />
      <div className="mx-auto max-w-[1400px] px-6 py-14">
        <div className="grid gap-10 lg:grid-cols-[1.3fr_repeat(5,1fr)]">
          <div>
            <Wordmark />
            <p className="mt-4 max-w-xs text-sm leading-relaxed text-slate-500">
              {lang === "en"
                ? "AI-powered threat intelligence and proactive defense for the world's most critical organizations."
                : "Inteligencia de amenazas con IA y defensa proactiva para las organizaciones más críticas del mundo."}
            </p>
            <p className="mt-6 text-[11px] font-semibold uppercase tracking-[0.2em] text-slate-500">
              {lang === "en" ? "A safer digital world. Together." : "Un mundo digital más seguro. Juntos."}
            </p>
          </div>
          {cols.map((col) => (
            <div key={col.title}>
              <div className="text-[11px] font-semibold uppercase tracking-wider text-slate-300">{col.title}</div>
              <ul className="mt-4 space-y-2.5 text-[13px]">
                {col.links.map((l) => (
                  <li key={l}>
                    <Link href="/solutions" className="transition-colors hover:text-cyan-300">
                      {l}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="mt-12 flex flex-col-reverse items-start justify-between gap-6 border-t border-white/10 pt-6 md:flex-row md:items-center">
          <p className="text-xs text-slate-500">
            © {new Date().getFullYear()} NEXUS Cybersecurity Global.{" "}
            {lang === "en" ? "All rights reserved." : "Todos los derechos reservados."}
          </p>
          <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-[11px] text-slate-500">
            {LEGAL[lang].map((l) => (
              <Link key={l} href="/#trust" className="transition-colors hover:text-slate-300">
                {l}
              </Link>
            ))}
          </div>
          <div className="flex items-center gap-4 text-slate-400">
            <Link href="/#" aria-label="LinkedIn" className="transition-colors hover:text-cyan-300">
              <LinkedinIcon className="size-[18px]" />
            </Link>
            <Link href="/#" aria-label="X" className="transition-colors hover:text-cyan-300">
              <TwitterIcon className="size-[18px]" />
            </Link>
            <Link href="/#" aria-label="YouTube" className="transition-colors hover:text-cyan-300">
              <YoutubeIcon className="size-[18px]" />
            </Link>
            <Link href="/#" aria-label="GitHub" className="transition-colors hover:text-cyan-300">
              <GithubIcon className="size-[18px]" />
            </Link>
          </div>
        </div>
      </div>
    </footer>
  );
}
