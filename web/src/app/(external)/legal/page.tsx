"use client";

import Link from "next/link";

import { ArrowRightIcon } from "lucide-react";

import { useSiteI18n } from "../_site/i18n";
import { PageHeader, Section } from "../_site/sections";
import { LEGAL_DOCS, LEGAL_ORDER } from "./legal-content";

// Docs that are planned but not yet drafted — listed honestly as pending, with no
// link to empty content.
const PENDING_DOCS = {
  en: [
    { t: "Data Processing Agreement (DPA)" },
    { t: "Service Level Agreement (SLA)" },
    { t: "Subprocessor List" },
    { t: "Legal Notice" },
  ],
  es: [
    { t: "Acuerdo de Tratamiento de Datos (DPA)" },
    { t: "Acuerdo de Nivel de Servicio (SLA)" },
    { t: "Lista de Subencargados" },
    { t: "Aviso Legal" },
  ],
};

export default function LegalIndex() {
  const { lang } = useSiteI18n();
  return (
    <>
      <PageHeader
        kicker="Legal"
        title={lang === "en" ? "Legal & Policies" : "Legal y Políticas"}
        body={
          lang === "en"
            ? "Our policies and agreements. Published documents are preliminary drafts pending legal review; others are listed as pending."
            : "Nuestras políticas y acuerdos. Los documentos publicados son borradores preliminares pendientes de revisión legal; los demás se listan como pendientes."
        }
      />
      <Section>
        <div className="grid gap-4 sm:grid-cols-2">
          {LEGAL_ORDER.map((key) => (
            <Link
              key={key}
              href={`/legal/${key}`}
              className="group flex items-center justify-between gap-3 rounded-xl border border-white/10 bg-white/[0.03] p-5 transition-colors hover:border-cyan-400/40"
            >
              <div>
                <div className="text-sm font-semibold text-white">{LEGAL_DOCS[key].title[lang]}</div>
                <span className="mt-1 inline-block rounded-full border border-amber-400/30 bg-amber-400/5 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-amber-200">
                  {lang === "en" ? "Draft" : "Borrador"}
                </span>
              </div>
              <ArrowRightIcon className="size-4 text-slate-500 transition-transform group-hover:translate-x-1" />
            </Link>
          ))}
          {PENDING_DOCS[lang].map((d) => (
            <div
              key={d.t}
              className="flex items-center justify-between gap-3 rounded-xl border border-white/5 bg-white/[0.01] p-5 opacity-70"
            >
              <div>
                <div className="text-sm font-semibold text-slate-300">{d.t}</div>
                <span className="mt-1 inline-block rounded-full border border-slate-500/30 bg-slate-500/5 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-400">
                  {lang === "en" ? "Pending" : "Pendiente"}
                </span>
              </div>
            </div>
          ))}
        </div>
      </Section>
    </>
  );
}
