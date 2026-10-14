"use client";

import { t } from '@/lib/i18n';

import * as React from "react";

import { CheckIcon, CopyIcon } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { cn, copyText } from "@/lib/utils";

type CopyButtonProps = {
  // Texto a copiar;Si está vacío, el botón está deshabilitado。
  text: string | null | undefined;
  // Después de una copia exitosa toast redacción publicitaria,”Copiado” predeterminado」。
  successMessage?: string;
  label?: React.ReactNode;
  size?: React.ComponentProps<typeof Button>["size"];
  variant?: React.ComponentProps<typeof Button>["variant"];
  className?: string;
};

// CopyButton Botón unificado ”Copiar al portapapeles”:Incorporación exitosa/retroalimentación de falla,y en HTTP contexto no relacionado con la seguridad
// Bajar de categoría automáticamente(Ver copyText)。
export function CopyButton({
  text,
  successMessage = t("ui.d8dd878fb14359e9"),
  label = "copia",
  size = "sm",
  variant = "outline",
  className,
}: CopyButtonProps) {
  const [copied, setCopied] = React.useState(false);
  const timer = React.useRef<ReturnType<typeof setTimeout> | null>(null);

  React.useEffect(() => {
    return () => {
      if (timer.current) clearTimeout(timer.current);
    };
  }, []);

  async function handleCopy() {
    if (!text) return;
    const ok = await copyText(text);
    if (ok) {
      setCopied(true);
      toast.success(successMessage);
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(() => setCopied(false), 1500);
    } else {
      toast.error(t("ui.65dbb2e5ffe4bc8b"));
    }
  }

  return (
    <Button
      type="button"
      size={size}
      variant={variant}
      className={cn(className)}
      disabled={!text}
      onClick={handleCopy}
    >
      {copied ? <CheckIcon /> : <CopyIcon />}
      {label}
    </Button>
  );
}
