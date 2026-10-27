"use client";

import { AlertTriangleIcon } from "lucide-react";

import { legalDocsStatus } from "@/config/company";

import type { LegalDocContent } from "../legal/legal-content";
import { useSiteI18n } from "./i18n";
import { PageHeader } from "./sections";

function DraftBanner({ lang, version }: { lang: "en" | "es"; version?: string }) {
  return (
    <div className="flex items-start gap-3 rounded-xl border border-amber-400/30 bg-amber-400/5 p-4">
      <AlertTriangleIcon className="mt-0.5 size-5 shrink-0 text-amber-400" />
      <div className="text-xs leading-relaxed text-amber-100/80">
        <strong className="font-semibold text-amber-200">
          {lang === "en" ? "Draft — pending legal review." : "Borrador — pendiente de revisión legal."}
        </strong>{" "}
        {lang === "en"
          ? "This document is preliminary and not yet legally binding. Version and effective date are confirmed once reviewed."
          : "Este documento es preliminar y aún no es jurídicamente vinculante. La versión y la fecha de entrada en vigor se confirman tras la revisión."}
        <span className="ml-1 text-amber-100/60">
          {lang === "en" ? "Version:" : "Versión:"} {version ?? "PENDIENTE"}
        </span>
      </div>
    </div>
  );
}

export function LegalDoc({ docKey, content }: { docKey: string; content: LegalDocContent }) {
  const { lang } = useSiteI18n();
  const meta = legalDocsStatus[docKey];
  const sections = content.sections[lang];
  return (
    <>
      <PageHeader kicker="Legal" title={content.title[lang]} body={content.intro[lang]} />
      <section className="mx-auto max-w-3xl px-6 py-12">
        <DraftBanner lang={lang} version={meta?.version} />
        <div className="mt-8 space-y-7">
          {sections.map((s) => (
            <div key={s.h}>
              <h2 className="text-lg font-semibold text-white">{s.h}</h2>
              {s.p.map((para) => (
                <p key={para} className="mt-2 text-sm leading-relaxed text-slate-400">
                  {para}
                </p>
              ))}
            </div>
          ))}
        </div>
      </section>
    </>
  );
}
