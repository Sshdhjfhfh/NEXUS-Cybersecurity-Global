"use client";

import Link from "next/link";

import { type SiteLang, useSiteI18n } from "./i18n";

function LangToggle() {
  const { lang, setLang } = useSiteI18n();
  const opts: SiteLang[] = ["es", "en"];
  return (
    <div className="inline-flex overflow-hidden rounded-full border border-cyan-400/30 text-xs font-medium">
      {opts.map((l) => (
        <button
          key={l}
          type="button"
          onClick={() => setLang(l)}
          aria-pressed={lang === l}
          className={`px-3 py-1 uppercase transition-colors ${
            lang === l ? "bg-cyan-400 text-[#061024]" : "text-cyan-100/70 hover:text-white"
          }`}
        >
          {l}
        </button>
      ))}
    </div>
  );
}

export function SiteHeader() {
  const { t } = useSiteI18n();
  return (
    <header className="sticky top-0 z-40 border-b border-white/10 bg-[#061024]/90 backdrop-blur">
      <div className="mx-auto flex h-16 max-w-6xl items-center justify-between gap-4 px-4">
        <Link href="/" className="flex items-center gap-2">
          <span className="grid size-8 place-items-center rounded-md bg-gradient-to-br from-cyan-400 to-blue-600 font-black text-[#061024]">
            N
          </span>
          <span className="text-lg font-semibold tracking-tight text-white">
            NEXUS <span className="text-cyan-400">Cybersecurity</span>
          </span>
        </Link>
        <nav className="hidden items-center gap-6 text-sm text-cyan-100/80 md:flex">
          <Link href="/solutions" className="hover:text-white">
            {t("nav.solutions")}
          </Link>
          <Link href="/#research" className="hover:text-white">
            {t("nav.research")}
          </Link>
          <Link href="/#trust" className="hover:text-white">
            {t("nav.platform")}
          </Link>
          <Link href="/function/tasks" className="hover:text-white">
            {t("nav.console")}
          </Link>
        </nav>
        <div className="flex items-center gap-3">
          <LangToggle />
          <Link
            href="/request-demo"
            className="rounded-full bg-cyan-400 px-4 py-2 text-sm font-semibold text-[#061024] transition-colors hover:bg-cyan-300"
          >
            {t("nav.demo")}
          </Link>
        </div>
      </div>
    </header>
  );
}

export function SiteFooter() {
  const { t } = useSiteI18n();
  const year = new Date().getFullYear();
  return (
    <footer className="border-t border-white/10 bg-[#061024] text-cyan-100/70">
      <div className="mx-auto max-w-6xl px-4 py-10">
        <div className="flex flex-col items-start justify-between gap-6 md:flex-row">
          <div>
            <div className="text-base font-semibold text-white">NEXUS Cybersecurity Global</div>
            <p className="mt-1 max-w-md text-sm">{t("footer.tagline")}</p>
          </div>
          <nav className="flex flex-col gap-2 text-sm">
            <Link href="/solutions" className="hover:text-white">
              {t("nav.solutions")}
            </Link>
            <Link href="/request-demo" className="hover:text-white">
              {t("nav.demo")}
            </Link>
            <Link href="/function/tasks" className="hover:text-white">
              {t("nav.console")}
            </Link>
          </nav>
        </div>
        <p className="mt-8 max-w-2xl text-xs text-cyan-100/50">{t("footer.note")}</p>
        <p className="mt-2 text-xs text-cyan-100/50">
          © {year} NEXUS Cybersecurity Global. {t("footer.rights")}
        </p>
      </div>
    </footer>
  );
}
