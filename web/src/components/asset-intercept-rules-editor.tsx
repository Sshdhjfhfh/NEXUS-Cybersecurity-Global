"use client";

import { t } from '@/lib/i18n';

import * as React from "react";

import { PlusIcon, Trash2Icon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { NativeSelect, NativeSelectOption } from "@/components/ui/native-select";
import type { AssetInterceptKind, AssetInterceptRuleInput } from "@/lib/types";

// uso NativeSelect（nativo <select>）en lugar de shadcn Select：Este editor se utilizará en Sheet Dentro del cajón，
// shadcn Select desplegable portal llegar body、Al hacer clic afuera, el cajón ”Haga clic afuera para cerrar” se cerrará por error; El menú desplegable nativo no tiene este problema.。
export const ASSET_INTERCEPT_KIND_OPTIONS: {
  value: AssetInterceptKind;
  label: string;
  placeholder: string;
}[] = [
  { value: "exact_domain", label: t("ui.2513c003e3c7f3fa"), placeholder: "example.gov.cn" },
  { value: "exact_ip", label: t("ui.4dbab2bb62808cdf"), placeholder: "203.0.113.10" },
  { value: "exact_url", label: t("ui.9982875e2c716850"), placeholder: "https://example.com/login" },
  { value: "fuzzy_domain", label: t("ui.b7365e9f3d365213"), placeholder: ".gov.cn" },
  { value: "fuzzy_ip", label: t("ui.e10333ae69a7c1e6"), placeholder: "203.0.113." },
  { value: "fuzzy_url", label: t("ui.228de89e3cd0f885"), placeholder: "/admin" },
  { value: "cidr", label: "segmento de red CIDR", placeholder: "192.168.0.0/16" },
];

// AssetInterceptRulesEditor es ”Interceptar/Área de edición multilínea controlada de ”Permitir regla” (interceptaciónblock/Permitidoallow +
// tipo + contenido coincidente + Comentarios), no viene con persistencia: el componente principal decide cuándo enviar。
export function AssetInterceptRulesEditor({
  value,
  onChange,
}: {
  value: AssetInterceptRuleInput[];
  onChange: (v: AssetInterceptRuleInput[]) => void;
}) {
  function update(i: number, patch: Partial<AssetInterceptRuleInput>) {
    onChange(value.map((r, idx) => (idx === i ? { ...r, ...patch } : r)));
  }
  function remove(i: number) {
    onChange(value.filter((_, idx) => idx !== i));
  }
  function add() {
    onChange([...value, { action: "block", kind: "fuzzy_domain", pattern: "", note: "", enabled: true }]);
  }
  return (
    <div className="grid gap-2">
      {value.map((r, i) => {
        const ph = ASSET_INTERCEPT_KIND_OPTIONS.find((o) => o.value === r.kind)?.placeholder ?? "";
        return (
          // biome-ignore lint/suspicious/noArrayIndexKey: línea inestable id，solo control por índice
          <div key={i} className="flex items-center gap-2">
            <NativeSelect
              size="sm"
              className="w-[84px] shrink-0"
              value={r.action}
              onChange={(e) => update(i, { action: e.target.value as "block" | "allow" })}
            >
              <NativeSelectOption value="block">{t("nav.intercept")}</NativeSelectOption>
              <NativeSelectOption value="allow">{t("ui.f3fb44a969080b74")}</NativeSelectOption>
            </NativeSelect>
            <NativeSelect
              size="sm"
              className="w-[120px] shrink-0"
              value={r.kind}
              onChange={(e) => update(i, { kind: e.target.value as AssetInterceptKind })}
            >
              {ASSET_INTERCEPT_KIND_OPTIONS.map((o) => (
                <NativeSelectOption key={o.value} value={o.value}>
                  {o.label}
                </NativeSelectOption>
              ))}
            </NativeSelect>
            <Input
              className="flex-1"
              placeholder={ph}
              value={r.pattern}
              onChange={(e) => update(i, { pattern: e.target.value })}
            />
            <Input
              className="w-[120px] shrink-0"
              placeholder={t("ui.ecd7c2ad14b6487f")}
              value={r.note}
              onChange={(e) => update(i, { note: e.target.value })}
            />
            <Button
              type="button"
              size="icon"
              variant="ghost"
              className="text-destructive hover:text-destructive size-8 shrink-0"
              onClick={() => remove(i)}
            >
              <Trash2Icon className="size-4" />
            </Button>
          </div>
        );
      })}
      <Button type="button" size="sm" variant="outline" className="w-fit" onClick={add}>
        <PlusIcon className="size-4" />  {t("ui.7c595962a1ee20b4")} </Button>
    </div>
  );
}
