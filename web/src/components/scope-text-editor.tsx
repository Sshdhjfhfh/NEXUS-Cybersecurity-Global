"use client";

import { t } from '@/lib/i18n';

import * as React from "react";

import { Badge } from "@/components/ui/badge";
import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import { Textarea } from "@/components/ui/textarea";
import type { ParsedCompanyScopeText } from "@/lib/company-scope";
import type { CompanyScopeKind } from "@/lib/types";

const SCOPE_KIND_LABELS: Record<CompanyScopeKind, string> = {
  domain: "nombre de dominio",
  ip: "IP",
  cidr: "CIDR",
  icp: "ICP",
  keyword: "Palabras clave",
};

export function ScopeTextEditor({
  id,
  value,
  onValueChange,
  parsed,
  label = t("ui.3f6e21ff1201fe11"),
  description = t("ui.a20aaaf32226cd6b"),
}: {
  id: string;
  value: string;
  onValueChange: (value: string) => void;
  parsed: ParsedCompanyScopeText;
  label?: string;
  description?: string;
}) {
  const counts = React.useMemo(() => {
    const result = new Map<CompanyScopeKind, number>();
    for (const rule of parsed.rules) result.set(rule.kind, (result.get(rule.kind) ?? 0) + 1);
    return result;
  }, [parsed.rules]);

  return (
    <Field data-invalid={parsed.errors.length > 0}>
      <FieldLabel htmlFor={id}>{label}</FieldLabel>
      <FieldDescription>{description}</FieldDescription>
      <Textarea
        id={id}
        rows={8}
        value={value}
        aria-invalid={parsed.errors.length > 0}
        placeholder={t("ui.316914332ff203d0")}
        className="min-h-36 resize-y font-mono text-sm"
        onChange={(event) => onValueChange(event.target.value)}
      />
      {parsed.rules.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5 text-muted-foreground text-xs">
          <span>{t("ui.bd34fe6fb3115d09")} {parsed.rules.length}  {t("ui.a1cb6bd814e4e98b")}</span>
          {Object.entries(SCOPE_KIND_LABELS).map(([kind, kindLabel]) => {
            const count = counts.get(kind as CompanyScopeKind) ?? 0;
            return count > 0 ? (
              <Badge key={kind} variant="secondary" className="font-mono tabular-nums">
                {kindLabel} {count}
              </Badge>
            ) : null;
          })}
        </div>
      )}
      {parsed.errors.length > 0 && (
        <FieldError>
          {parsed.errors.slice(0, 5).map((item) => (
            <span key={`${item.line}-${item.error}`} className="block">
               {"No."} {item.line}  {"OK:"}{item.error}
            </span>
          ))}
          {parsed.errors.length > 5 && <span className="block">{t("ui.20af4511c75c6d7b")} {parsed.errors.length - 5}  {t("ui.1cbf73191deeb56d")}</span>}
        </FieldError>
      )}
    </Field>
  );
}
