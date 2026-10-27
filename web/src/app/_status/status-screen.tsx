"use client";

import { type ReactNode, useEffect, useState } from "react";

import Link from "next/link";

type Lang = "es" | "en";
type Copy = { title: string; message: string; cta?: string };

// Reads the visitor's chosen site language (falls back to the browser language,
// then Spanish). Safe before hydration and offline.
export function useStatusLang(): Lang {
  const [lang, setLang] = useState<Lang>("es");
  useEffect(() => {
    try {
      const saved = localStorage.getItem("nexus_site_lang");
      if (saved === "en" || saved === "es") {
        setLang(saved);
        return;
      }
    } catch {
      /* ignore */
    }
    if (typeof navigator !== "undefined" && navigator.language?.toLowerCase().startsWith("en")) {
      setLang("en");
    }
  }, []);
  return lang;
}

function LogoMark({ spinner, tone }: { spinner?: boolean; tone: string }) {
  return (
    <div className="relative grid size-24 place-items-center">
      {spinner ? (
        <span
          className={`absolute inset-0 animate-spin rounded-full border-2 border-transparent ${tone}`}
          style={{ borderTopColor: "currentColor", borderRightColor: "currentColor" }}
        />
      ) : (
        <span className="absolute inset-0 rounded-full bg-[radial-gradient(circle,rgba(34,211,238,0.22),transparent_70%)]" />
      )}
      {/* biome-ignore lint/performance/noImgElement: brand asset on a standalone status screen */}
      <img src="/brand/mark.webp" alt="NEXUS" className="size-16 animate-pulse" />
    </div>
  );
}

// A full-screen branded status screen used for offline, errors, 404 and
// maintenance — so the user never sees a blank page.
export function StatusScreen({
  es,
  en,
  code,
  spinner,
  tone = "text-cyan-400",
  onAction,
  href,
  children,
}: {
  es: Copy;
  en: Copy;
  code?: string;
  spinner?: boolean;
  tone?: string;
  onAction?: () => void;
  href?: string;
  children?: ReactNode;
}) {
  const lang = useStatusLang();
  const c = lang === "en" ? en : es;
  return (
    <div className="grid min-h-screen place-items-center bg-[#060b18] px-6 text-center text-white">
      <div className="flex max-w-md flex-col items-center">
        <span className={tone}>
          <LogoMark spinner={spinner} tone={tone} />
        </span>
        {code ? (
          <span className="mt-6 rounded-full border border-white/15 px-3 py-0.5 text-xs font-semibold tracking-wider text-slate-300">
            {code}
          </span>
        ) : null}
        <h1 className="mt-5 text-2xl font-bold tracking-tight">{c.title}</h1>
        <p className="mt-3 text-sm leading-relaxed text-slate-400">{c.message}</p>
        {children}
        {c.cta ? (
          onAction ? (
            <button
              type="button"
              onClick={onAction}
              className="mt-7 rounded-lg bg-gradient-to-r from-cyan-400 to-cyan-500 px-6 py-2.5 text-sm font-semibold text-[#060b18] transition-all hover:shadow-[0_0_28px_rgba(34,211,238,0.5)]"
            >
              {c.cta}
            </button>
          ) : (
            <Link
              href={href ?? "/"}
              className="mt-7 rounded-lg bg-gradient-to-r from-cyan-400 to-cyan-500 px-6 py-2.5 text-sm font-semibold text-[#060b18] transition-all hover:shadow-[0_0_28px_rgba(34,211,238,0.5)]"
            >
              {c.cta}
            </Link>
          )
        ) : null}
      </div>
    </div>
  );
}
