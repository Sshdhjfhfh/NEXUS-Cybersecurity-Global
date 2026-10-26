"use client";

import Link from "next/link";

import { useSiteI18n } from "../_site/i18n";

function SolutionCard({ title, body, tag }: { title: string; body: string; tag?: string }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-lg font-semibold text-white">{title}</h3>
        {tag ? (
          <span className="rounded-full border border-violet-400/40 bg-violet-400/10 px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-violet-200">
            {tag}
          </span>
        ) : null}
      </div>
      <p className="mt-2 text-sm leading-relaxed text-cyan-100/70">{body}</p>
    </div>
  );
}

export default function Solutions() {
  const { t } = useSiteI18n();
  return (
    <section className="mx-auto max-w-6xl px-4 py-20">
      <h1 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">{t("solutions.title")}</h1>
      <p className="mt-3 max-w-2xl text-sm text-cyan-100/75">{t("solutions.subtitle")}</p>
      <div className="mt-10 grid gap-5 md:grid-cols-2">
        <SolutionCard title={t("sol.offensive.t")} body={t("sol.offensive.b")} />
        <SolutionCard title={t("sol.defensive.t")} body={t("sol.defensive.b")} />
        <SolutionCard title={t("sol.purple.t")} body={t("sol.purple.b")} tag="Beta" />
        <SolutionCard title={t("sol.research.t")} body={t("sol.research.b")} tag="Research" />
      </div>
      <div className="mt-12">
        <Link
          href="/request-demo"
          className="inline-block rounded-full bg-cyan-400 px-6 py-3 text-sm font-semibold text-[#061024] transition-colors hover:bg-cyan-300"
        >
          {t("nav.demo")}
        </Link>
      </div>
    </section>
  );
}
