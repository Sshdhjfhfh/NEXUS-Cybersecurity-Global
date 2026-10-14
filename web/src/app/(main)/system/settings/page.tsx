"use client";

import { t } from '@/lib/i18n';

import * as React from "react";

import { CpuIcon, FlaskConicalIcon, KeyboardIcon, RadioTowerIcon, SearchIcon, ShieldAlertIcon } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { api } from "@/lib/api";
import { CHAT_SEND_MODE_OPTIONS, type ChatSendMode, setChatSendMode, useChatSendMode } from "@/lib/chat-send-mode";
import type { Settings } from "@/lib/types";

import { UpdateCard } from "./_components/update-card";

export default function SystemSettingsPage() {
  const [trafficCapture, setTrafficCapture] = React.useState(false);
  const [agentTrafficBinding, setAgentTrafficBinding] = React.useState(false);
  const [webSearch, setWebSearch] = React.useState(false);
  const [backend, setBackend] = React.useState("ddgs");
  const [braveKeySet, setBraveKeySet] = React.useState(false);
  const [braveKeyInput, setBraveKeyInput] = React.useState("");
  const [tavilyKeySet, setTavilyKeySet] = React.useState(false);
  const [tavilyKeyInput, setTavilyKeyInput] = React.useState("");
  const [savingTavilyKey, setSavingTavilyKey] = React.useState(false);
  const [proxyInput, setProxyInput] = React.useState("");
  const [savingProxy, setSavingProxy] = React.useState(false);
  const [globalProxyInput, setGlobalProxyInput] = React.useState("");
  const [savingGlobalProxy, setSavingGlobalProxy] = React.useState(false);
  const [testing, setTesting] = React.useState(false);
  const [loaded, setLoaded] = React.useState(false);
  const [saving, setSaving] = React.useState(false);
  const [savingKey, setSavingKey] = React.useState(false);
  const [pyInterp, setPyInterp] = React.useState("");
  const [workers, setWorkers] = React.useState("3");
  const [savingWorkers, setSavingWorkers] = React.useState(false);
  // rango de inyección de restricciones operativas(Ambos están habilitados de forma predeterminada)。
  const [injectPlanner, setInjectPlanner] = React.useState(true);
  const [injectWorker, setInjectWorker] = React.useState(true);
  // función experimental:noa compresión de contexto(Desactivado por defecto)。
  const [noaCompaction, setNoaCompaction] = React.useState(false);
  // Preferencia pura de front-end: no vayas /api/settings，Lectura y escritura directa localStorage。
  const sendMode = useChatSendMode();

  const apply = React.useCallback((s: Settings) => {
    setTrafficCapture(!!s.traffic_capture);
    setAgentTrafficBinding(!!s.agent_traffic_binding);
    setWebSearch(!!s.web_search_enabled);
    setBackend(s.web_search_backend || "ddgs");
    setBraveKeySet(!!s.brave_key_set);
    setTavilyKeySet(!!s.tavily_key_set);
    setProxyInput(s.web_search_proxy ?? "");
    setGlobalProxyInput(s.global_proxy ?? "");
    setPyInterp(s.python_interpreter ?? "");
    setWorkers(String(s.workers ?? 3));
    setInjectPlanner(s.constraints_inject_planner !== false);
    setInjectWorker(s.constraints_inject_worker !== false);
    setNoaCompaction(!!s.noa_compaction);
  }, []);

  const saveWorkers = () => {
    const n = Number(workers);
    if (!Number.isInteger(n) || n <= 0) {
      toast.error(t("ui.2d39f5f710231995"));
      return;
    }
    setSavingWorkers(true);
    api
      .setSettings({ workers: n })
      .then((s) => {
        apply(s);
        toast.success(t("ui.e01a2ce46a7c823e"));
      })
      .catch((e) => toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message))
      .finally(() => setSavingWorkers(false));
  };

  const savePython = () => {
    setSaving(true);
    api
      .setSettings({ python_interpreter: pyInterp.trim() })
      .then((s) => {
        apply(s);
        toast.success(t("ui.aad739d2aeca5778"));
      })
      .catch((e) => toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message))
      .finally(() => setSaving(false));
  };
  const detectPython = () => {
    setSaving(true);
    api
      .detectPython()
      .then((r) => setPyInterp(r.python_interpreter))
      .catch(() => undefined)
      .finally(() => setSaving(false));
  };

  React.useEffect(() => {
    api
      .settings()
      .then(apply)
      .catch(() => undefined)
      .finally(() => setLoaded(true));
  }, [apply]);

  const toggleTraffic = (v: boolean) => {
    setTrafficCapture(v); // optimistic
    setSaving(true);
    api
      .setSettings({ traffic_capture: v })
      .then(apply)
      .catch(() => setTrafficCapture(!v)) // revert on failure
      .finally(() => setSaving(false));
  };

  const toggleInjectPlanner = (v: boolean) => {
    setInjectPlanner(v); // optimistic
    api
      .setSettings({ constraints_inject_planner: v })
      .then(apply)
      .catch(() => setInjectPlanner(!v)); // revert on failure
  };

  const toggleAgentTrafficBinding = (v: boolean) => {
    setAgentTrafficBinding(v);
    setSaving(true);
    api
      .setSettings({ agent_traffic_binding: v })
      .then((s) => {
        apply(s);
        toast.success(v ? t("ui.a2aa55f160059c1e") : t("ui.712c00b2a8d36915"));
      })
      .catch((e) => {
        setAgentTrafficBinding(!v);
        toast.error(`Error al guardar:${(e as Error).message}`);
      })
      .finally(() => setSaving(false));
  };

  const toggleInjectWorker = (v: boolean) => {
    setInjectWorker(v); // optimistic
    api
      .setSettings({ constraints_inject_worker: v })
      .then(apply)
      .catch(() => setInjectWorker(!v)); // revert on failure
  };

  const toggleNoaCompaction = (v: boolean) => {
    setNoaCompaction(v); // optimistic
    api
      .setSettings({ noa_compaction: v })
      .then((s) => {
        apply(s);
        toast.success(v ? t("ui.09eceab0733de377") : t("ui.26a1a7e6d22faefe"));
      })
      .catch((e) => {
        setNoaCompaction(!v); // revert on failure
        toast.error(`Error al guardar:${(e as Error).message}`);
      });
  };

  // Persist a web-search patch (enable and/or backend). Optimistic with refetch.
  const saveWebSearch = (patch: Partial<Settings>) => {
    setSaving(true);
    api
      .setSettings(patch)
      .then((s) => {
        apply(s);
        toast.success(t("ui.b420d52b48f938b3"));
      })
      .catch((e) => {
        toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message);
        api
          .settings()
          .then(apply)
          .catch(() => undefined);
      })
      .finally(() => setSaving(false));
  };

  const saveBraveKey = () => {
    setSavingKey(true);
    api
      .setSettings({ brave_search_api_key: braveKeyInput })
      .then((s) => {
        apply(s);
        setBraveKeyInput("");
        toast.success(t("ui.507f3238ae096ce7"));
      })
      .catch((e) => toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message))
      .finally(() => setSavingKey(false));
  };

  const saveTavilyKey = () => {
    setSavingTavilyKey(true);
    api
      .setSettings({ tavily_search_api_key: tavilyKeyInput })
      .then((s) => {
        apply(s);
        setTavilyKeyInput("");
        toast.success(t("ui.adc966f364071492"));
      })
      .catch((e) => toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message))
      .finally(() => setSavingTavilyKey(false));
  };

  const saveProxy = () => {
    setSavingProxy(true);
    api
      .setSettings({ web_search_proxy: proxyInput.trim() })
      .then((s) => {
        apply(s);
        toast.success(proxyInput.trim() ? t("ui.63a17a3873f2cdbf") : t("ui.5adefa649b16077e"));
      })
      .catch((e) => toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message))
      .finally(() => setSavingProxy(false));
  };

  const saveGlobalProxy = () => {
    setSavingGlobalProxy(true);
    api
      .setSettings({ global_proxy: globalProxyInput.trim() })
      .then((s) => {
        apply(s);
        toast.success(globalProxyInput.trim() ? t("ui.3b8585eac6f4707d") : t("ui.e77d6961c8641306"));
      })
      .catch((e) => toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message))
      .finally(() => setSavingGlobalProxy(false));
  };

  // Run a real "test" search ("test") against the CURRENT form values (backend +
  // proxy + entered key), falling back to saved values server-side. Toasts result.
  const runTest = () => {
    setTesting(true);
    api
      .testWebSearch({
        web_search_backend: backend,
        web_search_proxy: proxyInput.trim(),
        brave_search_api_key: braveKeyInput,
        tavily_search_api_key: tavilyKeyInput,
      })
      .then((r) => {
        if (r.ok) toast.success(`Prueba de búsqueda exitosa · ${r.backend} Regreso ${r.count}resultados`);
        else toast.error(t("ui.fd3eb67b762e7714") + (r.error || t("ui.683b5faec33d5f1c")));
      })
      .catch((e) => toast.error(t("ui.fd3eb67b762e7714") + (e as Error).message))
      .finally(() => setTesting(false));
  };

  // brave-free selected but no key stored and none being entered → tool stays off.
  const braveNeedsKey = webSearch && backend === "brave-free" && !braveKeySet;

  return (
    <div className="flex flex-1 flex-col gap-4 md:gap-6">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">{t("ui.65a323f0c17c4950")}</h1>
        <p className="text-muted-foreground text-sm">{t("ui.246af773fec4c75b")}</p>
      </div>

      {/* Varias columnas en lugar de grid：La tarjeta de búsqueda de red es varias veces más alta que el resto y la altura cambia con el backend seleccionado（brave/tavily
          de key La entrada es representación condicional）。grid El más alto llenará toda la fila, dejando un gran espacio en blanco al lado.，
          Varias columnas se completan automáticamente de manera equilibrada según la altura del contenido. espaciado de tarjetas mb en lugar de gap——bajo diseño de varias columnas
          column-gap Independientemente del espaciado de las columnas, el espaciado de las filas debe ser establecido por los propios elementos secundarios.。 */}
      <div className="columns-1 gap-4 md:gap-6 lg:columns-2">
        <UpdateCard />

        <Card className="mb-4 break-inside-avoid md:mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <RadioTowerIcon className="size-4" />
               {t("ui.e2ad72a341aaf597")} </CardTitle>
            <CardDescription>
               {t("ui.cfa4122b7c9c0be9")} <br />
               {t("ui.b56b9b511d326bbf")} <b>{"No"}</b>{t("ui.9e3b41fa49b3b401")}<b>{t("ui.53a426d29ee783f1")}</b>{t("ui.98626f7de3a74ca9")} </CardDescription>
          </CardHeader>
          <CardContent className="flex items-center justify-between gap-4">
            <Label htmlFor="traffic-capture" className="text-sm font-normal text-muted-foreground">
              {trafficCapture ? t("ui.d0e3a38e11f27a0a") : t("ui.e3fba6907102f13e")}
            </Label>
            <Switch
              id="traffic-capture"
              checked={trafficCapture}
              disabled={!loaded || saving}
              onCheckedChange={toggleTraffic}
            />
          </CardContent>
        </Card>

        <Card className="mb-4 break-inside-avoid md:mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <RadioTowerIcon className="size-4" />
               {t("ui.0578c192393e59ba")} </CardTitle>
            <CardDescription id="agent-traffic-binding-description">
               {t("ui.0b5ee7046b69ba3e")} <b>{t("ui.1f790b0fd5c60f12")}</b>
              <br />
               {t("ui.d22cbd04b051a8cc")} </CardDescription>
          </CardHeader>
          <CardContent className="flex items-center justify-between gap-4">
            <Label htmlFor="agent-traffic-binding" className="text-sm font-normal text-muted-foreground">
              {agentTrafficBinding ? t("ui.c0846ce74d93793e") : t("ui.bbcc0dd6acaa81b0")}
            </Label>
            <Switch
              id="agent-traffic-binding"
              aria-describedby="agent-traffic-binding-description"
              checked={agentTrafficBinding}
              disabled={!loaded || saving}
              onCheckedChange={toggleAgentTrafficBinding}
            />
          </CardContent>
        </Card>

        <Card className="mb-4 break-inside-avoid md:mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <RadioTowerIcon className="size-4" />
               {t("ui.bd51633a4c65c828")} </CardTitle>
            <CardDescription>
               {t("ui.9a2307e8dc657970")}<b>{t("ui.281d705bb2f03abf")}</b>{t("ui.c0b0e8c636d0da9e")} <b>http / https / socks5</b>{t("ui.73af2a1c4d3d4c58")}{" "}
              <code>user:pass</code>  {t("ui.5b44c2fcf7f9e17a")} <br />
               {t("ui.b4a71696a3bd7de8")}<b>{t("ui.e2ad72a341aaf597")}</b>{t("ui.0824fcffed483728")}<b>{t("ui.eaa9b4707a85e539")}</b>{t("ui.ce683365bdff1fc8")} <br />
              <b>consejo</b>{t("ui.ad29830760f833de")}<b>{t("ui.c0717410c96ab83a")}</b>{t("ui.f30d2ed18040317d")} <code>ALL_PROXY</code>  {t("ui.c4db85c956669731")} </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-2">
            <Label htmlFor="global-proxy" className="text-sm font-normal text-muted-foreground">
               {t("ui.aabc0191ec10a19a")} </Label>
            <div className="flex items-center gap-2">
              <Input
                id="global-proxy"
                autoComplete="off"
                placeholder={t("ui.95dd7c891f3a417c")}
                value={globalProxyInput}
                disabled={!loaded || savingGlobalProxy}
                onChange={(e) => setGlobalProxyInput(e.target.value)}
              />
              <Button type="button" onClick={saveGlobalProxy} disabled={!loaded || savingGlobalProxy}>
                guardar </Button>
            </div>
            <p className="text-muted-foreground text-xs">
              {globalProxyInput.trim() ? t("ui.1e7d8cfe67759046") : t("ui.d5033fb2bd19b1d1")}
            </p>
          </CardContent>
        </Card>

        <Card className="mb-4 break-inside-avoid md:mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <ShieldAlertIcon className="size-4" />
               {t("ui.77a1cc8e38b7d4f1")} </CardTitle>
            <CardDescription>
               {t("ui.8a429159eabdb5bc")}<b>{t("ui.649d63e04b5f7c2e")}</b>{t("ui.0db2e05bcd16711c")} <br />
               {t("ui.efff635b4d89e57b")} <b>{t("ui.654c0c0fe86bd0af")}</b>{"con"} <b>{t("ui.03ca10b088927879")}</b>
               {t("ui.3488385167831036")} </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <div className="flex items-center justify-between gap-4">
              <Label htmlFor="inject-planner" className="text-sm font-normal text-muted-foreground">
                 {t("ui.7140eda47d04dc6f")}{injectPlanner ? t("ui.09a484d927b5e548") : "· Cerrado"}
              </Label>
              <Switch
                id="inject-planner"
                checked={injectPlanner}
                disabled={!loaded}
                onCheckedChange={toggleInjectPlanner}
              />
            </div>
            <div className="flex items-center justify-between gap-4">
              <Label htmlFor="inject-worker" className="text-sm font-normal text-muted-foreground">
                 {t("ui.0a34f77b8747be69")}{injectWorker ? t("ui.09a484d927b5e548") : "· Cerrado"}
              </Label>
              <Switch
                id="inject-worker"
                checked={injectWorker}
                disabled={!loaded}
                onCheckedChange={toggleInjectWorker}
              />
            </div>
          </CardContent>
        </Card>

        <Card className="mb-4 break-inside-avoid md:mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <FlaskConicalIcon className="size-4" />
               {t("ui.2573f1eb38a833b6")} </CardTitle>
            <CardDescription>
               {t("ui.f015fb81e0ac3fef")} <br />
              <b>{t("ui.75269a2adb956498")}</b>{t("ui.f1f8970334743ad6")} <b>{t("ui.d59a1228c5dd238d")}</b>{t("ui.2aa4710f9861675d")} </CardDescription>
          </CardHeader>
          <CardContent className="flex items-center justify-between gap-4">
            <Label htmlFor="noa-compaction" className="text-sm font-normal text-muted-foreground">
               {t("ui.75269a2adb956498")}{noaCompaction ? t("ui.09a484d927b5e548") : "· Cerrado"}
            </Label>
            <Switch
              id="noa-compaction"
              checked={noaCompaction}
              disabled={!loaded}
              onCheckedChange={toggleNoaCompaction}
            />
          </CardContent>
        </Card>

        <Card className="mb-4 break-inside-avoid md:mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <SearchIcon className="size-4" />
               {t("ui.faad93a241392801")} </CardTitle>
            <CardDescription>
               {t("ui.858778cea24e377f")}<b>{t("ui.303f3eadf132b976")}</b>{t("ui.dadd21f16e0114f4")}<b>{t("ui.07bda7fdebe5b2c9")}</b>{t("ui.563e89163e2b2944")} <b>web_search</b>{t("ui.13794052c499c7e0")}<b>{t("ui.0b5ffa61a0bb3887")}</b>
               {t("ui.bfc69700d08734d1")} <br />
               {t("ui.0a7d12734174e8ca")} <b>DuckDuckGo（ddgs）</b>{t("ui.26125f6e8a435d8a")}<b>{t("ui.3bd901346d2d7ed8")}</b>{t("ui.0f205d6177d11776")}{" "}
              <b>Tavily</b>{t("ui.f1f43e6594ff072b")} <b>DeepSeek</b>{t("ui.aac804d7d955f00e")} </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <div className="flex items-center justify-between gap-4">
              <Label htmlFor="web-search" className="text-sm font-normal text-muted-foreground">
                {webSearch ? t("ui.5fd9d04555b6d006") : t("ui.d3df7e1334d3205f")}
              </Label>
              <Switch
                id="web-search"
                checked={webSearch}
                disabled={!loaded || saving}
                onCheckedChange={(v) => {
                  setWebSearch(v); // optimistic
                  saveWebSearch({ web_search_enabled: v });
                }}
              />
            </div>

            {webSearch && (
              <div className="flex items-center justify-between gap-4">
                <Label className="text-sm font-normal text-muted-foreground">{t("ui.02bd03f508f4de47")}</Label>
                <Select
                  value={backend}
                  disabled={!loaded || saving}
                  onValueChange={(v) => {
                    setBackend(v); // optimistic
                    saveWebSearch({ web_search_backend: v });
                  }}
                >
                  <SelectTrigger className="w-48 shrink-0">
                    <SelectValue placeholder={t("ui.5d4b07b72a5d72f9")} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="ddgs">{t("ui.7219f09275a0f5e0")}</SelectItem>
                    <SelectItem value="brave-free">{t("ui.f07806beee62c9e1")}</SelectItem>
                    <SelectItem value="tavily">{t("ui.3f0555007eeb205a")}</SelectItem>
                    <SelectItem value="deepseek">{t("ui.5cde2ebfa5d40170")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            )}

            {webSearch && backend === "deepseek" && (
              <div className="border-border/60 bg-muted/30 flex flex-col gap-2 rounded-md border p-3">
                <p className="text-sm font-medium">{t("ui.7fe4b4f4436f5ee3")}</p>
                <p className="text-muted-foreground text-xs leading-relaxed">
                   {t("ui.6e6a4db096ee375c")}<b>{t("ui.905846a4cae991f1")}</b>{t("ui.f3691797248d63d0")} <b>{t("ui.89001691e27c7971")}</b>{t("ui.344d300c4b5c2f7c")}<b>{t("ui.a0ab82278c6d89d8")}</b>
                   {t("ui.aa3b1b52774a4a51")} </p>
                <p className="text-muted-foreground text-xs leading-relaxed">
                   {t("ui.a4765beeb0e9c217")} <b>{t("ui.ebc9e9579bc402e1")}</b>{t("ui.1236e40b0c480a7f")}<b>{t("ui.3082f1e4694f5eb7")}</b>{t("ui.a877860398f99b28")}<b>{t("ui.832fc771098ad394")}</b>{t("ui.1f00f79fc48fde93")}<b>{t("ui.6ba07701cc8fb6c3")}</b>
                   {t("ui.2fad037f888b4eed")} </p>
                <p className="text-muted-foreground text-xs leading-relaxed">
                   {t("ui.11d57702926c21d3")} </p>
              </div>
            )}

            {webSearch && backend === "brave-free" && (
              <div className="flex flex-col gap-2">
                <Label htmlFor="brave-key" className="text-sm font-normal text-muted-foreground">
                  Brave Search API Key
                  {braveKeySet && <span className="ml-2 text-xs text-emerald-500">{t("ui.54a0560489f8d8ab")}</span>}
                </Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="brave-key"
                    type="password"
                    autoComplete="off"
                    placeholder={braveKeySet ? t("ui.22baa5f106b466b0") : t("ui.005bb5eac6700550")}
                    value={braveKeyInput}
                    disabled={!loaded || savingKey}
                    onChange={(e) => setBraveKeyInput(e.target.value)}
                  />
                  <Button
                    type="button"
                    onClick={saveBraveKey}
                    disabled={!loaded || savingKey || braveKeyInput.trim() === ""}
                  >
                    guardar </Button>
                </div>
                {braveNeedsKey && (
                  <p className="text-xs text-amber-500">
                     {t("ui.c14b32626d869e4c")} </p>
                )}
                <p className="text-muted-foreground text-xs">
                   {t("ui.7b847ff34b1ee29d")} </p>
              </div>
            )}

            {webSearch && backend === "tavily" && (
              <div className="flex flex-col gap-2">
                <Label htmlFor="tavily-key" className="text-sm font-normal text-muted-foreground">
                  Tavily Search API Key
                  {tavilyKeySet && <span className="ml-2 text-xs text-emerald-500">{t("ui.54a0560489f8d8ab")}</span>}
                </Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="tavily-key"
                    type="password"
                    autoComplete="off"
                    placeholder={tavilyKeySet ? t("ui.22baa5f106b466b0") : t("ui.9b33f6aa6b6df084")}
                    value={tavilyKeyInput}
                    disabled={!loaded || savingTavilyKey}
                    onChange={(e) => setTavilyKeyInput(e.target.value)}
                  />
                  <Button
                    type="button"
                    onClick={saveTavilyKey}
                    disabled={!loaded || savingTavilyKey || tavilyKeyInput.trim() === ""}
                  >
                    guardar </Button>
                </div>
                {webSearch && backend === "tavily" && !tavilyKeySet && (
                  <p className="text-xs text-amber-500">
                     {t("ui.1b0c5c86462a68ef")} </p>
                )}
                <p className="text-muted-foreground text-xs">{t("ui.8441809101c2992d")}</p>
              </div>
            )}

            {webSearch && (
              <div className="flex flex-col gap-2">
                <Label htmlFor="ws-proxy" className="text-sm font-normal text-muted-foreground">
                   {t("ui.bf6a56d69e9e4ea0")} </Label>
                <div className="flex items-center gap-2">
                  <Input
                    id="ws-proxy"
                    autoComplete="off"
                    placeholder={t("ui.f975fd378f9e9ba0")}
                    value={proxyInput}
                    disabled={!loaded || savingProxy}
                    onChange={(e) => setProxyInput(e.target.value)}
                  />
                  <Button type="button" onClick={saveProxy} disabled={!loaded || savingProxy}>
                    guardar </Button>
                </div>
                <p className="text-muted-foreground text-xs">
                   {t("ui.b6607088b1d46053")} </p>
              </div>
            )}

            {webSearch && (
              <div className="flex items-center justify-between gap-4 border-t pt-4">
                <p className="text-muted-foreground text-xs">
                   {t("ui.b0ea1021180cbc39")} </p>
                <Button
                  type="button"
                  variant="outline"
                  onClick={runTest}
                  disabled={!loaded || testing}
                  className="shrink-0"
                >
                  {testing ? t("ui.a3da3a488d2e46b0") : t("ui.0f8b632a15bac08a")}
                </Button>
              </div>
            )}
          </CardContent>
        </Card>

        <Card className="mb-4 break-inside-avoid md:mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <RadioTowerIcon className="size-4" />
               {t("ui.fc785a371037d056")} </CardTitle>
            <CardDescription>
               {t("ui.5f2afd3490042aa8")} <b>script</b>  {t("ui.ae21f2da93d573c3")} </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <div className="flex items-center gap-2">
              <Input
                className="font-mono text-sm"
                placeholder={t("ui.892d0ba9e204c6d6")}
                value={pyInterp}
                disabled={!loaded || saving}
                onChange={(e) => setPyInterp(e.target.value)}
              />
              <Button variant="outline" onClick={detectPython} disabled={!loaded || saving}>
                 {t("ui.2b1dc566bd63833f")} </Button>
              <Button onClick={savePython} disabled={!loaded || saving}>
                guardar </Button>
            </div>
          </CardContent>
        </Card>

        <Card className="mb-4 break-inside-avoid md:mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <CpuIcon className="size-4" />
               {t("ui.ee6ee2c0fd0bb27c")} </CardTitle>
            <CardDescription>
               {t("ui.001eae2bea54319e")} <b>{t("ui.0018407837d7ba2c")}</b>{t("ui.78490d5ac9bcac8c")} </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <div className="flex items-center gap-2">
              <Input
                type="number"
                min={1}
                className="w-32 font-mono text-sm"
                placeholder="3"
                value={workers}
                disabled={!loaded || savingWorkers}
                onChange={(e) => setWorkers(e.target.value)}
              />
              <Button onClick={saveWorkers} disabled={!loaded || savingWorkers}>
                guardar </Button>
            </div>
          </CardContent>
        </Card>

        <Card className="mb-4 break-inside-avoid md:mb-6">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <KeyboardIcon className="size-4" />
               {t("ui.c3cf7ee3d00c02ca")} </CardTitle>
            <CardDescription>
               {t("ui.6ed75089327579b2")} <br />
               {t("ui.9f0591c096cfc216")}<b>{t("ui.c6aaab92c327e564")}</b>{t("ui.6177d3b89cb32f78")} </CardDescription>
          </CardHeader>
          <CardContent className="flex items-center justify-between gap-4">
            <Label htmlFor="chat-send-mode" className="text-sm font-normal text-muted-foreground">
               {t("ui.5ff320ca72c8d5aa")} </Label>
            <Select value={sendMode} onValueChange={(v) => setChatSendMode(v as ChatSendMode)}>
              <SelectTrigger id="chat-send-mode" className="w-72">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {CHAT_SEND_MODE_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
