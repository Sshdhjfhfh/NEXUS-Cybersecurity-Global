// Nexus IA domain model — types used across the UI.
// Derived from the functional spec (section 7: forma de datos clave).

export type TaskStatus = "created" | "queued" | "running" | "paused" | "done" | "failed" | "timeout";
export type EngineMode = "exploring" | "paused" | "stalled" | "idle";

export interface Task {
  id: string;
  name?: string; // nombre de tarea opcional;vacío/predeterminado=sin nombre,Volver a la descripción al mostrar
  category_id?: number;
  category_name?: string;
  pinned?: boolean;
  pinned_at?: string | null;
  description: string;
  goal: string;
  status: TaskStatus;
  created_at: string;
  created_unix?: number; // created_at as unix seconds (run-duration calc)
  completed_at?: string; // RFC3339 finish time (done/failed); "" if unfinished
  completed_unix?: number; // completed_at as unix seconds (0/undef if unfinished)
  last_activity_unix?: number; // unix seconds of the last activity (0/undef if none)
  paused?: boolean;
  queued?: boolean;
  active?: boolean;
  in_flight?: number;
  findings?: { critical: number; high: number; medium: number; low: number }; // Número de vulnerabilidades registradas(agrupación por gravedad)
  last_activity?: string;
  stalled?: boolean;
  goals_total?: number;
  goals_met?: number;
  engine_mode?: EngineMode;
  tokens?: TokenTotal; // whole-task token consumption
  llm_profile_id?: number; // LLM profile used; absent = default profile
  llm_profile_ids?: number[]; // ordered task-level failover chain
  active_llm_profile_id?: number; // profile used by the next LLM call
  llm_failover_state?: "default" | "ready" | "chain_exhausted" | string;
  llm_failover_reason?: string;
  source_task_ids?: string[]; // directly related tasks inherited as read-only context
  archive_blocked_by_task_id?: string; // live direct dependent that must be archived first
  company_ids?: number[]; // associated company scopes; current company assets join the task at creation
  coverage_enabled?: boolean; // cambio de función de cobertura de activos(Crear sincronización,activado de forma predeterminada)；false=No calculado/No mostrar cobertura
}

export interface TaskCategory {
  id: number;
  name: string;
  task_count: number;
  created_at: string;
  updated_at: string;
}

export interface TaskTemplate {
  id: number;
  name: string;
  description: string;
  goal: string;
  category_id?: number | null; // categoría predeterminada；null/predeterminado=Ninguno
  intercept_rules?: AssetInterceptRuleInput[]; // interceptación predeterminada a nivel de tarea/Permitir reglas
  created_at: string;
  updated_at: string;
}

export interface DeleteTaskOptions {
  delete_assets: boolean;
  delete_traffic: boolean;
  delete_files: boolean;
  delete_findings: boolean;
  delete_llm_records: boolean;
}

export interface DeleteTaskResult {
  deleted: string;
  assets_deleted: number;
  assets_detached: number;
  traffic_deleted: number;
  files_deleted: boolean;
  findings_deleted: number;
  llm_records_deleted: number;
  cleanup_warning?: string;
}

export type TaskArchiveState =
  | "archive_queued"
  | "archiving"
  | "archive_failed"
  | "ready"
  | "restore_queued"
  | "restoring"
  | "restore_failed"
  | "delete_queued"
  | "deleting"
  | "delete_failed";

export interface TaskArchiveTokenStats {
  calls?: number;
  input_tokens?: number;
  output_tokens?: number;
  cache_read_tokens?: number;
  cache_write_tokens?: number;
}

export interface TaskArchive {
  id: number;
  task_id: number;
  state: TaskArchiveState;
  phase: string;
  progress: number;
  error?: string;
  warnings?: string[];
  format_version: number;
  sha256?: string;
  original_size: number;
  compressed_size: number;
  task_name: string;
  task_description: string;
  task_goal: string;
  original_status: TaskStatus;
  category_id?: number;
  category_name?: string;
  source_task_ids: number[];
  remaining_timeout_seconds: number;
  data_counts: Record<string, number>;
  aggregate_stats: {
    tokens?: TaskArchiveTokenStats;
    skills?: Record<string, number>;
    tools?: Record<string, number>;
    findings?: Record<string, number>;
  };
  archived_at?: string;
  requested_at: string;
  created_at: string;
  updated_at: string;
}

export interface TaskArchivePage {
  items: TaskArchive[];
  total: number;
  page: number;
  size: number;
}

export interface ArchiveBatchItem {
  id: string;
  archive_id?: number;
  ok: boolean;
  queued: boolean;
  error?: string;
}

// ---- Asset graph (global, shared across tasks) ----
export type AssetType =
  | "company"
  | "domain"
  | "ip"
  | "port"
  | "service"
  | "site"
  | "endpoint"
  | "parameter"
  | "tech"
  | "credential"
  | "data";

export type NodeState = "observed" | "confirmed" | "tombstoned";

export interface AssetNode {
  id: string;
  type: AssetType;
  name: string;
  key: string; // nkey
  value?: string;
  company_id?: string; // Activos conferidos a la empresa id；vacío=No adquirido
  state: NodeState;
  confidence: number; // 0..1
  attrs?: Record<string, unknown>;
  first_seen: string;
  last_seen: string;
}

export type AssetRel =
  | "owns"
  | "resolves"
  | "exposes"
  | "runs"
  | "serves"
  | "has_endpoint"
  | "has_param"
  | "fingerprinted"
  | "authenticates_as"
  | "reachable"
  | "has_subdomain";

export interface Edge {
  src: string;
  dst: string;
  rel: AssetRel | ExploreRel;
}

// Task asset view — server-side enriched, paginated.
export interface TaskAssetRef {
  id: string;
  name?: string;
  key: string;
  attrs?: Record<string, unknown>;
}

export interface TaskAssetItem extends AssetNode {
  techs?: TaskAssetRef[];
  auth?: TaskAssetRef[];
  params?: TaskAssetRef[];
}

export interface TaskAssetView {
  counts: Record<string, number>;
  total: number;
  items: TaskAssetItem[];
}

// ---- New unified asset model (new backend) ----
export type NewAssetType = "root_domain" | "ip" | "subdomain" | "app" | "service" | "endpoint";

export interface Asset {
  id: number;
  type: NewAssetType;
  company_id?: number;
  task_ids: number[];
  domain?: string;
  root_domain?: string;
  ip?: string;
  c_segment?: string;
  port?: number;
  icp?: string;
  bound_domains?: string[];
  open_ports?: { port: number; service?: string }[];
  record_type?: string;
  record_value?: string[] | string;
  bundle_id?: string;
  app_name?: string;
  category?: string;
  app_description?: string;
  app_icp?: string;
  url?: string;
  service_type?: string;
  service_name?: string;
  favicon_mmh3?: string;
  status_code?: number;
  content_length?: number;
  page_title?: string;
  technologies?: string[];
  auth?: Record<string, unknown>[];
  method?: string;
  params?: Record<string, unknown>[];
  extra?: Record<string, unknown>;
  last_seen: string;
  task_source?: string;
  task_source_summary?: string;
  task_source_node_id?: number;
}

