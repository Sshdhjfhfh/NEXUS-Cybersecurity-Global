"use client";

import { t } from '@/lib/i18n';

import * as React from "react";
import { toast } from "sonner";
import { EyeIcon, GitCompareIcon, InfoIcon, PencilIcon, RotateCcwIcon, SaveIcon, Trash2Icon, XIcon } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Markdown } from "@/components/markdown";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import type { Agent, AgentDetail, AgentTrigger, MCPServer, PromptVar, PromptVersion, Settings, SkillItem, Tool } from "@/lib/types";

// Traffic tools are host tools gated by the global captura de tráfico switch: bindable, but
// only usable when capture is on. Keep this list in sync with traffic.SeedToolMetas.
const TRAFFIC_TOOL_KEYS = new Set(["traffic_search", "traffic_get"]);

// AgentEditor is the tabbed editor for one agent, used inside the agents-page
// drawer (and reused full-page for deep links). Tabs: Configuración y palabras de aviso / MCP / Skill /
// Tools. Config + prompt save as before; visibility + tool bindings toggle live.
export function AgentEditor({ agentKey, onSaved }: { agentKey: string; onSaved?: () => void }) {
  const [detail, setDetail] = React.useState<AgentDetail | null>(null);
  const [versions, setVersions] = React.useState<PromptVersion[]>([]);
  const [variables, setVariables] = React.useState<PromptVar[]>([]);
  const [mcp, setMcp] = React.useState<MCPServer[]>([]);
  const [skills, setSkills] = React.useState<SkillItem[]>([]);
  const [tools, setTools] = React.useState<Tool[]>([]);
  const [loaded, setLoaded] = React.useState(false);
  const [viewVer, setViewVer] = React.useState<PromptVersion | null>(null);
  const [diffVer, setDiffVer] = React.useState<PromptVersion | null>(null);

  const [prompt, setPrompt] = React.useState("");
  const [mcpVisible, setMcpVisible] = React.useState<number[]>([]);
  const [skillVisible, setSkillVisible] = React.useState<string[]>([]);
  const [preview, setPreview] = React.useState("");
  const [maxTurns, setMaxTurns] = React.useState("0");
  const [runSecs, setRunSecs] = React.useState("600");
  // "" = seguir(no vinculado)；De lo contrario profile id cadena
  const [llmProfileId, setLlmProfileId] = React.useState("");
  const [llmProfiles, setLlmProfiles] = React.useState<NonNullable<AgentDetail["llm_profiles"]>>([]);
  const [webSearch, setWebSearch] = React.useState(false);
  const [interactiveShell, setInteractiveShell] = React.useState(false);
  const [wrapup, setWrapup] = React.useState("");
  const [wrapupDefault, setWrapupDefault] = React.useState("");
  const [wrapupTurns, setWrapupTurns] = React.useState("0");
  const [wrapupTurnsDefault, setWrapupTurnsDefault] = React.useState(5);
  // Palabra final de tiempo de espera a nivel de tarea(solo worker/planner)
  const [ttSupported, setTtSupported] = React.useState(false);
  const [ttWrapup, setTtWrapup] = React.useState("");
  const [ttWrapupDefault, setTtWrapupDefault] = React.useState("");
  const [ttTurns, setTtTurns] = React.useState("0");
  const [ttTurnsDefault, setTtTurnsDefault] = React.useState(5);
  const [settings, setSettings] = React.useState<Settings | null>(null);

  React.useEffect(() => {
    api.mcpServers().then(setMcp).catch(() => { /* optional catalog: the editor still works with an empty list */ });
    api.skills().then(setSkills).catch(() => { /* optional catalog: the editor still works with an empty list */ });
    api.tools().then(setTools).catch(() => { /* optional catalog: the editor still works with an empty list */ });
    api.settings().then(setSettings).catch(() => { /* gates fall back to their defaults */ });
  }, []);
  // global gates: traffic tools need captura de tráfico, web search needs the master switch.
  const captureOn = !!settings?.traffic_capture;
  const webSearchGlobalOn = !!settings?.web_search_enabled;

  const reload = React.useCallback(() => {
    api
      .getAgent(agentKey)
      .then((d) => {
        setDetail(d);
        setPrompt(d.prompt ?? "");
        setVariables(d.variables ?? []);
        setVersions(d.versions ?? []);
        setMcpVisible(d.visibility?.mcp ?? []);
        setSkillVisible(d.visibility?.skill ?? []);
        setMaxTurns(String(d.agent?.max_turns ?? 0));
        setRunSecs(String(d.agent?.run_seconds ?? 600));
        setLlmProfileId(d.agent?.llm_profile_id != null ? String(d.agent.llm_profile_id) : "");
        setLlmProfiles(d.llm_profiles ?? []);
        setWebSearch(!!d.agent?.web_search);
        setInteractiveShell(!!d.agent?.interactive_shell);
        setWrapup(d.wrapup_prompt ?? "");
        setWrapupDefault(d.wrapup_default ?? "");
        setWrapupTurns(String(d.wrapup_max_turns ?? 0));
        setWrapupTurnsDefault(d.wrapup_max_turns_default ?? 5);
        setTtSupported(!!d.task_timeout_wrapup_supported);
        setTtWrapup(d.task_timeout_wrapup_prompt ?? "");
        setTtWrapupDefault(d.task_timeout_wrapup_default ?? "");
        setTtTurns(String(d.task_timeout_wrapup_max_turns ?? 0));
        setTtTurnsDefault(d.task_timeout_wrapup_max_turns_default ?? 5);
      })
      .catch(() => setDetail(null))
      .finally(() => setLoaded(true));
  }, [agentKey]);
  React.useEffect(() => {
    reload();
  }, [reload]);

  async function doPreview() {
    try {
      const r = await api.previewAgentPrompt(agentKey, prompt);
      setPreview(r.error ? t("ui.01699728b45ef3ac") + r.error : r.rendered);
    } catch (e) {
      setPreview(t("ui.37acbe69270661e9") + (e as Error).message);
    }
  }
  async function savePrompt() {
    try {
      const r = await api.saveAgentPrompt(agentKey, prompt);
      toast.success(`Guardado como versión v${r.version}`);
      reload();
      onSaved?.();
    } catch (e) {
      toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message);
    }
  }
  async function resetPrompt() {
    try {
      const r = await api.resetAgentPrompt(agentKey);
      toast.success(`Restaurado al valor predeterminado incorporado (v${r.version}）`);
      reload();
    } catch (e) {
      toast.error(t("ui.e144eee698dc30eb") + (e as Error).message);
    }
  }
  async function saveWrapup() {
    try {
      const turns = Math.max(0, Math.floor(Number(wrapupTurns) || 0));
      await api.saveAgentWrapup(agentKey, wrapup, turns);
      toast.success(wrapup.trim() || turns > 0 ? t("ui.6b1237127c50a675") : t("ui.35621fd92d3a47f9"));
      reload();
    } catch (e) {
      toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message);
    }
  }
  async function resetWrapup() {
    try {
      await api.resetAgentWrapup(agentKey);
      toast.success(t("ui.023b09446077d427"));
      reload();
    } catch (e) {
      toast.error(t("ui.e144eee698dc30eb") + (e as Error).message);
    }
  }
  async function saveTaskTimeoutWrapup() {
    try {
      const turns = Math.max(0, Math.floor(Number(ttTurns) || 0));
      await api.saveAgentTaskTimeoutWrapup(agentKey, ttWrapup, turns);
      toast.success(t("ui.140c990aace7efde"));
      reload();
    } catch (e) {
      toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message);
    }
  }
  async function resetTaskTimeoutWrapup() {
    try {
      await api.resetAgentTaskTimeoutWrapup(agentKey);
      toast.success(t("ui.023b09446077d427"));
      reload();
    } catch (e) {
      toast.error(t("ui.e144eee698dc30eb") + (e as Error).message);
    }
  }
  async function saveConfig() {
    try {
      // Enviar sólo esto agent campos mostrados reales, evitar elementos no mostrados(si goals de max_turns)sobrescribir el valor predeterminado。
      const patch: Parameters<typeof api.saveAgentConfig>[1] = {
        llm_profile_id: llmProfileId === "" ? null : Number(llmProfileId),
      };
      if (showConfig) {
        patch.max_turns = Math.max(0, Math.floor(Number(maxTurns) || 0));
        patch.run_seconds = Math.max(0, Math.floor(Number(runSecs) || 0));
      }
      if (showWebSearch) patch.web_search = webSearch;
      if (showInteractiveShell) patch.interactive_shell = interactiveShell;
      await api.saveAgentConfig(agentKey, patch);
      toast.success(t("ui.00901a097adacc4d"));
      reload();
    } catch (e) {
      toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message);
    }
  }
  // applyVis optimistically updates, persists, and toasts success/failure. On
  // failure it reverts to the prior selection so the UI never lies about state.
  async function applyVis(nextMcp: number[], nextSkill: string[], okMsg: string) {
    const prevMcp = mcpVisible;
    const prevSkill = skillVisible;
    setMcpVisible(nextMcp);
    setSkillVisible(nextSkill);
    try {
      await api.setAgentVisibility(agentKey, nextMcp, nextSkill);
      toast.success(okMsg);
      onSaved?.(); // refresh the list so the card's MCP/Skill counts stay in sync
    } catch (e) {
      setMcpVisible(prevMcp);
      setSkillVisible(prevSkill);
      toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message);
    }
  }
  function toggleMcp(id: number) {
    const on = mcpVisible.includes(id);
    const name = mcp.find((m) => m.id === id)?.name ?? String(id);
    void applyVis(
      on ? mcpVisible.filter((x) => x !== id) : [...mcpVisible, id],
      skillVisible,
      `${on ? t("ui.f407384e2f304cf0") : t("ui.56f90b9199cea3b0")} MCP「${name}"Visible`,
    );
  }
  function toggleSkill(name: string) {
    const on = skillVisible.includes(name);
    void applyVis(
      mcpVisible,
      on ? skillVisible.filter((x) => x !== name) : [...skillVisible, name],
      `${on ? t("ui.f407384e2f304cf0") : t("ui.56f90b9199cea3b0")} Skill「${name}"Visible`,
    );
  }
  async function toggleTool(i18nItem: Tool) {
    const on = i18nItem.agents.includes(agentKey);
    const nextAgents = on ? i18nItem.agents.filter((k) => k !== agentKey) : [...i18nItem.agents, agentKey];
    // optimistic update
    setTools((ts) => ts.map((x) => (x.key === i18nItem.key ? { ...x, agents: nextAgents } : x)));
    try {
      await api.saveTool(i18nItem.key, {
        description: i18nItem.description,
        schema: i18nItem.schema,
        agents: nextAgents,
        enabled: i18nItem.enabled,
      });
      toast.success(`${on ? t("ui.78765d041c2b607c") : t("ui.bfe1df5580a0f59e")}Herramientas${i18nItem.key}」`);
      onSaved?.(); // refresh the list so the card's Herramientas count stays in sync
    } catch (e) {
      toast.error(t("ui.9506df170156d591") + (e as Error).message);
      reload();
      api.tools().then(setTools).catch(() => { /* optional catalog: the editor still works with an empty list */ });
    }
  }

  if (loaded && !detail) {
    return <div className="text-muted-foreground p-6 text-center text-sm">{t("ui.65cbf0dae8a90f9c")}{agentKey}</div>;
  }
  // config is meaningless for the conversational main agent and the fixed-budget
  // goals decomposer; every other agent (workers, custom assistants) honors it.
  const showConfig = agentKey !== "mainagent" && agentKey !== "goals";
  // web search applies to every conversational/executing agent except the one-shot
  // goals decomposer; it's gated by the global master switch.
  const showWebSearch = agentKey !== "goals";
  // interactive shell (duradero PTY familia de herramientas de conversación) Dividir de manera similar goals afuera agent abierto;Sin puerta global。
  const showInteractiveShell = agentKey !== "goals";
  // cada uno agent(Incluyendo goals/mainagent)Todos corriendo a alguna parte LLM arriba,Por lo tanto, el enlace del ”modelo predeterminado” es para todos agent abierto。
  const showLLM = true;
  // triggers (P3) only attach to custom agents.
  const isCustom = !!detail && !detail.agent?.builtin;

  return (
    <Tabs defaultValue="prompt" className="flex min-h-0 flex-1 flex-col">
      <TabsList className="mx-4 mt-2 w-fit">
        <TabsTrigger value="prompt">{t("ui.01f026d5841760f8")}</TabsTrigger>
        <TabsTrigger value="wrapup">{t("ui.9d52605842992f5f")}</TabsTrigger>
        <TabsTrigger value="mcp">MCP</TabsTrigger>
        <TabsTrigger value="skill">Skill</TabsTrigger>
        <TabsTrigger value="tools">Tools</TabsTrigger>
        {isCustom && <TabsTrigger value="triggers">{t("ui.b8734b28075b3344")}</TabsTrigger>}
      </TabsList>

      {/* Configuración + palabra rápida */}
      <TabsContent value="prompt" className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">
        <div className="grid gap-4">
          {(showLLM || showConfig || showWebSearch || showInteractiveShell) && (
            <div className="grid gap-3 rounded-md border p-3">
              {showLLM && (
                <div className="grid gap-1.5">
                  <Label htmlFor="llm-profile" className="text-xs">{t("ui.da42658d54830f25")}</Label>
                  <div className="flex flex-wrap items-center gap-3">
                    <Select value={llmProfileId || "__follow__"} onValueChange={(v) => setLlmProfileId(v === "__follow__" ? "" : v)}>
                      <SelectTrigger id="llm-profile" className="h-8 w-72">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="__follow__">{t("ui.70ffcf1646bc70ab")}</SelectItem>
                        {llmProfiles.map((p) => (
                          <SelectItem key={p.id} value={String(p.id)}>
                            {p.name}（{p.model}）{p.is_default ? t("ui.d1c51289de8beaa5") : ""}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <span className="text-muted-foreground max-w-md text-xs">
                      Vincule la configuración fija de LLM al Agente (haga clic en "Guardar configuración" para que surta efecto). Prioridad: Enlace de agente &gt; Especificación de tarea/sesión &gt; Activación global. </span>
                  </div>
                </div>
              )}
              <div className="flex flex-wrap items-end gap-3">
                {showConfig && (
                  <>
                    <div className="grid gap-1.5">
                      <Label htmlFor="max-turns" className="text-xs">{t("ui.297a9a18bbc9b625")}</Label>
                      <Input id="max-turns" type="number" min={0} className="h-8 w-32"
                        value={maxTurns} onChange={(e) => setMaxTurns(e.target.value)} />
                    </div>
                    <div className="grid gap-1.5">
                      <Label htmlFor="run-seconds" className="text-xs">{t("ui.9a4256e111e36c30")}</Label>
                      <Input id="run-seconds" type="number" min={0} className="h-8 w-32"
                        value={runSecs} onChange={(e) => setRunSecs(e.target.value)} />
                    </div>
                  </>
                )}
                <Button size="sm" variant="outline" onClick={saveConfig}>
                  <SaveIcon />  {t("ui.c753c07393e625c4")} </Button>
              </div>
              {showWebSearch && (
                <div className="flex items-center gap-3 border-t pt-3">
                  <Switch
                    id="web-search"
                    checked={webSearch}
                    disabled={!webSearchGlobalOn}
                    onCheckedChange={setWebSearch}
                  />
                  <div className="grid gap-0.5">
                    <Label htmlFor="web-search" className="text-sm">{t("ui.faad93a241392801")}</Label>
                    <span className="text-muted-foreground text-xs">
                      {webSearchGlobalOn
                        ? t("ui.c4ccad0f97c29381")
                        : t("ui.4444cc733137eb1d")}
                    </span>
                  </div>
                </div>
              )}
              {showInteractiveShell && (
                <div className="flex items-center gap-3 border-t pt-3">
                  <Switch
                    id="interactive-shell"
                    checked={interactiveShell}
                    onCheckedChange={setInteractiveShell}
                  />
                  <div className="grid gap-0.5">
                    <Label htmlFor="interactive-shell" className="text-sm">{t("ui.870afbb3f57f7ef9")}</Label>
                    <span className="text-muted-foreground text-xs">
                       {t("ui.682ef2caf86068b0")} </span>
                  </div>
                </div>
              )}
            </div>
          )}

          <div className="grid gap-2">
            <Label className="text-muted-foreground text-xs">{t("ui.516d331a97accfe4")}</Label>
            <div className="flex flex-wrap gap-2">
              {variables.map((v) => (
                <Tooltip key={v.name}>
                  <TooltipTrigger asChild>
                    <button type="button" onClick={() => setPrompt((p) => `${p}{{.${v.name}}}`)}
                      className="hover:bg-muted inline-flex items-center gap-1 rounded-md border bg-muted/40 px-2 py-1 font-mono text-xs">
                      {`{{.${v.name}}}`}
                      <Badge variant="secondary" className="px-1 py-0 text-[10px]">{v.source}</Badge>
                    </button>
                  </TooltipTrigger>
                  <TooltipContent className="max-w-xs">
                    <p className="font-medium">{v.description}</p>
                    <p className="text-muted-foreground mt-1">{t("ui.265eadbe8414b0f9")}{v.example}</p>
                  </TooltipContent>
                </Tooltip>
              ))}
              {variables.length === 0 && <span className="text-muted-foreground text-xs">{t("ui.cfc0e0abeb9128cb")}</span>}
            </div>
          </div>

          <Textarea className="font-mono text-xs" rows={16} value={prompt}
            placeholder={t("ui.cd828a364209a241")} onChange={(e) => setPrompt(e.target.value)} />

          <div className="flex flex-wrap gap-2">
            <Dialog>
              <DialogTrigger asChild>
                <Button variant="outline" size="sm" onClick={doPreview}>
                  <EyeIcon />  {t("ui.237a4275ffc28f76")} </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-2xl">
                <DialogHeader>
                  <DialogTitle>{t("ui.237a4275ffc28f76")}</DialogTitle>
                  <DialogDescription>{t("ui.c8f7ce276036333f")} {`{{.Var}}`}  {t("ui.33da465f69a1105d")}</DialogDescription>
                </DialogHeader>
                <div className="max-h-[60vh] overflow-auto rounded-md border bg-muted/30 p-3">
                  <Markdown text={preview} />
                </div>
              </DialogContent>
            </Dialog>
            <Button size="sm" onClick={savePrompt}>
              <SaveIcon />  {t("ui.d0120cfe84918955")} </Button>
            <Button variant="outline" size="sm" onClick={resetPrompt}>
              <RotateCcwIcon />  {t("ui.c48a43ea113759e3")} </Button>
          </div>


          <Separator />
          <div className="grid gap-2">
            <Label className="text-muted-foreground text-xs">{t("ui.86bd5274f8c26bbe")}</Label>
            <ul className="grid gap-1">
              {versions.map((ver, i) => (
                <li key={ver.version} className="flex items-center gap-2 rounded-md px-1 py-0.5 text-xs hover:bg-muted/50">
                  <span className="font-mono shrink-0">v{ver.version}</span>
                  {i === 0 && <Badge variant="secondary" className="px-1.5 py-0 shrink-0">{t("ui.e5c6fde86910ded7")}</Badge>}
                  <span className="text-muted-foreground truncate flex-1">{ver.note}</span>
                  {ver.ts && (
                    <span className="text-muted-foreground/60 shrink-0 tabular-nums">
                      {new Date(ver.ts).toLocaleDateString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })}
                    </span>
                  )}
                  <Button variant="ghost" size="icon-sm" className="size-6 shrink-0" onClick={() => setViewVer(ver)}>
                    <EyeIcon className="size-3" />
                  </Button>
                  {i > 0 && versions[0] && (
                    <Button variant="ghost" size="icon-sm" className="size-6 shrink-0" onClick={() => setDiffVer(ver)}>
                      <GitCompareIcon className="size-3" />
                    </Button>
                  )}
                </li>
              ))}
              {versions.length === 0 && (
                <li className="text-muted-foreground text-xs">{t("ui.b4c40b9dd3ee747f")}</li>
              )}
            </ul>
          </div>

          {/* Ver versión dialog */}
          <Dialog open={!!viewVer} onOpenChange={(o) => { if (!o) setViewVer(null); }}>
            <DialogContent className="sm:max-w-2xl">
              <DialogHeader>
                <DialogTitle>
                  v{viewVer?.version}
                  {viewVer?.version === versions[0]?.version && (
                    <Badge variant="secondary" className="ml-2 px-1.5 py-0 align-middle">{t("ui.e5c6fde86910ded7")}</Badge>
                  )}
                </DialogTitle>
                <DialogDescription>
                  {viewVer?.note || t("ui.c3ceb515983ee344")}
                  {viewVer?.ts && (
                    <span className="ml-2 text-muted-foreground/60">
                      {new Date(viewVer.ts).toLocaleString("zh-CN")}
                    </span>
                  )}
                </DialogDescription>
              </DialogHeader>
              <pre className="bg-muted max-h-[55vh] overflow-auto whitespace-pre-wrap rounded-md p-3 font-mono text-xs">
                {viewVer?.template_text || t("ui.43a5359157cf2a94")}
              </pre>
              <div className="flex gap-2 justify-end">
                {viewVer && viewVer.version !== versions[0]?.version && (
                  <Button variant="outline" size="sm" onClick={() => {
                    if (viewVer) { setDiffVer(viewVer); setViewVer(null); }
                  }}>
                    <GitCompareIcon className="mr-1 size-3.5" />  {t("ui.6f843474065773ba")} </Button>
                )}
                <Button size="sm" onClick={() => {
                  if (viewVer) { setPrompt(viewVer.template_text); setViewVer(null); toast.success(`v cargado${viewVer.version}Vaya al editor, confirme y haga clic en "Guardar como nueva versión"`); }
                }}>
                   {t("ui.0281d5d04de17049")} </Button>
              </div>
            </DialogContent>
          </Dialog>

          {/* comparación de versiones dialog */}
          <Dialog open={!!diffVer} onOpenChange={(o) => { if (!o) setDiffVer(null); }}>
            <DialogContent className="sm:max-w-3xl">
              <DialogHeader>
                <DialogTitle>{t("ui.4013b7abc1609421")}{diffVer?.version} → v{versions[0]?.version}{t("ui.092cb444b688ec21")}</DialogTitle>
                <DialogDescription>
                  <span className="inline-flex items-center gap-3 text-xs">
                    <span className="rounded bg-red-500/15 px-1.5 py-0.5 text-red-600 dark:text-red-400">{t("ui.ae0c6b5d9a48d9ae")}</span>
                    <span className="rounded bg-green-500/15 px-1.5 py-0.5 text-green-600 dark:text-green-400">{t("ui.7aaf1aab7b562ebe")}</span>
                  </span>
                </DialogDescription>
              </DialogHeader>
              <DiffView oldText={diffVer?.template_text ?? ""} newText={versions[0]?.template_text ?? ""} />
            </DialogContent>
          </Dialog>
        </div>
      </TabsContent>

      {/* Palabra de mensaje de cierre */}
      <TabsContent value="wrapup" className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">
        <div className="grid gap-3">
          <p className="text-muted-foreground text-xs leading-relaxed">
             {t("ui.3dc2ceef23035c18")}<b>{t("ui.be448e0f234d8960")}</b>{"o"}<b>{t("ui.150156354f6ed675")}</b>{t("ui.822c3d319bf37c7c")} </p>
          <div className="flex items-center gap-2">
            <Label className="text-xs">{t("ui.00f4d42f9fce6bac")}</Label>
            {wrapup.trim() ? (
              <Badge variant="secondary" className="px-1.5 py-0">{t("ui.5f2afd3490042aa8")}</Badge>
            ) : (
              <Badge variant="outline" className="px-1.5 py-0">{t("ui.c7055d1619dfd6dc")}</Badge>
            )}
          </div>
          <Textarea
            className="font-mono text-xs"
            rows={10}
            value={wrapup}
            placeholder={wrapupDefault || t("ui.dc0c82489f439f82")}
            onChange={(e) => setWrapup(e.target.value)}
          />
          <div className="grid gap-1.5">
            <Label htmlFor="wrapup-turns" className="text-xs">
               {t("ui.b8b7449cdd6c9e32")} {wrapupTurnsDefault}  {t("ui.4f4659978ac89c7d")} </Label>
            <Input id="wrapup-turns" type="number" min={0} className="h-8 w-32"
              value={wrapupTurns} onChange={(e) => setWrapupTurns(e.target.value)} />
          </div>
          <div className="flex gap-2">
            <Button size="sm" onClick={saveWrapup}>{t("common.save")}</Button>
            <Button size="sm" variant="outline" onClick={resetWrapup}>{t("ui.c48a43ea113759e3")}</Button>
          </div>
          {wrapupDefault && (
            <>
              <Separator />
              <div className="grid gap-1.5">
                <Label className="text-muted-foreground text-xs">{t("ui.1e73d4b67658783e")}</Label>
                <pre className="text-muted-foreground max-h-40 overflow-y-auto rounded-md border bg-muted/30 p-2 text-xs whitespace-pre-wrap">
                  {wrapupDefault}
                </pre>
              </div>
            </>
          )}

          {ttSupported && (
            <>
              <Separator className="my-2" />
              <p className="text-muted-foreground text-xs leading-relaxed">
                <b>{t("ui.175358e6a4748e8d")}</b>{t("ui.adb5ef9aeaa01e06")}<b>{t("ui.32b92e1fd0e65732")}</b>{t("ui.dee785ddb58e51d5")}<b>{t("ui.5c4bd4026e925170")}</b>{t("ui.c856584a8eae2c6e")} </p>
              <div className="flex items-center gap-2">
                <Label className="text-xs">{t("ui.1e5df4aa6377d328")}</Label>
                {ttWrapup.trim() ? (
                  <Badge variant="secondary" className="px-1.5 py-0">{t("ui.5f2afd3490042aa8")}</Badge>
                ) : (
                  <Badge variant="outline" className="px-1.5 py-0">{t("ui.c7055d1619dfd6dc")}</Badge>
                )}
              </div>
              <Textarea
                className="font-mono text-xs"
                rows={10}
                value={ttWrapup}
                placeholder={ttWrapupDefault || t("ui.f4cfc406dfd8047e")}
                onChange={(e) => setTtWrapup(e.target.value)}
              />
              <div className="grid gap-1.5">
                <Label htmlFor="tt-turns" className="text-xs">
                   {t("ui.23652c67aac758f2")} {ttTurnsDefault}  {t("ui.4f4659978ac89c7d")} </Label>
                <Input id="tt-turns" type="number" min={0} className="h-8 w-32"
                  value={ttTurns} onChange={(e) => setTtTurns(e.target.value)} />
              </div>
              <div className="flex gap-2">
                <Button size="sm" onClick={saveTaskTimeoutWrapup}>{t("common.save")}</Button>
                <Button size="sm" variant="outline" onClick={resetTaskTimeoutWrapup}>{t("ui.c48a43ea113759e3")}</Button>
              </div>
              {ttWrapupDefault && (
                <div className="grid gap-1.5">
                  <Label className="text-muted-foreground text-xs">{t("ui.1e73d4b67658783e")}</Label>
                  <pre className="text-muted-foreground max-h-40 overflow-y-auto rounded-md border bg-muted/30 p-2 text-xs whitespace-pre-wrap">
                    {ttWrapupDefault}
                  </pre>
                </div>
              )}
            </>
          )}
        </div>
      </TabsContent>

      {/* MCP Visibilidad */}
      <TabsContent value="mcp" className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">
        <p className="text-muted-foreground mb-3 text-xs">{t("ui.abae5d3a6be1f8db")}</p>
        <div className="grid gap-2">
          {mcp.map((m) => (
            <label key={m.id} className="flex items-center gap-2 rounded-md border p-2 text-sm">
              <Checkbox checked={mcpVisible.includes(m.id)} onCheckedChange={() => toggleMcp(m.id)} />
              {m.name}
              <span className="text-muted-foreground ml-auto text-xs">{m.transport}</span>
            </label>
          ))}
          {mcp.length === 0 && <span className="text-muted-foreground text-xs">{t("ui.591a6b6cd9662a81")}</span>}
        </div>
      </TabsContent>

      {/* Skill Visibilidad */}
      <TabsContent value="skill" className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">
        <p className="text-muted-foreground mb-3 text-xs">{t("ui.e6c34a6b3e2f54c5")}</p>
        <div className="grid gap-2">
          {skills.map((s) => (
            <label key={s.name} className="flex items-center gap-2 rounded-md border p-2 text-sm">
              <Checkbox checked={skillVisible.includes(s.name)} onCheckedChange={() => toggleSkill(s.name)} />
              <span className="font-mono text-xs">{s.name}</span>
              {s.description && <span className="text-muted-foreground ml-auto truncate text-xs">{s.description}</span>}
            </label>
          ))}
          {skills.length === 0 && <span className="text-muted-foreground text-xs">{t("ui.bfc44de7d9f81c38")}</span>}
        </div>
      </TabsContent>

      {/* Tools vinculante */}
      <TabsContent value="tools" className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">
        <p className="text-muted-foreground mb-3 text-xs">{t("ui.4b0fdd25a83ffe08")}</p>
        <div className="grid gap-2">
          {tools.map((tool) => {
            const isTraffic = TRAFFIC_TOOL_KEYS.has(tool.key);
            const gated = isTraffic && !captureOn; // traffic tools need captura de tráfico on
            return (
              <label
                key={tool.key}
                className={cn(
                  "flex items-center gap-2 rounded-md border p-2 text-sm",
                  gated && "opacity-60",
                )}
              >
                <Checkbox
                  checked={tool.agents.includes(agentKey)}
                  disabled={gated}
                  onCheckedChange={() => toggleTool(tool)}
                />
                <span className="font-mono text-xs">{tool.key}</span>
                {isTraffic && (
                  <Badge variant="secondary" className="px-1 py-0 text-[9px]">{t("nav.traffic")}</Badge>
                )}
                {!tool.enabled && (
                  <Badge variant="outline" className="text-destructive px-1 py-0 text-[9px]">{t("ui.68de79aa60784d31")}</Badge>
                )}
                {gated ? (
                  <span className="text-muted-foreground ml-auto text-xs">{t("ui.d9db47f91e456409")}</span>
                ) : (
                  tool.description && (
                    <span className="text-muted-foreground ml-auto line-clamp-1 max-w-[55%] text-xs">
                      {tool.description}
                    </span>
                  )
                )}
              </label>
            );
          })}
          {tools.length === 0 && <span className="text-muted-foreground text-xs">{t("ui.b527d8feccae77a4")}</span>}
        </div>
      </TabsContent>

      {/* disparador(P3, Sólo personalizado agent) */}
      {isCustom && (
        <TabsContent value="triggers" className="min-h-0 flex-1 overflow-y-auto px-4 pb-4">
          <AgentTriggersTab agentKey={agentKey} agent={detail?.agent} />
        </TabsContent>
      )}
    </Tabs>
  );
}

