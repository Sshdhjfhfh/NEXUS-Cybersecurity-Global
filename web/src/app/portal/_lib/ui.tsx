"use client";

import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactElement, ReactNode } from "react";

import Link from "next/link";

export function PortalAuthShell({
  title,
  subtitle,
  children,
  footer,
}: {
  title: string;
  subtitle?: string;
  children: ReactNode;
  footer?: ReactElement;
}) {
  return (
    <div className="grid min-h-screen place-items-center bg-[#060b18] px-4 py-12 text-white">
      <div className="w-full max-w-md">
        <Link href="/" className="mb-8 flex justify-center">
          {/* biome-ignore lint/performance/noImgElement: brand asset */}
          <img src="/brand/logo.webp" alt="NEXUS Cybersecurity Global" className="h-10 w-auto" />
        </Link>
        <div className="rounded-2xl border border-white/10 bg-white/[0.03] p-7">
          <h1 className="text-xl font-bold">{title}</h1>
          {subtitle ? <p className="mt-1 text-sm text-slate-400">{subtitle}</p> : null}
          <div className="mt-6">{children}</div>
        </div>
        {footer ? <div className="mt-5 text-center text-sm text-slate-400">{footer}</div> : null}
      </div>
    </div>
  );
}

export function Field({ label, ...props }: { label: string } & InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label className="block text-xs font-medium text-slate-400">
      {label}
      <input
        {...props}
        className="mt-1.5 w-full rounded-lg border border-white/15 bg-[#0a1324] px-3 py-2.5 text-sm text-white placeholder:text-slate-600 focus:border-cyan-400/60 focus:outline-none"
      />
    </label>
  );
}

export function PrimaryButton({ children, ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      {...props}
      className="w-full rounded-lg bg-gradient-to-r from-cyan-400 to-cyan-500 px-4 py-2.5 text-sm font-semibold text-[#060b18] transition-all hover:shadow-[0_0_24px_rgba(34,211,238,0.5)] disabled:opacity-60"
    >
      {children}
    </button>
  );
}
