"use client";

// Floating glass dashboards overlaid on the hero's operations-center photo. The
// numbers are illustrative sample UI (a product mockup), not live operational data.

function Stat({ value, label, delta, up }: { value: string; label: string; delta: string; up: boolean }) {
  return (
    <div>
      <div className="text-[10px] uppercase tracking-wide text-slate-400">{label}</div>
      <div className="mt-0.5 flex items-baseline gap-1.5">
        <span className="text-lg font-bold text-white">{value}</span>
        <span className={`text-[10px] font-semibold ${up ? "text-emerald-400" : "text-rose-400"}`}>
          {up ? "▲" : "▼"} {delta}
        </span>
      </div>
    </div>
  );
}

export function AnalysisPanel() {
  const rows = [
    { label: "Scanning 12.4M events/s", on: true },
    { label: "Analyzing indicators", on: true },
    { label: "Correlating threats", on: true },
    { label: "Generating insights", on: false },
  ];
  return (
    <div className="w-56 rounded-xl border border-cyan-400/20 bg-[#0a1324]/85 p-4 shadow-2xl backdrop-blur-md">
      <div className="mb-3 text-[11px] font-semibold uppercase tracking-wider text-cyan-300">AI Threat Analysis</div>
      <ul className="space-y-2.5">
        {rows.map((r) => (
          <li key={r.label} className="flex items-center gap-2.5 text-[12px] text-slate-200">
            <span
              className={`size-1.5 rounded-full ${r.on ? "bg-cyan-400 shadow-[0_0_8px_#22d3ee]" : "bg-slate-600"} ${
                r.on ? "animate-pulse" : ""
              }`}
            />
            {r.label}
          </li>
        ))}
      </ul>
    </div>
  );
}

export function OverviewPanel() {
  const bars = [40, 65, 48, 80, 58, 72, 90, 62, 76, 52, 68, 84];
  return (
    <div className="w-72 rounded-xl border border-cyan-400/20 bg-[#0a1324]/85 p-4 shadow-2xl backdrop-blur-md">
      <div className="mb-3 text-[11px] font-semibold uppercase tracking-wider text-cyan-300">
        Global Threat Overview
      </div>
      <div className="grid grid-cols-3 gap-3">
        <Stat value="1,328" label="Threats Detected" delta="12%" up />
        <Stat value="23" label="Active Incidents" delta="35%" up={false} />
        <Stat value="1,305" label="Contained Today" delta="48%" up />
      </div>
      <div className="mt-4 flex items-center gap-4">
        <div
          className="size-16 shrink-0 rounded-full"
          style={{
            background:
              "conic-gradient(#f43f5e 0 32%, #a855f7 32% 56%, #f59e0b 56% 74%, #22d3ee 74% 88%, #38bdf8 88% 96%, #64748b 96% 100%)",
          }}
        >
          <div className="grid size-full place-items-center">
            <div className="size-9 rounded-full bg-[#0a1324]" />
          </div>
        </div>
        <ul className="grid grid-cols-2 gap-x-3 gap-y-1 text-[10px] text-slate-300">
          <li className="flex items-center gap-1.5">
            <span className="size-1.5 rounded-sm bg-rose-500" />
            Malware 32%
          </li>
          <li className="flex items-center gap-1.5">
            <span className="size-1.5 rounded-sm bg-purple-500" />
            Phishing 24%
          </li>
          <li className="flex items-center gap-1.5">
            <span className="size-1.5 rounded-sm bg-amber-500" />
            Ransom. 18%
          </li>
          <li className="flex items-center gap-1.5">
            <span className="size-1.5 rounded-sm bg-cyan-400" />
            Zero-Day 14%
          </li>
          <li className="flex items-center gap-1.5">
            <span className="size-1.5 rounded-sm bg-sky-400" />
            DDoS 8%
          </li>
          <li className="flex items-center gap-1.5">
            <span className="size-1.5 rounded-sm bg-slate-500" />
            Other 4%
          </li>
        </ul>
      </div>
      <div className="mt-4">
        <div className="mb-1.5 text-[10px] uppercase tracking-wide text-slate-400">Live Global Activity</div>
        <div className="flex h-10 items-end gap-1">
          {bars.map((h, i) => (
            <div
              key={`${h}-${i}`}
              className="flex-1 rounded-sm bg-gradient-to-t from-cyan-500/40 to-cyan-400"
              style={{ height: `${h}%` }}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

// HeroVisual floats the two dashboards over the hero photo (which the section
// paints as a full-bleed background).
export function HeroVisual() {
  return (
    <div className="relative hidden h-full min-h-[420px] w-full lg:block">
      <div className="absolute right-0 top-0">
        <OverviewPanel />
      </div>
      <div className="absolute bottom-2 left-0 hidden xl:block">
        <AnalysisPanel />
      </div>
    </div>
  );
}