export interface IntentAsset {
  intent_id: number | string;
  asset_id: number;
  type: NewAssetType;
  label: string;
  source: string;
  source_summary: string;
  source_node_id?: number;
  source_task_id: number;
  inherited: boolean;
}

export interface TaskAssetMutation {
  requested: number;
  attached: number;
  existing: number;
}

export interface TaskAssetScopeMutation {
  requested: number;
  assets_linked: number;
  assets_existing: number;
  scopes_added: number;
  scopes_existing: number;
}

// ---- Asset coverage graph (per task) ----
// Fuerza dirigida a un nodo en la superposición de activos。key Sólo: Activo="a:<id>"、Empresa="c:<id>"、
// nombre de dominio raíz sin fila de activos="r:<domain>"。in_scope=false es un nodo de contexto gris que se usa solo para cableado。
export interface CoverageGraphNode {
  key: string;
  kind: "company" | "root_domain" | "subdomain" | "ip" | "service" | "app" | "endpoint";
  label: string;
  tested: boolean;
  in_scope: boolean;
  asset_id?: number;
  company_id?: number;
  domain?: string;
  root_domain?: string;
  ip?: string;
  url?: string;
  port?: number;
  service_type?: string;
  app_name?: string;
  page_title?: string;
  status_code?: number;
}

export interface CoverageGraphEdge {
  src: string;
  dst: string;
}

export interface CoverageGraphData {
  nodes: CoverageGraphNode[];
  edges: CoverageGraphEdge[];
}

// La intención asociada con un activo en el mapa de exploración de esta tarea/hecho/Descubrimiento (para cajón de nodos de gráficos superpuestos)）。
export interface CoverageAssetRef {
  id: number;
  kind: string;
  state: string;
  summary: string;
  source_task_id?: string;
  inherited?: boolean;
}
export interface CoverageAssetRefs {
  intents: CoverageAssetRef[];
  facts: CoverageAssetRef[];
  findings: CoverageAssetRef[];
}

// ---- Workspace file manager (workDir) ----
export interface WorkspaceEntry {
  name: string;
  path: string; // workspace-relative, forward slashes
  dir: boolean;
  size: number;
  mtime: number; // unix millis
}
export interface WorkspaceListing {
  path: string;
  entries: WorkspaceEntry[];
}
export interface WorkspaceFile {
  path: string;
  size: number;
  binary: boolean;
  too_large?: boolean;
  content?: string;
}

// Un ítem del rango de la prueba de tareas (denominador de cobertura + Límites de autorización）。
export interface TaskScopeRow {
  id: number;
  task_id: number;
  kind: "company" | "root_domain" | "subdomain" | "ip" | "cidr" | "icp" | "keyword";
  company_id?: number;
  company_name?: string; // backend JOIN companies Analizar, sólo kind=company valioso
  domain?: string;
  net?: string;
  value?: string;
  source: "auto" | "agent" | "manual";
  reason?: string;
}

export type CompanyScopeKind = "domain" | "ip" | "cidr" | "icp" | "keyword";

// Reglas de rango de activos estructurados enviadas al agregar una nueva empresa。
export interface CompanyScopeRule {
  kind: CompanyScopeKind;
  value: string;
}

// Resultado de la escritura del rango de activos。errors Esta es una línea ilegal en este envío.；warnings No tiene nada que ver con este envío.、
// Pero existen problemas de datos que hacen que los resultados de la atribución no sean los esperados (p. ej. ip El campo almacena el activo del nombre de host.）。
export interface CompanyScopeMutation {
  added: number;
  skipped: number;
  invalid: number;
  errors?: string[];
  warnings?: string[];
}

// Una de las reglas de rango de activos de la empresa (atribuida a una única fuente de verdad）。
export interface ScopeRow {
  id: number;
  company_id: number;
  kind: CompanyScopeKind;
  domain?: string; // kind=domain A veces valioso
  net?: string; // kind=ip|cidr A veces valioso
  value?: string; // kind=icp|keyword puede devolverse directamente desde el backend
  raw: string; // Entrada de usuario sin procesar, utilizada para visualización y relleno
  reason?: string;
}

// empresa：type=company nodo de activos + icono + Recuento de activos + Reglas de alcance de activos。
export interface Company {
  id: number;
  name: string;
  logo?: string; // icono remoto URL；Si está vacío, la interfaz usa la primera letra del nombre.
  asset_count: number;
  scope?: ScopeRow[];
}

// ---- Exploration graph (per task) ----
export type ExploreKind = "task" | "begin" | "goal" | "intent" | "fact" | "finding" | "hint" | "digest";
export type GoalState = "open" | "met" | "abandoned";
export type IntentState = "open" | "running" | "paused" | "done" | "blocked" | "exhausted" | "stopped";
export type FindingState = "confirmed" | "dismissed";
export type HintState = "active" | "consumed";
export type ExploreRel = "spawns" | "derived_from" | "yields" | "proves" | "covers";

export interface TaskNode {
  id: string;
  type: ExploreKind;
  payload?: string;
  priority: number; // 0..10
  state: string; // GoalState | IntentState | FindingState | HintState
  origin: string;
  ts: string;
  source_task_id?: string;
  inherited?: boolean;
  delete_reason?: string; // Eliminación falsa intencional(state='deleted')Motivo de la eliminación
}

// página del tablero de anuncios:Nodos paginados en orden de creación. + Los bordes involucrados en esta página. + Nodo en el otro extremo del borde(refs,prensa id índice),
// De esta forma, cada emisión podrá explicar claramente “de dónde viene y qué produce”.」,Sin bajar la imagen completa。
export interface ExplorationNodePage {
  items: TaskNode[];
  total: number;
  page: number;
  size: number;
  edges: Edge[];
  refs: Record<string, TaskNode>;
  // nodo id → Activos anclados por este nodo(Se muestra cuando se expande el tablero de anuncios.,Contiene el nodo en esta página y sus vecinos.)。
  assets: Record<string, FindingAsset[]>;
}

export interface ExplorationNodeQuery {
  page?: number;
  size?: number;
  kinds?: ExploreKind[];
  states?: string[];
  q?: string;
  order?: "asc" | "desc";
}

// Tarjeta de gestión de objetivos para objetivos.(El backend ha sido payload Dividido en text/vulnclass)。
export interface TaskGoal {
  id: string;
  text: string;
  vulnclass?: string;
  state: string; // GoalState
  origin?: string;
  ts: string;
}

// Restricciones operativas utilizadas por las tarjetas de gestión de restricciones.(allow=Permitido / deny=Prohibido)。
export type ConstraintKind = "allow" | "deny";
export interface TaskConstraint {
  id: string;
  kind: ConstraintKind;
  text: string;
  origin?: string;
  ts?: string;
}

