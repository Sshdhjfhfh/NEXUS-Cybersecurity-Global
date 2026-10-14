import { t } from '@/lib/i18n';

// Centralised status → color/label semantics, reused across the whole app.
// Spec §8.3: Intención / portada / misión / Gravedad each have a consistent color set.

export type Tone = "neutral" | "blue" | "green" | "amber" | "red" | "rose" | "violet" | "slate";

export const toneClasses: Record<Tone, string> = {
  neutral: "bg-muted text-muted-foreground border-transparent",
  blue: "bg-blue-500/15 text-blue-600 dark:text-blue-400 border-blue-500/20",
  green: "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 border-emerald-500/20",
  amber: "bg-amber-500/15 text-amber-600 dark:text-amber-400 border-amber-500/20",
  red: "bg-red-500/15 text-red-600 dark:text-red-400 border-red-500/20",
  // rose Usado como ”serio” - énfasis alto y sólido,un contorno rojo suave que visualmente es obviamente más alto que ”alto riesgo”。
  rose: "bg-rose-600 text-white border-rose-600 dark:bg-rose-600 dark:text-white",
  violet: "bg-violet-500/15 text-violet-600 dark:text-violet-400 border-violet-500/20",
  slate: "bg-slate-500/15 text-slate-600 dark:text-slate-400 border-slate-500/20",
};

export const toneDot: Record<Tone, string> = {
  neutral: "bg-muted-foreground",
  blue: "bg-blue-500",
  green: "bg-emerald-500",
  amber: "bg-amber-500",
  red: "bg-red-500",
  rose: "bg-white",
  violet: "bg-violet-500",
  slate: "bg-slate-500",
};

interface StatusMeta {
  label: string;
  tone: Tone;
}

const intent: Record<string, StatusMeta> = {
  open: { label: t("ui.6e349ee9eb69d533"), tone: "slate" },
  running: { label: t("ui.99644756494482e8"), tone: "blue" },
  paused: { label: t("ui.0b5ba9cc57aedf6a"), tone: "amber" },
  done: { label: t("status.completed"), tone: "green" },
  // blocked = modelo/API/Se agotaron los reintentos de falla de la red y esta intención básicamente no se ha detectado (interceptación no objetivo)）。
  blocked: { label: t("ui.23d3da8d886b8aec"), tone: "red" },
  // exhausted = número de pasos alcanzado/El presupuesto de tiempo se cortó a mitad de camino y solo se reescribieron resultados parciales (se han explorado las no direcciones)）。
  exhausted: { label: t("ui.e349cf55735a6c21"), tone: "violet" },
  // stopped = Estado histórico de eliminación temporal (reservado,Datos históricos）。
  stopped: { label: t("status.stopped"), tone: "slate" },
  // deleted = El usuario eliminó falsamente la intención (conservando el nodo y la relación consanguínea, consulte el motivo de la eliminación) delete_reason campo）。
  deleted: { label: t("ui.8133b28f315f32c7"), tone: "slate" },
};

const task: Record<string, StatusMeta> = {
  created: { label: t("ui.1bba71a51144f309"), tone: "slate" },
  queued: { label: t("status.queued"), tone: "amber" },
  running: { label: t("status.running"), tone: "blue" },
  paused: { label: t("ui.0b5ba9cc57aedf6a"), tone: "amber" },
  done: { label: t("status.completed"), tone: "green" },
  failed: { label: t("status.failed"), tone: "red" },
  timeout: { label: "tiempo agotado", tone: "amber" },
};

const severity: Record<string, StatusMeta> = {
  critical: { label: t("finding.critical"), tone: "rose" },
  high: { label: t("finding.high"), tone: "red" },
  medium: { label: t("finding.medium"), tone: "amber" },
  low: { label: t("finding.low"), tone: "slate" },
};

const finding: Record<string, StatusMeta> = {
  pending: { label: t("status.pending"), tone: "amber" },
  in_progress: { label: t("ui.5e598cabd0dffa5f"), tone: "blue" },
  confirmed: { label: t("ui.710bb383bba5b65d"), tone: "red" },
  resolved: { label: t("ui.564b17b9ab24b30e"), tone: "green" },
  fixed: { label: t("finding.fixed"), tone: "green" },
  false_positive: { label: t("ui.ab940618809b3de2"), tone: "slate" },
  ignored: { label: t("ui.d4f0d2d2efa4c16d"), tone: "neutral" },
  duplicate: { label: t("ui.4b56d912aa76d05b"), tone: "neutral" },
  risk_accepted: { label: t("ui.ceb9ce0696ccceac"), tone: "violet" },
};

const engine: Record<string, StatusMeta> = {
  exploring: { label: t("ui.8d768e226cb414c2"), tone: "blue" },
  paused: { label: t("ui.0b5ba9cc57aedf6a"), tone: "amber" },
  stalled: { label: t("ui.e5e354b5652e672d"), tone: "red" },
  idle: { label: t("ui.99acacf317cc8ef9"), tone: "neutral" },
};

const goal: Record<string, StatusMeta> = {
  open: { label: t("status.in_progress"), tone: "blue" },
  met: { label: t("ui.065f40291feccd46"), tone: "green" },
  abandoned: { label: t("ui.88befda7da352042"), tone: "slate" },
};

const audit: Record<string, StatusMeta> = {
  allow: { label: t("ui.22acfabb18e435b6"), tone: "green" },
  block: { label: t("nav.intercept"), tone: "red" },
};

const node: Record<string, StatusMeta> = {
  observed: { label: t("ui.edffe72ec80220b7"), tone: "slate" },
  confirmed: { label: t("common.confirm"), tone: "green" },
  tombstoned: { label: t("ui.88befda7da352042"), tone: "neutral" },
};

// estado de entrega push。sending uso blue en lugar de amber：No es ”problemático”」，
// Pero ”ha sido recibido y está siendo emitido”, que es lo mismo que pending La semántica de espera de。
const delivery: Record<string, StatusMeta> = {
  pending: { label: t("ui.eeaca59a2410cd7d"), tone: "amber" },
  sending: { label: t("ui.68f1e141257c51e3"), tone: "blue" },
  sent: { label: t("ui.d15f39d60e24e2f4"), tone: "green" },
  failed: { label: t("status.failed"), tone: "red" },
  skipped: { label: t("ui.94200d3abd8d561a"), tone: "neutral" },
};

const maps = {
  intent,
  task,
  severity,
  finding,
  engine,
  goal,
  audit,
  node,
  delivery,
} as const;

export type StatusDomain = keyof typeof maps;

export function statusMeta(domain: StatusDomain, key: string): StatusMeta {
  return maps[domain][key] ?? { label: key, tone: "neutral" };
}
