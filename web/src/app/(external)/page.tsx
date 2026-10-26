"use client";

import Link from "next/link";

import { useSiteI18n } from "./_site/i18n";

function Pillar({ title, body, accent }: { title: string; body: string; accent: string }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-6 transition-colors hover:border-cyan-400/40">
      <div className={`mb-4 h-1 w-10 rounded-full ${accent}`} />
      <h3 className="text-lg font-semibold text-white">{title}</h3>
      <p className="mt-2 text-sm leading-relaxed text-cyan-100/70">{body}</p>
    </div>
  );
}

export default function Home() {
  const { t } = useSiteI18n();
  return (
    <>
      {/* Hero */}
      <section className="relative overflow-hidden">
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(60%_60%_at_50%_0%,rgba(34,211,238,0.18),transparent_70%)]" />
        <div className="mx-auto max-w-6xl px-4 py-24 text-center">
          <span className="inline-flex items-center gap-2 rounded-full border border-cyan-400/30 bg-cyan-400/10 px-4 py-1.5 text-xs font-medium text-cyan-200">
            <span className="size-1.5 animate-pulse rounded-full bg-cyan-400" />
            {t("hero.badge")}
          </span>
          <h1 className="mx-auto mt-6 max-w-4xl text-balance text-4xl font-bold leading-tight tracking-tight text-white sm:text-5xl">
            {t("hero.title")}
          </h1>
          <p className="mx-auto mt-6 max-w-2xl text-pretty text-base leading-relaxed text-cyan-100/80 sm:text-lg">
            {t("hero.subtitle")}
          </p>
          <div className="mt-9 flex flex-wrap items-center justify-center gap-3">
            <Link
              href="/request-demo"
              className="rounded-full bg-cyan-400 px-6 py-3 text-sm font-semibold text-[#061024] transition-colors hover:bg-cyan-300"
            >
              {t("hero.cta.demo")}
            </Link>
            <Link
              href="/solutions"
              className="rounded-full border border-white/15 px-6 py-3 text-sm font-semibold text-white transition-colors hover:border-cyan-400/50"
            >
              {t("hero.cta.solutions")}
            </Link>
          </div>
          <p className="mx-auto mt-6 max-w-xl text-xs text-cyan-100/50">{t("hero.note")}</p>
        </div>
      </section>

      {/* Pillars */}
      <section id="research" className="mx-auto max-w-6xl px-4 py-16">
        <h2 className="text-center text-2xl font-bold tracking-tight text-white sm:text-3xl">{t("pillars.title")}</h2>
        <div className="mt-10 grid gap-5 md:grid-cols-3">
          <Pillar title={t("pillars.red.title")} body={t("pillars.red.body")} accent="bg-rose-400" />
          <Pillar title={t("pillars.blue.title")} body={t("pillars.blue.body")} accent="bg-cyan-400" />
          <Pillar title={t("pillars.research.title")} body={t("pillars.research.body")} accent="bg-violet-400" />
        </div>
      </section>

      {/* Trust */}
      <section id="trust" className="border-y border-white/10 bg-white/[0.02]">
        <div className="mx-auto max-w-6xl px-4 py-16">
          <div className="grid items-center gap-10 md:grid-cols-2">
            <div>
              <h2 className="text-2xl font-bold tracking-tight text-white sm:text-3xl">{t("trust.title")}</h2>
              <p className="mt-4 text-sm leading-relaxed text-cyan-100/75">{t("trust.body")}</p>
            </div>
            <div className="flex flex-wrap gap-3">
              {[t("trust.badge1"), t("trust.badge2"), t("trust.badge3")].map((b) => (
                <span
                  key={b}
                  className="rounded-xl border border-cyan-400/20 bg-cyan-400/5 px-4 py-3 text-sm font-medium text-cyan-100"
                >
                  {b}
                </span>
              ))}
            </div>
          </div>
        </div>
      </section>

      {/* CTA */}
      <section className="mx-auto max-w-6xl px-4 py-20 text-center">
        <h2 className="text-2xl font-bold tracking-tight text-white sm:text-3xl">{t("cta.title")}</h2>
        <p className="mx-auto mt-3 max-w-xl text-sm text-cyan-100/75">{t("cta.body")}</p>
        <Link
          href="/request-demo"
          className="mt-8 inline-block rounded-full bg-cyan-400 px-8 py-3 text-sm font-semibold text-[#061024] transition-colors hover:bg-cyan-300"
        >
          {t("cta.button")}
        </Link>
      </section>
    </>
  );
}
