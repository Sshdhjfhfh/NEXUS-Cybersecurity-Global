// Shared, locale-aware date/time formatting. Fixes two review findings:
//  1. Dates were hardcoded to the Chinese locale (`toLocaleString("zh-CN")` /
//     `Intl.DateTimeFormat("zh-CN", …)` across ~40 files), so evidence and report
//     timestamps rendered in zh-CN regardless of the UI language.
//  2. A hand-built relative-time template rendered broken strings like "5hace s".
//
// Every screen should format dates through here. Do NOT hardcode "zh-CN" or build
// relative-time strings by hand — use Intl via these helpers, with the user's
// locale resolved from <html lang> (set by the app's i18n).

type DateInput = Date | string | number;

function toDate(v: DateInput): Date {
  return v instanceof Date ? v : new Date(v);
}

/** The UI locale: <html lang> first, then the browser, then Spanish. */
export function uiLocale(): string {
  if (typeof document !== "undefined") {
    const l = document.documentElement.lang?.trim();
    if (l) return l;
  }
  if (typeof navigator !== "undefined" && navigator.language) return navigator.language;
  return "es";
}

/** e.g. "17 nov 2026, 14:30" in the user's locale. "—" for an invalid date. */
export function formatDateTime(v: DateInput, locale = uiLocale()): string {
  const d = toDate(v);
  if (Number.isNaN(d.getTime())) return "—";
  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(d);
}

/** Date only, e.g. "17 nov 2026". */
export function formatDate(v: DateInput, locale = uiLocale()): string {
  const d = toDate(v);
  if (Number.isNaN(d.getTime())) return "—";
  return new Intl.DateTimeFormat(locale, { year: "numeric", month: "short", day: "2-digit" }).format(d);
}

const DIVISIONS: [Intl.RelativeTimeFormatUnit, number][] = [
  ["year", 60 * 60 * 24 * 365],
  ["month", 60 * 60 * 24 * 30],
  ["day", 60 * 60 * 24],
  ["hour", 60 * 60],
  ["minute", 60],
  ["second", 1],
];

/**
 * A correct, localized relative time ("hace 5 minutos", "in 2 days"), via
 * Intl.RelativeTimeFormat — never a hand-built "Nhace s" string.
 */
export function relativeTime(v: DateInput, locale = uiLocale()): string {
  const d = toDate(v);
  if (Number.isNaN(d.getTime())) return "—";
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
  const diffSec = (d.getTime() - Date.now()) / 1000;
  const abs = Math.abs(diffSec);
  for (const [unit, secs] of DIVISIONS) {
    if (abs >= secs || unit === "second") {
      return rtf.format(Math.round(diffSec / secs), unit);
    }
  }
  return rtf.format(0, "second");
}