// ---- Findings ----
export type Severity = "critical" | "high" | "medium" | "low";

// estado de manejo de vulnerabilidades:Pendiente / Procesamiento / confirmado / Procesado / Fijo / Falsa alarma / ignorar / repetir / Aceptación de riesgos。
export type FindingStatus =
  | "pending"
  | "in_progress"
  | "confirmed"
  | "resolved"
  | "fixed"
  | "false_positive"
  | "ignored"
  | "duplicate"
  | "risk_accepted";

// FindingAsset Es un activo ligado a una vulnerabilidad(Ya prerenderizado en el backend label)。
export interface FindingAsset {
  id: string;
  type: string;
  label: string;
}

export interface Finding {
  traffic_count?: number;
  evidence_version?: number;
  report_evidence_version?: number;
  report_stale?: boolean;
  id: string;
  finding_id?: string; // Independiente findings fila de la tabla id,Manejo de actualización de estado(Es posible que falte el nodo antiguo en la tarea)
  vulnclass: string;
  name?: string; // nombre de la vulnerabilidad;Cuando está vacío, la pantalla vuelve a mostrar vulnclass
  severity: Severity;
  status: FindingStatus;
  summary: string;
  evidence: string;
  report?: string; // informe detallado(Markdown);Solo regresa la interfaz de detalles,la lista está vacía
  intent_id?: string;
  param_id?: string;
  task_id?: string;
  task_description?: string;
  source_task_id?: string;
  inherited?: boolean;
  assets?: FindingAsset[];
  ts: string;
}

// FindingsPage es la respuesta de paginación del lado del servidor de la lista de descubrimiento。
export interface FindingsPage {
  items: Finding[];
  total: number;
  page: number;
  page_size: number;
}

export interface FindingGroup {
  task_id: string | number | null;
  task_name?: string; // nombre de tarea opcional;vacío/predeterminado=sin nombre
  task_description: string;
  task_status: string;
  count: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  last_found_at: string;
}

export interface FindingGroupsPage {
  items: FindingGroup[];
  total: number;
  finding_total: number;
  page: number;
  page_size: number;
}

export interface FindingDeepenResponse {
  task_id: string;
  intent_id: string;
  state: IntentState;
  queued: boolean;
}

// FindingStats Descubra la agregación de tablas completa(tarjeta de estadísticas + menú desplegable de tipo de vulnerabilidad),informática del lado del servidor,No afectado por la paginación。
export interface FindingStats {
  total: number;
  pending: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  vulnclasses: string[];
  tasks: FindingTaskOption[];
}

// FindingTaskOption es un elemento de la lista desplegable ”filtrar por tarea” en la página de descubrimiento.:misión con errores(Una descripción vacía significa que la tarea ha sido eliminada,
// visualización de retroceso en el front-end id)y el número de vulnerabilidades。
export interface FindingTaskOption {
  id: string | number;
  name?: string; // nombre de tarea opcional;vacío/predeterminado=sin nombre
  description: string;
  count: number;
}

// FindingQuery es paginación de la lista de descubrimiento/filtro/Ordenación de parámetros。
export interface FindingQuery {
  page: number;
  pageSize: number;
  severity?: "all" | Severity;
  status?: "all" | FindingStatus;
  vulnclass?: string;
  task?: string; // misión id;"all"/vacío = No filtrar por tarea
  query?: string;
  sort?: "severity" | "time";
  // nodo del árbol de activos key;Seleccione un nodo = seleccione todo su subárbol. vacío = No filtrar por activos。
  assetScope?: string;
}

// ---- Findings by asset (Vista de activos) ----
export type FindingAssetKind = "company" | "root_domain" | "subdomain" | "ip" | "service" | "app" | "endpoint" | "none";

// FindingAssetNode Es un nodo del árbol de activos。key con forma a:<id>(Activos)、c:<id>(empresa)、
// r:<domain>(No hay ningún nombre de dominio raíz de la fila de activos en la biblioteca.)、__none__(Activos no asociados)。
export interface FindingAssetNode {
  key: string;
  parent?: string;
  kind: FindingAssetKind;
  label: string;
  asset_id?: number;
  company_id?: number;
  self: number; // el número de descubrimientos directamente relacionados con este activo
  total: number; // Incluir descendientes, eliminar duplicados tal como se encuentran
  critical: number;
  high: number;
  medium: number;
  low: number;
  last_found_at: string;
}

export interface FindingAssetTree {
  nodes: FindingAssetNode[];
  finding_total: number;
  truncated: boolean;
  dropped_kinds?: string[];
}

// FINDING_UNASSIGNED_ASSET Con el backend db.FindingUnassignedAsset correspondiente。
export const FINDING_UNASSIGNED_ASSET = "__none__";

// ---- Activity / sessions ----
export type ActivityKind =
  | "tool_use"
  | "tool_result"
  | "text"
  | "thinking"
  | "result"
  | "user"
  | "intent" // LLM-generated exploration objective leading a worker session (UI-synthesized)
  | "round" // planner round boundary marker (engine-emitted)
  | "usage" // live cumulative token usage (per model turn); not rendered
  | "llm_switch" // automatic/manual task-level LLM switch
  | "llm_failover" // task-level provider switch / chain exhaustion audit event
  | "intercept_request"; // user-approval request from the intercept layer

// ChatAttachment Es un archivo subido una vez:path relativo a esta sesión/directorio de trabajo de tareas(eso es agent de CWD)。
export interface ChatAttachment {
  name: string;
  path: string;
  size: number;
  abs?: string; // camino absoluto(scope=staging Devuelto durante la carga temporal;Escríbelo en la descripción antes de crear la tarea.)
}

export interface Activity {
  seq: number;
  intent_id?: string;
  worker: string; // session owner: planner | mainagent | work#1 ...
  ts: string;
  kind: ActivityKind;
  tool?: string;
  tool_use_id?: string;
  is_error?: boolean;
  summary: string;
  detail?: string;
  metadata?: {
    llm_transition?: LLMTransition;
  };
  source_task_id?: string;
  inherited?: boolean;
  main_seg?: number; // main-agent conversation segment (present only on worker="mainagent" rows)
  // token usage (present only on kind='result')
  input_tokens?: number;
  output_tokens?: number;
  cache_read_tokens?: number;
  cache_write_tokens?: number;
}

export interface LLMAuditProfile {
  id: number;
  name: string;
  format: string;
  model: string;
}

export interface LLMTransition {
  mode: "automatic" | "manual" | "exhausted";
  reason: string;
  previous?: LLMAuditProfile;
  next?: LLMAuditProfile;
}

export interface TaskLLMResolution {
  profile_id?: number;
  name: string;
  format: string;
  model: string;
  source: "task_chain" | "agent_binding" | "global_profile" | "environment" | "global";
  available: boolean;
  reason?: string;
}