// ---------- Diff helpers ----------

type DiffLine = { type: "same" | "add" | "del"; text: string };

function computeDiff(oldText: string, newText: string): DiffLine[] {
  const a = oldText.split("\n");
  const b = newText.split("\n");
  const m = a.length;
  const n = b.length;
  const dp: number[][] = Array.from({ length: m + 1 }, () => new Array(n + 1).fill(0));
  for (let i = 1; i <= m; i++)
    for (let j = 1; j <= n; j++)
      dp[i][j] = a[i - 1] === b[j - 1] ? dp[i - 1][j - 1] + 1 : Math.max(dp[i - 1][j], dp[i][j - 1]);
  const result: DiffLine[] = [];
  let i = m;
  let j = n;
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && a[i - 1] === b[j - 1]) {
      result.unshift({ type: "same", text: a[i - 1] });
      i--;
      j--;
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      result.unshift({ type: "add", text: b[j - 1] });
      j--;
    } else {
      result.unshift({ type: "del", text: a[i - 1] });
      i--;
    }
  }
  return result;
}

function DiffView({ oldText, newText }: { oldText: string; newText: string }) {
  const lines = React.useMemo(() => computeDiff(oldText, newText), [oldText, newText]);
  return (
    <pre className="max-h-[60vh] overflow-auto rounded-md border bg-muted/30 p-2 font-mono text-xs leading-5">
      {lines.map((l, idx) => (
        <div
          key={idx}
          className={cn(
            "whitespace-pre-wrap px-1",
            l.type === "del" && "bg-red-500/15 text-red-700 dark:text-red-400",
            l.type === "add" && "bg-green-500/15 text-green-700 dark:text-green-400",
            l.type === "same" && "text-muted-foreground",
          )}
        >
          <span className="select-none mr-1 opacity-50">{l.type === "del" ? "-" : l.type === "add" ? "+" : " "}</span>
          {l.text}
        </div>
      ))}
    </pre>
  );
}

