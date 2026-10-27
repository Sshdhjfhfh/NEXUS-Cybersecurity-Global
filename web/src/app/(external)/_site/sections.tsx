"use client";

import type { ReactElement, ReactNode } from "react";

import Link from "next/link";

import { ArrowRightIcon } from "lucide-react";

import { Reveal } from "./motion";

// Shared building blocks for the public marketing subpages, so every page keeps
// the same NEXUS rhythm, spacing and motion.

export function PageHeader({
  kicker,
  title,
  body,
  img,
}: {
  kicker: string;
  title: string;
  body?: string;
  img?: string;
}) {
  return (
    <section className="relative overflow-hidden border-b border-white/5">
      {img ? (
        <>
          {/* biome-ignore lint/performance/noImgElement: static marketing asset */}
          <img
            src={img}
            alt=""
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 size-full object-cover opacity-25"
          />
          <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,#060b18_8%,rgba(6,11,24,0.7)_60%,rgba(6,11,24,0.85)_100%)]" />
        </>
      ) : (
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(80%_60%_at_20%_0%,rgba(34,211,238,0.1),transparent_60%)]" />
      )}
      <div className="relative mx-auto max-w-[1400px] px-6 py-20 lg:py-24">
        <Reveal className="max-w-3xl">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{kicker}</p>
          <h1 className="mt-3 text-4xl font-bold leading-tight tracking-tight text-white sm:text-5xl">{title}</h1>
          {body ? <p className="mt-5 max-w-2xl text-base leading-relaxed text-slate-300">{body}</p> : null}
        </Reveal>
      </div>
    </section>
  );
}

export function Section({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <section className={`mx-auto max-w-[1400px] px-6 py-16 ${className}`}>{children}</section>;
}

export function SectionHeading({ kicker, title, body }: { kicker?: string; title: string; body?: string }) {
  return (
    <Reveal className="max-w-2xl">
      {kicker ? (
        <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{kicker}</p>
      ) : null}
      <h2 className="mt-2 text-3xl font-bold tracking-tight text-white sm:text-4xl">{title}</h2>
      {body ? <p className="mt-4 text-sm leading-relaxed text-slate-400">{body}</p> : null}
    </Reveal>
  );
}

export function InfoCard({
  icon,
  title,
  body,
  badge,
}: {
  icon?: ReactElement;
  title: string;
  body: string;
  badge?: string;
}) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-6 transition-colors hover:border-cyan-400/40">
      <div className="flex items-center justify-between gap-2">
        {icon ? <span className="text-cyan-300">{icon}</span> : null}
        {badge ? (
          <span className="rounded-full border border-violet-400/40 bg-violet-400/10 px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-violet-200">
            {badge}
          </span>
        ) : null}
      </div>
      <h3 className="mt-4 text-lg font-semibold text-white">{title}</h3>
      <p className="mt-2 text-sm leading-relaxed text-slate-400">{body}</p>
    </div>
  );
}

export function Band({
  kicker,
  title,
  body,
  img,
  reverse,
  children,
}: {
  kicker?: string;
  title: string;
  body?: string;
  img: string;
  reverse?: boolean;
  children?: ReactElement;
}) {
  return (
    <section className="border-t border-white/5">
      <div className="mx-auto grid max-w-[1400px] items-center gap-10 px-6 py-16 lg:grid-cols-2">
        <Reveal className={reverse ? "lg:order-2" : ""}>
          {kicker ? (
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-300/80">{kicker}</p>
          ) : null}
          <h2 className="mt-2 text-2xl font-bold tracking-tight text-white sm:text-3xl">{title}</h2>
          {body ? <p className="mt-4 max-w-md text-sm leading-relaxed text-slate-400">{body}</p> : null}
          {children ? <div className="mt-5">{children}</div> : null}
        </Reveal>
        <Reveal delay={120} className={reverse ? "lg:order-1" : ""}>
          {/* biome-ignore lint/performance/noImgElement: static marketing asset */}
          <img
            src={img}
            alt=""
            aria-hidden="true"
            className="aspect-video w-full rounded-2xl border border-white/10 object-cover shadow-2xl"
          />
        </Reveal>
      </div>
    </section>
  );
}

export function CtaBand({ title, body, cta, href }: { title: string; body: string; cta: string; href: string }) {
  return (
    <section className="border-t border-white/5 bg-[#070d1c]">
      <div className="mx-auto max-w-[1400px] px-6 py-16 text-center">
        <Reveal>
          <h2 className="text-2xl font-bold tracking-tight text-white sm:text-3xl">{title}</h2>
          <p className="mx-auto mt-3 max-w-xl text-sm text-slate-400">{body}</p>
          <Link
            href={href}
            className="mt-7 inline-flex items-center gap-2 rounded-lg bg-gradient-to-r from-cyan-400 to-cyan-500 px-6 py-3 text-sm font-semibold text-[#060b18] shadow-[0_0_24px_rgba(34,211,238,0.4)] transition-all hover:shadow-[0_0_36px_rgba(34,211,238,0.6)]"
          >
            {cta} <ArrowRightIcon className="size-4" />
          </Link>
        </Reveal>
      </div>
    </section>
  );
}