export interface TaskLLMResolutions {
  mainagent: TaskLLMResolution;
  planner: TaskLLMResolution;
  worker: TaskLLMResolution;
}

// ---- Agent triggers (P3 Programación, solo personalizada agent) ----
export interface AgentTrigger {
  id: number;
  agent_key: string;
  enabled: boolean;
  interval_sec: number; // tiempo:cada N segundos(0=irregular)
  on_finding: boolean; // Cualquier tarea encontrada finding Se activa cuando
  on_goal_met: boolean; // Se activa cuando cualquier tarea alcanza su objetivo.
  on_task_timeout: boolean; // Se activa cuando se agota el tiempo de espera de cualquier tarea
  on_tool_call: boolean; // Se llama a la herramienta seleccionada(Ejecución completada)Se activa cuando
  on_task_create: boolean; // Se activa cuando se crea cualquier tarea.
  interval_message: string; // Mensajes de usuario independientes para cada condición de activación
  finding_message: string;
  goal_message: string;
  task_timeout_message: string;
  tool_call_message: string;
  task_create_message: string;
  tool_names: string[]; // on_tool_call herramienta seleccionada key(al menos uno)
  last_fire?: string;
}

// ---- Conversations (chat page) ----
export interface ActiveFindingRetest {
  id: number;
  finding_id: string;
  conversation_id: number;
  status: "pending" | "running";
}

export interface FindingRetest {
  id: number;
  finding_id: number;
  conversation_id: number | null;
  status: "pending" | "running" | "completed" | "failed" | "stopped";
  verdict: "" | "reproduced" | "fixed" | "inconclusive";
  notes: string;
  summary: string;
  evidence: string;
  error: string;
  created_at: string;
  started_at: string | null;
  finished_at: string | null;
}

export interface Conversation {
  id: number;
  running?: boolean; // live server state, returned with the conversation list
  agent_key: string;
  title: string;
  llm_profile_id?: number;
  pinned?: boolean;
  pinned_at?: string | null;
  created_at: string;
  updated_at: string;
}

// ---- Backend logs (/logs page) ----
export interface LogLine {
  seq: number;
  db_id?: number; // server_logs.id; present for DB-persisted lines
  ts: string;
  level: "info" | "warn" | "error";
  tag: string;
  text: string;
}

export type SessionRole = "mainagent" | "planner" | "worker" | "system";
export type SessionStatus = "running" | "paused" | "done" | "blocked" | "exhausted" | "pending" | "stopped" | "deleted";

