"use client";

import Link from "next/link";

import { LifeBuoyIcon, LockIcon, MailIcon, ShieldAlertIcon } from "lucide-react";

import { company } from "@/config/company";

import { useSiteI18n } from "../_site/i18n";
import { PageHeader, Section } from "../_site/sections";

const COPY = {
  en: {
    kicker: "Contact",
    title: "Talk to NEXUS.",
    body: "Reach the right team directly, or request a guided demo of the platform.",
    channels: [
      { icon: "mail", t: "General", d: "Questions, partnerships and press.", email: company.email.general },
      { icon: "support", t: "Support", d: "Help for existing customers.", email: company.email.support },
      { icon: "privacy", t: "Privacy", d: "Data requests and privacy matters.", email: company.email.privacy },
      {
        icon: "security",
        t: "Security",
        d: "Report a vulnerability in NEXUS responsibly.",
        email: company.email.security,
      },
    ],
    provisional: "Email addresses are provisional and will be replaced with monitored mailboxes before launch.",
    demoTitle: "Prefer a walkthrough?",
    demoBody: "Request a demo and we'll tailor it to your environment.",
    demo: "Request a demo",
  },
  es: {
    kicker: "Contacto",
    title: "Habla con NEXUS.",
    body: "Llega directo al equipo correcto, o solicita una demo guiada de la plataforma.",
    channels: [
      { icon: "mail", t: "General", d: "Consultas, alianzas y prensa.", email: company.email.general },
      { icon: "support", t: "Soporte", d: "Ayuda para clientes actuales.", email: company.email.support },
      { icon: "privacy", t: "Privacidad", d: "Solicitudes de datos y privacidad.", email: company.email.privacy },
      {
        icon: "security",
        t: "Seguridad",
        d: "Reporta una vulnerabilidad en NEXUS de forma responsable.",
        email: company.email.security,
      },
    ],
    provisional:
      "Las direcciones de correo son provisionales y se reemplazarán por buzones monitoreados antes del lanzamiento.",
    demoTitle: "¿Prefieres un recorrido?",
    demoBody: "Solicita una demo y la adaptamos a tu entorno.",
    demo: "Solicitar demo",
  },
};

const ICON = { mail: MailIcon, support: LifeBuoyIcon, privacy: LockIcon, security: ShieldAlertIcon } as const;

export default function Contact() {
  const { lang } = useSiteI18n();
  const c = COPY[lang];
  return (
    <>
      <PageHeader kicker={c.kicker} title={c.title} body={c.body} />
      <Section>
        <div className="grid gap-5 sm:grid-cols-2">
          {c.channels.map((ch) => {
            const Icon = ICON[ch.icon as keyof typeof ICON];
            return (
              <div key={ch.t} className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
                <Icon className="size-6 text-cyan-300" />
                <h3 className="mt-4 text-lg font-semibold text-white">{ch.t}</h3>
                <p className="mt-1 text-sm text-slate-400">{ch.d}</p>
                <a
                  href={`mailto:${ch.email}`}
                  className="mt-3 inline-block text-sm font-medium text-cyan-300 hover:text-cyan-200"
                >
                  {ch.email}
                </a>
              </div>
            );
          })}
        </div>
        {company.emailsAreProvisional ? <p className="mt-6 text-xs text-slate-500">{c.provisional}</p> : null}

        <div className="mt-12 flex flex-col items-start justify-between gap-5 rounded-2xl border border-cyan-400/20 bg-cyan-400/5 p-8 sm:flex-row sm:items-center">
          <div>
            <h3 className="text-lg font-bold text-white">{c.demoTitle}</h3>
            <p className="mt-1 text-sm text-slate-400">{c.demoBody}</p>
          </div>
          <Link
            href="/request-demo"
            className="shrink-0 rounded-lg bg-gradient-to-r from-cyan-400 to-cyan-500 px-6 py-3 text-sm font-semibold text-[#060b18] shadow-[0_0_24px_rgba(34,211,238,0.4)]"
          >
            {c.demo}
          </Link>
        </div>
      </Section>
    </>
  );
}
