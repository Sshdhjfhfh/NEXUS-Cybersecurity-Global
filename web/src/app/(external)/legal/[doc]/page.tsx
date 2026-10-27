"use client";

import Link from "next/link";
import { useParams } from "next/navigation";

import { useSiteI18n } from "../../_site/i18n";
import { LegalDoc } from "../../_site/legal";
import { PageHeader } from "../../_site/sections";
import { LEGAL_DOCS } from "../legal-content";

export default function LegalDocPage() {
  const params = useParams();
  const { lang } = useSiteI18n();
  const doc = String(params?.doc ?? "");
  const content = LEGAL_DOCS[doc];

  if (!content) {
    return (
      <>
        <PageHeader
          kicker="Legal"
          title={lang === "en" ? "Document not found" : "Documento no encontrado"}
          body={
            lang === "en"
              ? "This legal document does not exist or has not been published yet."
              : "Este documento legal no existe o aún no se ha publicado."
          }
        />
        <section className="mx-auto max-w-3xl px-6 py-12">
          <Link href="/legal" className="text-sm font-medium text-cyan-300 hover:text-cyan-200">
            {lang === "en" ? "← Back to Legal" : "← Volver a Legal"}
          </Link>
        </section>
      </>
    );
  }

  return <LegalDoc docKey={doc} content={content} />;
}