// AgentTriggersTab manages a custom agent's P3 triggers: list + add + delete.
// Each trigger fires (tiempo/Descubrimientofinding/Objetivo cumplido/tiempo de espera de la tarea/Llamada de herramienta, múltiples selecciones posibles) → a new conversation runs
// in parallel with the base user message + auto context appended by the backend.
function AgentTriggersTab({ agentKey, agent }: { agentKey: string; agent?: Agent }) {
  const [triggers, setTriggers] = React.useState<AgentTrigger[]>([]);
  const [tools, setTools] = React.useState<Tool[]>([]);
  // estrategia de procesamiento posterior al disparo(cada agent);El valor inicial proviene de agent detail,guardar cambios。
  const [runMode, setRunMode] = React.useState<"serial" | "parallel">(agent?.trigger_run_mode ?? "serial");
  const [mergeMode, setMergeMode] = React.useState<"by_task" | "all" | "none">(agent?.trigger_merge_mode ?? "all");
  const [maxParallel, setMaxParallel] = React.useState(String(agent?.trigger_max_parallel ?? 5));
  React.useEffect(() => {
    setRunMode(agent?.trigger_run_mode ?? "serial");
    setMergeMode(agent?.trigger_merge_mode ?? "all");
    setMaxParallel(String(agent?.trigger_max_parallel ?? 5));
  }, [agent?.trigger_run_mode, agent?.trigger_merge_mode, agent?.trigger_max_parallel]);

  async function saveBehavior(patch: {
    trigger_run_mode?: "serial" | "parallel";
    trigger_merge_mode?: "by_task" | "all" | "none";
    trigger_max_parallel?: number;
  }) {
    try {
      await api.saveAgentConfig(agentKey, patch);
    } catch (e) {
      toast.error(t("ui.a09179782a41a469") + (e as Error).message);
    }
  }
  const [onInterval, setOnInterval] = React.useState(false);
  const [intervalSec, setIntervalSec] = React.useState("60");
  const [onFinding, setOnFinding] = React.useState(false);
  const [onGoalMet, setOnGoalMet] = React.useState(false);
  const [onTaskTimeout, setOnTaskTimeout] = React.useState(false);
  const [onToolCall, setOnToolCall] = React.useState(false);
  const [onTaskCreate, setOnTaskCreate] = React.useState(false);
  const [intervalMsg, setIntervalMsg] = React.useState("");
  const [findingMsg, setFindingMsg] = React.useState("");
  const [goalMsg, setGoalMsg] = React.useState("");
  const [taskTimeoutMsg, setTaskTimeoutMsg] = React.useState("");
  const [toolCallMsg, setToolCallMsg] = React.useState("");
  const [taskCreateMsg, setTaskCreateMsg] = React.useState("");
  const [toolNames, setToolNames] = React.useState<string[]>([]);
  const [saving, setSaving] = React.useState(false);
  // null = Nuevo modo; no null = Editando esto id Activador de。
  const [editingId, setEditingId] = React.useState<number | null>(null);

  const reload = React.useCallback(() => {
    api.agentTriggers(agentKey).then(setTriggers).catch(() => setTriggers([]));
  }, [agentKey]);
  React.useEffect(() => {
    reload();
  }, [reload]);
  React.useEffect(() => {
    api.tools().then(setTools).catch(() => setTools([]));
  }, []);

  function toggleTool(key: string) {
    setToolNames((prev) => (prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]));
  }

  // resetForm Borrar el formulario y volver al modo ”Agregar”。
  function resetForm() {
    setEditingId(null);
    setOnInterval(false);
    setIntervalSec("60");
    setOnFinding(false);
    setOnGoalMet(false);
    setOnTaskTimeout(false);
    setOnToolCall(false);
    setOnTaskCreate(false);
    setIntervalMsg("");
    setFindingMsg("");
    setGoalMsg("");
    setTaskTimeoutMsg("");
    setToolCallMsg("");
    setTaskCreateMsg("");
    setToolNames([]);
  }

  // startEdit complete un activador existente en el formulario,Ingrese al modo ”Editar”。
  function startEdit(t: AgentTrigger) {
    setEditingId(t.id);
    setOnInterval(t.interval_sec > 0);
    setIntervalSec(t.interval_sec > 0 ? String(t.interval_sec) : "60");
    setOnFinding(t.on_finding);
    setOnGoalMet(t.on_goal_met);
    setOnTaskTimeout(t.on_task_timeout);
    setOnToolCall(t.on_tool_call);
    setOnTaskCreate(t.on_task_create);
    setIntervalMsg(t.interval_message);
    setFindingMsg(t.finding_message);
    setGoalMsg(t.goal_message);
    setTaskTimeoutMsg(t.task_timeout_message);
    setToolCallMsg(t.tool_call_message);
    setTaskCreateMsg(t.task_create_message);
    setToolNames(t.tool_names ?? []);
  }

  // submit Según editingId Vaya a ”Agregar” o ”Guardar cambios”」;Mantenga este disparador habilitado mientras edita。
  async function submit() {
    const n = onInterval ? Math.max(1, Math.floor(Number(intervalSec) || 0)) : 0;
    if (n === 0 && !onFinding && !onGoalMet && !onTaskTimeout && !onToolCall && !onTaskCreate) {
      toast.error(t("ui.5fd88edee77ead85"));
      return;
    }
    if (onToolCall && toolNames.length === 0) {
      toast.error(t("ui.9b9d05f43996691b"));
      return;
    }
    const body = {
      interval_sec: n,
      on_finding: onFinding,
      on_goal_met: onGoalMet,
      on_task_timeout: onTaskTimeout,
      on_tool_call: onToolCall,
      on_task_create: onTaskCreate,
      interval_message: intervalMsg.trim(),
      finding_message: findingMsg.trim(),
      goal_message: goalMsg.trim(),
      task_timeout_message: taskTimeoutMsg.trim(),
      tool_call_message: toolCallMsg.trim(),
      task_create_message: taskCreateMsg.trim(),
      tool_names: onToolCall ? toolNames : [],
    };
    setSaving(true);
    try {
      if (editingId != null) {
        const cur = triggers.find((x) => x.id === editingId);
        await api.updateTrigger(editingId, { ...body, enabled: cur?.enabled ?? true });
        toast.success(t("ui.a184979c9c67e174"));
      } else {
        await api.createTrigger(agentKey, { ...body, enabled: true });
        toast.success(t("ui.5f3d53b81cfcd120"));
      }
      resetForm();
      reload();
    } catch (e) {
      toast.error((editingId != null ? t("ui.d97c6497d08cd1ab") : t("ui.bc6ac348470494fd")) + (e as Error).message);
    } finally {
      setSaving(false);
    }
  }
  async function toggleEnabled(i18nItem: AgentTrigger) {
    try {
      await api.updateTrigger(i18nItem.id, {
        enabled: !i18nItem.enabled,
        interval_sec: i18nItem.interval_sec,
        on_finding: i18nItem.on_finding,
        on_goal_met: i18nItem.on_goal_met,
        on_task_timeout: i18nItem.on_task_timeout,
        on_tool_call: i18nItem.on_tool_call,
        on_task_create: i18nItem.on_task_create,
        interval_message: i18nItem.interval_message,
        finding_message: i18nItem.finding_message,
        goal_message: i18nItem.goal_message,
        task_timeout_message: i18nItem.task_timeout_message,
        tool_call_message: i18nItem.tool_call_message,
        task_create_message: i18nItem.task_create_message,
        tool_names: i18nItem.tool_names,
      });
      reload();
    } catch (e) {
      toast.error(t("ui.d97c6497d08cd1ab") + (e as Error).message);
    }
  }
  async function del(id: number) {
    try {
      await api.deleteTrigger(id);
      if (editingId === id) resetForm();
      reload();
    } catch (e) {
      toast.error(t("ui.ac60bde833d4bb98") + (e as Error).message);
    }
  }

  function condLabel(i18nItem: AgentTrigger): string {
    const parts: string[] = [];
    if (i18nItem.interval_sec > 0) parts.push(`cada ${i18nItem.interval_sec}s`);
    if (i18nItem.on_finding) parts.push(t("ui.fc319b4d0585e6e0"));
    if (i18nItem.on_goal_met) parts.push(t("ui.8820f5368c27f9ef"));
    if (i18nItem.on_task_timeout) parts.push(t("ui.b9f6f6c53d77eb8a"));
    if (i18nItem.on_tool_call) parts.push(`Llamada a herramienta (${i18nItem.tool_names.length})`);
    if (i18nItem.on_task_create) parts.push(t("ui.a350194c34061de0"));
    return parts.join(" · ") || t("ui.7102efd66446febc");
  }

  return (
    <div className="grid gap-4">
      <p className="text-muted-foreground text-xs">
         {t("ui.ea9cd77f2a83b601")}<b>{t("ui.660734655c8e940b")}</b>{t("ui.6d4505310d8bcf77")} </p>

      {/* estrategia de procesamiento posterior al disparo */}
      <div className="grid gap-3 rounded-md border p-3">
        <Label className="text-muted-foreground text-xs">{t("ui.643716438b9155d2")}</Label>
        <div className="flex flex-wrap items-center gap-4">
          <div className="grid gap-1">
            <Label className="text-xs">{t("ui.cf04341b3b7c21de")}</Label>
            <Select
              value={runMode}
              onValueChange={(v) => {
                const rm = v as "serial" | "parallel";
                setRunMode(rm);
                void saveBehavior({ trigger_run_mode: rm });
              }}
            >
              <SelectTrigger size="sm" className="h-8 w-40">
                <SelectValue />
              </SelectTrigger>
              <SelectContent position="popper">
                <SelectItem value="serial">{t("ui.3427e3b19654ef45")}</SelectItem>
                <SelectItem value="parallel">{t("ui.1b0a17a1b6624598")}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="grid gap-1">
            <Label className="text-xs">{t("ui.04276fd40efac31e")}</Label>
            <Select
              value={mergeMode}
              disabled={runMode === "parallel"}
              onValueChange={(v) => {
                const mm = v as "by_task" | "all" | "none";
                setMergeMode(mm);
                void saveBehavior({ trigger_merge_mode: mm });
              }}
            >
              <SelectTrigger size="sm" className="h-8 w-44">
                <SelectValue />
              </SelectTrigger>
              <SelectContent position="popper">
                <SelectItem value="by_task">{t("ui.0f8994a403aec0f6")}</SelectItem>
                <SelectItem value="all">{t("ui.e1c79a61d7aa3286")}</SelectItem>
                <SelectItem value="none">{t("ui.ab6b5708a86f21ed")}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {runMode === "parallel" && (
            <div className="grid gap-1">
              <Label htmlFor="tr-maxpar" className="text-xs">{t("ui.9bb927201d3c2866")}</Label>
              <Input
                id="tr-maxpar"
                type="number"
                min={0}
                className="h-8 w-28"
                value={maxParallel}
                onChange={(e) => setMaxParallel(e.target.value)}
                onBlur={() => {
                  const n = Math.max(0, Math.floor(Number(maxParallel) || 0));
                  setMaxParallel(String(n));
                  void saveBehavior({ trigger_max_parallel: n });
                }}
              />
            </div>
          )}
        </div>
        <p className="text-muted-foreground text-xs">
          {runMode === "parallel"
            ? t("ui.96adf6dfceddc90b")
            : mergeMode === "by_task"
              ? t("ui.979cdf690dc833cd")
              : mergeMode === "all"
                ? t("ui.57792f153a842cb5")
                : t("ui.4d0c799dbda3f306")}
        </p>
      </div>

      {/* nuevo / Editar disparador */}
      <div className="grid gap-3 rounded-md border p-3">
        <Label className="text-muted-foreground text-xs">
          {editingId != null
            ? `Editar disparador #${editingId}(Haga clic en "Guardar cambios" después de realizar cambios)`
            : t("ui.dfd5c660b67a5982")}
        </Label>

        {/* tiempo */}
        <div className="grid gap-1.5">
          <label className="flex items-center gap-2 text-sm">
            <Checkbox checked={onInterval} onCheckedChange={(v) => setOnInterval(!!v)} />  {t("ui.eb3856a2d5557371")} </label>
          {onInterval && (
            <div className="grid gap-1.5">
              <div className="flex items-center gap-2">
                <Label htmlFor="tr-interval" className="text-xs">{t("ui.74767fb71638182a")}</Label>
                <Input id="tr-interval" type="number" min={1} className="h-8 w-24"
                  value={intervalSec} onChange={(e) => setIntervalSec(e.target.value)} />
                <span className="text-muted-foreground text-xs">{t("ui.324ea98f7cd80cd7")}</span>
              </div>
              <Textarea className="text-xs" rows={2} value={intervalMsg}
                placeholder={t("ui.294f5f9f857da2af")} onChange={(e) => setIntervalMsg(e.target.value)} />
            </div>
          )}
        </div>

        {/* finding */}
        <div className="grid gap-1.5">
          <label className="flex items-center gap-2 text-sm">
            <Checkbox checked={onFinding} onCheckedChange={(v) => setOnFinding(!!v)} />  {t("ui.744310650cfdb143")} </label>
          {onFinding && (
            <Textarea className="text-xs" rows={2} value={findingMsg}
              placeholder={t("ui.c08d29dafd7de9b6")} onChange={(e) => setFindingMsg(e.target.value)} />
          )}
        </div>

        {/* Objetivo cumplido */}
        <div className="grid gap-1.5">
          <label className="flex items-center gap-2 text-sm">
            <Checkbox checked={onGoalMet} onCheckedChange={(v) => setOnGoalMet(!!v)} />  {t("ui.fb7c7b83fefabc9e")} </label>
          {onGoalMet && (
            <Textarea className="text-xs" rows={2} value={goalMsg}
              placeholder={t("ui.1a5fb30e3e92f553")} onChange={(e) => setGoalMsg(e.target.value)} />
          )}
        </div>

        {/* tiempo de espera de la tarea */}
        <div className="grid gap-1.5">
          <label className="flex items-center gap-2 text-sm">
            <Checkbox checked={onTaskTimeout} onCheckedChange={(v) => setOnTaskTimeout(!!v)} />  {t("ui.01e0e6b2f0727904")} </label>
          {onTaskTimeout && (
            <Textarea className="text-xs" rows={2} value={taskTimeoutMsg}
              placeholder={t("ui.b9ae1df43cad9038")} onChange={(e) => setTaskTimeoutMsg(e.target.value)} />
          )}
        </div>

        {/* llamada de herramienta */}
        <div className="grid gap-1.5">
          <label className="flex items-center gap-2 text-sm">
            <Checkbox checked={onToolCall} onCheckedChange={(v) => setOnToolCall(!!v)} />  {t("ui.b3e8b74fcc976e16")} </label>
          {onToolCall && (
            <div className="grid gap-1.5">
              <div className="text-muted-foreground text-xs">
                 {t("ui.214228d4eb94a889")}<b>{t("ui.c308f16d1c5b71bf")}</b>{t("ui.5daaa42d45ea62c5")} {toolNames.length}  {"ud."} </div>
              <div className="max-h-40 overflow-y-auto rounded-md border p-2">
                {tools.length === 0 && <span className="text-muted-foreground text-xs">{t("ui.ab105594098570a6")}</span>}
                <div className="grid gap-1">
                  {tools.map((tool) => (
                    <label key={tool.key} className="flex items-start gap-2 text-xs">
                      <Checkbox className="mt-0.5" checked={toolNames.includes(tool.key)}
                        onCheckedChange={() => toggleTool(tool.key)} />
                      <span className="min-w-0">
                        <span className="font-medium">{tool.key}</span>
                        {tool.description && <span className="text-muted-foreground line-clamp-1"> {tool.description}</span>}
                      </span>
                    </label>
                  ))}
                </div>
              </div>
              <Textarea className="text-xs" rows={2} value={toolCallMsg}
                placeholder={t("ui.2b33f91c7c73523e")} onChange={(e) => setToolCallMsg(e.target.value)} />
            </div>
          )}
        </div>

        {/* creación de tarea */}
        <div className="grid gap-1.5">
          <label className="flex items-center gap-2 text-sm">
            <Checkbox checked={onTaskCreate} onCheckedChange={(v) => setOnTaskCreate(!!v)} />  {t("ui.755883c43b32a7fd")} </label>
          {onTaskCreate && (
            <Textarea className="text-xs" rows={2} value={taskCreateMsg}
              placeholder={t("ui.639bad272681a45c")} onChange={(e) => setTaskCreateMsg(e.target.value)} />
          )}
        </div>

        <div className="flex items-center gap-2">
          <Button size="sm" onClick={submit} disabled={saving}>
            <SaveIcon /> {editingId != null ? "guardar cambios" : t("ui.49f38217a14faa2e")}
          </Button>
          {editingId != null && (
            <Button size="sm" variant="ghost" onClick={resetForm} disabled={saving}>
              <XIcon />  {t("ui.4ccf118d16da15f8")} </Button>
          )}
        </div>
      </div>

      {/* El disparador ya existe */}
      <div className="grid gap-2">
        <Label className="text-muted-foreground text-xs">{t("ui.85fe85a88001ee3e")}</Label>
        {triggers.length === 0 && <span className="text-muted-foreground text-xs">{t("ui.26814bfa70c30bf9")}</span>}
        {triggers.map((trigger) => (
          <div
            key={trigger.id}
            className={cn(
              "flex items-start gap-2 rounded-md border p-2 text-sm",
              editingId === trigger.id && "border-primary bg-primary/5",
            )}
          >
            <Switch checked={trigger.enabled} onCheckedChange={() => toggleEnabled(trigger)} className="mt-0.5" />
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-1.5">
                <span className="font-medium">{condLabel(trigger)}</span>
                {!trigger.enabled && (
                  <Badge variant="outline" className="text-destructive px-1 py-0 text-[9px]">{t("ui.68de79aa60784d31")}</Badge>
                )}
              </div>
              <div className="text-muted-foreground grid gap-0.5 text-xs">
                {trigger.interval_sec > 0 && trigger.interval_message && <div className="line-clamp-1">{t("ui.7cffd82d69edbcbb")}{trigger.interval_message}</div>}
                {trigger.on_finding && trigger.finding_message && <div className="line-clamp-1">finding：{trigger.finding_message}</div>}
                {trigger.on_goal_met && trigger.goal_message && <div className="line-clamp-1">{t("ui.a75cd602a12543f3")}{trigger.goal_message}</div>}
                {trigger.on_task_timeout && trigger.task_timeout_message && <div className="line-clamp-1">{t("ui.096cc93952014fe9")}{trigger.task_timeout_message}</div>}
                {trigger.on_task_create && trigger.task_create_message && <div className="line-clamp-1">{t("ui.3b4d81b92ddfed8b")}{trigger.task_create_message}</div>}
                {trigger.on_tool_call && (
                  <>
                    <div className="line-clamp-1">{t("ui.09049c37cc10515f")}{trigger.tool_names.join("、") || t("ui.d0052c4dd0bcb3b3")}</div>
                    {trigger.tool_call_message && <div className="line-clamp-1">{t("ui.dfbda3bc77541609")}{trigger.tool_call_message}</div>}
                  </>
                )}
              </div>
            </div>
            <Button variant="ghost" size="icon-sm" className="text-muted-foreground hover:text-foreground"
              onClick={() => startEdit(trigger)} title={t("common.edit")}>
              <PencilIcon className="size-3.5" />
            </Button>
            <Button variant="ghost" size="icon-sm" className="text-muted-foreground hover:text-destructive"
              onClick={() => del(trigger.id)} title={t("common.delete")}>
              <Trash2Icon className="size-3.5" />
            </Button>
          </div>
        ))}
      </div>
    </div>
  );
}
