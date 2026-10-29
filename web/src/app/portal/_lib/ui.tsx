"use client";

import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactElement, ReactNode } from "react";

import Link from "next/link";

import { LanguageSwitcher, usePortalI18n } from "./i18n";

// Soft, friendly client-portal design system. Light background, white cards with
// gentle shadows and rounded corners, calm cyan/sky accent — the welcoming look of
// a modern B2B SaaS portal. The NEXUS logo sits on a small dark brand strip so it
// stays legible on the light theme.

const authBg = "bg-gradient-to-br from-slate-50 via-[#eef4fb] to-[#e6eefb]";

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
    <div className={`grid min-h-screen place-items-center px-4 py-12 ${authBg}`}>
      <div className="w-full max-w-md">
        <div className="mb-3 flex justify-end">
          <LanguageSwitcher />
        </div>
        <div className="overflow-hidden rounded-2xl border border-slate-200/70 bg-white shadow-xl shadow-slate-300/30">
          <Link
            href="/"
            className="flex justify-center bg-gradient-to-r from-[#0b1b3a] to-[#0a2540] px-7 py-6"
            aria-label="NEXUS Cybersecurity Global"
          >
            {/* biome-ignore lint/performance/noImgElement: brand asset */}
            <img src="/brand/logo.webp" alt="NEXUS Cybersecurity Global" className="h-9 w-auto" />
          </Link>
          <div className="p-7">
            <h1 className="text-xl font-bold text-slate-900">{title}</h1>
            {subtitle ? <p className="mt-1 text-sm text-slate-500">{subtitle}</p> : null}
            <div className="mt-6">{children}</div>
          </div>
        </div>
        {footer ? <div className="mt-5 text-center text-sm text-slate-500">{footer}</div> : null}
      </div>
    </div>
  );
}

// PortalLoading is the full-page placeholder used while a portal page (or its
// Suspense boundary) resolves.
export function PortalLoading() {
  const { t } = usePortalI18n();
  return (
    <div role="status" className={`grid min-h-screen place-items-center text-sm text-slate-500 ${authBg}`}>
      {t.loading}
    </div>
  );
}

export function FormError({ children }: { children: ReactNode }) {
  return (
    <p role="alert" className="text-sm text-rose-600">
      {children}
    </p>
  );
}

export const authLink = "text-cyan-700 hover:text-cyan-800";

export function Field({ label, ...props }: { label: string } & InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label className="block text-xs font-medium text-slate-600">
      {label}
      <input
        {...props}
        className="mt-1.5 w-full rounded-xl border border-slate-200 bg-slate-50 px-3.5 py-2.5 text-sm text-slate-800 placeholder:text-slate-400 transition focus:border-cyan-400 focus:bg-white focus:outline-none focus:ring-4 focus:ring-cyan-100 disabled:bg-slate-100 disabled:text-slate-500"
      />
    </label>
  );
}

export function PrimaryButton({ children, ...props }: ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      {...props}
      className="w-full rounded-xl bg-gradient-to-r from-cyan-700 to-sky-700 px-4 py-2.5 text-sm font-semibold text-white shadow-sm shadow-cyan-500/20 transition hover:shadow-md hover:brightness-[1.05] focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-cyan-200 disabled:cursor-not-allowed disabled:opacity-60"
    >
      {children}
    </button>
  );
}
