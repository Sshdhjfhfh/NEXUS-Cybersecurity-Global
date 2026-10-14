"use client";

import { t } from '@/lib/i18n';

import * as React from "react";

import {
  CheckCircle2Icon,
  DownloadIcon,
  ExternalLinkIcon,
  RefreshCwIcon,
  RotateCcwIcon,
  TriangleAlertIcon,
} from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { api, sseUrl } from "@/lib/api";
import type { UpdateCheck, UpdateProgress } from "@/lib/types";

/** El tiempo máximo de espera para que una nueva versión esté en línea. Una actualización requiere tres inicios de proceso (almacenamiento temporal → disfrazarse → Nueva versión），
 *  Cada vez que se necesitan segundos, tres minutos son suficientes para cubrir el disco lento y Docker reconstrucción de contenedores。 */
const RESTART_TIMEOUT_MS = 180_000;

function humanSize(n?: number): string {
  if (!n || n <= 0) return "";
  const units = ["B", "KB", "MB", "GB"];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

export function UpdateCard() {
  const [info, setInfo] = React.useState<UpdateCheck | null>(null);
  const [checking, setChecking] = React.useState(true);
  const [progress, setProgress] = React.useState<UpdateProgress | null>(null);
  // con progress Separado: el proceso finalizará una vez que se complete el almacenamiento temporal.，SSE Interrumpirá, cambiará a sondeo en este momento /api/health。
  const [restarting, setRestarting] = React.useState(false);
  const [busy, setBusy] = React.useState(false);

  // quiet también decida si desea omitir el caché de back-end: verifique automáticamente el caché al ingresar a la página (la barra superior acaba de marcar），
  // El usuario hace clic manualmente en ”Buscar actualizaciones” para forzar el regreso a la fuente. De lo contrario, la versión recién lanzada no será visible hasta que expire el caché.。
  const check = React.useCallback((quiet = false) => {
    setChecking(true);
    api
      .checkUpdate(!quiet)
      .then((r) => {
        setInfo(r);
        if (!quiet) {
          if (r.error) toast.error(t("ui.7a06611e317941d8") + r.error);
          else if (r.has_update) toast.success(`Nueva versión encontrada ${r.latest}`);
          else if (r.comparable) toast.success(t("ui.fb72de3691de9a60"));
        }
      })
      .catch((e) => {
        if (!quiet) toast.error(t("ui.7a06611e317941d8") + (e as Error).message);
      })
      .finally(() => setChecking(false));
  }, []);

  React.useEffect(() => {
    check(true);
  }, [check]);

  // sondeo /api/health Hasta que cambie el número de versión。
  //
  // El criterio debe ser"Versión cambiada"en lugar de"Se puede conectar"：Durante el proceso de cambio, la versión anterior se reiniciará brevemente.
  // （Esa vez solo era responsable de nexus.new Reemplácelo y salga inmediatamente), solo mirar la conectividad juzgará mal el éxito.。
  const waitForNewVersion = React.useCallback(async (fromVersion: string) => {
    setRestarting(true);
    const deadline = Date.now() + RESTART_TIMEOUT_MS;
    while (Date.now() < deadline) {
      await sleep(2000);
      try {
        const r = await fetch("/api/health", { cache: "no-store" });
        if (r.ok) {
          const j = (await r.json()) as { version?: string };
          if (j.version && j.version !== fromVersion) {
            toast.success(`Actualizado a ${j.version}, recargando la página`);
            await sleep(800);
            window.location.reload();
            return;
          }
        }
      } catch {
        // Se espera que no se pueda conectar dentro de la ventana de reinicio, continúe con el sondeo。
      }
    }
    setRestarting(false);
    toast.error("Tiempo de espera de reinicio del servicio. Verifique los registros del backend o confirme que nexus se inició a través de start.sh/start.bat.");
  }, []);

  // progreso de actualización de la suscripción。SSE No salir Next de /api Reescribir (esa capa se almacenará en el búfer y los eventos no se podrán eliminar)）。
  const openStream = React.useCallback(
    (fromVersion: string) => {
      const es = new EventSource(sseUrl("/api/update/stream"));
      es.onmessage = (ev) => {
        let p: UpdateProgress;
        try {
          p = JSON.parse(ev.data) as UpdateProgress;
        } catch {
          return;
        }
        setProgress(p);
        if (p.phase === "failed") {
          es.close();
          setBusy(false);
          toast.error("Error en la actualización:" + (p.error || p.message));
          return;
        }
        if (p.phase === "staged") {
          es.close();
          void waitForNewVersion(fromVersion);
        }
      };
      es.onerror = () => {
        // Cuando el proceso sale SSE Debe estar desconectado. Si ya estás esperando un reinicio, esto es normal.，
        // entregado /api/health Simplemente continúe sondeando para determinar。
        es.close();
      };
      return es;
    },
    [waitForNewVersion],
  );

  const doUpdate = () => {
    if (!info) return;
    const from = info.current;
    const ok = window.confirm(
      `Confirmar para actualizar a ${info.latest}？\n\n` +
        t("ui.8ebdc22a2d98bdd0") +
        (info.mode === "docker"
          ? t("ui.87f513136164ef85") +
            t("ui.12b64e719d1e7ed4")
          : ""),
    );
    if (!ok) return;

    setBusy(true);
    setProgress({ phase: "downloading", percent: 0, message: t("ui.d2b925965ea8d2b9") });
    const es = openStream(from);
    api.applyUpdate().catch((e) => {
      es.close();
      setBusy(false);
      setProgress(null);
      toast.error(t("ui.492340856c6772a4") + (e as Error).message);
    });
  };

  const doRollback = () => {
    if (!info) return;
    if (
      !window.confirm(
        t("ui.8c2e4c7ca64aabf8"),
      )
    )
      return;
    const from = info.current;
    setBusy(true);
    api
      .rollbackUpdate()
      .then(() => {
        toast.success(t("ui.12c8b2d62acc1280"));
        void waitForNewVersion(from);
      })
      .catch((e) => {
        setBusy(false);
        toast.error(t("ui.bd4304807b9a91bf") + (e as Error).message);
      });
  };

  const phase = progress?.phase;
  const showProgress = busy || restarting;
  // Sólo la fase de descarga puede obtener el porcentaje real (presione Content-Length Conde). verificar/descomprimir/Esperando reinicio
  // Una etapa con duración desconocida, la barra de progreso se llena y se agrega una animación de pulso para indicarlo."Ocupado pero no estoy seguro de cuánto tiempo será"。
  const downloading = !restarting && phase === "downloading";
  const pct = downloading ? Math.max(progress?.percent ?? 0, 0) : 100;

  return (
    // La página de configuración es un diseño de flujo en cascada de varias columnas. La propia tarjeta es responsable del espacio entre líneas y prohíbe la desconexión entre columnas (ver page.tsx Comentarios）。
    <Card className="mb-4 break-inside-avoid md:mb-6">
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <DownloadIcon className="size-4" />
           {t("ui.6134047d5622d344")} </CardTitle>
        <CardDescription>{t("ui.447d9a2670e4c90a")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <span className="text-muted-foreground">{t("ui.498a54b716f55fc9")}</span>
          <Badge variant="secondary" className="font-mono">
            {info?.current ?? "…"}
          </Badge>
          {info && (
            <>
              <Badge variant="outline" className="font-mono">
                {info.os}/{info.arch}
              </Badge>
              <Badge variant="outline">{info.mode === "docker" ? "Docker" : t("ui.bbac29c4df618fe6")}</Badge>
            </>
          )}
          {info?.latest && (
            <>
              <span className="text-muted-foreground">{t("ui.b410efcdec91c718")}</span>
              <Badge variant={info.has_update ? "default" : "secondary"} className="font-mono">
                {info.latest}
              </Badge>
            </>
          )}
          {info?.html_url && (
            <a
              href={info.html_url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-xs text-muted-foreground underline-offset-4 hover:underline"
            >
               {t("ui.fafc2b715647d0e8")} <ExternalLinkIcon className="size-3" />
            </a>
          )}
        </div>

        {info?.boot_notice && (
          <p className="flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 p-2 text-xs text-amber-700 dark:text-amber-400">
            <TriangleAlertIcon className="mt-0.5 size-3.5 shrink-0" />
            {info.boot_notice}
          </p>
        )}

        {info?.error && (
          <p className="flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/10 p-2 text-xs text-destructive">
            <TriangleAlertIcon className="mt-0.5 size-3.5 shrink-0" />
             {t("ui.647fb107715cebf2")}{info.error}
            {"　"}{t("ui.bbdaa6650ff4dac3")} </p>
        )}

        {info && !info.comparable && info.reason && <p className="text-xs text-muted-foreground">{info.reason}</p>}

        {info?.has_update && info.asset_available === false && (
          <p className="flex items-start gap-2 rounded-md border border-destructive/40 bg-destructive/10 p-2 text-xs text-destructive">
            <TriangleAlertIcon className="mt-0.5 size-3.5 shrink-0" />
            {info.latest}  {t("ui.f9564379705d9be7")} {info.os}/{info.arch}  {t("ui.86349a447c2831b8")} {info.asset}{t("ui.2bd66882a6a6d044")} </p>
        )}

        {info?.has_update && info.asset_available !== false && (
          <p className="text-xs text-muted-foreground">
             {t("ui.897e4ec52665c6d3")} <span className="font-mono">{info.asset}</span>
            {info.size ? `（${humanSize(info.size)}）` : ""}{t("ui.e4423f1b2cfb58b3")} </p>
        )}

        {info && !info.has_update && info.comparable && !info.error && (
          <p className="flex items-center gap-2 text-xs text-muted-foreground">
            <CheckCircle2Icon className="size-3.5 text-emerald-600" />
             {t("ui.8548b4f9f89f49b4")} </p>
        )}

        {info?.mode === "docker" && info.has_update && (
          <p className="text-xs text-muted-foreground">
             {t("ui.b9d56fe9b157db3e")} <span className="font-mono"> docker compose up -d </span>
             {t("ui.793e77bf37eb50b3")} <span className="font-mono"> docker compose build orchestrator &amp;&amp; docker compose up -d orchestrator</span>。
          </p>
        )}

        {showProgress && (
          <div className="space-y-1.5">
            <Progress value={pct} className={downloading ? undefined : "animate-pulse"} />
            <p className="text-xs text-muted-foreground">
              {restarting ? t("ui.2d2a683991cd21c6") : progress?.message}
            </p>
          </div>
        )}

        <div className="flex flex-wrap gap-2">
          <Button variant="outline" size="sm" onClick={() => check(false)} disabled={checking || busy || restarting}>
            <RefreshCwIcon className={checking ? "size-4 animate-spin" : "size-4"} />
             {t("ui.066381ff326bdf15")} </Button>
          <Button
            size="sm"
            onClick={doUpdate}
            disabled={busy || restarting || !info?.has_update || info?.asset_available === false}
          >
            <DownloadIcon className="size-4" />
            {info?.has_update ? `actualizado a ${info.latest}` : t("ui.d9074c0652cf1e87")}
          </Button>
          {info?.has_backup && (
            <Button variant="ghost" size="sm" onClick={doRollback} disabled={busy || restarting}>
              <RotateCcwIcon className="size-4" />
               {t("ui.b872134bed694c89")} </Button>
          )}
        </div>

        <p className="text-xs text-muted-foreground">
           {t("ui.77b65941d237f2cf")} <span className="font-mono">start.sh</span>{t("ui.b6bc72152ab7ad4c")} <span className="font-mono"> start.bat</span>) Iniciar nexus; Al ejecutar nexus body directamente, el programa no se iniciará automáticamente después de salir. </p>
      </CardContent>
    </Card>
  );
}
