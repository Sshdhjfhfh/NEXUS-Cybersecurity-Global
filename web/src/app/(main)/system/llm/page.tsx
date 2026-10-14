"use client";

import { t } from '@/lib/i18n';

import * as React from "react";

import {
  Loader2Icon,
  PlugZapIcon,
  PlusIcon,
  RefreshCwIcon,
  RotateCcwIcon,
  SaveIcon,
  StarIcon,
  Trash2Icon,
  ZapIcon,
} from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { api } from "@/lib/api";
import type { LLMPoolMember, LLMPoolStatus, LLMProfile, LLMRetryOverride } from "@/lib/types";
import { cn } from "@/lib/utils";

import { ProfileRetryFields, RetryPolicyPanel, ZERO_OVERRIDE } from "./_components/retry";

// pensar en cambiar(thinking.type)e intensidad del pensamiento(reasoning_effort)son dos campos [mutuamente independientes]，
// Configure cada uno individualmente; algunas interfaces no lo hacen thinking Los campos y los parámetros de intensidad por sí solos pueden activar el pensamiento, por lo que es necesario desacoplarlos.。
// cadena vacía del repositorio = Este campo [No enviar】；Radix Select No aceptar nulo value，Por lo tanto UI uso "none"
// Sentinel significa no enviar y al acceder "" intercambio（NONE / fromStore / toStore）。
const NONE = "none";
const fromStore = (v?: string) => (v ? v : NONE);
const toStore = (v: string) => (v === NONE ? "" : v);
const THINKING_TYPES: { value: string; label: string }[] = [
  { value: NONE, label: t("ui.4b7c8c1cb99b9856") },
  { value: "disabled", label: t("common.close") },
  { value: "enabled", label: t("ui.b4a71696a3bd7de8") },
];
// ¿Qué nombre de campo de solicitud se utiliza para generar el límite superior (solo openai El formato tiene sentido）。NONE ↔ "" Tome el mismo conjunto de conversiones centinela。
const MAX_TOKENS_FIELDS: { value: string; label: string }[] = [
  { value: NONE, label: t("ui.8724d0e9a2f9d492") },
  { value: "max_completion_tokens", label: "max_completion_tokens" },
];
// Los otros dos formatos tienen nombres de campos fijos respectivamente y las opciones no tienen sentido para ellos. Por favor, déjelo claro en la copia de la descripción.。
const MAX_TOKENS_FIELD_HINTS: Record<string, string> = {
  openai:
    t("ui.3f5542b292de133e"),
  anthropic: t("ui.1fda51a025afb42d"),
  "openai-responses": t("ui.196ae272602ff94c"),
};
const EFFORT_LEVELS: { value: string; label: string }[] = [
  { value: NONE, label: t("ui.4b7c8c1cb99b9856") },
  { value: "low", label: "low" },
  { value: "medium", label: "medium" },
  { value: "high", label: "high" },
  { value: "xhigh", label: "xhigh" },
  { value: "max", label: "max" },
];

function cooldownText(secs: number) {
  if (secs <= 0) return "";
  if (secs < 60) return `${secs}s`;
  return `${Math.ceil(secs / 60)}min`;
}

// Se muestra una configuración en la tarjeta como ”es normal”. No completado Key La configuración no puede enviar ninguna solicitud. Cabe mencionar en primer lugar el disyuntor.；
// El resto del estado proviene del registro del fusible de sondeo (no se generarán nuevos registros cuando el sondeo esté desactivado. En este momento, ”normal”」= Ninguna falla conocida）。
type Health = { label: string; cls: string; hint?: string };
function healthOf(p: LLMProfile, m?: LLMPoolMember): Health {
  if (!p.api_key_hint) {
    return {
      label: t("ui.ada98977b8d5846d"),
      cls: "border-muted-foreground/40 text-muted-foreground",
      hint: t("ui.86b7d6775334666e"),
    };
  }
  if (m?.state === "tripped") {
    return {
      label: m.cooldown_secs > 0 ? `Fusionado · ${cooldownText(m.cooldown_secs)}` : t("ui.da72859eac4dcae4"),
      cls: "border-destructive/50 text-destructive",
      hint: m.last_error,
    };
  }
  if (m?.state === "degraded") {
    return {
      label: `Excepción · Fallo ${m.fails}veces`,
      cls: "border-amber-500/50 text-amber-600 dark:text-amber-400",
      hint: m.last_error,
    };
  }
  return { label: t("ui.a008009bce4f9b61"), cls: "border-emerald-500/50 text-emerald-600 dark:text-emerald-400" };
}