// Daily token aggregate bucket (GET /api/tokens/daily).
export interface DailyTokenBucket {
  date: string; // "YYYY-MM-DD"
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// Per-worker token usage (GET /api/exploration/tokens).
export interface TokenUsage {
  worker: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

export interface SessionTokenUsage {
  session: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

export interface BatchControlItem {
  id: string;
  ok: boolean;
  status?: string;
  queued?: boolean;
  error?: string;
}

// Resultados tarea por tarea de la reclasificación de lotes. El fallo solo puede ser que la tarea haya sido eliminada, la escritura de la categoría en sí es atómica。
export interface BatchCategoryItem {
  id: string;
  ok: boolean;
  error?: string;
}

// Whole-task (all agents) token aggregate.
export interface TokenTotal {
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// Global per-profile token spend from the llm_usage ledger (GET /api/tokens/usage).
export interface ProfileUsage {
  profile_name: string;
  calls: number;
  tasks: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// One (profile, UTC day) token bucket for the dashboard's daily chart (new source).
export interface ProfileDayUsage {
  profile_name: string;
  date: string; // YYYY-MM-DD
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
}

// Response of GET /api/tokens/usage — the dashboard's "new" (llm_usage) token view.
export interface UsageStats {
  by_profile: ProfileUsage[];
  daily: ProfileDayUsage[];
}

// Per-model token usage for one task (GET /api/llm/records/by-model), from the
// always-on llm_usage metering ledger. calls = number of LLM calls on this model.
export interface ModelTokenStat {
  model: string;
  calls: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

export interface Session {
  id: string;
  role: SessionRole;
  title: string;
  status: SessionStatus;
  live: boolean;
  last_activity: string;
  intent_id?: string;
  source_task_id?: string;
  inherited?: boolean;
  seg?: number; // main-agent session: which conversation segment (0 = original)
}

// ---- Security ----
export interface AuditEntry {
  ts: string;
  tool: string;
  action: "allow" | "block";
  reason?: string;
  command?: string;
}

export interface Audit {
  entries?: AuditEntry[];
  attributions?: Record<string, number>;
}

// ---- Traffic ----
export interface TrafficExchange {
  id: string;
  ts: string;
  host: string;
  method: string;
  url: string;
  status: number;
  content_type: string;
  resp_len: number;
}

export interface TrafficResp {
  enabled: boolean;
  proxy?: string;
  count?: number; // global total (unfiltered)
  total?: number; // rows matching the current filter (for pagination)
  page?: number;
  size?: number;
  exchanges?: TrafficExchange[];
}

// Full raw request/response of one exchange (lazy-loaded on row select).
export interface TrafficDetail {
  req: string;
  resp: string;
}

// One distinct recorded host with its exchange count (target picker).
export interface TrafficHost {
  host: string;
  count: number;
}

// ---- App settings (runtime toggles) ----
export interface Settings {
  traffic_capture: boolean;
  agent_traffic_binding: boolean; // Agent vincular automáticamente la evidencia de tráfico, desactivado de forma predeterminada; no afecta la encuadernación manual
  llm_record: boolean; // LLM interruptor de grabación (apagado predeterminado); no se grabará nada cuando esté apagado LLM llamada
  // Web search. brave_key_set / tavily_key_set reflect whether a key is stored
  // (the values are never returned). On PUT, send the corresponding field to set/clear.
  web_search_enabled: boolean;
  web_search_backend: string; // "ddgs" | "brave-free" | "tavily" | "deepseek"
  brave_key_set: boolean;
  tavily_key_set: boolean;
  // write-only: only sent on PUT to store/clear the key.
  brave_search_api_key?: string;
  tavily_search_api_key?: string;
  // Agente exportador independiente(http/https/socks5)，Se utiliza para acceder al punto final de búsqueda; y registrar el tráfico MITM Agente irrelevante. vacío=conexión directa。
  web_search_proxy?: string;
  // Agente exportador global(http/https/socks5，se puede traer user:pass)，Todo el tráfico objetivo pasa por él. Cuando la captura de tráfico está activada como
  // MITM aguas arriba; inyección directa cuando la captura está desactivada agent de bash/WebFetch。vacío=conexión directa。
  global_proxy?: string;
  python_interpreter?: string; // herramienta de secuencia de comandos personalizada python ruta del intérprete(vacío=Detección de tiempo de ejecución)
  workers?: number; // trabajo concurrente agent número(predeterminado3)；Válido para tareas iniciadas más tarde
  // límite superior de simultaneidad de tareas:el límite superior del número de tareas ”en ejecución” al mismo tiempo. cerrar=Sin límite;Después de la apertura, las nuevas tareas se pondrán en cola si superan el límite.,Iniciar automáticamente cuando hay espacio disponible。
  task_concurrency_enabled?: boolean; // predeterminado false
  task_concurrency_limit?: number; // predeterminado después de la apertura 5
  // LLM sondeo(Conmutación por error)。Desactivado de forma predeterminada; cuando está encendido, ”modelo no especificado” agent No disponible en la configuración actual
  // （Saldo insuficiente/key no válido/limitación de corriente/Cambia automáticamente a la siguiente configuración cuando hay una excepción de servicio)。
  llm_pool_enabled?: boolean; // predeterminado false
  // vinculado con la configuración especificada agent/Si se debe recurrir a la cadena de sondeo cuando falla la tarea. Predeterminado false = Vinculante significa exclusivo。
  llm_pool_bind_fallback?: boolean;
  // rango de inyección de restricciones operativas(Ambos están habilitados de forma predeterminada):dar la tarea allow/deny Correspondencia ortográfica de restricciones agent Solicitud del sistema para。
  constraints_inject_planner?: boolean;
  constraints_inject_worker?: boolean;
  // función experimental:noa compresión de contexto basada en modelos(Desactivado por defecto)。Cuatro tipos de acceso a la plataforma después de la apertura agent(planner/
  // worker/Señor agent/Diálogo)por noa hacerse cargo de la compresión del contexto,Reemplaza incorporado compaction;cada run leer una vez,Sí
  // comenzó más tarde run Efectivo。
  noa_compaction?: boolean;
  // ---- Vulnerabilidad IM Push (el canal en sí es un recurso independiente, ver /api/notify/*，Aquí solo hay tres configuraciones globales）----
  notify_enabled?: boolean; // Pulsar el interruptor principal, encendido por defecto; Se utiliza para la hemostasia con un solo clic durante el período de mantenimiento.
  notify_public_base_url?: string; // dirección de acceso externo del enlace de detalles de la vulnerabilidad; vacío=El mensaje no regresa a la cadena.
  notify_digest_interval_min?: number; // Período del modo resumen (minutos), predeterminado 30
}

// ---- Vulnerabilidad IM Empujar ----

// NotificationFilter Es la condición de filtrado del canal. Todos los campos son opcionales. El valor predeterminado no es ningún filtrado.。
// El backend no verifica todos los campos: cuando la configuración tiene un formato incorrecto, se trata como ”acertada” (es mejor presionar más que perder el impulso)）。
export interface NotificationFilter {
  min_severity?: string; // "" | low | medium | high | critical
  task_ids?: number[]; // vacío=Sin límite; si no está vacío, debe superponerse con la tarea a la que pertenece la vulnerabilidad.
  asset_ids?: number[]; // vacío=Sin límite; si no está vacío, debe cruzarse con el activo anclado a la vulnerabilidad
  vulnclass_include?: string[]; // vacío=Recoge todo; si no está vacío, se requiere el tipo de vulnerabilidad para acceder a cualquier palabra clave (subcadena que no distingue entre mayúsculas y minúsculas）
  vulnclass_exclude?: string[]; // Excluir si se acierta alguna palabra clave (la exclusión tiene prioridad sobre la inclusión)）
  on_status_change?: boolean; // Si se deben recibir también eventos de cambio de estado de eliminación de vulnerabilidades
}

// NotificationChannel es una instancia de canal。config Los campos de kind Varía，
// y el campo de credencial se reemplaza con "__masked__" El valor de la máscara al principio; devolverlo tal como está significa ”no cambiarlo”」。
export interface NotificationChannel {
  id: number;
  name: string;
  kind: string;
  enabled: boolean;
  mode: "realtime" | "digest";
  config: Record<string, unknown>;
  filter: NotificationFilter;
  rate_per_min: number;
  created_at: string;
  updated_at: string;
  // secret_keys Lo proporciona el back-end según el tipo de canal, y el front-end muestra el cuadro de contraseña y el mensaje ”déjelo en blanco o no cambie” en consecuencia.，
  // No codifique ningún conocimiento del canal。
  secret_keys: string[];
}

// NotificationKind Sí /api/notify/meta Metadatos de tipo de canal devueltos。
export interface NotificationKind {
  kind: string;
  default_rate_per_min: number;
  secret_keys: string[];
}

export interface NotificationMeta {
  kinds: NotificationKind[];
  enabled: boolean;
  public_base_url: string;
  digest_interval_min: string;
  defaults: { digest_interval_min: number };
  stats: {
    channels: number;
    channels_on: number;
    pending: number;
    failed: number;
    sent_today: number;
    backlog_age_ms: number;
  };
}

// NotificationDelivery Es un registro de entrega, utilizado para el historial de entrega y reenvío fallido.。
export interface NotificationDelivery {
  id: number;
  finding_id: string;
  event_kind: string; // finding_created | finding_status_changed
  channel_id: number;
  channel_name: string;
  channel_kind: string;
  state: "pending" | "sending" | "sent" | "failed" | "skipped";
  attempts: number;
  last_error: string;
  batch_id?: number;
  created_at: string;
  sent_at?: string;
  next_attempt_at: string;
  title: string;
  severity: string;
}

// ---- LLM config ----
export interface LLMProfile {
  id: string;
  name: string;
  format: "openai" | "anthropic" | "openai-responses";
  base_url?: string;
  proxy?: string;
  model: string;
  api_key_hint?: string;
  rate_per_second: number;
  rate_per_minute: number;
  context_window_k?: number;
  // pensar en cambiar(thinking.type): ""=No enviar(predeterminado) | "disabled"=Cerrar | "enabled"=encendido
  thinking_type?: string;
  // intensidad del pensamiento: ""=No enviar(predeterminado) | "low"/"medium"/"high"/"xhigh"/"max"
  reasoning_effort?: string;
  is_default: boolean;
  // Orden de sondeo: Cuanto mayor sea el número, se seleccionará primero. La configuración de activación es siempre la cabeza de la cadena y no tiene nada que ver con este valor.。
  priority?: number;
  // true = No es un objetivo de conmutación por error (todavía puede serlo) agent/Se utiliza enlace explícito de tarea）。
  pool_exclude?: boolean;
  // true（predeterminado）= Transmisión(SSE) | false = verdadero·sin transmisión(stream:false，devolución única)。
  streaming?: boolean;
  // el límite superior de salida para una única respuesta(token)。0 = No enviar este campo, determinado por el valor predeterminado del servidor。
  // Atención y context_window_k Distinción: Esta última es la capacidad total del modelo y solo se usa localmente para los umbrales de compresión.。
  max_tokens?: number;
  // qué nombre de campo de solicitud se utiliza solo para el límite superior format="openai" significativo：
  // ""=max_tokens(predeterminado) | "max_completion_tokens"(OpenAI El modelo de inferencia solo lo reconoce.)
  max_tokens_field?: string;
  // Nombre del encabezado de sesión personalizado: Cuando no esté vacío, se incluirá en cada solicitud. HTTP cabeza, valor de cabeza=sesión actual/intencional session id。
  // ""=No enviado. Solía ​​presionar session-id caché de solicitud de encabezado/puerta de enlace para enrutamiento fijo。
  session_header_key?: string;
  // Esta configuración cubre el reintento (establecimiento de conexión/respuesta vacía/Mismo provider ventana segura). dejar en blanco/completo 0 = Sigue la estrategia global。
  retry?: LLMRetryOverride;
}

// ---- LLM estrategia de reintento ----
// Dos botones para una capa de reintentos. ambos「0 = No configurado」：
//   attempts    0=Usar tiempos predeterminados | -1=Cierra esta capa y vuelve a intentarlo. | >0=Número de reintentos
//   interval_ms 0=Usar retroceso exponencial predeterminado | >0=utilice este intervalo fijo de milisegundos en su lugar
export interface LLMRetryRule {
  attempts: number;
  interval_ms: number;
}

// único LLM Configure las tres capas que se pueden cubrir (todas son reintentos de ”seguir el punto final”)）。
export interface LLMRetryOverride {
  connect: LLMRetryRule; // Reintento de establecimiento de conexión: restablecimiento de conexión/tiempo de espera/429/5xx，antes de que comience la transmisión
  empty: LLMRetryRule; // Reintento de respuesta vacía: completado sin nada (solo openai Formato）
  stream: LLMRetryRule; // Mismo provider Reintento de ventana seguro: reproducción de secuencia antes de entregar la salida
}

// estrategia global = valores predeterminados para las tres capas anteriores + Las dos capas solo tienen global：
//   breaker disyuntor de sondeo（attempts=Varias fallas instantáneas consecutivas de fusibles，interval_ms=tiempo de enfriamiento fijo）
//   intent  Intención de volver a correr（worker con model_error Después del final, toda la intención es volver a correr.）
export interface LLMRetryPolicy extends LLMRetryOverride {
  breaker: LLMRetryRule;
  intent: LLMRetryRule;
}

// ---- LLM sondeo (conmutación por error）----
// Una posición configurada y un estado de salud en la cadena de sondeo。state:
//   ok       normal
//   degraded Hay fallas consecutivas pero no se alcanza el umbral del disyuntor.
//   tripped  Fusible fundido, omitido durante el período de enfriamiento（cooldown_secs son los segundos restantes）
export interface LLMPoolMember {
  profile_id: string;
  name: string;
  model: string;
  format: string;
  priority: number;
  active: boolean; // Si es la configuración actualmente activa (siempre la cabeza de la cadena)）
  excluded: boolean; // pool_exclude：No participar en las encuestas
  state: "ok" | "degraded" | "tripped";
  fails: number;
  trips: number;
  cooldown_secs: number;
  last_error?: string;
  last_at?: string;
}

export interface LLMPoolStatus {
  enabled: boolean;
  bind_fallback: boolean;
  chain: LLMPoolMember[];
}

// ---- Agents ----
export interface Agent {
  id: string;
  key: string; // Incorporado como goals/planner/mainagent/worker；personalizado según lo definido por el usuario key
  name: string;
  description?: string;
  role: string;
  builtin: boolean;
  enabled: boolean;
  llm_profile_id?: number | null; // encuadernado LLM Configuración；null/absent = sigue la misión/sesión/Mundial
  max_turns?: number; // 0 = Ilimitado
  run_seconds?: number; // worker El límite superior del reloj de pared en una sola operación.(segundos)；0 = Ilimitado
  web_search?: boolean; // si se debe habilitar la búsqueda de red(Cerrado por el interruptor global del sistema)
  interactive_shell?: boolean; // Si se debe habilitar interactivo shell(duradero PTY familia de herramientas de conversación)
  // P3 estrategia de procesamiento posterior al disparo(Sólo personalizado agent significativo)
  trigger_run_mode?: "serial" | "parallel"; // cola en serie / Cada disparador desencadena una sesión simultánea.
  trigger_merge_mode?: "by_task" | "all" | "none"; // solo serial：fusionar con la misma tarea / fusionar todo / No fusionar
  trigger_max_parallel?: number; // solo parallel：cada agent límite superior de concurrencia；0=Sin límite
  // cantidad vinculante(Solo regresa la interfaz de lista)：Visibles MCP / Visibles Skill / herramienta de encuadernación
  mcp_count?: number;
  skill_count?: number;
  tool_count?: number;
}

export interface PromptVar {
  name: string;
  description: string;
  example: string;
  source: "exploration" | "runtime" | "distilled";
}

export interface PromptVersion {
  version: number;
  ts: string;
  note: string;
  template_text: string;
}

export interface AgentDetail {
  agent: Agent;
  prompt: string;
  variables: PromptVar[];
  versions: PromptVersion[];
  visibility: { mcp: number[]; skill: string[] };
  // vinculable LLM candidato de configuración(Proporcionar menú desplegable ”Modelo predeterminado”)；Enlace actual ver agent.llm_profile_id
  llm_profiles?: { id: number; name: string; model: string; is_default: boolean }[];
  wrapup_prompt?: string; // Palabra de mensaje de cierre guardada(vacío=Usar valor predeterminado incorporado)
  wrapup_default?: string; // palabra de aviso de cierre predeterminada incorporada(marcador de posición/Restaurar valor predeterminado)
  wrapup_max_turns?: number; // Número de rondas finales guardadas(0=Usar valor predeterminado incorporado)
  wrapup_max_turns_default?: number; // Número predeterminado incorporado de rondas de cierre(suministro "0=predeterminadoN" propina)
  // Palabra final de tiempo de espera a nivel de tarea(solo worker/planner，task_timeout_wrapup_supported=true Esta partición se mostrará sólo cuando)
  task_timeout_wrapup_supported?: boolean;
  task_timeout_wrapup_prompt?: string;
  task_timeout_wrapup_default?: string;
  task_timeout_wrapup_max_turns?: number;
  task_timeout_wrapup_max_turns_default?: number;
}

// ---- MCP ----
export interface MCPServer {
  id: number;
  name: string;
  transport: "stdio" | "http" | "sse";
  command?: string;
  args: string[];
  env: Record<string, string>;
  url?: string;
  enabled: boolean;
  insecure?: boolean; // http: skip TLS cert verification (self-signed servers)
  tools?: string[]; // mcp_tools_cache (names only, for the count)
}

export interface MCPTool {
  name: string;
  description: string;
}

// ---- Skills ----
// Fields align with the agentskills.io open specification.
// description covers both "what the skill does" and "when to use it".
export interface SkillItem {
  name: string; // unique key = directory name
  description?: string; // required per spec; covers what + when to use
  license?: string; // optional: SPDX identifier or free text
  compatibility?: string; // optional: environment requirements
  mcps?: string[]; // MCP server names this skill unlocks on load
  files: string[]; // files in the skill directory
  // Estadísticas de llamadas（skill_usage Libro mayor). nunca llamé skill：calls=0、last_used predeterminado。
  calls: number;
  tasks: number; // Número de tareas que lo han cargado（chat Las sesiones no se cuentan）
  usage_agents: string[]; // lo cargué agent key
  last_used?: string;
}

// SkillCall Fue una vez Skill() Llamada (única skill Lista de llamadas recientes）。
export interface SkillCall {
  ts: string;
  agent_key: string;
  task_id: number; // 0 = Escenario sin tareas (sesión de diálogo）
  session_id: string;
  args_len: number;
}

// MissingSkill tiene nombre pero no existe skill —— "Quieres usarlo pero no lo tienes"Brecha de。
export interface MissingSkill {
  skill: string;
  calls: number;
  agents: string[];
  last_used?: string;
}

// ---- Tools (directorio de herramientas incorporado) ----
// key + handler live in Go; only these fields are page-editable. system tools lock
// the key and the parameter *structure* (name/type/required) — the per-param
// description/default and the agent binding are what move.
export interface Tool {
  key: string;
  system: boolean;
  description: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  schema: Record<string, any>; // full JSON-Schema (object with properties)
  agents: string[]; // bound agent keys
  enabled: boolean;
  kind?: "builtin" | "shell" | "command" | "script" | "http"; // tipo de herramienta personalizada
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  exec?: Record<string, any>; // Especificaciones de ejecución de herramientas personalizadas(kind!=builtin)
  deferred?: boolean; // schema Retraso(SearchExtraTools/ExecuteExtraTool)
  calls?: number; // persistent runtime invocation count (older APIs may omit it)
}

// ---- Stats ----
export interface Stats {
  assets: number;
  engine_mode: EngineMode;
  llm_configured: boolean;
  roe_enabled: boolean;
  findings_confirmed: number;
  active_task?: Partial<Task>;
}

// ---- Intercept Rules ----
export type InterceptAction = "allow" | "deny" | "ask";
export type InterceptMatchTarget = "tool_name" | "tool_input";
export type InterceptMatchType = "string" | "regex";

export interface InterceptRule {
  id: number;
  name: string;
  enabled: boolean;
  priority: number;
  match_target: InterceptMatchTarget;
  match_type: InterceptMatchType;
  pattern: string;
  action: InterceptAction;
  message: string;
  timeout_enabled: boolean;
  timeout_seconds: number;
  timeout_action: "deny" | "allow";
  created_at: string;
  updated_at: string;
}

// ---- Asset Intercept Rules（Interceptación de activos: Lista negra global） ----
export type AssetInterceptKind =
  | "exact_domain"
  | "exact_ip"
  | "exact_url"
  | "fuzzy_domain"
  | "fuzzy_ip"
  | "fuzzy_url"
  | "cidr";

// action solo para reglas a nivel de tarea：block=Intercepción(Desactivar pruebas) allow=Permitido(Lista blanca)。
export type AssetInterceptAction = "block" | "allow";

// interceptación de activos a nivel de tarea/Permitir la entrada de reglas (crear tareas, editar detalles de tareas para su uso)）。
export interface AssetInterceptRuleInput {
  action: AssetInterceptAction;
  kind: AssetInterceptKind;
  pattern: string;
  note: string;
  enabled: boolean;
}

export interface AssetInterceptRule {
  id: number;
  enabled: boolean;
  action?: AssetInterceptAction; // Las reglas globales no tienen este campo (siempre interceptación); Se distinguen las reglas a nivel de tarea. block/allow
  kind: AssetInterceptKind;
  pattern: string;
  note: string;
  builtin: boolean;
  created_at: string;
  updated_at: string;
}

export interface InterceptPending {
  decision_source?: "rule" | "model" | "unknown" | "";
  id: number;
  rule_id?: number;
  conversation_id?: number;
  task_id?: string;
  agent_name: string;
  tool_name: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  tool_input: Record<string, any>;
  status: "pending" | "allowed" | "denied" | "timeout";
  reason: string; // Reglas message O razón de juicio modelo(zona de juicio modelo [modelo] prefijo)
  decided_at?: string;
  created_at: string;
}

// JudgeConfig: Aprobación de modelo(Determinado por el modelo sólo si no se cumplen las reglas de intercepción)Configuración global de。
export interface JudgeConfig {
  enabled: boolean;
  profile_id: number; // 0 = Seguir activación/configuración predeterminada
  prompt: string; // Palabra de indicación de determinación;GET Cuando no está configurado, el backend rellena el texto completo de la plantilla incorporada.
  timeout_seconds: number; // tiempo de espera de llamada del modelo
  fail_action: "allow" | "ask" | "deny"; // error de modelo/tiempo de espera/retroceso cuando no se puede resolver
  ask_timeout_seconds: number; // Juicio modelo ask Tiempo de espera de aprobación después de la transferencia al manual
  ask_timeout_action: "allow" | "deny"; // Acción predeterminada después del tiempo de espera de aprobación
}

// JudgeUsage: Aprobación de modelo(judge canal)Acumulación de token Dosis + cerca N Secuencia diaria de días。
export interface JudgeDayUsage {
  date: string; // YYYY-MM-DD (UTC)
  calls: number;
  input_tokens: number;
  output_tokens: number;
}
export interface JudgeUsage {
  calls: number;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
  daily: JudgeDayUsage[];
}

export interface InterceptApprovalFilter {
  status?: InterceptPending["status"];
  decision_source?: "rule" | "model" | "unknown";
}

// InterceptApprovalRow enriches InterceptPending with conversation/task and rule context.
export interface InterceptApprovalRow extends InterceptPending {
  conv_title: string; // "" if no linked conversation
  conv_agent_key: string; // "" if no linked conversation
  rule_name: string; // "" if rule was deleted
}

// ── sincronización de activos (ScopeSentry fuente de datos) ──────────────────────────────────────────────
export interface SSProject {
  id: string; // MongoDB ObjectID — used as filter.project
  name: string;
  logo?: string;
  AssetCount?: number;
  tag?: string;
}

export interface SSTask {
  id: string;
  name: string; // used as filter.task
  status?: number;
  progress?: number;
  creatTime?: string;
  endTime?: string;
}

// ConvTokenSummary — one conversation's token total (+ profile/date) for merging
// chat usage into the dashboard token stats. GET /api/tokens/conversations.
export interface ConvTokenSummary {
  llm_profile_id: number | null;
  created_at: string;
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
}

// ---- Command recording (Bash execution history) ----
export interface CommandRecord {
  id: number;
  exploration_id: number;
  worker: string;
  tool: string;
  command: string; // raw tool input (JSON)
  output: string;
  is_error: boolean;
  created_at: string;
}

// Estadísticas de llamadas de una sola herramienta.（/commands/stats）；errors El número de fallos.。
export interface ToolStat {
  tool: string;
  total: number;
  errors: number;
}

// ---- LLM recording ----
export interface LLMRecordItem {
  id: number;
  ts: string;
  model: string;
  profile_name: string;
  session_id: string;
  task_id: string;
  worker: string;
  latency_ms: number;
  input_tokens: number;
  output_tokens: number;
  cache_read: number;
  cache_write: number;
  status: string;
  error?: string;
}

export interface LLMRecordDetail extends LLMRecordItem {
  request_body: string;
  response_body: string;
  // provider envío y recepción reales HTTP Texto original: La solicitud es buildBody() Emitido completo body（Herramientas incluidas
  // schema），La respuesta es cruda SSE cuadro. arriba request_body/response_body Es una vista normalizada，
  // Herramienta descartada schema con tool_use Bloquear. El registro antiguo está vacío。
  raw_request?: string;
  raw_response?: string;
}

// One distinct task with its LLM-record count (task picker on the records page).
export interface LLMTask {
  task_id: string;
  count: number;
}

// The exact JSON sent to the review model, retained for all model verdicts.
export interface InterceptReviewInput {
  version: number;
  background?: {
    // worker_summary is retained only for immutable v2/v3 snapshots.
    source: "user_message" | "worker_summary";
    text: string;
    truncated?: boolean;
  };
  // Version 1 snapshots are immutable and remain readable in historical audits.
  task?: {
    task_id: number;
    description: string;
    goal: string;
    constraints: { id: number; kind: string; text: string; origin: string; created_at: number }[];
    truncated?: boolean;
  };
  working_directory?: string;
  worker_intent?: string;
  turn_input?: string;
  background_truncated?: boolean;
  // Legacy v1/v2 snapshots only; v3 never sends execution history.
  history?: {
    tool_use_id: string;
    tool: string;
    arguments_preview: string;
    result: string;
    status: "succeeded" | "failed";
    truncated?: boolean;
  }[];
  history_truncated?: boolean;
  correlation?: "exact" | "ambiguous" | "unavailable";
  tool_name: string;
  arguments: Record<string, unknown>;
}

// Immutable review snapshot plus separately recorded execution outcome.
export interface InterceptAudit {
  model_input?: InterceptReviewInput;
  model_input_digest?: string;
  run_id?: string;
  tool_use_id?: string;
  correlation: "exact" | "ambiguous" | "unavailable";
  input_digest: string;
  user_message: string;
  user_truncated?: boolean;
  context:
    | { kind: string; tool?: string; tool_use_id?: string; text: string; is_error?: boolean; truncated?: boolean }[]
    | null;
  context_truncated?: boolean;
  captured_at: string;
  model_fallback?: boolean;
  initial_action: "allow" | "ask" | "deny";
  initial_reason: string;
  effective_action?: "allow" | "deny";
  decision_reason?: string;
  rule_name?: string;
  config_digest?: string;
  profile_id?: number;
  execution_status: "not_started" | "not_executed" | "awaiting_result" | "succeeded" | "failed" | "unknown";
  output?: string;
  output_truncated?: boolean;
  execution_ended_at?: string;
}
export interface InterceptDetail extends InterceptApprovalRow {
  audit: InterceptAudit | null;
}

export type TrafficEvidenceRole = "baseline" | "proof" | "verification" | "supporting";
export interface TrafficEvidenceRef {
  traffic_id: string;
  role?: TrafficEvidenceRole;
  note?: string;
}
export interface TrafficEvidenceSnapshot {
  id: string;
  source_traffic_id: string;
  captured_at: number;
  url: string;
  method: string;
  status: number;
  content_type: string;
  req_head?: string;
  resp_head?: string;
  req_hash: string;
  resp_hash: string;
  req_len: number;
  resp_len: number;
}
export interface FindingTrafficBinding {
  id: string;
  finding_id: string;
  snapshot_id: string;
  role: TrafficEvidenceRole;
  note: string;
  position: number;
  created_at: string;
  snapshot: TrafficEvidenceSnapshot;
}
export interface FindingTraffic {
  finding_id: string;
  version: number;
  report_version: number;
  bindings: FindingTrafficBinding[];
}
export interface EvidenceBodyPreview {
  content: string;
  offset: number;
  total: number;
  next_offset: number;
  truncated: boolean;
  binary: boolean;
}
export interface FindingTrafficDetail {
  binding: FindingTrafficBinding;
  request: EvidenceBodyPreview;
  response: EvidenceBodyPreview;
}

/** GET /api/update/check —— La versión actual es la misma que GitHub Resultados comparativos de la última versión oficial.。 */
export interface UpdateCheck {
  /** la versión actualmente en ejecución; las construcciones de desarrollo son "dev" o git describe Forma con sufijo de。 */
  current: string;
  /** estado de ejecución。docker La descarga y reinstalación solo afecta la capa de escritura del contenedor. La reconstrucción del contenedor devolverá la versión de la imagen.。 */
  mode: "docker" | "binary";
  os: string;
  arch: string;
  repo: string;
  /** ¿Existe una versión anterior que se pueda revertir?（artex.old）。 */
  has_backup: boolean;
  /** La conclusión del arranque de actualización automática durante este inicio (el reemplazo falló / revertido, etc.), vacío cuando no sucede nada。 */
  boot_notice?: string;
  rolled_back?: boolean;
  /** consulta GitHub Da la razón cuando falla. En este momento, ninguno de los siguientes campos estará disponible.。 */
  error?: string;
  latest?: string;
  notes?: string;
  html_url?: string;
  published_at?: string;
  /** El nombre del paquete de lanzamiento correspondiente a la plataforma actual y el Release ¿Realmente lo trajiste?。 */
  asset?: string;
  asset_available?: boolean;
  size?: number;
  has_update?: boolean;
  /** Si los números de versión de ambas partes son comparables; la construcción de desarrollo es false，Las actualizaciones con un clic están deshabilitadas en este momento。 */
  comparable?: boolean;
  /** comparable para false Descripción del tiempo。 */
  reason?: string;
}

/** /api/update/stream progreso de actualización empujado。 */
export interface UpdateProgress {
  phase: "idle" | "downloading" | "verifying" | "extracting" | "staged" | "failed";
  /** Sólo la fase de descarga es significativa（0-100）；Las etapas restantes son -1。 */
  percent: number;
  message: string;
  version?: string;
  error?: string;
}

// Original execution selected from an approval, never submitted to the reviewer.
export interface InterceptExecution {
  conversation_id: number | null;
  task_id: string | null;
  session: string;
  seq: number;
  items: Activity[];
}
