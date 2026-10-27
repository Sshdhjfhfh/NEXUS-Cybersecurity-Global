"use client";

import { useState } from "react";

import Link from "next/link";

import { ChevronDownIcon, GlobeIcon, MenuIcon, XIcon } from "lucide-react";

import { openCookiePreferences } from "./cookie-consent";
import { useSiteI18n } from "./i18n";

// Public marketing site chrome, matching the NEXUS Cybersecurity Global design:
// a slim dark header with a mega-nav and a rich multi-column footer.

const NAV = {
  es: [
    { label: "Plataforma", href: "/platform", caret: false },
    { label: "Soluciones", href: "/solutions", caret: false },
    { label: "Investigación IA", href: "/research", caret: false },
    { label: "Servicios", href: "/services", caret: false },
    { label: "Trust Center", href: "/#trust", caret: false },
    { label: "Nosotros", href: "/about", caret: false },
    { label: "Contacto", href: "/contact", caret: false },
  ],
  en: [
    { label: "Platform", href: "/platform", caret: false },
    { label: "Solutions", href: "/solutions", caret: false },
    { label: "AI Research", href: "/research", caret: false },
    { label: "Services", href: "/services", caret: false },
    { label: "Trust Center", href: "/#trust", caret: false },
    { label: "About", href: "/about", caret: false },
    { label: "Contact", href: "/contact", caret: false },
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

// Footer links point only to routes that actually exist (no decorative or 404 links).
const FOOTER_COLS = {
  en: [
    {
      title: "Platform",
      links: [
        { label: "Overview", href: "/platform" },
        { label: "Security Engine", href: "/platform" },
        { label: "Evolution Lab", href: "/research" },
        { label: "Pricing", href: "/pricing" },
      ],
    },
    {
      title: "Solutions",
      links: [
        { label: "Red Team", href: "/services" },
        { label: "Blue Team", href: "/services" },
        { label: "Purple Team", href: "/services" },
        { label: "AI Research", href: "/research" },
      ],
    },
    {
      title: "Services",
      links: [
        { label: "Professional Services", href: "/services" },
        { label: "Solutions", href: "/solutions" },
        { label: "Pricing", href: "/pricing" },
        { label: "Request a demo", href: "/request-demo" },
      ],
    },
    {
      title: "Company",
      links: [
        { label: "About", href: "/about" },
        { label: "Research", href: "/research" },
        { label: "Trust Center", href: "/#trust" },
        { label: "Contact", href: "/contact" },
      ],
    },
    {
      title: "Resources",
      links: [
        { label: "Platform", href: "/platform" },
        { label: "Trust Center", href: "/#trust" },
        { label: "Support", href: "/contact" },
        { label: "Request a demo", href: "/request-demo" },
      ],
    },
  ],
  es: [
    {
      title: "Plataforma",
      links: [
        { label: "Visión general", href: "/platform" },
        { label: "Motor de seguridad", href: "/platform" },
        { label: "Evolution Lab", href: "/research" },
        { label: "Precios", href: "/pricing" },
      ],
    },
    {
      title: "Soluciones",
      links: [
        { label: "Red Team", href: "/services" },
        { label: "Blue Team", href: "/services" },
        { label: "Purple Team", href: "/services" },
        { label: "Investigación IA", href: "/research" },
      ],
    },
    {
      title: "Servicios",
      links: [
        { label: "Servicios profesionales", href: "/services" },
        { label: "Soluciones", href: "/solutions" },
        { label: "Precios", href: "/pricing" },
        { label: "Solicitar demo", href: "/request-demo" },
      ],
    },
    {
      title: "Empresa",
      links: [
        { label: "Nosotros", href: "/about" },
        { label: "Investigación", href: "/research" },
        { label: "Trust Center", href: "/#trust" },
        { label: "Contacto", href: "/contact" },
      ],
    },
    {
      title: "Recursos",
      links: [
        { label: "Plataforma", href: "/platform" },
        { label: "Trust Center", href: "/#trust" },
        { label: "Soporte", href: "/contact" },
        { label: "Solicitar demo", href: "/request-demo" },
      ],
    },
  ],
};

const LEGAL = {
  en: [
    { label: "Legal", href: "/legal" },
    { label: "Terms of Service", href: "/legal/terms" },
    { label: "Privacy Policy", href: "/legal/privacy" },
    { label: "Cookie Policy", href: "/legal/cookies" },
    { label: "Acceptable Use", href: "/legal/aup" },
    { label: "Security", href: "/legal/security" },
    { label: "Vulnerability Disclosure", href: "/legal/vdp" },
  ],
  es: [
    { label: "Legal", href: "/legal" },
    { label: "Términos del servicio", href: "/legal/terms" },
    { label: "Privacidad", href: "/legal/privacy" },
    { label: "Política de Cookies", href: "/legal/cookies" },
    { label: "Uso aceptable", href: "/legal/aup" },
    { label: "Seguridad", href: "/legal/security" },
    { label: "Divulgación de vulnerabilidades", href: "/legal/vdp" },
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
                  <li key={l.label}>
                    <Link href={l.href} className="transition-colors hover:text-cyan-300">
                      {l.label}
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
              <Link key={l.label} href={l.href} className="transition-colors hover:text-slate-300">
                {l.label}
              </Link>
            ))}
            <button
              type="button"
              onClick={() => openCookiePreferences()}
              className="transition-colors hover:text-slate-300"
            >
              {lang === "en" ? "Cookie preferences" : "Preferencias de cookies"}
            </button>
          </div>
        </div>
      </div>
    </footer>
  );
}