// ─────────────────────────────────────────────────────────────────────────────
// sondear el cajón de configuración
// ─────────────────────────────────────────────────────────────────────────────

function PoolSheet({
  open,
  onOpenChange,
  pool,
  onReload,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  pool: LLMPoolStatus | null;
  onReload: () => Promise<void>;
}) {
  const [busy, setBusy] = React.useState(false);

  // La cuenta regresiva de enfriamiento son los segundos restantes calculados por el backend; solo cuando el cajón esté abierto y la configuración sea anormal, sáquelo regularmente para dejar que comience.。
  React.useEffect(() => {
    if (!open || !pool?.enabled || !pool.chain.some((m) => m.state !== "ok")) return;
    const t = setInterval(() => void onReload(), 10_000);
    return () => clearInterval(t);
  }, [open, pool, onReload]);

  async function toggle(patch: { llm_pool_enabled?: boolean; llm_pool_bind_fallback?: boolean }) {
    if (busy) return;
    setBusy(true);
    try {
      await api.setSettings(patch);
      await onReload();
      if (patch.llm_pool_enabled !== undefined) {
        toast.success(patch.llm_pool_enabled ? t("ui.473ab7ded523ec20") : t("ui.e70951bf9705956a"));
      } else {
        toast.success(t("ui.a510ce156e560bf1"));
      }
    } catch (e) {
      toast.error(`Error de configuración:${(e as Error).message}`);
    } finally {
      setBusy(false);
    }
  }

  async function recover(id?: string) {
    try {
      await api.resetLLMPool(id);
      await onReload();
      toast.success(id ? t("ui.9a662fb733210fa5") : t("ui.436dbf4cf8a776ad"));
    } catch (e) {
      toast.error(`La recuperación falló:${(e as Error).message}`);
    }
  }

  const enabled = pool?.enabled ?? false;
  const chain = pool?.chain ?? [];
  // Miembros que participan en la encuesta (excluidos aquellos marcados como ”no participan en la encuesta”), el orden es el orden real de los intentos del backend。
  const inChain = chain.filter((m) => m.active || !m.excluded);
  const tripped = chain.filter((m) => m.state === "tripped");

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="flex flex-col gap-0 p-0 data-[side=right]:sm:max-w-lg">
        <SheetHeader className="px-4">
          <SheetTitle className="flex items-center gap-2">
            <ZapIcon className="size-4" />  {t("ui.4375920a8d234744")} </SheetTitle>
          <SheetDescription>
             {t("ui.9d9664cd7ebbb372")}<b>{t("ui.c268d78ba991fbff")}</b>{t("ui.8eab7dd33abcd509")} </SheetDescription>
        </SheetHeader>

        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4 pb-6">
          <div className="flex items-center justify-between gap-4 rounded-lg border p-3">
            <div className="grid gap-0.5">
              <Label className="text-sm">{t("ui.097f7cd18e45f853")}</Label>
              <p className="text-muted-foreground text-xs">{t("ui.d5edaaf2866f66f3")}</p>
            </div>
            <Switch
              checked={enabled}
              disabled={busy}
              onCheckedChange={(v) => void toggle({ llm_pool_enabled: v })}
              aria-label={t("ui.8a119091653c41ad")}
            />
          </div>

          {enabled && (
            <>
              <div className="flex items-center justify-between gap-4 rounded-lg border p-3">
                <div className="grid gap-0.5">
                  <Label className="text-sm">{t("ui.9f8fcb9fd88c1c93")}</Label>
                  <p className="text-muted-foreground text-xs">
                     {t("ui.1cf3866750fae1c0")} </p>
                </div>
                <Switch
                  checked={pool?.bind_fallback ?? false}
                  disabled={busy}
                  onCheckedChange={(v) => void toggle({ llm_pool_bind_fallback: v })}
                  aria-label={t("ui.27b092107c3f5b30")}
                />
              </div>

              <Separator />

              <div className="grid gap-2">
                <div className="flex items-center justify-between">
                  <Label className="text-sm">{t("ui.13ccd29eb49d020a")}</Label>
                  {tripped.length > 0 && (
                    <Button size="sm" variant="ghost" onClick={() => void recover()}>
                      <RotateCcwIcon />  {t("ui.c06c2e5f08dfabc0")} </Button>
                  )}
                </div>
                {inChain.length < 2 && (
                  <p className="text-muted-foreground text-xs">
                     {t("ui.34021e9b4f861ced")} {inChain.length}  {t("ui.9eeed2aba0b19a25")} </p>
                )}
                {chain.map((m) => {
                  const excluded = m.excluded && !m.active;
                  const order = excluded ? null : inChain.findIndex((x) => x.profile_id === m.profile_id) + 1;
                  return (
                    <div
                      key={m.profile_id}
                      className={cn(
                        "grid gap-1 rounded-lg border p-2.5 text-sm",
                        excluded && "opacity-55",
                        m.state === "tripped" && "border-destructive/40",
                      )}
                    >
                      <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                        <span className="w-5 shrink-0 text-center font-mono text-muted-foreground text-xs">
                          {order ?? "—"}
                        </span>
                        <span className="font-medium">{m.name}</span>
                        {m.active && (
                          <Badge variant="outline" className="border-amber-400/50 text-amber-500">
                             {t("ui.8e38c2571d87f6ee")} </Badge>
                        )}
                        {excluded && <Badge variant="outline">{t("ui.051731d26324c6ba")}</Badge>}
                        <div className="ml-auto flex items-center gap-2">
                          {m.state === "tripped" && m.cooldown_secs > 0 && (
                            <span className="text-muted-foreground text-xs">{t("ui.8ec6d3e74181f474")} {cooldownText(m.cooldown_secs)}</span>
                          )}
                          {m.state === "degraded" && (
                            <span className="text-muted-foreground text-xs">{t("ui.47e7e065a8dc5e76")} {m.fails}  {t("ui.d4c028ac38863872")}</span>
                          )}
                          {m.state !== "ok" && (
                            <Button
                              size="icon"
                              variant="ghost"
                              className="size-7"
                              aria-label={t("ui.e5eb2eabf66dcd19")}
                              title={t("ui.46fe4be6b7fae400")}
                              onClick={() => void recover(m.profile_id)}
                            >
                              <RotateCcwIcon className="size-3.5" />
                            </Button>
                          )}
                        </div>
                      </div>
                      <div className="flex flex-wrap items-center gap-x-3 pl-7 text-muted-foreground text-xs">
                        <code className="truncate font-mono">{m.model}</code>
                        {!m.active && <span>{t("ui.dbae0b2a1a745bfc")} {m.priority}</span>}
                      </div>
                      {m.last_error && (
                        <p className="truncate pl-7 font-mono text-muted-foreground text-xs" title={m.last_error}>
                          {m.last_error}
                        </p>
                      )}
                    </div>
                  );
                })}
                {chain.length === 0 && (
                  <div className="rounded-lg border border-dashed p-4 text-center text-muted-foreground text-sm">
                     {t("ui.98ea2fb76ceb2aa0")} </div>
                )}
              </div>

              <div className="rounded-lg border border-dashed p-3 text-muted-foreground text-xs leading-relaxed">
                 {t("ui.de057ee6138dd8f6")} </div>
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Cajón de configuración de modelo (nuevo / Los editores comparten el mismo conjunto de formularios.）
// ─────────────────────────────────────────────────────────────────────────────

function ProfileSheet({
  profile,
  open,
  onOpenChange,
  onSaved,
}: {
  profile: LLMProfile | null; // null = nuevo
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onSaved: (id: string) => void;
}) {
  const isNew = !profile;
  const [name, setName] = React.useState("");
  const [format, setFormat] = React.useState<"anthropic" | "openai" | "openai-responses">("anthropic");
  const [model, setModel] = React.useState("");
  const [baseUrl, setBaseUrl] = React.useState("");
  const [proxy, setProxy] = React.useState("");
  const [apiKey, setApiKey] = React.useState("");
  const [keyHint, setKeyHint] = React.useState("");
  const [rps, setRps] = React.useState("0");
  const [rpm, setRpm] = React.useState("0");
  const [cw, setCw] = React.useState("0"); // ventana de contexto(K tokens);0=predeterminado200K
  const [thinkingType, setThinkingType] = React.useState(NONE);
  const [effort, setEffort] = React.useState(NONE);
  const [priority, setPriority] = React.useState("0"); // orden de votación;Lo más grande es primero
  const [poolExclude, setPoolExclude] = React.useState(false);
  const [streaming, setStreaming] = React.useState(true); // true=Transmisión(predeterminado);false=sin transmisión
  const [maxTokens, setMaxTokens] = React.useState("0"); // límite superior de salida de respuesta única;0=No enviar
  const [maxTokensField, setMaxTokensField] = React.useState(NONE); // ¿Qué nombre de campo se debe utilizar para el límite superior?;NONE=max_tokens
  const [sessionHeaderKey, setSessionHeaderKey] = React.useState(""); // nombre del encabezado de sesión personalizado;vacío=No enviar
  const [retry, setRetry] = React.useState<LLMRetryOverride>(ZERO_OVERRIDE); // Reintentar cobertura para esta configuración;completo 0=Sigue el panorama general
  const [testing, setTesting] = React.useState(false);
  const [saving, setSaving] = React.useState(false);
  const [models, setModels] = React.useState<string[]>([]);
  const [loadingModels, setLoadingModels] = React.useState(false);
  const [modelsOpen, setModelsOpen] = React.useState(false);

  // Cada vez que se abre desde el entrante profile Completar el formulario (restablecer los valores predeterminados al crear uno nuevo). Cierra y abre el cajón.
  // Simplemente un comienzo limpio, sin dejar restos de la configuración anterior.。
  React.useEffect(() => {
    if (!open) return;
    setName(profile?.name ?? "");
    setFormat(profile?.format === "openai" || profile?.format === "openai-responses" ? profile.format : "anthropic");
    setModel(profile?.model ?? "");
    setBaseUrl(profile?.base_url ?? "");
    setProxy(profile?.proxy ?? "");
    setRps(String(profile?.rate_per_second ?? 0));
    setRpm(String(profile?.rate_per_minute ?? 0));
    setCw(String(profile?.context_window_k ?? 0));
    setThinkingType(fromStore(profile?.thinking_type));
    setEffort(fromStore(profile?.reasoning_effort));
    setPriority(String(profile?.priority ?? 0));
    setPoolExclude(profile?.pool_exclude ?? false);
    setStreaming(profile?.streaming ?? true);
    setMaxTokens(String(profile?.max_tokens ?? 0));
    setMaxTokensField(fromStore(profile?.max_tokens_field));
    setSessionHeaderKey(profile?.session_header_key ?? "");
    setRetry(profile?.retry ?? ZERO_OVERRIDE);
    setApiKey("");
    setKeyHint(profile?.api_key_hint ?? "");
    setModels([]);
    setModelsOpen(false);
  }, [open, profile]);

  const profileId = profile ? Number(profile.id) : undefined;

  async function loadModels() {
    if (loadingModels) return;
    setLoadingModels(true);
    setModels([]);
    try {
      const r = await api.fetchLLMModels(format, baseUrl, apiKey, proxy, profileId);
      if (r.ok && r.models && r.models.length > 0) {
        setModels(r.models);
        setModelsOpen(true);
        toast.success(`cargado ${r.models.length}modelos`);
      } else {
        toast.error(`No se pudo cargar el modelo:${r.error ?? t("ui.fc1083d06e55e9e8")}`);
      }
    } catch (e) {
      toast.error(`Error al cargar el modelo:${(e as Error).message}`);
    } finally {
      setLoadingModels(false);
    }
  }

  async function testConnection() {
    if (testing) return;
    setTesting(true);
    try {
      // Pruebe configurando los parámetros de pensamiento que realmente se ejecutarán, de modo que los modelos que no admitan este campo fallarán aquí.，
      // En lugar de esperar hasta ejecutar una misión para explotar. pasar profile id：Key Utilice el guardado cuando el cuadro de entrada quede vacío Key。
      const r = await api.testLLM(
        format,
        model,
        baseUrl,
        apiKey,
        proxy,
        toStore(thinkingType),
        toStore(effort),
        profileId,
        streaming,
        sessionHeaderKey.trim(),
      );
      // Muestra el contenido de la respuesta juntos: se puede ver que el modelo realmente habló y se considera lo mismo que la conversación.。
      if (r.ok)
        toast.success(`Conexión exitosa · ${r.latency_ms ?? "?"}ms · ${r.model ?? model}`, {
          description: r.reply ? `Respuesta:${r.reply}` : undefined,
        });
      else toast.error(`Error de conexión:${r.error ?? t("ui.2669e033a1213e64")}`);
    } catch (e) {
      toast.error(`Error de prueba:${(e as Error).message}`);
    } finally {
      setTesting(false);
    }
  }

  async function save() {
    if (!name.trim() || !model.trim()) {
      toast.error(t("ui.ca82547edf264421"));
      return;
    }
    if (saving) return;
    setSaving(true);
    try {
      const { id } = await api.saveLLMProfile({
        ...(profile ? { id: Number(profile.id) } : {}),
        name: name.trim(),
        format,
        model: model.trim(),
        base_url: baseUrl.trim(),
        proxy: proxy.trim(),
        api_key: apiKey,
        rate_per_second: Number(rps) || 0,
        rate_per_minute: Number(rpm) || 0,
        context_window_k: Number(cw) || 0,
        thinking_type: toStore(thinkingType),
        reasoning_effort: toStore(effort),
        priority: Number(priority) || 0,
        pool_exclude: poolExclude,
        streaming,
        max_tokens: Math.max(0, Number(maxTokens) || 0),
        // El cambio de nombre de campo es solo para openai(Chat Completions) Significativo, otros formatos volverán a los valores predeterminados.；
        // El backend también realizará la misma normalización nuevamente, pero no lo permitas. UI Enviar valor contradictorio。
        max_tokens_field: format === "openai" ? toStore(maxTokensField) : "",
        session_header_key: sessionHeaderKey.trim(),
        retry,
      });
      if (isNew) toast.success(`Creado:${name.trim()}(Establecer como activo en la tarjeta para habilitar)`);
      else toast.success(profile?.is_default ? t("ui.9f70e0decc8157df") : t("ui.3a660ff5fb7c1989"));
      onSaved(String(id));
      onOpenChange(false);
    } catch (e) {
      toast.error(`Error al guardar:${(e as Error).message}`);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="flex flex-col gap-0 p-0 data-[side=right]:min-w-[420px] data-[side=right]:sm:max-w-xl"
      >
        <SheetHeader className="px-4">
          <SheetTitle className="flex items-center gap-2">
            {isNew ? t("ui.4601b473507f1a0b") : `Editar:${profile?.name}`}
            {profile?.is_default && (
              <Badge variant="outline" className="border-amber-400/50 text-amber-500">
                 {t("ui.f34b2412f9ade4f0")} </Badge>
            )}
          </SheetTitle>
          <SheetDescription>
            {isNew
              ? t("ui.75717cfd0a644409")
              : t("ui.a94ed13244858e67")}
          </SheetDescription>
        </SheetHeader>

        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4 pb-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="p-name">{t("common.name")}</Label>
              <Input
                id="p-name"
                placeholder={t("ui.5b0ee74701a9ddf6")}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label>{t("ui.aa32c15eaf920e0f")}</Label>
              <Select value={format} onValueChange={(v) => setFormat(v as "anthropic" | "openai" | "openai-responses")}>
                <SelectTrigger>
                  <SelectValue placeholder={t("ui.55b923a3f72d92e4")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="anthropic">Anthropic</SelectItem>
                  <SelectItem value="openai">OpenAI (Chat Completions)</SelectItem>
                  <SelectItem value="openai-responses">OpenAI (Responses API)</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="p-model">{t("ui.8cce676e2f83bf17")}</Label>
            <div className="flex gap-2">
              <Input
                id="p-model"
                className="font-mono"
                placeholder="claude-opus-4-8"
                value={model}
                onChange={(e) => setModel(e.target.value)}
              />
              {/* modal: esto Popover El contenido de portal llegar <body>，en Sheet Fuera del bloqueo de desplazamiento，
                  Sin adición modal La lista se puede representar pero no se puede desplazar.。modal deja que mantenga el bloqueo de desplazamiento superior por sí solo。 */}
              <Popover open={modelsOpen} onOpenChange={setModelsOpen} modal>
                <PopoverTrigger asChild>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="shrink-0"
                    disabled={loadingModels}
                    onClick={loadModels}
                    title={t("ui.fd9c465809ac0ce2")}
                  >
                    {loadingModels ? <Loader2Icon className="animate-spin" /> : <RefreshCwIcon />}
                  </Button>
                </PopoverTrigger>
                {models.length > 0 && (
                  <PopoverContent className="max-h-72 w-72 gap-0 overflow-y-auto overscroll-contain p-1" align="end">
                    {models.map((m) => (
                      <button
                        key={m}
                        type="button"
                        className="w-full shrink-0 rounded-md px-2 py-1.5 text-left font-mono text-xs hover:bg-accent hover:text-accent-foreground"
                        onClick={() => {
                          setModel(m);
                          setModelsOpen(false);
                        }}
                      >
                        {m}
                      </button>
                    ))}
                  </PopoverContent>
                )}
              </Popover>
            </div>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="p-base-url">{t("ui.3125cf89e6e0c399")}</Label>
            <Input
              id="p-base-url"
              className="font-mono"
              placeholder="https://api.openai.com/v1"
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="p-proxy">{t("ui.9279b0701269e845")}</Label>
            <Input
              id="p-proxy"
              className="font-mono"
              placeholder="socks5://user:pass@127.0.0.1:1080 · http://127.0.0.1:8080"
              value={proxy}
              onChange={(e) => setProxy(e.target.value)}
            />
            <p className="text-muted-foreground text-xs">
               {t("ui.bc876ca20a2c2efb")} </p>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="p-session-header">{t("ui.50456bda3fb8b22e")}</Label>
            <Input
              id="p-session-header"
              className="font-mono"
              placeholder={t("ui.4477cb2e4fbd369d")}
              value={sessionHeaderKey}
              onChange={(e) => setSessionHeaderKey(e.target.value)}
            />
            <p className="text-muted-foreground text-xs">
               {t("ui.c72bd615ca024340")} <b>{t("ui.2175355e087bcc3e")}</b>{t("ui.0c2f68cfbcbcb965")} </p>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="p-api-key">API Key</Label>
            <Input
              id="p-api-key"
              type="password"
              placeholder={keyHint ? `Ya configurado (${keyHint}), déjelo en blanco y permanezca sin cambios.` : "sk-…"}
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-3">
            <div className="grid gap-2">
              <Label htmlFor="p-rps">{t("ui.104bab2c815adad2")}</Label>
              <Input id="p-rps" type="number" min={0} value={rps} onChange={(e) => setRps(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="p-rpm">{t("ui.d809cdee63718579")}</Label>
              <Input id="p-rpm" type="number" min={0} value={rpm} onChange={(e) => setRpm(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="p-cw">{t("ui.d6d9539d247c930b")}</Label>
              <Input
                id="p-cw"
                type="number"
                min={0}
                max={1000}
                value={cw}
                onChange={(e) => setCw(e.target.value)}
                placeholder="200"
              />
            </div>
          </div>
          <p className="-mt-2 text-muted-foreground text-xs">
             {t("ui.824894ced29df847")} </p>

          <div className="grid gap-3 rounded-lg border p-3">
            <div className="flex items-center justify-between gap-4">
              <div className="grid gap-0.5">
                <Label htmlFor="p-priority" className="text-sm">
                   {t("ui.81a247fcc8f3ec64")} </Label>
                <p className="text-muted-foreground text-xs">
                   {t("ui.cc295dd97ca2c14b")} </p>
              </div>
              <Input
                id="p-priority"
                type="number"
                className="w-24 shrink-0"
                value={priority}
                onChange={(e) => setPriority(e.target.value)}
              />
            </div>
            <div className="flex items-center justify-between gap-4 border-t pt-3">
              <div className="grid gap-0.5">
                <Label className="text-sm">{t("ui.051731d26324c6ba")}</Label>
                <p className="text-muted-foreground text-xs">
                   {t("ui.7c96cf41298d5e23")} </p>
              </div>
              <Switch checked={poolExclude} onCheckedChange={setPoolExclude} aria-label={t("ui.051731d26324c6ba")} />
            </div>
            <div className="flex items-center justify-between gap-4 border-t pt-3">
              <div className="grid gap-0.5">
                <Label className="text-sm">{t("ui.fe2209d97b319c33")}</Label>
                <p className="text-muted-foreground text-xs">
                   {t("ui.653d4bdea22614b9")} </p>
              </div>
              <Switch checked={streaming} onCheckedChange={setStreaming} aria-label={t("ui.7fffc90fc71b76e5")} />
            </div>
          </div>

          <div className="grid gap-3 rounded-lg border p-3">
            <div className="flex items-center justify-between gap-4">
              <div className="grid gap-0.5">
                <Label htmlFor="p-max-tokens" className="text-sm">
                   {t("ui.2f03ef3c2b3473c2")} </Label>
                <p className="text-muted-foreground text-xs">
                   {t("ui.906e4be67ebba156")} </p>
              </div>
              <Input
                id="p-max-tokens"
                type="number"
                min={0}
                className="w-28 shrink-0"
                value={maxTokens}
                onChange={(e) => setMaxTokens(e.target.value)}
                placeholder="0"
              />
            </div>
            <div className="flex items-center justify-between gap-4 border-t pt-3">
              <div className="grid gap-0.5">
                <Label className="text-sm">{t("ui.29345e4e3cd5dbdc")}</Label>
                <p className="text-muted-foreground text-xs">{MAX_TOKENS_FIELD_HINTS[format]}</p>
              </div>
              <Select
                value={format === "openai" ? maxTokensField : NONE}
                onValueChange={setMaxTokensField}
                disabled={format !== "openai"}
              >
                <SelectTrigger className="w-56 shrink-0">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {MAX_TOKENS_FIELDS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid gap-3 rounded-lg border p-3">
            <div className="flex items-center justify-between gap-4">
              <div className="grid gap-0.5">
                <Label className="text-sm">{t("ui.aa262a05e208ceb1")}</Label>
                <p className="text-muted-foreground text-xs">
                   {t("ui.9716c56b9277ce23")} </p>
              </div>
              <Select value={thinkingType} onValueChange={setThinkingType}>
                <SelectTrigger className="w-32 shrink-0">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {THINKING_TYPES.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center justify-between gap-4 border-t pt-3">
              <div className="grid gap-0.5">
                <Label className="text-sm">{t("ui.7863257c1e5c683b")}</Label>
                <p className="text-muted-foreground text-xs">
                   {t("ui.248ae9e9b0b7c0dd")} </p>
              </div>
              <Select value={effort} onValueChange={setEffort}>
                <SelectTrigger className="w-32 shrink-0">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {EFFORT_LEVELS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <ProfileRetryFields value={retry} onChange={setRetry} />
        </div>

        <div className="flex gap-2 border-t px-4 py-3">
          <Button variant="outline" onClick={testConnection} disabled={testing}>
            {testing ? <Loader2Icon className="animate-spin" /> : <PlugZapIcon />}
            {testing ? t("ui.a3da3a488d2e46b0") : t("ui.692506e62db7b7ec")}
          </Button>
          <Button onClick={save} disabled={saving} className="flex-1">
            {saving && <Loader2Icon className="animate-spin" />}
            {!saving && (isNew ? <PlusIcon /> : <SaveIcon />)}
            {isNew ? t("ui.238b4429a65d9a87") : t("common.save")}
          </Button>
        </div>
      </SheetContent>
    </Sheet>
  );
}

// ─────────────────────────────────────────────────────────────────────────────

export default function LLMPage() {
  const [profiles, setProfiles] = React.useState<LLMProfile[]>([]);
  const [pool, setPool] = React.useState<LLMPoolStatus | null>(null);
  const [poolOpen, setPoolOpen] = React.useState(false);
  // El interruptor y el contenido del cajón se guardan por separado: cuando está cerrado editing Dejar sin cambios, de lo contrario el título cambiará de
  // 「Editar X」Flash a ”Nuevo」。editing = null Indica nuevo。
  const [editOpen, setEditOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<LLMProfile | null>(null);
  const openEditor = React.useCallback((p: LLMProfile | null) => {
    setEditing(p);
    setEditOpen(true);
  }, []);

  const loadPool = React.useCallback(async () => {
    try {
      setPool(await api.llmPool());
    } catch {
      /* ignore */
    }
  }, []);

  const load = React.useCallback(async () => {
    try {
      setProfiles(await api.llmProfiles());
    } catch {
      /* ignore */
    }
    await loadPool();
  }, [loadPool]);

  React.useEffect(() => {
    void load();
  }, [load]);

  // Botón de insignia de salud en la tarjeta profile id Obtener estado de sondeo。
  const health = React.useMemo(() => {
    const m = new Map<string, LLMPoolMember>();
    for (const c of pool?.chain ?? []) m.set(c.profile_id, c);
    return m;
  }, [pool]);

  async function activate(id: string, name: string) {
    try {
      await api.activateLLMProfile(id);
      toast.success(`Activado:${name}`);
      await load();
    } catch (e) {
      toast.error(`Falló la activación:${(e as Error).message}`);
    }
  }

  async function remove(p: LLMProfile) {
    if (p.is_default) {
      toast.error(t("ui.636312d753ab03e0"));
      return;
    }
    try {
      await api.deleteLLMProfile(p.id);
      toast.success(`Eliminado:${p.name}`);
      await load();
    } catch (e) {
      toast.error(`Error al eliminar:${(e as Error).message}`);
    }
  }

  const poolOn = pool?.enabled ?? false;

  return (
    <div className="flex flex-1 flex-col gap-4 md:gap-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="font-semibold text-xl tracking-tight">LLM</h1>
          <p className="text-muted-foreground text-sm">
             {t("ui.635e4ae4b56b5c08")} </p>
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => setPoolOpen(true)}>
            <ZapIcon />  {t("ui.2083e1b2e833429b")} {poolOn && (
              <Badge variant="outline" className="ml-1 border-emerald-500/50 text-emerald-600 dark:text-emerald-400">
                 {t("ui.56f90b9199cea3b0")} </Badge>
            )}
          </Button>
          <Button size="sm" variant="outline" onClick={() => openEditor(null)}>
            <PlusIcon />  {t("ui.238b4429a65d9a87")} </Button>
        </div>
      </div>

      <Tabs defaultValue="profiles" className="flex-1">
        <TabsList>
          <TabsTrigger value="profiles">{t("ui.0f96520c58640cab")}</TabsTrigger>
          <TabsTrigger value="retry">{t("ui.224a599b831159da")}</TabsTrigger>
        </TabsList>

        <TabsContent value="profiles" className="mt-4">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
            {profiles.map((p) => {
              const h = healthOf(p, health.get(p.id));
              return (
                // biome-ignore lint/a11y/useSemanticElements: La tarjeta contiene su propio botón de operación, usando nativo <button> Provocará anidamiento de botones (ilegal). HTML）
                <Card
                  key={p.id}
                  role="button"
                  tabIndex={0}
                  onClick={() => openEditor(p)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      openEditor(p);
                    }
                  }}
                  className={cn(
                    "cursor-pointer gap-0 py-4 outline-none transition-colors hover:border-foreground/30",
                    p.is_default && "border-amber-400/50 bg-amber-400/5",
                  )}
                >
                  <CardContent className="grid gap-2 px-4">
                    <div className="flex items-start gap-2">
                      <StarIcon
                        className={cn(
                          "mt-0.5 size-4 shrink-0",
                          p.is_default ? "fill-amber-400 text-amber-400" : "text-muted-foreground",
                        )}
                      />
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="truncate font-medium text-sm">{p.name}</span>
                          <Badge variant="outline" className="uppercase">
                            {p.format}
                          </Badge>
                          <Badge variant="outline" className={cn("ml-auto", h.cls)} title={h.hint}>
                            {h.label}
                          </Badge>
                        </div>
                        <code className="mt-1 block truncate font-mono text-muted-foreground text-xs">{p.model}</code>
                      </div>
                    </div>

                    <div className="flex flex-wrap gap-x-3 gap-y-0.5 pl-6 text-muted-foreground text-xs">
                      {p.api_key_hint && <span>{p.api_key_hint}</span>}
                      <span>
                        {p.rate_per_second}/s · {p.rate_per_minute}/min
                      </span>
                      {p.proxy && <span className="truncate">{t("ui.1e068f0819ee6905")} {p.proxy}</span>}
                      {p.reasoning_effort && (
                        <span>{t("ui.a42667a60252a963")} {p.reasoning_effort === "off" ? t("ui.32dcc0ec9aae9984") : p.reasoning_effort}</span>
                      )}
                      {/* Los dos campos relacionados con el sondeo solo tienen significado cuando el sondeo está activado y no ocupan espacio cuando está desactivado. */}
                      {poolOn &&
                        !p.is_default &&
                        (p.pool_exclude ? <span>{t("ui.051731d26324c6ba")}</span> : <span>{t("ui.dbae0b2a1a745bfc")} {p.priority ?? 0}</span>)}
                    </div>

                    <div className="mt-1 flex gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        className="flex-1"
                        disabled={p.is_default}
                        onClick={(e) => {
                          e.stopPropagation();
                          void activate(p.id, p.name);
                        }}
                      >
                        {p.is_default ? t("ui.0a0ca661486c7ebc") : t("ui.f94a1c2b15df8f0e")}
                      </Button>
                      <Button
                        size="icon"
                        variant="outline"
                        aria-label={t("ui.516600f464f37ffc")}
                        onClick={(e) => {
                          e.stopPropagation();
                          void remove(p);
                        }}
                      >
                        <Trash2Icon className="text-destructive" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
            {profiles.length === 0 && (
              <div className="col-span-full rounded-lg border border-dashed p-10 text-center text-muted-foreground text-sm">
                 {t("ui.d7b86ad5f6106b0e")} </div>
            )}
          </div>
        </TabsContent>

        <TabsContent value="retry" className="mt-4">
          <RetryPolicyPanel />
        </TabsContent>
      </Tabs>

      <ProfileSheet profile={editing} open={editOpen} onOpenChange={setEditOpen} onSaved={() => void load()} />
      <PoolSheet open={poolOpen} onOpenChange={setPoolOpen} pool={pool} onReload={loadPool} />
    </div>
  );
}
