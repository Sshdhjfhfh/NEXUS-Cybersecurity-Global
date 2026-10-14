import { t } from '@/lib/i18n';

import type { NewAssetType } from "@/lib/types";

const ASSET_TYPE_LABELS: Record<NewAssetType, string> = {
  app: t("ui.766d27f411cd76f2"),
  endpoint: "interfaz",
  ip: "IP",
  root_domain: "nombre de dominio raíz",
  service: t("ui.1efa3b455a06245e"),
  subdomain: "nombre de subdominio",
};

const TASK_ASSET_SOURCE_LABELS: Record<string, string> = {
  agent: t("ui.4dd244b9c2bc0799"),
  anchor: t("ui.f79c64b5c57de773"),
  api: t("ui.5c46fa81ee62ecac"),
  company: t("ui.e8fc49517d93e6e4"),
  legacy: t("ui.68c8309de2349c90"),
  manual: t("ui.b8033bc9dcde8238"),
  system: t("ui.b621219079ae23e4"),
  task: t("ui.bfb2b7164bb0080a"),
};

export function taskAssetTypeLabel(type: NewAssetType): string {
  return ASSET_TYPE_LABELS[type];
}

export function taskAssetSourceLabel(source: string): string {
  return TASK_ASSET_SOURCE_LABELS[source] ?? source;
}
