"use client";

import { t } from '@/lib/i18n';

import * as React from "react";

import Link from "next/link";

import { ArrowUpCircleIcon } from "lucide-react";

import { api } from "@/lib/api";

/**
 * barra superior"Hay una nueva versión."Consejo: Verifique una vez cuando esté cargada la página completa. Si hay actualizaciones, aparecerán junto al número de versión.，
 * Haga clic en la tarjeta ”Versión y actualizaciones” para ir directamente a la página de configuración del sistema。
 *
 * par de servidores GitHub Los resultados de la consulta son 30 Caché de minutos, por lo que es seguro comprobarlo cada vez que se monta.
 * ——no certificado GitHub API solo 60 veces/hora/IP，Si no existe dicha capa de caché, abra algunas pestañas más
 * La cuota se agotará y luego tengo muchas ganas de actualizar pero no puedo encontrarla.。
 *
 * Si la consulta falla, quedará en silencio: la barra superior no es el lugar para informar errores. El usuario verá el motivo yendo a la página de configuración y haciendo clic en ”Buscar actualizaciones”.。
 */
export function UpdateBadge() {
  const [latest, setLatest] = React.useState("");

  React.useEffect(() => {
    let alive = true;
    api
      .checkUpdate()
      .then((r) => {
        // has_update Ya incluido"los números de versión son comparables"Según el criterio, la versión de desarrollo no mostrará este mensaje.。
        if (alive && r.has_update && r.latest) setLatest(r.latest.replace(/^v(?=\d)/, ""));
      })
      .catch(() => {
        // Silencioso: Sin red / GitHub El error no debería aparecer en la barra superior incluso si la corriente es limitada.。
      });
    return () => {
      alive = false;
    };
  }, []);

  if (!latest) return null;

  return (
    <Link
      href="/system/settings"
      title={`Nueva versión encontrada ${latest}, haga clic para actualizar`}
      className="inline-flex items-center gap-1.5 rounded-full bg-primary px-2.5 py-1 font-medium text-primary-foreground text-xs transition-opacity hover:opacity-90"
    >
      {/* Punto de respiro: hay muchos elementos en la barra superior y es fácil ignorar el texto puro. La animación lo hace visible de un vistazo.。 */}
      <span className="relative flex size-1.5">
        <span className="absolute inline-flex size-full animate-ping rounded-full bg-primary-foreground opacity-75" />
        <span className="relative inline-flex size-1.5 rounded-full bg-primary-foreground" />
      </span>
      <ArrowUpCircleIcon className="size-3.5" />
      <span className="hidden sm:inline">{t("ui.88c9bb2d67e09616")} {latest}</span>
      <span className="sm:hidden">{t("ui.88c9bb2d67e09616")}</span>
    </Link>
  );
}
