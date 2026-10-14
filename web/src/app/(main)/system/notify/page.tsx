"use client";

import { t } from '@/lib/i18n';

import * as React from "react";

import { BellIcon, PlusIcon, SendIcon, Trash2Icon } from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { api } from "@/lib/api";
import type { NotificationChannel, NotificationFilter, NotificationMeta } from "@/lib/types";

import {
  CHANNEL_FIELDS,
  type ChannelForm,
  emptyForm,
  KIND_LABEL,
  parseIDs,
  parseKeywords,
  parseKV,
  SEVERITY_OPTIONS,
} from "./_components/channel-fields";
import { ConfigField, FilterSummary } from "./_components/channel-form";
import { DeliveryList } from "./_components/delivery-list";
import { formatBacklog, StatTile } from "./_components/stat-tile";

// esta página solo es responsable del diseño: cargar datos, mantener el estado del formulario y llamar a las interfaces.。
// Definición y análisis de campos en _components/channel-fields.ts，Controles y resúmenes de filtros en
// _components/channel-form.tsx，Entrega registrada en _components/delivery-list.tsx——
// Dividido porque cada uno de ellos se puede leer por separado y, cuando se comprime en un solo archivo, esta página está cerca de 1100 OK。
export default function NotifyPage() {
  const [meta, setMeta] = React.useState<NotificationMeta | null>(null);
  const [channels, setChannels] = React.useState<NotificationChannel[]>([]);
  const [tab, setTab] = React.useState<"channels" | "deliveries">("channels");

  const [open, setOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<NotificationChannel | null>(null);
  const [form, setForm] = React.useState<ChannelForm>(emptyForm("dingtalk"));
  const [saving, setSaving] = React.useState(false);
  const [testing, setTesting] = React.useState(false);

  const [globalSaving, setGlobalSaving] = React.useState(false);
  const [baseURL, setBaseURL] = React.useState("");
  const [digestMin, setDigestMin] = React.useState("");

  const load = React.useCallback(() => {
    api
      .notifyMeta()
      .then((m) => {
        setMeta(m);
        setBaseURL(m.public_base_url);
        setDigestMin(m.digest_interval_min);
      })
      .catch((e) => toast.error(t("ui.795be4ac85c7bf15") + (e as Error).message));
    // Se debe informar la falla al cargar la lista de canales: la falla silenciosa se mostrará como ”Ni un solo canal”.」，
    // Los usuarios pensarán que la configuración se perdió, lo cual es más alarmante que informar un error directamente.。
    api
      .notifyChannels()
      .then(setChannels)
      .catch((e) => toast.error(t("ui.a6358eaec77c4739") + (e as Error).message));
  }, []);
  React.useEffect(() => {
    load();
  }, [load]);

  function setF(patch: Partial<ChannelForm>) {
    setForm((f) => ({ ...f, ...patch }));
  }
  function setCfg(key: string, value: unknown) {
    setForm((f) => ({ ...f, config: { ...f.config, [key]: value } }));
  }

  function openAdd() {
    setEditing(null);
    setForm(emptyForm(meta?.kinds[0]?.kind ?? "dingtalk"));
    setOpen(true);
  }

  function openEdit(ch: NotificationChannel) {
    setEditing(ch);
    // filter En el backend está Go Estructura, siempre serializada en objetos (no null），Entonces no hay necesidad de decir la verdad.。
    const f = ch.filter;
    setForm({
      name: ch.name,
      kind: ch.kind,
      mode: ch.mode,
      enabled: ch.enabled,
      ratePerMin: String(ch.rate_per_min),
      // repetido por el backend config Las credenciales aquí son valores de máscara; póngalos en el formulario tal como están y envíelos de vuelta tal como estaban al enviarlos.，
      // El backend conserva el valor original en la biblioteca en consecuencia。
      config: { ...ch.config },
      minSeverity: f.min_severity ?? "",
      includeText: (f.vulnclass_include ?? []).join("\n"),
      excludeText: (f.vulnclass_exclude ?? []).join("\n"),
      taskIDsText: (f.task_ids ?? []).join(","),
      assetIDsText: (f.asset_ids ?? []).join(","),
      onStatusChange: f.on_status_change ?? false,
    });
    setOpen(true);
  }

  // buildConfig Convertir estado de formulario a canal config。
  //
  // regla única, dos tipos de valores：
  //   - valor de máscara（"__masked__..."）Devolver tal cual → El servidor lo interpreta como ”Este campo no se ha modificado y se conserva el valor original en la biblioteca”.」
  //   - El resto se enviará según la entrada del usuario. Una cadena vacía significa ”borrar este campo”.」
  //
  // La razón por la que no prestamos especial atención al campo de credencial (como ”deje la credencial en blanco para omitirla”) es porque eso hará que el usuario**No se puede borrar**
  // Una clave mal configurada: no hay ninguna acción en la interfaz que diga ”Quiero eliminarla”. Bajo las reglas actuales，
  // Borrar el cuadro de entrada equivale a borrar el campo. La semántica es única y controlable por el usuario.。
  // El valor de la máscara no aparecerá en el cuadro de entrada (ver ConfigField），Entonces ”hay palabras en el cuadro” siempre es igual a
  // 「Completado por el usuario activamente」。
  function buildConfig(): Record<string, unknown> {
    const defs = CHANNEL_FIELDS[form.kind] ?? [];
    const out: Record<string, unknown> = {};
    for (const d of defs) {
      const raw = form.config[d.key];
      if (d.kind === "switch") {
        out[d.key] = raw === true;
        continue;
      }
      if (typeof raw === "string" && raw.startsWith("__masked__")) {
        out[d.key] = raw;
        continue;
      }
      if (d.kind === "number") {
        const n = Number(raw);
        out[d.key] = Number.isFinite(n) && n > 0 ? n : 0;
        continue;
      }
      if (d.kind === "kv") {
        out[d.key] = parseKV(String(raw ?? ""));
        continue;
      }
      if (d.kind === "list") {
        out[d.key] = String(raw ?? "")
          .split(/[\s,，]+/)
          .map((s) => s.trim())
          .filter(Boolean);
        continue;
      }
      out[d.key] = String(raw ?? "").trim();
    }
    return out;
  }

  function buildFilter(): NotificationFilter {
    return {
      min_severity: form.minSeverity || undefined,
      vulnclass_include: parseKeywords(form.includeText),
      vulnclass_exclude: parseKeywords(form.excludeText),
      task_ids: parseIDs(form.taskIDsText),
      asset_ids: parseIDs(form.assetIDsText),
      on_status_change: form.onStatusChange,
    };
  }

  async function saveForm() {
    if (!form.name.trim()) {
      toast.error(t("ui.5ba6a867f9ef3bc8"));
      return;
    }
    setSaving(true);
    try {
      const payload = {
        name: form.name.trim(),
        kind: form.kind,
        mode: form.mode,
        enabled: form.enabled,
        config: buildConfig(),
        filter: buildFilter(),
        rate_per_min: form.ratePerMin.trim() === "" ? undefined : Number(form.ratePerMin),
      };
      if (editing) {
        await api.notifyUpdateChannel(editing.id, payload);
        toast.success(t("ui.3a660ff5fb7c1989"));
        setOpen(false);
      } else {
        await api.notifyCreateChannel(payload);
        toast.success(t("ui.7dc1f0ece2f2aba3"));
        setOpen(false);
      }
      load();
    } catch (e) {
      toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message);
    } finally {
      setSaving(false);
    }
  }

  async function testChannel() {
    if (!editing) return;
    setTesting(true);
    try {
      const r = await api.notifyTestChannel(editing.id);
      toast.success(`Mensaje de prueba enviado (${r.latency_ms}ms), por favor confirmar en el grupo`);
    } catch (e) {
      // El backend devuelve el error original devuelto por el canal con sinceridad. Esta es la única pista para solucionar problemas de configuración. Muéstralo tal como es.。
      toast.error(t("ui.35d6765479f4220c") + (e as Error).message, { duration: 12000 });
    } finally {
      setTesting(false);
    }
  }

  async function removeChannel(ch: NotificationChannel) {
    try {
      await api.notifyDeleteChannel(ch.id);
      toast.success(`Eliminado:${ch.name}`);
      setOpen(false);
      load();
    } catch (e) {
      toast.error(t("ui.ac60bde833d4bb98") + (e as Error).message);
    }
  }

  async function toggleEnabled(ch: NotificationChannel) {
    try {
      await api.notifyUpdateChannel(ch.id, { enabled: !ch.enabled });
      load();
    } catch (e) {
      toast.error(t("ui.b4fb4e8016ec762e") + (e as Error).message);
    }
  }

  async function toggleGlobal(on: boolean) {
    setGlobalSaving(true);
    try {
      await api.setSettings({ notify_enabled: on });
      setMeta((m) => (m ? { ...m, enabled: on } : m));
      toast.success(on ? t("ui.c823ddb1c401ee7c") : t("ui.5a5bfccd96b442da"));
    } catch (e) {
      toast.error(t("ui.b4fb4e8016ec762e") + (e as Error).message);
    } finally {
      setGlobalSaving(false);
    }
  }

  async function saveGlobal() {
    setGlobalSaving(true);
    try {
      const patch: Record<string, unknown> = { notify_public_base_url: baseURL.trim() };
      const n = Number(digestMin);
      if (Number.isFinite(n) && n > 0) patch.notify_digest_interval_min = n;
      await api.setSettings(patch);
      toast.success(t("ui.3a660ff5fb7c1989"));
      load();
    } catch (e) {
      toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message);
    } finally {
      setGlobalSaving(false);
    }
  }

  const fields = CHANNEL_FIELDS[form.kind] ?? [];
  const secretKeys = new Set(meta?.kinds.find((k) => k.kind === form.kind)?.secret_keys ?? []);
  const defaultRate = meta?.kinds.find((k) => k.kind === form.kind)?.default_rate_per_min ?? 0;

  return (
    <div className="flex flex-1 flex-col gap-4 md:gap-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">{t("ui.a27efbf78b65937a")}</h1>
          <p className="text-muted-foreground text-sm">
             {t("ui.d0997eaa2f465cca")} </p>
        </div>
        {meta && (
          // uso div en lugar de label：Switch Trae el tuyo aria-label，Pon otra capa en el exterior. label
          // No se puede asociar con ningún control nativo y hace que parezca que puedes cambiar haciendo clic en el texto.。
          <div className="flex shrink-0 items-center gap-2 text-sm">
            <span className="text-muted-foreground">{t("ui.9cb9e99f61f43e93")}</span>
            <Switch
              checked={meta.enabled}
              disabled={globalSaving}
              onCheckedChange={toggleGlobal}
              aria-label={t("ui.512496db0e4dbd53")}
            />
          </div>
        )}
      </div>

      {meta && (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
          <StatTile label={t("ui.1501284e7c8472b9")} value={`${meta.stats.channels_on} / ${meta.stats.channels}`} hint={t("ui.96c8dde1244a5a86")} />
          <StatTile label={t("ui.07f0ec7c7d15b7fc")} value={String(meta.stats.sent_today)} />
          <StatTile label={t("ui.eeaca59a2410cd7d")} value={String(meta.stats.pending)} />
          <StatTile label={t("status.failed")} value={String(meta.stats.failed)} tone={meta.stats.failed > 0 ? "red" : undefined} />
          <StatTile
            label={t("ui.9b10f9458b76cdac")}
            value={formatBacklog(meta.stats.backlog_age_ms)}
            // La edad del trabajo pendiente es mucho más útil que el número del trabajo pendiente: Trabajo pendiente 3 Las barras pueden ser de 3 segundos arriba 3 hora。
            hint={meta.stats.backlog_age_ms > 5 * 60_000 ? t("ui.c2c1ec3b7812ecd7") : undefined}
            tone={meta.stats.backlog_age_ms > 5 * 60_000 ? "red" : undefined}
          />
        </div>
      )}

      <Card className="gap-3">
        <CardHeader>
          <CardTitle className="text-base">{t("ui.2c8860f2036e15bd")}</CardTitle>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <div className="grid gap-2">
            <Label htmlFor="n-base">{t("ui.5010d669653bd676")}</Label>
            <Input
              id="n-base"
              placeholder="https://nexus.example.com"
              value={baseURL}
              onChange={(e) => setBaseURL(e.target.value)}
            />
            <p className="text-muted-foreground text-xs">{t("ui.b7d9ccb4f9f1e297")}</p>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="n-digest">{t("ui.7f43e33b1e03cc89")}</Label>
            <Input
              id="n-digest"
              type="number"
              min={1}
              max={1440}
              placeholder="30"
              value={digestMin}
              onChange={(e) => setDigestMin(e.target.value)}
            />
            <p className="text-muted-foreground text-xs">{t("ui.903e2a22f0208d55")}</p>
          </div>
          <div className="sm:col-span-2">
            <Button onClick={saveGlobal} disabled={globalSaving}>
               {t("ui.99fc522df5180369")} </Button>
          </div>
        </CardContent>
      </Card>

      <Tabs value={tab} onValueChange={(v) => setTab(v as "channels" | "deliveries")} className="flex flex-col gap-4">
        <TabsList>
          <TabsTrigger value="channels">{t("ui.1501284e7c8472b9")}</TabsTrigger>
          <TabsTrigger value="deliveries">{t("ui.9eefc4935128e9b4")}</TabsTrigger>
        </TabsList>

        <TabsContent value="channels">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <button
              type="button"
              onClick={openAdd}
              className="text-foreground/70 border-foreground/70 hover:bg-muted/60 hover:shadow-sm flex min-h-[130px] flex-col items-center justify-center gap-2 rounded-xl border border-dashed transition"
            >
              <PlusIcon className="size-6" />
              <span className="text-sm">{t("ui.8c2ac66227a00c2d")}</span>
            </button>

            {channels.map((ch) => (
              <Card
                key={ch.id}
                onClick={() => openEdit(ch)}
                className="hover:border-primary/60 cursor-pointer gap-3 transition hover:shadow-sm"
              >
                <CardHeader>
                  <div className="flex items-center gap-2">
                    <BellIcon className="text-muted-foreground size-4 shrink-0" />
                    <CardTitle className="truncate text-base">{ch.name}</CardTitle>
                    {/* Se puede hacer clic en toda la tarjeta (para ingresar a la edición), por lo que estos dos controles deben tragarse las burbujas por separado.，
                        De lo contrario cambiar/La eliminación provocará la edición incidental. poner stopPropagation Espera el control mismo
                        Sobre el cuerpo en lugar de una capa div：conjunto div Creará una ”apariencia interactiva pero no
                        Elemento estático de ”Carácter”, que desencadena a11y Alarma, no tiene sentido semánticamente.。 */}
                    <div className="ml-auto flex items-center gap-2">
                      <Switch
                        checked={ch.enabled}
                        onCheckedChange={() => toggleEnabled(ch)}
                        onClick={(e) => e.stopPropagation()}
                        aria-label={t("ui.2d7376f810894b02")}
                      />
                      <Button
                        size="icon"
                        variant="outline"
                        aria-label={t("common.delete")}
                        onClick={(e) => {
                          e.stopPropagation();
                          // void Descartar explícitamente Promise：removeChannel yo mismo catch y toast，
                          // No es necesario aquí await（onClick No async）。
                          void removeChannel(ch);
                        }}
                      >
                        <Trash2Icon className="text-destructive" />
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="grid gap-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge variant="outline">{KIND_LABEL[ch.kind] ?? ch.kind}</Badge>
                    <Badge variant="outline">{ch.mode === "digest" ? t("ui.ed7bdc67c84b3ed6") : t("ui.8863a5170d55d1d1")}</Badge>
                    {!ch.enabled && <Badge variant="outline">{t("ui.68de79aa60784d31")}</Badge>}
                  </div>
                  <FilterSummary filter={ch.filter} />
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="deliveries">
          <DeliveryList channels={channels} />
        </TabsContent>
      </Tabs>

      <Sheet open={open} onOpenChange={setOpen}>
        <SheetContent side="right" className="w-full data-[side=right]:sm:max-w-lg">
          <SheetHeader>
            <SheetTitle>{editing ? editing.name : t("ui.1eeef1555bc67583")}</SheetTitle>
            <SheetDescription>
              {KIND_LABEL[form.kind] ?? form.kind}
              {defaultRate > 0 ? ` · Límite de corriente predeterminado ${defaultRate}bares/minuto` : t("ui.1b24389f44506fb2")}
            </SheetDescription>
          </SheetHeader>

          <div className="flex min-h-0 flex-1 flex-col overflow-y-auto px-4">
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label>{t("ui.9411c7c4d3c5802b")}</Label>
                <Select
                  value={form.kind}
                  onValueChange={(v) => {
                    // Cambiar el tipo equivale a cambiar un conjunto de campos de credenciales y la configuración anterior no se puede fusionar.。
                    setF({ kind: v, config: {} });
                  }}
                  disabled={!!editing}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {(meta?.kinds ?? []).map((k) => (
                      <SelectItem key={k.kind} value={k.kind}>
                        {KIND_LABEL[k.kind] ?? k.kind}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {editing && (
                  <p className="text-muted-foreground text-xs">
                     {t("ui.f6a04fa08a08b65c")} </p>
                )}
              </div>

              <div className="grid gap-2">
                <Label htmlFor="n-name">{t("ui.3ead58798d48bff8")}</Label>
                <Input
                  id="n-name"
                  placeholder={t("ui.aef0e76f38d03b48")}
                  value={form.name}
                  onChange={(e) => setF({ name: e.target.value })}
                />
              </div>

              {fields.length === 0 ? (
                <p className="text-muted-foreground text-sm">
                   {t("ui.3a8f73daecf84bcb")} </p>
              ) : (
                fields.map((d) => (
                  <ConfigField
                    key={d.key}
                    def={d}
                    value={form.config[d.key]}
                    isSecret={secretKeys.has(d.key)}
                    onChange={(v) => setCfg(d.key, v)}
                  />
                ))
              )}

              <div className="grid gap-2">
                <Label>{t("ui.d644139601abdde5")}</Label>
                <Select value={form.mode} onValueChange={(v) => setF({ mode: v as "realtime" | "digest" })}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="realtime">{t("ui.7856dca63369a1ba")}</SelectItem>
                    <SelectItem value="digest">{t("ui.b95889dc6d04b7d7")}</SelectItem>
                  </SelectContent>
                </Select>
                <p className="text-muted-foreground text-xs">
                   {t("ui.c1c14cb17b9bfa54")} </p>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="n-rate">{t("ui.f274fddf6448bd00")}</Label>
                <Input
                  id="n-rate"
                  type="number"
                  min={0}
                  placeholder={defaultRate > 0 ? String(defaultRate) : t("ui.975d03a768f1c563")}
                  value={form.ratePerMin}
                  onChange={(e) => setF({ ratePerMin: e.target.value })}
                />
                <p className="text-muted-foreground text-xs">
                   {t("ui.0d220ebef293cce3")} </p>
              </div>

              <div className="border-t pt-4">
                <p className="mb-3 text-sm font-medium">{t("ui.7beab3c8a5d90a73")}</p>
                <div className="grid gap-4">
                  <div className="grid gap-2">
                    <Label>{t("ui.09b39085a2c7b338")}</Label>
                    <Select
                      value={form.minSeverity || "all"}
                      onValueChange={(v) => setF({ minSeverity: v === "all" ? "" : v })}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {SEVERITY_OPTIONS.map((o) => (
                          <SelectItem key={o.value || "all"} value={o.value || "all"}>
                            {o.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="n-inc">{t("ui.86d05d8bc61c0797")}</Label>
                    <Textarea
                      id="n-inc"
                      placeholder={t("ui.783d71e73c5a9ce9")}
                      value={form.includeText}
                      onChange={(e) => setF({ includeText: e.target.value })}
                    />
                    <p className="text-muted-foreground text-xs">
                       {t("ui.eab396abc078c994")} </p>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="n-exc">{t("ui.870ce1f2fc4ea66f")}</Label>
                    <Textarea
                      id="n-exc"
                      placeholder={t("ui.58c7fd9c200bb923")}
                      value={form.excludeText}
                      onChange={(e) => setF({ excludeText: e.target.value })}
                    />
                    <p className="text-muted-foreground text-xs">{t("ui.dad7d3bdbe416e3b")}</p>
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="n-tasks">{t("ui.7d9b754c7af35c7a")}</Label>
                    <Input
                      id="n-tasks"
                      placeholder="1, 2, 3"
                      value={form.taskIDsText}
                      onChange={(e) => setF({ taskIDsText: e.target.value })}
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label htmlFor="n-assets">{t("ui.e1729e5067963061")}</Label>
                    <Input
                      id="n-assets"
                      placeholder="10, 11"
                      value={form.assetIDsText}
                      onChange={(e) => setF({ assetIDsText: e.target.value })}
                    />
                    <p className="text-muted-foreground text-xs">{t("ui.0f1b2ebd46a72956")}</p>
                  </div>
                  <div className="flex items-center gap-2 text-sm">
                    <Switch
                      checked={form.onStatusChange}
                      onCheckedChange={(v) => setF({ onStatusChange: v })}
                      aria-label={t("ui.e9362d2fcdd81a1a")}
                    />
                     {t("ui.b94c463be76638cc")} </div>
                </div>
              </div>

              <div className="flex items-center gap-2 text-sm">
                <Switch checked={form.enabled} onCheckedChange={(v) => setF({ enabled: v })} aria-label={t("ui.2d7376f810894b02")} />
                 {t("ui.2ec3742cc916b48c")} </div>
            </div>

            <div className="flex gap-2 pt-2 pb-6">
              <Button onClick={saveForm} disabled={saving}>
                {editing ? t("common.save") : t("common.add")}
              </Button>
              {editing && (
                <Button variant="outline" onClick={testChannel} disabled={testing}>
                  <SendIcon />  {t("ui.82da38739ebe99ff")} </Button>
              )}
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}
