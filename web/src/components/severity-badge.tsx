import { cn } from "@/lib/utils";

// SeverityBadge is the single, shared severity signal for the whole product —
// the same colors and shape in the client Portal and the operator Console, so a
// "High" finding never looks like two different things. Severity is encoded by
// color AND a letter + label (not color alone), so it survives color-blindness
// and grayscale printing. Colors come from the --sev-* design tokens.

export type Severity = "critical" | "high" | "medium" | "low" | "info";

const META: Record<Severity, { label: string; letter: string; cls: string }> = {
  critical: { label: "Crítica", letter: "C", cls: "bg-sev-critical text-sev-critical-foreground" },
  high: { label: "Alta", letter: "A", cls: "bg-sev-high text-sev-high-foreground" },
  medium: { label: "Media", letter: "M", cls: "bg-sev-medium text-sev-medium-foreground" },
  low: { label: "Baja", letter: "B", cls: "bg-sev-low text-sev-low-foreground" },
  info: { label: "Info", letter: "I", cls: "bg-sev-info text-sev-info-foreground" },
};

const ORDER: Severity[] = ["critical", "high", "medium", "low", "info"];

// normalizeSeverity maps the backend's severity strings (en or es, any case) to
// the canonical ramp; anything unknown is treated as "info".
export function normalizeSeverity(v?: string | null): Severity {
  const s = (v ?? "").trim().toLowerCase();
  if (s.startsWith("crit") || s === "crítica" || s === "critica") return "critical";
  if (s.startsWith("high") || s === "alta") return "high";
  if (s.startsWith("med") || s === "media") return "medium";
  if (s.startsWith("low") || s === "baja") return "low";
  return "info";
}

export function SeverityBadge({
  severity,
  label,
  className,
}: {
  severity: string | Severity;
  /** Localized label override; defaults to the built-in Spanish label. */
  label?: string;
  className?: string;
}) {
  const sev = ORDER.includes(severity as Severity)
    ? (severity as Severity)
    : normalizeSeverity(String(severity));
  const m = META[sev];
  const text = label ?? m.label;
  return (
    <span
      role="status"
      aria-label={`Severidad: ${text}`}
      title={text}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md px-2 py-0.5 text-xs font-semibold tracking-wide",
        m.cls,
        className,
      )}
    >
      <span
        aria-hidden
        className="grid h-4 w-4 place-items-center rounded-sm bg-white/25 text-[10px] font-bold leading-none"
      >
        {m.letter}
      </span>
      {text}
    </span>
  );
}
