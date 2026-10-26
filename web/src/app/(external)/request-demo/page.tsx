"use client";

import { type FormEvent, useState } from "react";

import { useSiteI18n } from "../_site/i18n";

const SERVICES = ["offensive", "defensive", "purple", "ai-research", "managed", "other"] as const;

type Status = { kind: "idle" } | { kind: "sending" } | { kind: "ok" } | { kind: "err"; msg: string };

const inputCls =
  "mt-1 w-full rounded-lg border border-white/15 bg-white/[0.04] px-3 py-2 text-sm text-white placeholder:text-cyan-100/30 focus:border-cyan-400/60 focus:outline-none";

export default function RequestDemo() {
  const { t } = useSiteI18n();
  const [status, setStatus] = useState<Status>({ kind: "idle" });

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const get = (k: string) => String(fd.get(k) ?? "").trim();
    const body = {
      organization: get("organization"),
      contact_name: get("contact_name"),
      email: get("email"),
      country: get("country"),
      service: get("service"),
      message: get("message"),
      consent: fd.get("consent") === "on",
      website: get("website"), // honeypot: real users leave it empty
    };
    if (!body.organization || !body.contact_name || !body.email) {
      setStatus({ kind: "err", msg: t("form.required") });
      return;
    }
    setStatus({ kind: "sending" });
    try {
      const res = await fetch("/api/public/leads", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        setStatus({ kind: "err", msg: data?.error ? `${t("form.err")} (${data.error})` : t("form.err") });
        return;
      }
      setStatus({ kind: "ok" });
    } catch {
      setStatus({ kind: "err", msg: t("form.err") });
    }
  }

  if (status.kind === "ok") {
    return (
      <section className="mx-auto max-w-xl px-4 py-24 text-center">
        <div className="mx-auto grid size-12 place-items-center rounded-full bg-cyan-400/15 text-2xl text-cyan-300">
          ✓
        </div>
        <h1 className="mt-6 text-2xl font-bold text-white">{t("form.ok")}</h1>
      </section>
    );
  }

  return (
    <section className="mx-auto max-w-2xl px-4 py-20">
      <h1 className="text-3xl font-bold tracking-tight text-white sm:text-4xl">{t("form.title")}</h1>
      <p className="mt-3 text-sm text-cyan-100/75">{t("form.subtitle")}</p>

      <form onSubmit={onSubmit} className="mt-10 grid gap-5" noValidate>
        <div className="grid gap-5 sm:grid-cols-2">
          <label className="text-sm text-cyan-100/80">
            {t("form.org")} *
            <input name="organization" required maxLength={200} className={inputCls} />
          </label>
          <label className="text-sm text-cyan-100/80">
            {t("form.name")} *
            <input name="contact_name" required maxLength={200} className={inputCls} />
          </label>
          <label className="text-sm text-cyan-100/80">
            {t("form.email")} *
            <input name="email" type="email" required maxLength={200} className={inputCls} />
          </label>
          <label className="text-sm text-cyan-100/80">
            {t("form.country")}
            <input name="country" maxLength={200} className={inputCls} />
          </label>
        </div>
        <label className="text-sm text-cyan-100/80">
          {t("form.service")}
          <select name="service" defaultValue="" className={inputCls}>
            <option value="" className="bg-[#061024]">
              {t("svc.choose")}
            </option>
            {SERVICES.map((s) => (
              <option key={s} value={s} className="bg-[#061024]">
                {t(`svc.${s}`)}
              </option>
            ))}
          </select>
        </label>
        <label className="text-sm text-cyan-100/80">
          {t("form.message")}
          <textarea name="message" rows={5} maxLength={4000} className={inputCls} />
        </label>
        {/* Honeypot: hidden from humans, bots tend to fill it. */}
        <input
          name="website"
          tabIndex={-1}
          autoComplete="off"
          aria-hidden="true"
          className="absolute left-[-9999px] h-0 w-0 opacity-0"
        />
        <label className="flex items-start gap-3 text-sm text-cyan-100/80">
          <input name="consent" type="checkbox" required className="mt-1 size-4 accent-cyan-400" />
          <span>{t("form.consent")}</span>
        </label>
        {status.kind === "err" ? <p className="text-sm text-rose-300">{status.msg}</p> : null}
        <button
          type="submit"
          disabled={status.kind === "sending"}
          className="justify-self-start rounded-full bg-cyan-400 px-6 py-3 text-sm font-semibold text-[#061024] transition-colors hover:bg-cyan-300 disabled:opacity-60"
        >
          {status.kind === "sending" ? t("form.sending") : t("form.submit")}
        </button>
      </form>
    </section>
  );
}
