"use client";

import { t } from '@/lib/i18n';

import { CheckIcon } from "lucide-react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import type { NotificationFilter } from "@/lib/types";

// asText / inputType Es un valor auxiliar en este archivo (fuertemente relacionado con la representación de control) y no se coloca en channel-fields。
import { type FieldDef, type FieldKind, SEVERITY_OPTIONS } from "./channel-fields";

// asText Representa cualquier valor de configuración en una cadena disponible en el cuadro de entrada。
// config desde JSON，El valor puede ser string / number / boolean / array / null，
// La única preocupación aquí es ”si se puede incluir en el cuadro de texto”. La serialización específica se realiza mediante buildConfig Responsable。
function asText(v: unknown): string {
  if (typeof v === "string") return v;
  if (v === null || v === undefined) return "";
  return String(v);
}

// inputType Asignar tipos de campos a input de type Propiedades。
function inputType(kind: FieldKind): "text" | "password" | "number" {
  if (kind === "password") return "password";
  if (kind === "number") return "number";
  return "text";
}

// ConfigField Renderiza el control correspondiente según la definición del campo。
//
// El procesamiento de campos de máscara es lo único aquí: cuadro de entrada**No mostrar**El valor de la máscara en sí, solo se muestra una línea
// 「Mensaje guardado”. De esta manera, solo hay una regla en la interfaz: las palabras en el cuadro las completa el usuario.，
// Un cuadro vacío es un valor vacío. si pones "__masked__:…abc123" póngalo en el cuadro de entrada, el usuario pensará que es para él mismo
// El texto del marcador de posición eliminado hace que sea más fácil borrar las credenciales.。
export function ConfigField({
  def,
  value,
  isSecret,
  onChange,
}: {
  def: FieldDef;
  value: unknown;
  isSecret: boolean;
  onChange: (v: unknown) => void;
}) {
  const id = `n-cfg-${def.key}`;
  const raw = asText(value);
  // El valor de la máscara reflejado por el backend: en el formulario "__masked__:…abc123"，La cola es un fragmento identificable del valor original.。
  const masked = isSecret && raw.startsWith("__masked__");
  const maskedTail = masked ? (raw.split("…")[1] ?? "") : "";

  if (def.kind === "switch") {
    return (
      <div className="flex items-center gap-2 text-sm">
        <Switch checked={value === true} onCheckedChange={onChange} aria-label={def.label} />
        {def.label}
        {def.help && <span className="text-muted-foreground">（{def.help}）</span>}
      </div>
    );
  }

  if (def.kind === "select") {
    return (
      <div className="grid gap-2">
        <Label>{def.label}</Label>
        <Select value={raw || def.options?.[0]?.value} onValueChange={onChange}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {(def.options ?? []).map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    );
  }

  // Los controles se envían por tipo de campo. usar if Cadena en lugar de ternario anidado porque aquí se deben distinguir cuatro tipos de controles.，
  // Al leer el ternario de tres niveles, tengo que detenerme y contar los paréntesis.。
  function control() {
    if (def.kind === "textarea" || def.kind === "kv") {
      return (
        <Textarea
          id={id}
          className="font-mono"
          placeholder={def.placeholder}
          value={masked ? "" : raw}
          onChange={(e) => onChange(e.target.value)}
        />
      );
    }
    if (def.kind === "list") {
      return (
        <Input
          id={id}
          value={Array.isArray(value) ? (value as string[]).join(", ") : raw}
          onChange={(e) => onChange(e.target.value)}
          placeholder={def.placeholder}
        />
      );
    }
    return (
      <Input
        id={id}
        className={def.kind === "text" ? "font-mono" : ""}
        type={inputType(def.kind)}
        placeholder={def.placeholder}
        value={masked ? "" : raw}
        onChange={(e) => onChange(e.target.value)}
      />
    );
  }

  const hint = masked ? (
    <p className="text-muted-foreground flex items-center gap-1 text-xs">
      <CheckIcon className="size-3" />
       {t("ui.3a660ff5fb7c1989")}{maskedTail ? `(Número de cola ${maskedTail}）` : ""}  {t("ui.0b09307a688af358")} </p>
  ) : (
    def.help && <p className="text-muted-foreground text-xs">{def.help}</p>
  );

  return (
    <div className="grid gap-2">
      <Label htmlFor={id}>{def.label}</Label>
      {control()}
      {hint}
    </div>
  );
}

// FilterSummary Resuma las condiciones del filtro en una línea para que pueda ver lo que promueve este canal sin expandir la tarjeta.。
export function FilterSummary({ filter }: { filter: NotificationFilter }) {
  const parts: string[] = [];
  if (filter.min_severity) {
    parts.push(SEVERITY_OPTIONS.find((o) => o.value === filter.min_severity)?.label ?? filter.min_severity);
  }
  if (filter.vulnclass_include?.length) parts.push(`El tipo contiene ${filter.vulnclass_include.length}palabra`);
  if (filter.vulnclass_exclude?.length) parts.push(`excluir ${filter.vulnclass_exclude.length}palabra`);
  if (filter.task_ids?.length) parts.push(`${filter.task_ids.length}tareas`);
  if (filter.asset_ids?.length) parts.push(`${filter.asset_ids.length}activos`);
  if (filter.on_status_change) parts.push(t("ui.6016247aeeccc1ba"));
  if (parts.length === 0) {
    return <p className="text-muted-foreground text-sm">{t("ui.6ecbb036cda5ce07")}</p>;
  }
  return <p className="text-muted-foreground text-sm">{parts.join(" · ")}</p>;
}
