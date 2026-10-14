"use client";

import { t } from '@/lib/i18n';

import * as React from "react";

import Link from "next/link";

import {
  DndContext,
  type DragEndEvent,
  DragOverlay,
  type DragStartEvent,
  KeyboardSensor,
  PointerSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import {
  ArchiveIcon,
  ArrowDownIcon,
  ArrowUpDownIcon,
  ArrowUpIcon,
  ChevronRightIcon,
  EyeIcon,
  FolderInputIcon,
  GripVerticalIcon,
  Loader2Icon,
  PaperclipIcon,
  PauseIcon,
  PencilIcon,
  PinIcon,
  PinOffIcon,
  PlayIcon,
  PlusIcon,
  SaveIcon,
  SearchIcon,
  SlidersHorizontalIcon,
  StarIcon,
  TagsIcon,
  Trash2Icon,
  Undo2Icon,
  XIcon,
} from "lucide-react";
import { toast } from "sonner";

import { AssetInterceptRulesEditor } from "@/components/asset-intercept-rules-editor";
import { StatusBadge } from "@/components/status-badge";
import { TablePagination } from "@/components/table-pagination";
import { TaskLLMProfileChain } from "@/components/task-llm-profile-chain";
import { TaskTemplateControls } from "@/components/task-template-controls";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
} from "@/components/ui/combobox";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSet,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Item, ItemActions, ItemContent, ItemDescription, ItemGroup, ItemMedia, ItemTitle } from "@/components/ui/item";
import { Label } from "@/components/ui/label";
import { Progress } from "@/components/ui/progress";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { api } from "@/lib/api";
import { getLocalStorageValue, setLocalStorageValue } from "@/lib/local-storage.client";
import { type SortDirection, useStoredSortPreference } from "@/lib/sort-preference";
import type {
  AssetInterceptRuleInput,
  ChatAttachment,
  Company,
  DeleteTaskOptions,
  DeleteTaskResult,
  LLMProfile,
  Task,
  TaskArchive,
  TaskArchiveState,
  TaskCategory,
  TaskStatus,
} from "@/lib/types";
import { cn } from "@/lib/utils";

// fmtBytes renders a human file size for the upload manifest.
function fmtBytes(n: number): string {
  if (n >= 1 << 20) return `${(n / (1 << 20)).toFixed(1)} MB`;
  if (n >= 1 << 10) return `${(n / (1 << 10)).toFixed(1)} KB`;
  return `${n} B`;
}

// UPLOAD_MARKER labels the auto-appended block of uploaded-file paths inside the task
// description, so re-uploads append under the same block instead of adding a new header.
const UPLOAD_MARKER = t("ui.15a86b98fef530d8");

// appendUploads folds newly-uploaded files' ABSOLUTE paths into the description as a
// Read/Bash-friendly manifest — the worker opens them by path. Keeps one marked block:
// first upload adds the header, later uploads append bullets under it.
function appendUploads(desc: string, atts: ChatAttachment[]): string {
  const bullets = atts.map((a) => `- ${a.abs ?? a.path}（${fmtBytes(a.size)}）`).join("\n");
  if (desc.includes(UPLOAD_MARKER)) {
    return `${desc.replace(/\s*$/, "")}\n${bullets}\n`;
  }
  const head = desc.trim() ? `${desc.replace(/\s*$/, "")}\n\n` : "";
  return `${head}${UPLOAD_MARKER} El trabajador se puede abrir mediante Read/Bash según la ruta: ${bullets}\n`;
}

// POLL_MS is the task-list refresh interval. Task state moves on the server (planner /
// worker), so the list has to be pulled; 10s is plenty for status / Progreso / token cambio.
const POLL_MS = 10_000;
const MAX_SOURCE_TASKS = 8;

// fmtTokens renders a compact token count (1234 → 1.2k, 2_000_000 → 2M).
function fmtTokens(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(n >= 10_000_000 ? 0 : 1) + "M";
  if (n >= 1000) return (n / 1000).toFixed(n >= 10000 ? 0 : 1) + "k";
  return String(n);
}

// fmtDuration renders a run duration in seconds as a compact human string.
function fmtDuration(sec: number): string {
  if (sec <= 0) return "—";
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const m = Math.floor((sec % 3600) / 60);
  const s = sec % 60;
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
}

// taskDuration is the task's run duration in seconds: created → now while running,
// created → completion for a finished task, else created → last activity. 0 when it
// never ran (no activity yet).
function taskDuration(task: Task, nowSec: number): number {
  const start = task.created_unix ?? 0;
  if (!start) return 0;
  const end =
    task.status === "running"
      ? nowSec
      : task.completed_unix && task.completed_unix > 0
        ? task.completed_unix
        : (task.last_activity_unix ?? 0);
  return end > start ? end - start : 0;
}

// deleteDetails takes only the countable part of the result so callers can pass an
// aggregate accumulated over a bulk delete.
type DeleteCounts = Omit<DeleteTaskResult, "deleted" | "cleanup_warning">;

function deleteDetails(result: DeleteCounts): string[] {
  const details: string[] = [];
  if (result.assets_deleted > 0) details.push(`Eliminar activo ${result.assets_deleted}artículo`);
  if (result.assets_detached > 0) details.push(`Disociar activos compartidos ${result.assets_detached}artículo`);
  if (result.traffic_deleted > 0) details.push(`Eliminar tráfico ${result.traffic_deleted}artículo`);
  if (result.files_deleted) details.push(t("ui.805fd6675c5a40ef"));
  if (result.findings_deleted > 0) details.push(`Eliminar vulnerabilidad ${result.findings_deleted}artículo`);
  if (result.llm_records_deleted > 0) details.push(`Eliminar registros de solicitud/respuesta de LLM ${result.llm_records_deleted}artículo`);
  return details;
}

function deleteSummary(result: DeleteTaskResult): string {
  const details = deleteDetails(result);
  return details.length > 0 ? `Tarea eliminada (${details.join("，")}）` : t("ui.1f35edb8b57e679b");
}

// fmtDateTime renders a unix-seconds timestamp as a compact local date-time
// (MM-DD HH:mm), or "—" when unset.
function fmtDateTime(unix?: number): string {
  if (!unix || unix <= 0) return "—";
  const d = new Date(unix * 1000);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
}

const STATUS_OPTIONS: { value: TaskStatus; label: string }[] = [
  { value: "created", label: t("ui.1bba71a51144f309") },
  { value: "queued", label: t("status.queued") },
  { value: "running", label: t("status.running") },
  { value: "paused", label: t("ui.0b5ba9cc57aedf6a") },
  { value: "done", label: t("status.completed") },
  { value: "failed", label: t("status.failed") },
  { value: "timeout", label: "tiempo agotado" },
];

// Select cadena vacía no aceptada value,Entonces ”Sin clasificar” está en filtros, nuevos formularios y movimientos por lotes
// Utilice este valor centinela de manera uniforme,Traducir al backend al enviar null。
const UNCATEGORIZED_VALUE = "uncategorized";

// se puede pausar = No definitivo y no pausado,Con el backend applyTaskControlWithCause El control de puerta de
// (done/failed/timeout es el estado final);paused antes de continuar. Los botones en línea y el control por lotes comparten este juicio,
// No habrá diferencias entre los dos bandos que uno pueda señalar y el otro no.。
const PAUSABLE_STATUSES = new Set<TaskStatus>(["created", "queued", "running"]);
const ARCHIVABLE_STATUSES = new Set<TaskStatus>(["paused", "done", "failed", "timeout"]);

function taskControlAction(status: TaskStatus): "pause" | "resume" | null {
  if (status === "paused") return "resume";
  if (PAUSABLE_STATUSES.has(status)) return "pause";
  return null;
}

type TaskSortField = "id" | "created" | "duration" | "status";

const TASK_SORT_FIELDS: readonly TaskSortField[] = ["id", "created", "duration", "status"];
const TASK_SORT_PREFERENCE_KEY = "nexus_ia_task_list_sort";
const TASK_FILTER_PREFERENCE_KEY = "nexus_ia_task_list_filters";

const TASK_STATUS_RANK = new Map(STATUS_OPTIONS.map((option, index) => [option.value, index]));

function taskIsPinned(task: Task): boolean {
  return task.pinned ?? Boolean(task.pinned_at);
}

function compareTaskIDs(left: string, right: string): number {
  return left.localeCompare(right, undefined, { numeric: true, sensitivity: "base" });
}

function taskCreatedUnix(task: Task): number {
  if (task.created_unix) return task.created_unix;
  const parsed = Date.parse(task.created_at);
  return Number.isNaN(parsed) ? 0 : Math.floor(parsed / 1000);
}

function compareTasks(left: Task, right: Task, field: TaskSortField, direction: SortDirection, nowSec: number): number {
  const leftPinned = taskIsPinned(left);
  const rightPinned = taskIsPinned(right);
  if (leftPinned !== rightPinned) return leftPinned ? -1 : 1;
  if (leftPinned && left.pinned_at !== right.pinned_at) {
    return (right.pinned_at ?? "").localeCompare(left.pinned_at ?? "");
  }

  let compared = 0;
  switch (field) {
    case "created":
      compared = taskCreatedUnix(left) - taskCreatedUnix(right);
      break;
    case "duration":
      compared = taskDuration(left, nowSec) - taskDuration(right, nowSec);
      break;
    case "status":
      compared = (TASK_STATUS_RANK.get(left.status) ?? 0) - (TASK_STATUS_RANK.get(right.status) ?? 0);
      break;
    default:
      compared = compareTaskIDs(left.id, right.id);
  }
  if (compared !== 0) return direction === "asc" ? compared : -compared;
  return -compareTaskIDs(left.id, right.id);
}

export default function TasksPage() {
  const [activeTab, setActiveTab] = React.useState("current");
  const [tasks, setTasks] = React.useState<Task[]>([]);
  const [categories, setCategories] = React.useState<TaskCategory[]>([]);
  const [categoriesLoaded, setCategoriesLoaded] = React.useState(false);
  const [query, setQuery] = React.useState("");
  const [statusFilter, setStatusFilter] = React.useState<TaskStatus | "all">("all");
  const [categoryFilter, setCategoryFilter] = React.useState("all");
  const [filtersHydrated, setFiltersHydrated] = React.useState(false);
  const [sortPreference, setSortPreference] = useStoredSortPreference(
    TASK_SORT_PREFERENCE_KEY,
    TASK_SORT_FIELDS,
    "id",
    "desc",
  );
  const { field: sortField, direction: sortDirection } = sortPreference;
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState(20);
  const [nowSec, setNowSec] = React.useState(() => Math.floor(Date.now() / 1000));
  const [batchControlling, setBatchControlling] = React.useState<"pause" | "resume" | null>(null);
  const [movingCategory, setMovingCategory] = React.useState(false);

  React.useEffect(() => {
    const raw = getLocalStorageValue(TASK_FILTER_PREFERENCE_KEY);
    if (raw) {
      try {
        const parsed = JSON.parse(raw) as { status?: unknown; category?: unknown };
        if (parsed.status === "all" || STATUS_OPTIONS.some((option) => option.value === parsed.status)) {
          setStatusFilter(parsed.status as TaskStatus | "all");
        }
        if (
          typeof parsed.category === "string" &&
          (parsed.category === "all" || parsed.category === UNCATEGORIZED_VALUE || /^\d+$/.test(parsed.category))
        ) {
          setCategoryFilter(parsed.category);
        }
      } catch {
        // Ignore malformed or legacy preferences and retain the defaults.
      }
    }
    setFiltersHydrated(true);
  }, []);

  React.useEffect(() => {
    if (!filtersHydrated) return;
    setLocalStorageValue(
      TASK_FILTER_PREFERENCE_KEY,
      JSON.stringify({ status: statusFilter, category: categoryFilter }),
    );
  }, [categoryFilter, filtersHydrated, statusFilter]);

  const filtered = React.useMemo(() => {
    const q = query.trim().toLowerCase();
    return tasks.filter((t) => {
      if (statusFilter !== "all" && t.status !== statusFilter) return false;
      if (categoryFilter === UNCATEGORIZED_VALUE && t.category_id != null) return false;
      if (
        categoryFilter !== "all" &&
        categoryFilter !== UNCATEGORIZED_VALUE &&
        String(t.category_id) !== categoryFilter
      )
        return false;
      if (!q) return true;
      return (
        (t.name ?? "").toLowerCase().includes(q) ||
        (t.category_name ?? "").toLowerCase().includes(q) ||
        t.description.toLowerCase().includes(q) ||
        t.goal.toLowerCase().includes(q) ||
        t.id.toLowerCase().includes(q)
      );
    });
  }, [tasks, query, statusFilter, categoryFilter]);

  const sortNowSec = sortField === "duration" ? nowSec : 0;
  const ordered = React.useMemo(() => {
    return [...filtered].sort((left, right) => compareTasks(left, right, sortField, sortDirection, sortNowSec));
  }, [filtered, sortDirection, sortField, sortNowSec]);

  const sortTasksBy = React.useCallback(
    (field: TaskSortField) => {
      setSortPreference((current) => {
        if (current.field !== field) return { field, direction: "desc" };
        return { field, direction: current.direction === "asc" ? "desc" : "asc" };
      });
    },
    [setSortPreference],
  );

  // reset to page 1 whenever filters or ordering change
  // biome-ignore lint/correctness/useExhaustiveDependencies: both filters intentionally reset pagination.
  React.useEffect(() => {
    setPage(1);
  }, [query, statusFilter, categoryFilter, sortField, sortDirection]);

  const paginated = React.useMemo(
    () => ordered.slice((page - 1) * pageSize, page * pageSize),
    [ordered, page, pageSize],
  );

  // Eliminar múltiples selecciones:Seleccionar paso de página transversal/Filtrar y retener,Sólo cuando la tarea realmente desaparece(Eliminado o el backend ya no volverá)Convergencia temporal。
  const [selectedIds, setSelectedIds] = React.useState<Set<string>>(() => new Set());

  React.useEffect(() => {
    setSelectedIds((prev) => {
      if (prev.size === 0) return prev;
      const live = new Set(tasks.map((t) => t.id));
      const next = new Set([...prev].filter((id) => live.has(id)));
      return next.size === prev.size ? prev : next;
    });
  }, [tasks]);

  const toggleSelected = React.useCallback((id: string, checked: boolean) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  }, []);

  const pageIds = React.useMemo(() => paginated.map((t) => t.id), [paginated]);
  const pageSelectedCount = React.useMemo(
    () => pageIds.filter((id) => selectedIds.has(id)).length,
    [pageIds, selectedIds],
  );
  let headerChecked: boolean | "indeterminate" = false;
  if (pageIds.length > 0 && pageSelectedCount === pageIds.length) {
    headerChecked = true;
  } else if (pageSelectedCount > 0) {
    headerChecked = "indeterminate";
  }

  const toggleSelectedPage = React.useCallback(
    (checked: boolean) => {
      setSelectedIds((prev) => {
        const next = new Set(prev);
        for (const id of pageIds) {
          if (checked) next.add(id);
          else next.delete(id);
        }
        return next;
      });
    },
    [pageIds],
  );

  // lastRef holds the previous poll's serialized payload: the list is re-fetched every
  // POLL_MS but usually comes back unchanged, and setTasks on an identical payload would
  // re-render the whole page for nothing. Bail out when it matches.
  const lastRef = React.useRef<string>("");

  const load = React.useCallback(() => {
    api
      .tasks()
      .then((r) => {
        const next = r.tasks.map((t) => (t.id === r.active ? { ...t, active: true } : t));
        const sig = JSON.stringify(next);
        if (sig === lastRef.current) return;
        lastRef.current = sig;
        setTasks(next);
      })
      .catch(() => {
        // Polling is best-effort; the next interval retries automatically.
      });
  }, []);

  const loadCategories = React.useCallback(() => {
    api
      .taskCategories()
      .then((next) => {
        setCategories(next);
        setCategoriesLoaded(true);
      })
      .catch(() => {
        // Category management remains retryable without blocking the task list.
      });
  }, []);

  React.useEffect(() => {
    load();
    loadCategories();
    const i = setInterval(load, POLL_MS);
    return () => clearInterval(i);
  }, [load, loadCategories]);

  const refreshCategoriesAndTasks = React.useCallback(() => {
    lastRef.current = "";
    loadCategories();
    load();
  }, [load, loadCategories]);

  const applyTaskCategoryMove = React.useCallback(
    (taskID: string, category: TaskCategory | null) => {
      setTasks((current) => {
        const next = current.map((task) =>
          task.id === taskID ? { ...task, category_id: category?.id, category_name: category?.name } : task,
        );
        lastRef.current = JSON.stringify(next);
        return next;
      });
      loadCategories();
    },
    [loadCategories],
  );

  React.useEffect(() => {
    if (!categoriesLoaded) return;
    if (categoryFilter === "all" || categoryFilter === UNCATEGORIZED_VALUE) return;
    if (!categories.some((category) => String(category.id) === categoryFilter)) setCategoryFilter("all");
  }, [categories, categoriesLoaded, categoryFilter]);

  // Sólo si hay running tiempo de tarea por segundo tick——En otros casos, el ”tiempo de ejecución” es un valor estático,ralentí tick Será en vano
  // volver a renderizar toda la tabla。
  const hasRunning = React.useMemo(() => tasks.some((t) => t.status === "running"), [tasks]);

  // tick every second so running tasks' tiempo de ejecución counts up live.
  React.useEffect(() => {
    if (!hasRunning) return;
    setNowSec(Math.floor(Date.now() / 1000));
    const i = setInterval(() => setNowSec(Math.floor(Date.now() / 1000)), 1000);
    return () => clearInterval(i);
  }, [hasRunning]);

  const deleteTask = React.useCallback(
    async (id: string, options: DeleteTaskOptions) => {
      try {
        const result = await api.deleteTask(id, options);
        if (result.cleanup_warning) {
          toast.warning(`${deleteSummary(result)}; Parte de la limpieza de datos externos no se completa:${result.cleanup_warning}`);
        } else {
          toast.success(deleteSummary(result));
        }
        load();
      } catch (e) {
        toast.error(t("ui.ac60bde833d4bb98") + (e as Error).message);
        throw e;
      }
    },
    [load],
  );

  // controlTask Es una pausa en línea/continuar:Dejar en lotes controlTasksBatch,interfaz de tarea única de un solo recorrido,omitir
  // 「Verifique primero y luego haga clic en Lote.” Borrar lastRef Deja que la siguiente encuesta reciba el mismo orden incluso si la carga es la misma,Estado de lo contrario
  // La deduplicación bloqueará la reescritura y el botón parecerá no responder.。
  const controlTask = React.useCallback(
    async (id: string, action: "pause" | "resume") => {
      try {
        const result = await api.controlTask(id, action);
        toast.success(
          action === "pause" ? `Misión #${id}en pausa` : `Misión #${id} continúa${result.queued ? t("ui.f9110aaea90d2113") : ""}`,
        );
      } catch (e) {
        toast.error(`${action === "pause" ? t("status.paused") : t("ui.4eceb65dabcf2b7f")}Fallo:${(e as Error).message}`);
      } finally {
        // Actualizar independientemente del éxito o el fracaso:La falla se debe principalmente al cambio de estado.,Tire de él nuevamente para devolver el botón a su forma correcta.。
        lastRef.current = "";
        load();
      }
    },
    [load],
  );

  const renameTask = React.useCallback(
    async (task: Task, name: string) => {
      try {
        await api.renameTask(task.id, name);
        toast.success(`Misión #${task.id}renombrado`);
        lastRef.current = "";
        load();
      } catch (error) {
        toast.error(`Error al cambiar el nombre:${(error as Error).message}`);
        throw error;
      }
    },
    [load],
  );

  const toggleTaskPinned = React.useCallback(
    async (task: Task) => {
      const pinned = taskIsPinned(task);
      try {
        await api.pinTask(task.id, !pinned);
        toast.success(pinned ? `Misión #${task.id}Pin eliminado` : `Misión #${task.id}Fijado`);
        lastRef.current = "";
        load();
      } catch (error) {
        toast.error(`${pinned ? t("ui.4eea160fb2c96650") : t("ui.5596cd47498d7075")}Fallo:${(error as Error).message}`);
        throw error;
      }
    },
    [load],
  );

  const queueTaskArchive = React.useCallback(
    async (task: Task) => {
      try {
        await api.archiveTask(task.id);
        toast.success(`Misión #${task.id}Ingresó a la cola de archivo`);
        setActiveTab("archived");
        lastRef.current = "";
        load();
      } catch (error) {
        toast.error(`Error al archivar:${(error as Error).message}`);
        throw error;
      }
    },
    [load],
  );

  // deleteTasks Eliminar las tareas seleccionadas una por una:El backend no tiene interfaz por lotes,Y una única eliminación también limpiará los activos./Tráfico/Archivo,
  // Ejecución en serie para evitar hacer estallar el backend de una vez;Eliminado exitosamente del conjunto seleccionado,Fallo retenido para reintento。
  const deleteTasks = React.useCallback(
    async (ids: string[], options: DeleteTaskOptions, onProgress: (done: number) => void) => {
      const total: DeleteCounts = {
        assets_deleted: 0,
        assets_detached: 0,
        traffic_deleted: 0,
        files_deleted: false,
        findings_deleted: 0,
        llm_records_deleted: 0,
      };
      const deleted: string[] = [];
      const failed: { id: string; message: string }[] = [];
      const warnings: string[] = [];

      for (const id of ids) {
        try {
          const r = await api.deleteTask(id, options);
          total.assets_deleted += r.assets_deleted;
          total.assets_detached += r.assets_detached;
          total.traffic_deleted += r.traffic_deleted;
          total.files_deleted = total.files_deleted || r.files_deleted;
          total.findings_deleted += r.findings_deleted;
          total.llm_records_deleted += r.llm_records_deleted;
          if (r.cleanup_warning) warnings.push(`#${id}：${r.cleanup_warning}`);
          deleted.push(id);
        } catch (e) {
          failed.push({ id, message: (e as Error).message });
        }
        onProgress(deleted.length + failed.length);
      }

      if (deleted.length > 0) {
        setSelectedIds((prev) => {
          const next = new Set(prev);
          for (const id of deleted) next.delete(id);
          return next;
        });
        const details = deleteDetails(total);
        const summary = `eliminado ${deleted.length}tareas` + (details.length > 0 ? `（${details.join("，")}）` : "");
        if (warnings.length > 0) {
          toast.warning(`${summary}; Parte de la limpieza de datos externos no se completa:${warnings.join("；")}`);
        } else {
          toast.success(summary);
        }
      }
      if (failed.length > 0) {
        const head = failed
          .slice(0, 3)
          .map((f) => `#${f.id}（${f.message}）`)
          .join("；");
        toast.error(`${failed.length} Error al eliminar la tarea:${head}${failed.length > 3 ? t("ui.020c5079d229d923") : ""}`);
      }
      load();
    },
    [load],
  );

  const selectedTasks = React.useMemo(() => tasks.filter((task) => selectedIds.has(task.id)), [tasks, selectedIds]);
  const pausableTaskIDs = React.useMemo(
    () => selectedTasks.filter((task) => taskControlAction(task.status) === "pause").map((task) => task.id),
    [selectedTasks],
  );
  const resumableTaskIDs = React.useMemo(
    () => selectedTasks.filter((task) => taskControlAction(task.status) === "resume").map((task) => task.id),
    [selectedTasks],
  );
  const archivableTaskIDs = React.useMemo(
    () => selectedTasks.filter((task) => ARCHIVABLE_STATUSES.has(task.status) && !task.queued).map((task) => task.id),
    [selectedTasks],
  );

  const archiveSelectedTasks = React.useCallback(async () => {
    if (archivableTaskIDs.length === 0) return;
    const result = await api.archiveTasks(archivableTaskIDs);
    const succeeded = result.items.filter((item) => item.ok);
    const failed = result.items.filter((item) => !item.ok);
    if (succeeded.length > 0) toast.success(`ya ${succeeded.length}Tareas agregadas a la cola de archivo`);
    if (failed.length > 0) {
      toast.error(
        `${failed.length} Las tareas no se pueden archivar:${failed
          .slice(0, 3)
          .map((item) => `#${item.id}（${item.error || t("ui.d27b858f3d314d9e")}）`)
          .join("；")}`,
      );
    }
    setSelectedIds(new Set());
    setActiveTab("archived");
    lastRef.current = "";
    load();
  }, [archivableTaskIDs, load]);

  const controlSelectedTasks = React.useCallback(
    async (action: "pause" | "resume", ids: string[]) => {
      if (ids.length === 0 || batchControlling) return;
      if (ids.length > 100) {
        toast.error(t("ui.315d132b0ba29ff5"));
        return;
      }
      setBatchControlling(action);
      try {
        const result = await api.controlTasksBatch(ids, action);
        const succeeded = result.items.filter((item) => item.ok);
        const failed = result.items.filter((item) => !item.ok);
        if (succeeded.length > 0) {
          toast.success(
            action === "pause"
              ? `en pausa ${succeeded.length}tareas`
              : `Continuación ${succeeded.length} tareas${succeeded.some((item) => item.queued) ? t("ui.1200d319da48b36d") : ""}`,
          );
        }
        if (failed.length > 0) {
          const details = failed
            .slice(0, 3)
            .map((item) => `#${item.id}（${item.error || t("ui.d27b858f3d314d9e")}）`)
            .join("；");
          toast.error(`${failed.length} Error en la operación de tarea:${details}${failed.length > 3 ? t("ui.020c5079d229d923") : ""}`);
        }
        lastRef.current = "";
        load();
      } catch (error) {
        toast.error(`${action === "pause" ? t("ui.23e01b1b72d84b2a") : t("ui.916c20408a8f9c9b")}Fallo:${(error as Error).message}`);
      } finally {
        setBatchControlling(null);
      }
    },
    [batchControlling, load],
  );

  // El backend coloca todo el lote de escrituras clasificadas en una transacción, por lo que los elementos fallidos solo pueden ser tareas que se verifican y luego se eliminan.。
  const moveSelectedTasksCategory = React.useCallback(
    async (categoryID?: number) => {
      const ids = [...selectedIds];
      if (ids.length === 0 || movingCategory) return;
      if (ids.length > 100) {
        toast.error(t("ui.555d5ed9f4cd5792"));
        return;
      }
      setMovingCategory(true);
      try {
        const result = await api.updateTasksCategory(ids, categoryID);
        const succeeded = result.items.filter((item) => item.ok);
        const failed = result.items.filter((item) => !item.ok);
        const target = result.category?.name ?? "Sin categoría";
        if (succeeded.length > 0) {
          toast.success(`ya ${succeeded.length} Tareas movidas a "${target}」`);
          setSelectedIds(new Set());
        }
        if (failed.length > 0) {
          const details = failed
            .slice(0, 3)
            .map((item) => `#${item.id}（${item.error || t("ui.b8801c0c2dae2a7b")}）`)
            .join("；");
          toast.error(`${failed.length} Las tareas no se pudieron mover:${details}${failed.length > 3 ? t("ui.020c5079d229d923") : ""}`);
        }
        refreshCategoriesAndTasks();
      } catch (error) {
        toast.error(`No se pudo modificar la clasificación:${(error as Error).message}`);
      } finally {
        setMovingCategory(false);
      }
    },
    [movingCategory, refreshCategoriesAndTasks, selectedIds],
  );

  return (
    <Tabs value={activeTab} onValueChange={setActiveTab} className="gap-4">
      <TabsList className="mx-4 lg:mx-6">
        <TabsTrigger value="current">{t("ui.9afdbc0af4c775e5")}</TabsTrigger>
        <TabsTrigger value="archived">{t("ui.eac5386d4211a8c5")}</TabsTrigger>
      </TabsList>
      <TabsContent value="current">
        <Card>
          <CardContent className="flex flex-col gap-4 px-0 pt-6">
            <div className="flex flex-wrap items-center gap-2 px-4 lg:px-6">
              <div className="relative w-full sm:max-w-xs">
                <SearchIcon className="text-muted-foreground pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2" />
                <Input
                  placeholder={t("ui.0e2a51ce5668a132")}
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  className="pl-8"
                />
                {query && (
                  <button
                    type="button"
                    onClick={() => setQuery("")}
                    aria-label={t("ui.175e7c675f4a9fc0")}
                    className="text-muted-foreground hover:text-foreground absolute top-1/2 right-2 -translate-y-1/2"
                  >
                    <XIcon className="size-4" />
                  </button>
                )}
              </div>
              <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as TaskStatus | "all")}>
                <SelectTrigger className="w-36">
                  <SelectValue placeholder={t("common.status")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="all">{t("ui.ecda92faab01f5c1")}</SelectItem>
                    {STATUS_OPTIONS.map((s) => (
                      <SelectItem key={s.value} value={s.value}>
                        {s.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <Select value={categoryFilter} onValueChange={setCategoryFilter}>
                <SelectTrigger className="w-40">
                  <SelectValue placeholder={t("ui.62d216c0d8b232d5")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="all">{t("ui.e7d100e4d3d80e89")}</SelectItem>
                    <SelectItem value={UNCATEGORIZED_VALUE}>Sin categoría</SelectItem>
                    {categories.map((category) => (
                      <SelectItem key={category.id} value={String(category.id)}>
                        {category.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
              <span className="text-muted-foreground text-xs tabular-nums">
                {filtered.length}/{tasks.length}  {t("ui.a1cb6bd814e4e98b")} </span>
              {selectedIds.size > 0 && (
                <>
                  <span className="text-xs tabular-nums">{t("ui.df9a55ed63d1515b")} {selectedIds.size} </span>
                  <Button size="sm" variant="ghost" onClick={() => setSelectedIds(new Set())}>
                     {t("ui.708f57bc8c81b827")} </Button>
                  {pausableTaskIDs.length > 0 && (
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={batchControlling !== null}
                      onClick={() => void controlSelectedTasks("pause", pausableTaskIDs)}
                    >
                      {batchControlling === "pause" ? (
                        <Spinner data-icon="inline-start" />
                      ) : (
                        <PauseIcon data-icon="inline-start" />
                      )}
                       {t("ui.a04f890b2c4261ad")} {pausableTaskIDs.length}
                    </Button>
                  )}
                  {resumableTaskIDs.length > 0 && (
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={batchControlling !== null}
                      onClick={() => void controlSelectedTasks("resume", resumableTaskIDs)}
                    >
                      {batchControlling === "resume" ? (
                        <Spinner data-icon="inline-start" />
                      ) : (
                        <PlayIcon data-icon="inline-start" />
                      )}
                       {t("ui.4eceb65dabcf2b7f")} {resumableTaskIDs.length}
                    </Button>
                  )}
                  {archivableTaskIDs.length > 0 && (
                    <ArchiveConfirmDialog
                      count={archivableTaskIDs.length}
                      onConfirm={archiveSelectedTasks}
                      trigger={
                        <Button size="sm" variant="outline">
                          <ArchiveIcon data-icon="inline-start" />
                           {t("ui.75d6b4579e1d29eb")} {archivableTaskIDs.length}
                        </Button>
                      }
                    />
                  )}
                  <MoveTasksCategoryDialog
                    categories={categories}
                    count={selectedIds.size}
                    moving={movingCategory}
                    onMove={moveSelectedTasksCategory}
                  />
                  <BulkDeleteTasksDialog ids={[...selectedIds]} onDelete={deleteTasks} />
                </>
              )}
              <ConcurrencySettingsDialog />
              <CategoryManagementSheet
                categories={categories}
                tasks={tasks}
                onChanged={refreshCategoriesAndTasks}
                onTaskMoved={applyTaskCategoryMove}
              />
              <CreateTaskSheet
                tasks={tasks}
                categories={categories}
                onCreated={refreshCategoriesAndTasks}
                onCategoriesChanged={refreshCategoriesAndTasks}
              />
            </div>

            {tasks.length === 0 ? (
              <div className="text-muted-foreground mx-4 flex items-center justify-center rounded-lg border border-dashed py-20 text-sm lg:mx-6">
                 {t("ui.82382f1467ce1299")} </div>
            ) : filtered.length === 0 ? (
              <div className="text-muted-foreground mx-4 flex items-center justify-center rounded-lg border border-dashed py-20 text-sm lg:mx-6">
                 {t("ui.d07b9fad48f8ecd8")} </div>
            ) : (
              <Table className="**:data-[slot='table-cell']:px-4 **:data-[slot='table-head']:px-4">
                <TableHeader className="[&_tr]:border-t">
                  <TableRow>
                    <TableHead className="w-10">
                      <Checkbox
                        checked={headerChecked}
                        onCheckedChange={(checked) => toggleSelectedPage(checked === true)}
                        aria-label={t("ui.066ed4082e2d439e")}
                      />
                    </TableHead>
                    <SortableTaskHead
                      field="id"
                      label="ID"
                      activeField={sortField}
                      direction={sortDirection}
                      className="font-mono"
                      onSort={sortTasksBy}
                    />
                    <TableHead>{t("common.name")}</TableHead>
                    <TableHead>{t("common.description")}</TableHead>
                    <TableHead>{t("ui.dfe62c1beb52d3b3")}</TableHead>
                    <SortableTaskHead
                      field="status"
                      label={t("common.status")}
                      activeField={sortField}
                      direction={sortDirection}
                      onSort={sortTasksBy}
                    />
                    <TableHead className="text-center">{t("ui.c343709d855ec5b8")}</TableHead>
                    <TableHead className="text-center" title={t("ui.2b12299336e9564e")}>
                       {t("ui.4af9f64073ee2fa9")} <span className="text-muted-foreground font-normal">{t("ui.1542d231b0016cdd")}</span>
                    </TableHead>
                    <TableHead className="text-center">{t("ui.ef96f3af6ae56d66")}</TableHead>
                    <SortableTaskHead
                      field="created"
                      label={t("common.created_at")}
                      activeField={sortField}
                      direction={sortDirection}
                      align="right"
                      onSort={sortTasksBy}
                    />
                    <SortableTaskHead
                      field="duration"
                      label={t("ui.ad08d4b56f40b630")}
                      activeField={sortField}
                      direction={sortDirection}
                      align="right"
                      onSort={sortTasksBy}
                    />
                    <TableHead className="text-right">Token</TableHead>
                    <TableHead className="sticky right-0 z-10 bg-card text-right shadow-[-1px_0_0_0_hsl(var(--border))]">
                       {t("ui.1bf216d1df2388b7")} </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {paginated.map((task) => (
                    <TaskRow
                      key={task.id}
                      task={task}
                      // running come solo cuando estés en una misión nowSec;El resto de los Actos 0 —— props sin cambios,memo puede detenerse cada segundo tick
                      // Se vuelve a renderizar toda la tabla,Permitir el tiempo de caminata solo para quienes estén corriendo。
                      nowSec={task.status === "running" ? nowSec : 0}
                      onDelete={deleteTask}
                      onControl={controlTask}
                      onRename={renameTask}
                      onTogglePinned={toggleTaskPinned}
                      onArchive={queueTaskArchive}
                      selected={selectedIds.has(task.id)}
                      onSelectedChange={toggleSelected}
                    />
                  ))}
                </TableBody>
              </Table>
            )}
            <TablePagination
              page={page}
              pageSize={pageSize}
              total={filtered.length}
              onPageChange={setPage}
              onPageSizeChange={(nextPageSize) => {
                setPageSize(nextPageSize);
                setPage(1);
              }}
            />
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="archived">
        <TaskArchivesPanel onChanged={load} />
      </TabsContent>
    </Tabs>
  );
}

function taskSortIcon(active: boolean, direction: SortDirection) {
  if (!active) {
    return <ArrowUpDownIcon className="size-3.5 opacity-40 transition-opacity group-hover/sort:opacity-100" />;
  }
  if (direction === "asc") return <ArrowUpIcon className="size-3.5" />;
  return <ArrowDownIcon className="size-3.5" />;
}

function SortableTaskHead({
  field,
  label,
  activeField,
  direction,
  align = "left",
  className,
  onSort,
}: {
  field: TaskSortField;
  label: string;
  activeField: TaskSortField;
  direction: SortDirection;
  align?: "left" | "right";
  className?: string;
  onSort: (field: TaskSortField) => void;
}) {
  const active = activeField === field;
  let ariaSort: React.AriaAttributes["aria-sort"] = "none";
  if (active) ariaSort = direction === "asc" ? "ascending" : "descending";

  let actionLabel = `prensa${label}ordenar en orden inverso`;
  if (active) actionLabel = `${label}actual${direction === "asc" ? "secuencia positiva" : "orden inverso"}, haga clic para cambiar la dirección de clasificación`;

  return (
    <TableHead className={className} aria-sort={ariaSort}>
      <button
        type="button"
        className={cn(
          "group/sort inline-flex h-full w-full items-center gap-1 outline-none focus-visible:underline",
          align === "right" && "justify-end",
        )}
        aria-label={actionLabel}
        onClick={() => onSort(field)}
      >
        <span>{label}</span>
        {taskSortIcon(active, direction)}
      </button>
    </TableHead>
  );
}

function ConcurrencySettingsDialog() {
  const [open, setOpen] = React.useState(false);
  const [enabled, setEnabled] = React.useState(false);
  const [limit, setLimit] = React.useState("5");
  const [loading, setLoading] = React.useState(false);
  const [saving, setSaving] = React.useState(false);

  React.useEffect(() => {
    if (!open) return;
    setLoading(true);
    api
      .settings()
      .then((settings) => {
        setEnabled(!!settings.task_concurrency_enabled);
        setLimit(String(settings.task_concurrency_limit ?? 5));
      })
      .catch(() => {
        // Keep the dialog usable with defaults; reopening retries the request.
      })
      .finally(() => setLoading(false));
  }, [open]);

  async function save() {
    const nextLimit = Math.max(1, Math.floor(Number(limit) || 5));
    setSaving(true);
    try {
      await api.setSettings({ task_concurrency_enabled: enabled, task_concurrency_limit: nextLimit });
      toast.success(enabled ? `Límite de concurrencia habilitado: máximo de operaciones simultáneas ${nextLimit}tareas` : t("ui.095e755cd0861819"));
      setOpen(false);
    } catch (error) {
      toast.error(`Error al guardar:${(error as Error).message}`);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm" variant="outline" className="ml-auto" aria-label={t("ui.7e2a1c404a09e894")}>
          <SlidersHorizontalIcon />  {t("ui.a2a5a5bb160166d2")} </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("ui.898a806eeaa6c992")}</DialogTitle>
          <DialogDescription>
             {t("ui.82a66caaa434c343")} </DialogDescription>
        </DialogHeader>
        <div className="grid gap-5 py-2">
          <div className="flex items-center justify-between gap-4">
            <div className="grid gap-1">
              <Label htmlFor="task-concurrency-enabled">{t("ui.279cd059e2f67209")}</Label>
              <span className="text-muted-foreground text-xs">{t("ui.0e8744fdba4c0ddd")}</span>
            </div>
            <Switch id="task-concurrency-enabled" checked={enabled} onCheckedChange={setEnabled} disabled={loading} />
          </div>
          {enabled && (
            <div className="grid gap-2">
              <Label htmlFor="task-concurrency-limit">{t("ui.0d52f05880657c02")}</Label>
              <Input
                id="task-concurrency-limit"
                type="number"
                min={1}
                className="w-32"
                value={limit}
                onChange={(event) => setLimit(event.target.value)}
                disabled={loading}
              />
            </div>
          )}
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">{t("common.cancel")}</Button>
          </DialogClose>
          <Button onClick={save} disabled={loading || saving}>
            {saving && <Loader2Icon className="animate-spin" />} guardar </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// TaskRow renders one row of the task table. Memoized so the per-second tiempo de ejecución tick and
// the POLL_MS list refresh only re-render the rows whose data actually moved — a table page
// is 20 rows × (StatusBadge + Link + a Radix AlertDialog tree), far too heavy to rebuild
// wholesale on every parent render.
const TaskRow = React.memo(function TaskRow({
  task,
  nowSec,
  onDelete,
  onControl,
  onRename,
  onTogglePinned,
  onArchive,
  selected,
  onSelectedChange,
}: {
  task: Task;
  nowSec: number;
  onDelete: (id: string, options: DeleteTaskOptions) => Promise<void>;
  onControl: (id: string, action: "pause" | "resume") => Promise<void>;
  onRename: (task: Task, name: string) => Promise<void>;
  onTogglePinned: (task: Task) => Promise<void>;
  onArchive: (task: Task) => Promise<void>;
  selected: boolean;
  onSelectedChange: (id: string, checked: boolean) => void;
}) {
  return (
    <TableRow className="group border-border/60" data-state={selected ? "selected" : undefined}>
      <TableCell>
        <Checkbox
          checked={selected}
          onCheckedChange={(checked) => onSelectedChange(task.id, checked === true)}
          aria-label={`Seleccionar tarea ${task.id}`}
        />
      </TableCell>
      <TableCell>
        <code className="bg-muted rounded px-1.5 py-0.5 font-mono text-xs">{task.id}</code>
      </TableCell>
      <TableCell className="font-medium">
        <div className="flex max-w-xs items-center gap-2">
          <TaskNameEditor task={task} onRename={onRename} />
          {task.active && <StarIcon className="size-4 shrink-0 fill-amber-400 text-amber-400" />}
          {taskIsPinned(task) && <PinIcon className="text-primary size-4 shrink-0" aria-label={t("ui.3a7900b566d1cacb")} />}
        </div>
      </TableCell>
      <TableCell className="text-muted-foreground max-w-40">
        <Link
          href={`/function/tasks/detail?id=${encodeURIComponent(task.id)}`}
          className="block truncate rounded-sm hover:text-foreground hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          title={task.description}
        >
          {task.description}
        </Link>
      </TableCell>
      <TableCell className="text-muted-foreground max-w-40 truncate" title={task.goal}>
        {task.goal}
      </TableCell>
      <TableCell>
        <StatusBadge domain="task" value={task.status} dot />
      </TableCell>
      <TableCell className="text-muted-foreground text-center text-xs tabular-nums">
        {typeof task.goals_total === "number" && task.goals_total > 0 ? (
          `${task.goals_met}/${task.goals_total}`
        ) : (
          <span className="text-muted-foreground">—</span>
        )}
      </TableCell>
      <TableCell className="text-center text-xs tabular-nums">
        {(() => {
          const f = task.findings;
          const total = f ? f.critical + f.high + f.medium + f.low : 0;
          if (!f || total === 0) return <span className="text-muted-foreground">0</span>;
          const seg = (n: number, cls: string) => (
            <span className={n > 0 ? cls : "text-muted-foreground"}>{n}</span>
          );
          return (
            <span className="font-medium whitespace-nowrap" title={t("ui.2b12299336e9564e")}>
              {seg(f.critical, "text-rose-600 dark:text-rose-400")}
              <span className="text-muted-foreground">/</span>
              {seg(f.high, "text-red-600 dark:text-red-400")}
              <span className="text-muted-foreground">/</span>
              {seg(f.medium, "text-amber-600 dark:text-amber-400")}
              <span className="text-muted-foreground">/</span>
              {seg(f.low, "text-slate-600 dark:text-slate-400")}
            </span>
          );
        })()}
      </TableCell>
      <TableCell className="text-center text-xs tabular-nums">
        {task.in_flight && task.in_flight > 0 ? (
          <span className="text-foreground font-medium">{task.in_flight}</span>
        ) : (
          <span className="text-muted-foreground">0</span>
        )}
      </TableCell>
      <TableCell className="text-muted-foreground text-right text-xs whitespace-nowrap tabular-nums">
        {fmtDateTime(task.created_unix)}
      </TableCell>
      <TableCell className="text-right text-xs whitespace-nowrap tabular-nums">
        {(() => {
          const secs = taskDuration(task, nowSec);
          if (secs <= 0) return <span className="text-muted-foreground">—</span>;
          return (
            <span className={task.status === "running" ? "text-foreground" : "text-muted-foreground"}>
              {fmtDuration(secs)}
            </span>
          );
        })()}
      </TableCell>
      <TableCell
        className="text-right text-xs whitespace-nowrap tabular-nums"
        title={
          task.tokens
            ? `Entrada ${task.tokens.input_tokens} · Caché ${task.tokens.cache_read_tokens} · Salida ${task.tokens.output_tokens}`
            : undefined
        }
      >
        {task.tokens ? (
          <span className="text-muted-foreground">
             {t("ui.ff24436dc697b03f")} <span className="text-foreground">{fmtTokens(task.tokens.input_tokens)}</span>
            {t("ui.323b8bb346695a24")}
            <span className="text-foreground">{fmtTokens(task.tokens.cache_read_tokens)}</span>
            {t("ui.67b2687d6245c4e0")}
            <span className="text-foreground">{fmtTokens(task.tokens.output_tokens)}</span>
          </span>
        ) : (
          "—"
        )}
      </TableCell>
      <TableCell className="sticky right-0 z-10 bg-card text-right shadow-[-1px_0_0_0_hsl(var(--border))] group-hover:bg-muted/50">
        <div className="flex items-center justify-end gap-0.5">
          <Button size="icon" variant="ghost" asChild aria-label={t("ui.5f2eeb5dbd12ed6e")} title={t("ui.5f2eeb5dbd12ed6e")}>
            <Link href={`/function/tasks/detail?id=${encodeURIComponent(task.id)}`}>
              <EyeIcon />
            </Link>
          </Button>
          <TaskControlButton task={task} onControl={onControl} />
          <TaskPinAction task={task} onTogglePinned={onTogglePinned} />
          <TaskArchiveAction task={task} onArchive={onArchive} />
          <DeleteTaskDialog task={task} onDelete={onDelete} />
        </div>
      </TableCell>
    </TableRow>
  );
});

function TaskNameEditor({ task, onRename }: { task: Task; onRename: (task: Task, name: string) => Promise<void> }) {
  const [editing, setEditing] = React.useState(false);
  const [name, setName] = React.useState(task.name ?? "");
  const [saving, setSaving] = React.useState(false);
  const inputRef = React.useRef<HTMLInputElement>(null);
  const cancelOnBlurRef = React.useRef(false);

  React.useEffect(() => {
    if (!editing) setName(task.name ?? "");
  }, [editing, task.name]);

  async function finishEditing() {
    if (cancelOnBlurRef.current) {
      cancelOnBlurRef.current = false;
      setName(task.name ?? "");
      setEditing(false);
      return;
    }

    const next = name.trim();
    if (!next || next === task.name?.trim()) {
      setName(task.name ?? "");
      setEditing(false);
      return;
    }

    setSaving(true);
    try {
      await onRename(task, next);
      setEditing(false);
    } catch {
      requestAnimationFrame(() => inputRef.current?.focus());
    } finally {
      setSaving(false);
    }
  }

  if (editing) {
    return (
      <div className="flex min-w-0 items-center gap-1">
        <Input
          ref={inputRef}
          value={name}
          maxLength={200}
          autoFocus
          disabled={saving}
          aria-label={`Misión #${task.id}Nombre`}
          className="h-7 min-w-28 max-w-48 px-2 font-medium"
          onFocus={(event) => event.currentTarget.select()}
          onChange={(event) => setName(event.target.value)}
          onBlur={() => void finishEditing()}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              event.currentTarget.blur();
            } else if (event.key === "Escape") {
              event.preventDefault();
              cancelOnBlurRef.current = true;
              event.currentTarget.blur();
            }
          }}
        />
        {saving && <Spinner className="shrink-0" />}
      </div>
    );
  }

  return (
    <div className="flex min-w-0 items-center gap-1">
      <Link
        href={`/function/tasks/detail?id=${encodeURIComponent(task.id)}`}
        className="min-w-0 truncate rounded-sm hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        title={task.name?.trim() ? task.name : task.description}
      >
        {task.name?.trim() ? task.name : <span className="text-muted-foreground">{t("ui.7b2943156313d556")}</span>}
      </Link>
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="shrink-0 opacity-100 transition-opacity [@media(hover:hover)]:opacity-0 [@media(hover:hover)]:group-focus-within:opacity-100 [@media(hover:hover)]:group-hover:opacity-100"
        aria-label={`Cambiar el nombre de la tarea #${task.id}`}
        title={t("ui.de563c89c878c097")}
        onClick={() => {
          setName(task.name ?? "");
          setEditing(true);
        }}
      >
        <PencilIcon />
      </Button>
    </div>
  );
}

function taskPinIcon(pinning: boolean, pinned: boolean) {
  if (pinning) return <Spinner />;
  if (pinned) return <PinOffIcon />;
  return <PinIcon />;
}

function TaskPinAction({ task, onTogglePinned }: { task: Task; onTogglePinned: (task: Task) => Promise<void> }) {
  const [pinning, setPinning] = React.useState(false);
  const pinned = taskIsPinned(task);

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      disabled={pinning}
      aria-label={pinned ? `Cancelar tarea fijada #${task.id}` : `Tarea fijada #${task.id}`}
      title={pinned ? t("ui.4eea160fb2c96650") : t("ui.5596cd47498d7075")}
      onClick={async () => {
        setPinning(true);
        try {
          await onTogglePinned(task);
        } finally {
          setPinning(false);
        }
      }}
    >
      {taskPinIcon(pinning, pinned)}
    </Button>
  );
}

// Los íconos de tres estados se escriben por separado en lugar de íconos de tres estados anidados.:almacén Biome Línea de base prohibida noNestedTernary。
function taskControlIcon(pending: boolean, action: "pause" | "resume" | null) {
  if (pending) return <Loader2Icon className="animate-spin" />;
  if (action === "resume") return <PlayIcon />;
  return <PauseIcon />;
}

// TaskControlButton Es una pausa intralínea./Continuar cambiando. La tarea final se presenta como disabled en lugar de esconderse,
// De esta forma, el ancho de cada fila y columna es el mismo,La posición del botón no saltará con el estado.。
function TaskControlButton({
  task,
  onControl,
}: {
  task: Task;
  onControl: (id: string, action: "pause" | "resume") => Promise<void>;
}) {
  const [pending, setPending] = React.useState(false);
  const action = taskControlAction(task.status);
  const label = action === "resume" ? t("ui.ffe39c4a4ff41b12") : t("ui.e2a6072dc9371065");
  return (
    <Button
      size="icon"
      variant="ghost"
      aria-label={label}
      title={action ? label : t("ui.89452b0d838cf717")}
      disabled={!action || pending}
      onClick={async () => {
        if (!action) return;
        setPending(true);
        try {
          await onControl(task.id, action);
        } finally {
          setPending(false);
        }
      }}
    >
      {taskControlIcon(pending, action)}
    </Button>
  );
}

function archiveBlockReason(task: Task): string {
  if (task.queued) return "las tareas en cola deben pausarse primero";
  if (!ARCHIVABLE_STATUSES.has(task.status)) return t("ui.3f5b275a09da73f4");
  if (task.archive_blocked_by_task_id) {
    return `La tarea no está archivada #${task.archive_blocked_by_task_id}Herencia directa, archive primero las tareas dependientes`;
  }
  return "";
}

function ArchiveConfirmDialog({
  count,
  trigger,
  onConfirm,
}: {
  count: number;
  trigger: React.ReactNode;
  onConfirm: () => Promise<void>;
}) {
  const [open, setOpen] = React.useState(false);
  const [pending, setPending] = React.useState(false);
  return (
    <AlertDialog open={open} onOpenChange={(next) => !pending && setOpen(next)}>
      <AlertDialogTrigger asChild>{trigger}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{count === 1 ? t("ui.0ffa393b10f99002") : `Archivo ${count}tareas`}</AlertDialogTitle>
          <AlertDialogDescription className="[overflow-wrap:anywhere]">
             {t("ui.5aa2c48e883a96e2")} </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>{t("common.cancel")}</AlertDialogCancel>
          <AlertDialogAction
            disabled={pending}
            onClick={async (event) => {
              event.preventDefault();
              setPending(true);
              try {
                await onConfirm();
                setOpen(false);
              } finally {
                setPending(false);
              }
            }}
          >
            {pending && <Spinner data-icon="inline-start" />}
             {t("ui.13edcdee62a48aee")} </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function TaskArchiveAction({ task, onArchive }: { task: Task; onArchive: (task: Task) => Promise<void> }) {
  const reason = archiveBlockReason(task);
  const trigger = (
    <Button size="icon" variant="ghost" disabled={Boolean(reason)} aria-label={`Tarea de archivo #${task.id}`}>
      <ArchiveIcon />
    </Button>
  );
  if (reason) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="inline-flex">{trigger}</span>
        </TooltipTrigger>
        <TooltipContent>{reason}</TooltipContent>
      </Tooltip>
    );
  }
  return <ArchiveConfirmDialog count={1} trigger={trigger} onConfirm={() => onArchive(task)} />;
}

const ARCHIVE_PROCESSING_STATES = new Set<TaskArchiveState>([
  "archive_queued",
  "archiving",
  "restore_queued",
  "restoring",
  "delete_queued",
  "deleting",
]);

const ARCHIVE_FAILED_STATES = new Set<TaskArchiveState>(["archive_failed", "restore_failed", "delete_failed"]);

function archiveStateLabel(state: TaskArchiveState): string {
  const labels: Record<TaskArchiveState, string> = {
    archive_queued: t("ui.3b4a52acd83db931"),
    archiving: t("ui.eac5386d4211a8c5"),
    archive_failed: t("ui.eed1677953017d63"),
    ready: t("ui.b6ac39675ca5b0c0"),
    restore_queued: "esperando restauración",
    restoring: t("ui.bf5346a22525a8e5"),
    restore_failed: t("ui.6db4581e83646314"),
    delete_queued: t("ui.97dd6f403919bfad"),
    deleting: t("ui.8f48d409458d660d"),
    delete_failed: t("messages.deleted_failed"),
  };
  return labels[state];
}

function ArchiveStateBadge({ state }: { state: TaskArchiveState }) {
  let variant: "default" | "secondary" | "destructive" | "outline" = "outline";
  if (state === "ready") variant = "secondary";
  if (ARCHIVE_PROCESSING_STATES.has(state)) variant = "default";
  if (ARCHIVE_FAILED_STATES.has(state)) variant = "destructive";
  return <Badge variant={variant}>{archiveStateLabel(state)}</Badge>;
}

function formatArchiveBytes(bytes: number): string {
  if (bytes <= 0) return "—";
  if (bytes >= 1024 ** 3) return `${(bytes / 1024 ** 3).toFixed(1)} GB`;
  if (bytes >= 1024 ** 2) return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${bytes} B`;
}

function archiveCompressionLabel(archive: TaskArchive): string {
  if (archive.original_size <= 0 || archive.compressed_size <= 0) return "—";
  const saved = Math.max(0, 100 - (archive.compressed_size / archive.original_size) * 100);
  return `${formatArchiveBytes(archive.compressed_size)} · Guardar ${saved.toFixed(0)}%`;
}

function archiveDataTotal(archive: TaskArchive): number {
  return Object.values(archive.data_counts).reduce((sum, value) => sum + (Number(value) || 0), 0);
}

function archiveDate(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return date.toLocaleString();
}

function ArchiveDeleteDialog({
  archives,
  onConfirm,
  trigger,
}: {
  archives: TaskArchive[];
  onConfirm: () => Promise<void>;
  trigger: React.ReactNode;
}) {
  const [open, setOpen] = React.useState(false);
  const [pending, setPending] = React.useState(false);
  const rows = archives.reduce((sum, archive) => sum + archiveDataTotal(archive), 0);
  const bytes = archives.reduce((sum, archive) => sum + archive.compressed_size, 0);
  return (
    <AlertDialog open={open} onOpenChange={(next) => !pending && setOpen(next)}>
      <AlertDialogTrigger asChild>{trigger}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("ui.3d92915cfa507cae")} {archives.length}  {t("ui.d1c8f4b2cb32a207")}</AlertDialogTitle>
          <AlertDialogDescription className="[overflow-wrap:anywhere]">
             {t("ui.5844a703d753cf91")} {formatArchiveBytes(bytes)}  {t("ui.139840303b529c12")} {rows.toLocaleString()}  {t("ui.02c8eeb81f737866")} </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={pending}>{t("common.cancel")}</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={pending}
            onClick={async (event) => {
              event.preventDefault();
              setPending(true);
              try {
                await onConfirm();
                setOpen(false);
              } finally {
                setPending(false);
              }
            }}
          >
            {pending && <Spinner data-icon="inline-start" />}
             {t("ui.3d92915cfa507cae")} </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function TaskArchivesPanel({ onChanged }: { onChanged: () => void }) {
  const [archives, setArchives] = React.useState<TaskArchive[]>([]);
  const [query, setQuery] = React.useState("");
  const [stateFilter, setStateFilter] = React.useState("all");
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState(20);
  const [total, setTotal] = React.useState(0);
  const [selected, setSelected] = React.useState<Set<number>>(() => new Set());
  const [loading, setLoading] = React.useState(true);
  const pendingRestoreIDs = React.useRef(new Set<number>());

  const load = React.useCallback(async () => {
    try {
      const result = await api.taskArchives({
        page,
        size: pageSize,
        q: query.trim(),
        state: stateFilter === "all" ? undefined : stateFilter,
      });
      setArchives(result.items);
      setTotal(result.total);
      setSelected((current) => {
        const visible = new Set(result.items.map((archive) => archive.id));
        return new Set([...current].filter((id) => visible.has(id)));
      });
      const pending = [...pendingRestoreIDs.current];
      if (pending.length > 0) {
        const states = await Promise.allSettled(pending.map((id) => api.taskArchive(id)));
        let restored = false;
        states.forEach((state, index) => {
          if (state.status !== "rejected" || !(state.reason instanceof Error)) return;
          if (!state.reason.message.includes(t("ui.4ce32e30edc64359"))) return;
          pendingRestoreIDs.current.delete(pending[index]);
          restored = true;
        });
        if (restored) onChanged();
      }
    } catch (error) {
      toast.error(`No se pudo leer la lista de archivos:${(error as Error).message}`);
    } finally {
      setLoading(false);
    }
  }, [onChanged, page, pageSize, query, stateFilter]);

  React.useEffect(() => {
    void load();
    const timer = setInterval(() => void load(), 2_000);
    return () => clearInterval(timer);
  }, [load]);

  const selectable = React.useMemo(
    () => archives.filter((archive) => !ARCHIVE_PROCESSING_STATES.has(archive.state)),
    [archives],
  );
  const selectedArchives = React.useMemo(
    () => archives.filter((archive) => selected.has(archive.id)),
    [archives, selected],
  );
  const restorable = selectedArchives.filter(
    (archive) => archive.state === "ready" || archive.state === "restore_failed",
  );
  const deletable = selectedArchives.filter(
    (archive) => archive.state === "ready" || archive.state === "delete_failed",
  );
  const selectedAll = selectable.length > 0 && selectable.every((archive) => selected.has(archive.id));
  const selectedSome = selectable.some((archive) => selected.has(archive.id));

  const afterAction = React.useCallback(() => {
    setSelected(new Set());
    void load();
    onChanged();
  }, [load, onChanged]);

  async function restoreMany(items: TaskArchive[]) {
    try {
      const result =
        items.length === 1
          ? {
              items: [
                {
                  id: String(items[0].id),
                  ok: true,
                  queued: true,
                  archive_id: (await api.restoreTaskArchive(items[0].id)).id,
                },
              ],
            }
          : await api.restoreTaskArchives(items.map((archive) => archive.id));
      const succeeded = result.items.filter((item) => item.ok).length;
      const failed = result.items.length - succeeded;
      for (const item of result.items) {
        const archiveID = Number(item.archive_id ?? item.id);
        if (item.ok && Number.isSafeInteger(archiveID) && archiveID > 0) pendingRestoreIDs.current.add(archiveID);
      }
      if (succeeded > 0) toast.success(`ya ${succeeded}Tareas agregadas a la cola de restauración`);
      if (failed > 0) toast.error(`${failed}Las tareas no se pueden restaurar`);
      afterAction();
    } catch (error) {
      toast.error(`Error de restauración:${(error as Error).message}`);
    }
  }

  async function deleteMany(items: TaskArchive[]) {
    try {
      const result =
        items.length === 1
          ? {
              items: [
                {
                  id: String(items[0].id),
                  ok: true,
                  queued: true,
                  archive_id: (await api.deleteTaskArchive(items[0].id)).id,
                },
              ],
            }
          : await api.deleteTaskArchives(items.map((archive) => archive.id));
      const succeeded = result.items.filter((item) => item.ok).length;
      const failed = result.items.length - succeeded;
      if (succeeded > 0) toast.success(`ya ${succeeded}Archivos agregados a la cola de eliminación permanente`);
      if (failed > 0) toast.error(`${failed}Los archivos no se pueden eliminar`);
      afterAction();
    } catch (error) {
      toast.error(`Error en la eliminación permanente:${(error as Error).message}`);
      throw error;
    }
  }

  async function retry(archive: TaskArchive) {
    try {
      if (archive.state === "archive_failed") await api.archiveTask(String(archive.task_id));
      if (archive.state === "restore_failed") await api.restoreTaskArchive(archive.id);
      if (archive.state === "delete_failed") await api.deleteTaskArchive(archive.id);
      toast.success(t("ui.15560a0230f40f8f"));
      afterAction();
    } catch (error) {
      toast.error(`Reintento fallido:${(error as Error).message}`);
    }
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-4 px-0 pt-6">
        <div className="flex flex-wrap items-center gap-2 px-4 lg:px-6">
          <div className="relative w-full sm:max-w-xs">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={query}
              onChange={(event) => {
                setQuery(event.target.value);
                setPage(1);
              }}
              placeholder={t("ui.8a87a8ba08b02c11")}
              className="pl-8"
            />
          </div>
          <Select
            value={stateFilter}
            onValueChange={(value) => {
              setStateFilter(value);
              setPage(1);
            }}
          >
            <SelectTrigger className="w-36">
              <SelectValue placeholder={t("ui.9eefab413232ce69")} />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value="all">{t("ui.ecda92faab01f5c1")}</SelectItem>
                <SelectItem value="archive_queued">{t("ui.3b4a52acd83db931")}</SelectItem>
                <SelectItem value="archiving">{t("ui.eac5386d4211a8c5")}</SelectItem>
                <SelectItem value="ready">{t("ui.b6ac39675ca5b0c0")}</SelectItem>
                <SelectItem value="restore_queued">esperando restauración</SelectItem>
                <SelectItem value="restoring">{t("ui.bf5346a22525a8e5")}</SelectItem>
                <SelectItem value="delete_queued">{t("ui.97dd6f403919bfad")}</SelectItem>
                <SelectItem value="deleting">{t("ui.8f48d409458d660d")}</SelectItem>
                <SelectItem value="archive_failed">{t("ui.eed1677953017d63")}</SelectItem>
                <SelectItem value="restore_failed">{t("ui.6db4581e83646314")}</SelectItem>
                <SelectItem value="delete_failed">{t("messages.deleted_failed")}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
          <span className="text-muted-foreground text-xs tabular-nums">{total}  {t("ui.f7b3eac1cf97193d")}</span>
          {selectedArchives.length > 0 && (
            <>
              <span className="text-xs tabular-nums">{t("ui.df9a55ed63d1515b")} {selectedArchives.length} </span>
              {restorable.length > 0 && (
                <Button size="sm" variant="outline" onClick={() => void restoreMany(restorable)}>
                  <Undo2Icon data-icon="inline-start" />
                   {t("ui.eda02893d340c1be")} {restorable.length}
                </Button>
              )}
              {deletable.length > 0 && (
                <ArchiveDeleteDialog
                  archives={deletable}
                  onConfirm={() => deleteMany(deletable)}
                  trigger={
                    <Button size="sm" variant="destructive">
                      <Trash2Icon data-icon="inline-start" />
                       {t("ui.3d92915cfa507cae")} {deletable.length}
                    </Button>
                  }
                />
              )}
            </>
          )}
        </div>
        {loading ? (
          <div className="flex justify-center py-20">
            <Spinner />
          </div>
        ) : archives.length === 0 ? (
          <Empty className="mx-4 border border-dashed lg:mx-6">
            <EmptyHeader>
              <EmptyTitle>{t("ui.ae1f7f6d192faef7")}</EmptyTitle>
              <EmptyDescription>{t("ui.2369da7908a0609d")}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : (
          <Table className="**:data-[slot='table-cell']:px-4 **:data-[slot='table-head']:px-4">
            <TableHeader className="[&_tr]:border-t">
              <TableRow>
                <TableHead className="w-10">
                  <Checkbox
                    checked={selectedAll || (selectedSome ? "indeterminate" : false)}
                    onCheckedChange={(checked) =>
                      setSelected(checked === true ? new Set(selectable.map((archive) => archive.id)) : new Set())
                    }
                    aria-label={t("ui.f61ea65a3ea193a8")}
                  />
                </TableHead>
                <TableHead>{t("nav.tasks")}</TableHead>
                <TableHead>{t("ui.f971e4ec97f1b174")}</TableHead>
                <TableHead>{t("ui.773e03aa07df22d4")}</TableHead>
                <TableHead>{t("ui.52341755f1cf6218")}</TableHead>
                <TableHead>{t("ui.a17d428945abfae1")}</TableHead>
                <TableHead>{t("ui.3652146b327f0365")}</TableHead>
                <TableHead className="min-w-44">{t("ui.9eefab413232ce69")}</TableHead>
                <TableHead className="text-right">{t("common.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {archives.map((archive) => {
                const processing = ARCHIVE_PROCESSING_STATES.has(archive.state);
                const canRestore = archive.state === "ready" || archive.state === "restore_failed";
                const canDelete = archive.state === "ready" || archive.state === "delete_failed";
                return (
                  <TableRow key={archive.id} data-state={selected.has(archive.id) ? "selected" : undefined}>
                    <TableCell>
                      <Checkbox
                        checked={selected.has(archive.id)}
                        disabled={processing}
                        onCheckedChange={(checked) =>
                          setSelected((current) => {
                            const next = new Set(current);
                            if (checked === true) next.add(archive.id);
                            else next.delete(archive.id);
                            return next;
                          })
                        }
                        aria-label={`Seleccionar archivo de tarea #${archive.task_id}`}
                      />
                    </TableCell>
                    <TableCell className="max-w-sm">
                      <div className="flex min-w-0 flex-col gap-0.5">
                        <span className="truncate font-medium">
                          {archive.task_name || archive.task_description || `Misión #${archive.task_id}`}
                        </span>
                        <span className="text-muted-foreground truncate text-xs">
                          #{archive.task_id} · {archive.task_description}
                        </span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <StatusBadge domain="task" value={archive.original_status} dot />
                    </TableCell>
                    <TableCell className="text-muted-foreground">{archive.category_name || "Sin categoría"}</TableCell>
                    <TableCell className="text-muted-foreground whitespace-nowrap text-xs">
                      {archiveDate(archive.archived_at || archive.requested_at)}
                    </TableCell>
                    <TableCell
                      className="whitespace-nowrap text-xs"
                      title={`antes de la compresión ${formatArchiveBytes(archive.original_size)}`}
                    >
                      {archiveCompressionLabel(archive)}
                    </TableCell>
                    <TableCell className="text-xs tabular-nums">{archiveDataTotal(archive).toLocaleString()}</TableCell>
                    <TableCell>
                      <div className="flex min-w-0 flex-col gap-1.5">
                        <ArchiveStateBadge state={archive.state} />
                        {processing && <Progress value={archive.progress} />}
                        {archive.error && (
                          <p className="text-destructive text-xs [overflow-wrap:anywhere]">{archive.error}</p>
                        )}
                        {[...new Set(archive.warnings ?? [])].map((warning) => (
                          <p
                            key={warning}
                            className="text-amber-700 text-xs [overflow-wrap:anywhere] dark:text-amber-400"
                          >
                            {warning}
                          </p>
                        ))}
                        <span className="text-muted-foreground text-xs">
                          {archive.phase} · {archive.progress}%
                        </span>
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-0.5">
                        {ARCHIVE_FAILED_STATES.has(archive.state) && (
                          <Button
                            size="icon"
                            variant="ghost"
                            onClick={() => void retry(archive)}
                            aria-label={t("ui.20925ba24b869b66")}
                            title={t("ui.10636a10bc8b7ea1")}
                          >
                            <Undo2Icon />
                          </Button>
                        )}
                        {canRestore && (
                          <Button
                            size="icon"
                            variant="ghost"
                            onClick={() => void restoreMany([archive])}
                            aria-label={t("ui.7057877e76c2cccc")}
                            title={t("ui.7057877e76c2cccc")}
                          >
                            <Undo2Icon />
                          </Button>
                        )}
                        {canDelete && (
                          <ArchiveDeleteDialog
                            archives={[archive]}
                            onConfirm={() => deleteMany([archive])}
                            trigger={
                              <Button size="icon" variant="ghost" aria-label={t("ui.0688b2cbdf9792e7")} title={t("ui.3d92915cfa507cae")}>
                                <Trash2Icon />
                              </Button>
                            }
                          />
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
        <TablePagination
          page={page}
          pageSize={pageSize}
          total={total}
          onPageChange={setPage}
          onPageSizeChange={(size) => {
            setPageSize(size);
            setPage(1);
          }}
        />
      </CardContent>
    </Card>
  );
}

const emptyDeleteOptions = (): DeleteTaskOptions => ({
  delete_assets: false,
  delete_traffic: false,
  delete_files: false,
  delete_findings: false,
  delete_llm_records: false,
});

const deleteOptionKeys: (keyof DeleteTaskOptions)[] = [
  "delete_assets",
  "delete_traffic",
  "delete_files",
  "delete_findings",
  "delete_llm_records",
];

// DeleteOptionFields renders the「Limpiar datos relacionados al mismo tiempo.」checkbox block shared by the
// single-task and bulk delete dialogs. idPrefix keeps the label/input ids unique when
// several dialogs live in the same table.
function DeleteOptionFields({
  idPrefix,
  options,
  onOptionsChange,
  disabled,
}: {
  idPrefix: string;
  options: DeleteTaskOptions;
  onOptionsChange: React.Dispatch<React.SetStateAction<DeleteTaskOptions>>;
  disabled: boolean;
}) {
  const selectedCount = deleteOptionKeys.filter((key) => options[key]).length;
  let allChecked: boolean | "indeterminate" = false;
  if (selectedCount === deleteOptionKeys.length) {
    allChecked = true;
  } else if (selectedCount > 0) {
    allChecked = "indeterminate";
  }

  const updateOption = (key: keyof DeleteTaskOptions, checked: boolean) => {
    onOptionsChange((current) => ({ ...current, [key]: checked }));
  };

  const updateAllOptions = (checked: boolean) => {
    onOptionsChange({
      delete_assets: checked,
      delete_traffic: checked,
      delete_files: checked,
      delete_findings: checked,
      delete_llm_records: checked,
    });
  };

  return (
    <FieldSet disabled={disabled}>
      <FieldLegend variant="label">{t("ui.ae838337ecc325d5")}</FieldLegend>
      <FieldGroup className="gap-3">
        <Field orientation="horizontal">
          <Checkbox
            id={`delete-all-${idPrefix}`}
            checked={allChecked}
            onCheckedChange={(checked) => updateAllOptions(checked === true)}
          />
          <FieldContent>
            <FieldLabel htmlFor={`delete-all-${idPrefix}`}>{t("ui.a06fa1a5e42983b9")}</FieldLabel>
            <FieldDescription>{t("ui.08325b68c15bea1e")}</FieldDescription>
          </FieldContent>
        </Field>
        <Field orientation="horizontal">
          <Checkbox
            id={`delete-assets-${idPrefix}`}
            checked={options.delete_assets}
            onCheckedChange={(checked) => updateOption("delete_assets", checked === true)}
          />
          <FieldContent>
            <FieldLabel htmlFor={`delete-assets-${idPrefix}`}>{t("ui.107fdfd5c59d4e2f")}</FieldLabel>
            <FieldDescription>{t("ui.5ebfae13feeca387")}</FieldDescription>
          </FieldContent>
        </Field>
        <Field orientation="horizontal">
          <Checkbox
            id={`delete-traffic-${idPrefix}`}
            checked={options.delete_traffic}
            onCheckedChange={(checked) => updateOption("delete_traffic", checked === true)}
          />
          <FieldContent>
            <FieldLabel htmlFor={`delete-traffic-${idPrefix}`}>{t("ui.ebad840a32b40217")}</FieldLabel>
            <FieldDescription>{t("ui.75f4d8cfb015793d")}</FieldDescription>
          </FieldContent>
        </Field>
        <Field orientation="horizontal">
          <Checkbox
            id={`delete-files-${idPrefix}`}
            checked={options.delete_files}
            onCheckedChange={(checked) => updateOption("delete_files", checked === true)}
          />
          <FieldContent>
            <FieldLabel htmlFor={`delete-files-${idPrefix}`}>{t("ui.ba92822694c15d7f")}</FieldLabel>
            <FieldDescription>{t("ui.c283774eec2a4613")}</FieldDescription>
          </FieldContent>
        </Field>
        <Field orientation="horizontal">
          <Checkbox
            id={`delete-findings-${idPrefix}`}
            checked={options.delete_findings}
            onCheckedChange={(checked) => updateOption("delete_findings", checked === true)}
          />
          <FieldContent>
            <FieldLabel htmlFor={`delete-findings-${idPrefix}`}>{t("ui.8027c5d49f4ef8a0")}</FieldLabel>
            <FieldDescription>{t("ui.933f73553e8bcc07")}</FieldDescription>
          </FieldContent>
        </Field>
        <Field orientation="horizontal">
          <Checkbox
            id={`delete-llm-records-${idPrefix}`}
            checked={options.delete_llm_records}
            onCheckedChange={(checked) => updateOption("delete_llm_records", checked === true)}
          />
          <FieldContent>
            <FieldLabel htmlFor={`delete-llm-records-${idPrefix}`}>{t("ui.aa989a72d713b814")}</FieldLabel>
            <FieldDescription>{t("ui.d47c5a13a7546b35")}</FieldDescription>
          </FieldContent>
        </Field>
      </FieldGroup>
    </FieldSet>
  );
}

function DeleteTaskDialog({
  task,
  onDelete,
}: {
  task: Task;
  onDelete: (id: string, options: DeleteTaskOptions) => Promise<void>;
}) {
  const [open, setOpen] = React.useState(false);
  const [deleting, setDeleting] = React.useState(false);
  const [options, setOptions] = React.useState<DeleteTaskOptions>(emptyDeleteOptions);

  const handleOpenChange = (next: boolean) => {
    if (deleting) return;
    setOpen(next);
    if (next) setOptions(emptyDeleteOptions());
  };

  const handleDelete = async () => {
    setDeleting(true);
    try {
      await onDelete(task.id, options);
      setOpen(false);
    } finally {
      setDeleting(false);
    }
  };

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogTrigger asChild>
        <Button size="icon" variant="outline" aria-label={t("ui.8e9d1503c705483e")}>
          <Trash2Icon className="text-destructive" />
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("ui.82f25a28ce7d6b15")}{task.id}？</AlertDialogTitle>
          <AlertDialogDescription className="break-words">
            {task.description ? (
              <>
                「
                <span className="break-all">
                  {task.description.length > 80 ? `${task.description.slice(0, 80)}…` : task.description}
                </span>
                」
              </>
            ) : (
              t("ui.74236a7e41e2c506")
            )}
             {t("ui.d96ab0e3af10cbfe")} </AlertDialogDescription>
        </AlertDialogHeader>
        <DeleteOptionFields idPrefix={task.id} options={options} onOptionsChange={setOptions} disabled={deleting} />
        <AlertDialogFooter>
          <AlertDialogCancel disabled={deleting}>{t("common.cancel")}</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={deleting}
            onClick={(event) => {
              event.preventDefault();
              void handleDelete();
            }}
          >
            {deleting && <Spinner data-icon="inline-start" />}
            {deleting ? t("ui.8f48d409458d660d") : t("common.delete")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

// BulkDeleteTasksDialog deletes every checked task with one shared set of cleanup options.
// The backend has no batch endpoint, so deletion runs one task at a time (see deleteTasks)
// and the button shows live progress.
// MoveTasksCategoryDialog confirms a target before applying, so a mis-click on a
// large selection cannot silently re-file every task. UNCATEGORIZED_VALUE stands
// in for "no category" because Select rejects an empty string value.
function MoveTasksCategoryDialog({
  categories,
  count,
  moving,
  onMove,
}: {
  categories: TaskCategory[];
  count: number;
  moving: boolean;
  onMove: (categoryID?: number) => Promise<void>;
}) {
  const [open, setOpen] = React.useState(false);
  const [target, setTarget] = React.useState(UNCATEGORIZED_VALUE);

  const handleOpenChange = (next: boolean) => {
    if (moving) return;
    setOpen(next);
    if (next) setTarget(UNCATEGORIZED_VALUE);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button size="sm" variant="outline">
          <FolderInputIcon data-icon="inline-start" />  {t("ui.4ae58887d01f2e15")} </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("ui.f2b8cbca663d37fc")} {count}  {t("ui.3f77afe1c22c4706")}</DialogTitle>
          <DialogDescription>{t("ui.92db0eb82a6260f3")}</DialogDescription>
        </DialogHeader>
        <Field>
          <FieldLabel htmlFor="bulk-category">{t("ui.31ea1b681e83194b")}</FieldLabel>
          <Select value={target} onValueChange={setTarget} disabled={moving}>
            <SelectTrigger id="bulk-category" className="w-full">
              <SelectValue placeholder={t("ui.acbdd6d725728de0")} />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                <SelectItem value={UNCATEGORIZED_VALUE}>Sin categoría</SelectItem>
                {categories.map((category) => (
                  <SelectItem key={category.id} value={String(category.id)}>
                    {category.name}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          {categories.length === 0 && <FieldDescription>{t("ui.89a56841a0e839c4")}</FieldDescription>}
        </Field>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline" disabled={moving}>
               {t("ui.bb9dbb406dcbd524")} </Button>
          </DialogClose>
          <Button
            disabled={moving}
            onClick={() => {
              void onMove(target === UNCATEGORIZED_VALUE ? undefined : Number(target)).then(() => setOpen(false));
            }}
          >
            {moving && <Spinner data-icon="inline-start" />}
             {t("ui.6624c0d31888ecac")} </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function BulkDeleteTasksDialog({
  ids,
  onDelete,
}: {
  ids: string[];
  onDelete: (ids: string[], options: DeleteTaskOptions, onProgress: (done: number) => void) => Promise<void>;
}) {
  const [open, setOpen] = React.useState(false);
  const [deleting, setDeleting] = React.useState(false);
  const [done, setDone] = React.useState(0);
  const [options, setOptions] = React.useState<DeleteTaskOptions>(emptyDeleteOptions);

  const handleOpenChange = (next: boolean) => {
    if (deleting) return;
    setOpen(next);
    if (next) {
      setOptions(emptyDeleteOptions());
      setDone(0);
    }
  };

  const handleDelete = async () => {
    setDeleting(true);
    setDone(0);
    try {
      await onDelete(ids, options, setDone);
      setOpen(false);
    } finally {
      setDeleting(false);
    }
  };

  return (
    <AlertDialog open={open} onOpenChange={handleOpenChange}>
      <AlertDialogTrigger asChild>
        <Button size="sm" variant="outline">
          <Trash2Icon className="text-destructive" />  {t("ui.c08b2713c4c85128")} {ids.length}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent className="max-h-[85vh] overflow-y-auto">
        <AlertDialogHeader>
          <AlertDialogTitle>{t("ui.b02df26eee56226b")} {ids.length}  {t("ui.b0d55e373188d9d1")}</AlertDialogTitle>
          <AlertDialogDescription>
             {t("ui.f05009de1216cf03")} </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="text-muted-foreground flex flex-wrap gap-1 text-xs">
          {ids.slice(0, 30).map((id) => (
            <code key={id} className="bg-muted rounded px-1.5 py-0.5 font-mono">
              #{id}
            </code>
          ))}
          {ids.length > 30 && <span className="self-center">{t("ui.686fbe7df3f5ac18")} {ids.length} </span>}
        </div>
        <DeleteOptionFields idPrefix="bulk" options={options} onOptionsChange={setOptions} disabled={deleting} />
        <AlertDialogFooter>
          <AlertDialogCancel disabled={deleting}>{t("common.cancel")}</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            disabled={deleting}
            onClick={(event) => {
              event.preventDefault();
              void handleDelete();
            }}
          >
            {deleting && <Spinner data-icon="inline-start" />}
            {deleting ? `Eliminando ${done}/${ids.length}` : `Eliminar ${ids.length}tareas`}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

// CreateTaskSheet is the crear nueva tarea drawer. Its form state lives HERE, not in TasksPage: with
// Descripción/objetivo held by the page component every keystroke re-rendered the whole task table
// behind the drawer (plus its sticky column and 20 AlertDialog trees), which showed up as
// input lag. Now typing only re-renders the drawer.
function SourceTaskPicker({
  tasks,
  value,
  onValueChange,
  portalContainer,
}: {
  tasks: Task[];
  value: string[];
  onValueChange: (value: string[]) => void;
  portalContainer?: React.RefObject<HTMLElement | null>;
}) {
  const tasksByID = React.useMemo(() => new Map(tasks.map((task) => [task.id, task])), [tasks]);
  const taskIDs = React.useMemo(() => tasks.map((task) => task.id), [tasks]);
  const atLimit = value.length >= MAX_SOURCE_TASKS;

  const handleValueChange = (next: string[]) => {
    onValueChange(next.slice(0, MAX_SOURCE_TASKS));
  };

  return (
    <Combobox
      items={taskIDs}
      itemToStringValue={(taskID) => {
        const task = tasksByID.get(taskID);
        return task ? `${task.id} ${task.description} ${task.goal}` : taskID;
      }}
      multiple
      value={value}
      onValueChange={handleValueChange}
    >
      <ComboboxChips>
        <ComboboxValue>
          {value.map((taskID) => (
            <ComboboxChip key={taskID}>#{taskID}</ComboboxChip>
          ))}
        </ComboboxValue>
        <ComboboxChipsInput
          id="source-tasks"
          placeholder={atLimit ? `Más asociado ${MAX_SOURCE_TASKS}tareas` : t("ui.cd652e74b26ca007")}
          disabled={atLimit}
        />
      </ComboboxChips>
      <ComboboxContent portalContainer={portalContainer}>
        <ComboboxEmpty>{t("ui.1c25e5eacd900666")}</ComboboxEmpty>
        <ComboboxList>
          {(taskID) => {
            const task = tasksByID.get(taskID);
            return (
              <ComboboxItem key={taskID} value={taskID} disabled={atLimit && !value.includes(taskID)}>
                <div className="flex min-w-0 flex-1 items-center gap-2">
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">
                      #{taskID} · {task?.description ?? t("ui.ed27735ec46d95c7")}
                    </p>
                    {task?.goal && <p className="text-muted-foreground truncate text-xs">{task.goal}</p>}
                  </div>
                  {task && <StatusBadge domain="task" value={task.status} />}
                </div>
              </ComboboxItem>
            );
          }}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}

// CategoryPicker Es un selector de categoría de selección única en una nueva tarea: se pueden buscar categorías existentes; después de ingresar un nombre que no está en la biblioteca
// Presione Entrar (o haga clic en ”Crear” en la lista desplegable) para crear una nueva categoría y seleccionarla. Los elementos seleccionados se pueden eliminar. tag Pantalla. La clasificación es
// Recursos globales, la creación instantánea aquí equivale a la creación manual en ”Gestión de categorías”. Sólo se permite una categoría。
function CategoryPicker({
  categories,
  value,
  onValueChange,
  onCategoryCreated,
  portalContainer,
}: {
  categories: TaskCategory[];
  value?: number;
  onValueChange: (categoryID?: number) => void;
  onCategoryCreated: () => void;
  portalContainer?: React.RefObject<HTMLElement | null>;
}) {
  const [inputValue, setInputValue] = React.useState("");
  const [creating, setCreating] = React.useState(false);
  // La categoría recién creada tendrá que esperar a que se extraiga nuevamente la capa principal antes de regresar a ella. categories，Mantenga primero una copia local para evitar seleccionar chip y
  // La ventana desplegable muestra ”Categoría desconocida”」。
  const [localExtra, setLocalExtra] = React.useState<TaskCategory[]>([]);

  const allCategories = React.useMemo(() => {
    const byID = new Map<number, TaskCategory>();
    for (const category of categories) byID.set(category.id, category);
    for (const category of localExtra) if (!byID.has(category.id)) byID.set(category.id, category);
    return [...byID.values()];
  }, [categories, localExtra]);

  const byID = React.useMemo(() => new Map(allCategories.map((c) => [String(c.id), c])), [allCategories]);
  const categoryIDs = React.useMemo(() => allCategories.map((c) => String(c.id)), [allCategories]);
  const selectedIDs = value != null ? [String(value)] : [];

  const trimmed = inputValue.trim();
  const lower = trimmed.toLowerCase();
  // con base-ui el filtrado de subcadenas predeterminado sigue siendo coherente y se utiliza para determinar si hay categorías relacionadas」。
  const matchCount = trimmed
    ? allCategories.filter((c) => c.name.toLowerCase().includes(lower)).length
    : allCategories.length;

  const createAndSelect = async () => {
    if (!trimmed || creating) return;
    // Si ya existe exactamente el mismo nombre, selecciónelo directamente sin crearlo nuevamente.。
    const existing = allCategories.find((c) => c.name.toLowerCase() === lower);
    if (existing) {
      onValueChange(existing.id);
      setInputValue("");
      return;
    }
    setCreating(true);
    try {
      const created = await api.createTaskCategory(trimmed);
      setLocalExtra((prev) => [...prev, created]);
      onValueChange(created.id);
      setInputValue("");
      onCategoryCreated();
      toast.success(`Categoría creada${created.name}」`);
    } catch (e) {
      toast.error(`No se pudo crear la categoría:${(e as Error).message}`);
    } finally {
      setCreating(false);
    }
  };

  return (
    <Combobox
      items={categoryIDs}
      itemToStringValue={(id) => byID.get(id)?.name ?? id}
      multiple
      value={selectedIDs}
      onValueChange={(next: string[]) => {
        // Selección de radio: toma la última seleccionada; eliminar chip（claro) volverá a sin categoría。
        const last = next[next.length - 1];
        onValueChange(last ? Number(last) : undefined);
        setInputValue("");
      }}
      inputValue={inputValue}
      onInputValueChange={setInputValue}
    >
      <ComboboxChips>
        <ComboboxValue>
          {selectedIDs.map((id) => (
            <ComboboxChip key={id}>{byID.get(id)?.name ?? t("ui.b0af16919a4e6597")}</ComboboxChip>
          ))}
        </ComboboxValue>
        <ComboboxChipsInput
          id="task-category"
          placeholder={selectedIDs.length ? "" : t("ui.29659f3e6792d253")}
          onKeyDown={(e) => {
            // Ingrese cuando no haya ninguna coincidencia = Crear; conservar si hay una coincidencia base-ui ”Presione Enter para seleccionar el elemento resaltado”」。
            if (e.key === "Enter" && matchCount === 0 && trimmed) {
              e.preventDefault();
              void createAndSelect();
            }
          }}
        />
      </ComboboxChips>
      <ComboboxContent portalContainer={portalContainer}>
        <ComboboxList>
          {(id: string) => (
            <ComboboxItem key={id} value={id}>
              {byID.get(id)?.name ?? id}
            </ComboboxItem>
          )}
        </ComboboxList>
        {matchCount === 0 &&
          (trimmed ? (
            <button
              type="button"
              disabled={creating}
              onClick={() => void createAndSelect()}
              className="flex w-full items-center gap-2 px-2 py-2 text-left text-sm hover:bg-accent hover:text-accent-foreground disabled:opacity-50"
            >
              {creating ? <Spinner className="size-4" /> : <PlusIcon className="size-4" />}
               {t("ui.81f36409d2adb447")}{trimmed}」
            </button>
          ) : (
            <div className="px-2 py-2 text-sm text-muted-foreground">{t("ui.1db80c17ce06667a")}</div>
          ))}
      </ComboboxContent>
    </Combobox>
  );
}

const COMPANY_SCOPE_LABELS: Record<string, string> = {
  domain: "nombre de dominio",
  ip: "IP",
  cidr: "CIDR",
  icp: "ICP",
  keyword: "Palabras clave",
};

function companyScopeSummary(company: Company): string {
  const rows = company.scope ?? [];
  if (rows.length === 0) return t("ui.d7e51c182a26b2eb");
  const preview = rows.slice(0, 3).map((row) => {
    const value = row.raw || row.value || row.domain || row.net || "";
    return `${COMPANY_SCOPE_LABELS[row.kind] ?? row.kind}：${value}`;
  });
  return `${preview.join(" · ")}${rows.length > preview.length ? ` · Otro ${rows.length - preview.length}artículo` : ""}`;
}

function CompanyPicker({
  companies,
  value,
  onValueChange,
  portalContainer,
}: {
  companies: Company[];
  value: number[];
  onValueChange: (value: number[]) => void;
  portalContainer?: React.RefObject<HTMLElement | null>;
}) {
  const companiesByID = React.useMemo(
    () => new Map(companies.map((company) => [String(company.id), company])),
    [companies],
  );
  const companyIDs = React.useMemo(() => companies.map((company) => String(company.id)), [companies]);
  const selectedIDs = React.useMemo(() => value.map(String), [value]);

  return (
    <Combobox
      items={companyIDs}
      itemToStringValue={(companyID) => {
        const company = companiesByID.get(companyID);
        return company ? `${company.name} ${companyScopeSummary(company)}` : companyID;
      }}
      multiple
      value={selectedIDs}
      onValueChange={(next) => onValueChange(next.map(Number).filter(Number.isFinite))}
    >
      <ComboboxChips>
        <ComboboxValue>
          {selectedIDs.map((companyID) => (
            <ComboboxChip key={companyID}>{companiesByID.get(companyID)?.name ?? `Empresa #${companyID}`}</ComboboxChip>
          ))}
        </ComboboxValue>
        <ComboboxChipsInput id="task-companies" placeholder={t("ui.f5438e7be42e7fb3")} />
      </ComboboxChips>
      <ComboboxContent portalContainer={portalContainer}>
        <ComboboxEmpty>{t("ui.2ebcb3f4aa90fbac")}</ComboboxEmpty>
        <ComboboxList>
          {(companyID) => {
            const company = companiesByID.get(companyID);
            return (
              <ComboboxItem key={companyID} value={companyID}>
                <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <div className="flex min-w-0 items-center gap-2">
                    <span className="min-w-0 flex-1 truncate font-medium">{company?.name ?? `Empresa #${companyID}`}</span>
                    <span className="text-muted-foreground shrink-0 text-xs tabular-nums">
                      {company?.asset_count ?? 0}  {t("ui.25baaa016f821a4d")} </span>
                  </div>
                  {company && (
                    <span className="text-muted-foreground truncate text-xs" title={companyScopeSummary(company)}>
                      {companyScopeSummary(company)}
                    </span>
                  )}
                </div>
              </ComboboxItem>
            );
          }}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}

type CategoryManagementView = number | "uncategorized" | "new";

function CategoryDropTarget({
  value,
  name,
  count,
  selected,
  disabled,
  onSelect,
}: {
  value: string;
  name: string;
  count: number;
  selected: boolean;
  disabled: boolean;
  onSelect: () => void;
}) {
  const { isOver, setNodeRef } = useDroppable({ id: `category:${value}`, disabled });

  return (
    <button
      ref={setNodeRef}
      type="button"
      className={cn(
        "min-w-0 rounded-md border border-transparent px-2.5 py-2 text-left transition-colors",
        selected ? "bg-accent text-accent-foreground" : "hover:bg-accent/50",
        isOver && "border-primary bg-primary/10 text-foreground",
      )}
      onClick={onSelect}
    >
      <span className="block truncate font-medium text-sm">{name}</span>
      <span className="block truncate text-muted-foreground text-xs">{isOver ? t("ui.626b29567dbefc6e") : `${count}tareas`}</span>
    </button>
  );
}

function DraggableCategoryTask({ task, disabled, moving }: { task: Task; disabled: boolean; moving: boolean }) {
  const { attributes, isDragging, listeners, setNodeRef } = useDraggable({
    id: `task:${task.id}`,
    disabled,
  });

  return (
    <Item ref={setNodeRef} variant="outline" size="sm" className={cn(isDragging && "opacity-40")}>
      <ItemMedia className="group-has-data-[slot=item-description]/item:self-center group-has-data-[slot=item-description]/item:translate-y-0">
        {moving ? (
          <Spinner />
        ) : (
          <Button
            type="button"
            size="icon-xs"
            variant="ghost"
            className="touch-none cursor-grab active:cursor-grabbing"
            disabled={disabled}
            {...listeners}
            {...attributes}
            aria-label={`Arrastrar tarea #${task.id}`}
            title={t("ui.c7465e6168475339")}
          >
            <GripVerticalIcon />
          </Button>
        )}
      </ItemMedia>
      <ItemContent className="min-w-0">
        <ItemTitle className="w-full min-w-0">
          <Link href={`/function/tasks/detail?id=${encodeURIComponent(task.id)}`} className="truncate hover:underline">
            {task.name?.trim() || task.description || `Misión #${task.id}`}
          </Link>
        </ItemTitle>
        <ItemDescription className="line-clamp-1">
          #{task.id} · {task.description}
        </ItemDescription>
      </ItemContent>
      <ItemActions>
        <StatusBadge domain="task" value={task.status} />
      </ItemActions>
    </Item>
  );
}

function CategoryTaskDragPreview({ task }: { task: Task }) {
  return (
    <Item variant="outline" size="sm" className="w-80 bg-background shadow-lg">
      <ItemMedia className="group-has-data-[slot=item-description]/item:self-center group-has-data-[slot=item-description]/item:translate-y-0">
        <GripVerticalIcon className="size-4 text-muted-foreground" />
      </ItemMedia>
      <ItemContent className="min-w-0">
        <ItemTitle className="w-full min-w-0 truncate">
          {task.name?.trim() || task.description || `Misión #${task.id}`}
        </ItemTitle>
        <ItemDescription className="line-clamp-1">#{task.id}</ItemDescription>
      </ItemContent>
    </Item>
  );
}

function CategoryManagementSheet({
  categories,
  tasks,
  onChanged,
  onTaskMoved,
}: {
  categories: TaskCategory[];
  tasks: Task[];
  onChanged: () => void;
  onTaskMoved: (taskID: string, category: TaskCategory | null) => void;
}) {
  const [open, setOpen] = React.useState(false);
  const [selectedView, setSelectedView] = React.useState<CategoryManagementView>("new");
  const [draftName, setDraftName] = React.useState("");
  const [saving, setSaving] = React.useState(false);
  const [deleteOpen, setDeleteOpen] = React.useState(false);
  const [deleting, setDeleting] = React.useState(false);
  const [movingTaskID, setMovingTaskID] = React.useState<string | null>(null);
  const [activeTaskID, setActiveTaskID] = React.useState<string | null>(null);
  const wasOpen = React.useRef(false);
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor),
  );

  React.useEffect(() => {
    if (open && !wasOpen.current) {
      const first = categories[0];
      if (first) {
        setSelectedView(first.id);
        setDraftName(first.name);
      } else if (tasks.some((task) => task.category_id == null)) {
        setSelectedView("uncategorized");
        setDraftName("");
      } else {
        setSelectedView("new");
        setDraftName("");
      }
    }
    wasOpen.current = open;
  }, [categories, open, tasks]);

  const selectedCategory = React.useMemo(
    () =>
      typeof selectedView === "number" ? (categories.find((category) => category.id === selectedView) ?? null) : null,
    [categories, selectedView],
  );

  const uncategorizedCount = React.useMemo(() => tasks.filter((task) => task.category_id == null).length, [tasks]);

  const visibleTasks = React.useMemo(() => {
    if (selectedView === "uncategorized") return tasks.filter((task) => task.category_id == null);
    if (typeof selectedView === "number") return tasks.filter((task) => task.category_id === selectedView);
    return [];
  }, [selectedView, tasks]);

  const activeTask = React.useMemo(() => tasks.find((task) => task.id === activeTaskID) ?? null, [activeTaskID, tasks]);

  const selectCategory = (category: TaskCategory) => {
    setSelectedView(category.id);
    setDraftName(category.name);
  };

  const selectUncategorized = () => {
    setSelectedView("uncategorized");
    setDraftName("");
  };

  const startNew = () => {
    setSelectedView("new");
    setDraftName("");
  };

  async function saveCategory() {
    const name = draftName.trim();
    if (!name || saving || selectedView === "uncategorized") return;
    setSaving(true);
    try {
      if (selectedView === "new") {
        const created = await api.createTaskCategory(name);
        setSelectedView(created.id);
        setDraftName(created.name);
        toast.success(t("ui.e0a1c22f71a30a38"));
      } else {
        const updated = await api.renameTaskCategory(selectedView, name);
        setDraftName(updated.name);
        toast.success(t("ui.9864bf91a7b7b8b3"));
      }
      onChanged();
    } catch (error) {
      toast.error(`${selectedView === "new" ? t("common.create") : t("common.update")}Clasificación fallida:${(error as Error).message}`);
    } finally {
      setSaving(false);
    }
  }

  async function deleteCategory() {
    if (!selectedCategory || deleting) return;
    const deletedID = selectedCategory.id;
    setDeleting(true);
    try {
      await api.deleteTaskCategory(deletedID);
      const next = categories.find((category) => category.id !== deletedID);
      if (next) selectCategory(next);
      else selectUncategorized();
      toast.success(t("ui.ff02ea0543cfbcaf"));
      setDeleteOpen(false);
      onChanged();
    } catch (error) {
      toast.error(`No se pudo eliminar la categoría:${(error as Error).message}`);
    } finally {
      setDeleting(false);
    }
  }

  async function moveTask(task: Task, destination: string) {
    if (movingTaskID) return;
    const category =
      destination === "uncategorized" ? null : (categories.find((item) => item.id === Number(destination)) ?? null);
    if (destination !== "uncategorized" && !category) {
      toast.error(t("ui.7c4cd509e6a98541"));
      return;
    }
    if (task.category_id === category?.id || (task.category_id == null && category == null)) return;

    setMovingTaskID(task.id);
    try {
      await api.updateTaskCategory(task.id, category?.id);
      onTaskMoved(task.id, category);
      toast.success(`Misión #${task.id} Movido a "${category?.name ?? "Sin categoría"}」`);
    } catch (error) {
      toast.error(`Error al mover la tarea:${(error as Error).message}`);
    } finally {
      setMovingTaskID(null);
    }
  }

  function handleDragStart(event: DragStartEvent) {
    const id = String(event.active.id);
    setActiveTaskID(id.startsWith("task:") ? id.slice("task:".length) : null);
  }

  function handleDragEnd(event: DragEndEvent) {
    const activeID = String(event.active.id);
    const taskID = activeID.startsWith("task:") ? activeID.slice("task:".length) : null;
    setActiveTaskID(null);
    if (!taskID || !event.over) return;
    const destination = String(event.over.id);
    if (!destination.startsWith("category:")) return;
    const task = tasks.find((item) => item.id === taskID);
    if (!task) return;
    void moveTask(task, destination.slice("category:".length));
  }

  const saveLabel = saving ? "Ahorro" : selectedView === "new" ? t("ui.36a594ab1a3a9209") : "guardar cambios";

  return (
    <>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger asChild>
          <Button size="sm" variant="outline">
            <TagsIcon data-icon="inline-start" />
             {t("ui.a76190148c4b571a")} </Button>
        </SheetTrigger>
        <SheetContent className="grid h-full w-full! max-w-none! grid-rows-[auto_minmax(0,1fr)_auto] gap-0 overflow-hidden p-0 sm:w-[48rem]! sm:max-w-[48rem]!">
          <SheetHeader className="border-b px-6 py-5">
            <SheetTitle>{t("ui.91777a01eabe7abc")}</SheetTitle>
            <SheetDescription>{t("ui.1cf9a134f887cc03")}</SheetDescription>
          </SheetHeader>
          <DndContext
            sensors={sensors}
            onDragStart={handleDragStart}
            onDragCancel={() => setActiveTaskID(null)}
            onDragEnd={handleDragEnd}
          >
            <div className="grid min-h-0 overflow-y-auto lg:grid-cols-[15rem_minmax(0,1fr)] lg:overflow-hidden">
              <div className="flex min-h-0 flex-col border-b p-3 lg:border-r lg:border-b-0">
                <Button type="button" variant="outline" className="w-full" onClick={startNew}>
                  <PlusIcon data-icon="inline-start" />
                   {t("ui.732e615892521479")} </Button>
                <ScrollArea className="mt-2 max-h-44 lg:max-h-none lg:flex-1">
                  <div className="flex flex-col gap-1 pr-2">
                    <CategoryDropTarget
                      value="uncategorized"
                      name={"Sin categoría"}
                      count={uncategorizedCount}
                      selected={selectedView === "uncategorized"}
                      disabled={movingTaskID != null}
                      onSelect={selectUncategorized}
                    />
                    {categories.map((category) => (
                      <CategoryDropTarget
                        key={category.id}
                        value={String(category.id)}
                        name={category.name}
                        count={category.task_count}
                        selected={selectedView === category.id}
                        disabled={movingTaskID != null}
                        onSelect={() => selectCategory(category)}
                      />
                    ))}
                  </div>
                </ScrollArea>
              </div>
              <ScrollArea className="min-h-0">
                <FieldGroup className="p-6">
                  {selectedView !== "uncategorized" && (
                    <Field>
                      <FieldLabel htmlFor="task-category-name">{t("ui.3954e112a1b32f9a")}</FieldLabel>
                      <Input
                        id="task-category-name"
                        value={draftName}
                        onChange={(event) => setDraftName(event.target.value)}
                        placeholder={t("ui.dc071f82e0cadf3b")}
                        maxLength={80}
                        onKeyDown={(event) => {
                          if (event.key === "Enter") void saveCategory();
                        }}
                      />
                      <FieldDescription>
                        {selectedCategory
                          ? `actualmente disponible ${selectedCategory.task_count}Las tareas utilizan esta categoría. La lista de tareas se actualizará sincrónicamente después de cambiar el nombre.`
                          : t("ui.45632b43237a1dcf")}
                      </FieldDescription>
                    </Field>
                  )}
                  {selectedView !== "new" && (
                    <Field>
                      <div className="flex flex-wrap items-end justify-between gap-2">
                        <div className="flex min-w-0 flex-col gap-1">
                          <FieldLabel>{selectedCategory ? t("ui.938c0d79569263b8") : t("ui.4dd89a4345274e17")}</FieldLabel>
                          <FieldDescription>
                            {selectedCategory
                              ? `Esta categoría contiene ${visibleTasks.length}tareas.`
                              : `actualmente disponible ${visibleTasks.length}Las tareas aún no han sido clasificadas.`}
                          </FieldDescription>
                        </div>
                      </div>
                      {visibleTasks.length === 0 ? (
                        <Empty className="min-h-36 border">
                          <EmptyHeader>
                            <EmptyTitle>{selectedCategory ? t("ui.694c3be4715d7883") : t("ui.48ad65c544fa544f")}</EmptyTitle>
                            <EmptyDescription>{t("ui.ae9393d47f1fcbdc")}</EmptyDescription>
                          </EmptyHeader>
                        </Empty>
                      ) : (
                        <ItemGroup className="gap-2">
                          {visibleTasks.map((task) => (
                            <DraggableCategoryTask
                              key={task.id}
                              task={task}
                              moving={movingTaskID === task.id}
                              disabled={movingTaskID != null}
                            />
                          ))}
                        </ItemGroup>
                      )}
                    </Field>
                  )}
                </FieldGroup>
              </ScrollArea>
            </div>
            <DragOverlay>{activeTask ? <CategoryTaskDragPreview task={activeTask} /> : null}</DragOverlay>
          </DndContext>
          <SheetFooter className="border-t px-6 py-4 sm:flex-row sm:items-center">
            {selectedCategory && (
              <Button
                type="button"
                variant="destructive"
                className="sm:mr-auto"
                disabled={saving || deleting}
                onClick={() => setDeleteOpen(true)}
              >
                <Trash2Icon data-icon="inline-start" />
                 {t("ui.4930f416ee415cad")} </Button>
            )}
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
               {t("ui.aeccae342e4bd0a7")} </Button>
            {selectedView !== "uncategorized" && (
              <Button
                type="button"
                disabled={!draftName.trim() || saving || deleting}
                onClick={() => void saveCategory()}
              >
                {saving ? <Spinner data-icon="inline-start" /> : <SaveIcon data-icon="inline-start" />}
                {saveLabel}
              </Button>
            )}
          </SheetFooter>
        </SheetContent>
      </Sheet>
      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("ui.f7c3b1c7ef1e4038")}{selectedCategory?.name || t("ui.4a4de546af768074")}」？</AlertDialogTitle>
            <AlertDialogDescription>
               {t("ui.f482c00873f3329e")} {selectedCategory?.task_count ?? 0}  {t("ui.a6b59f3c38d2e46e")} </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>{t("common.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleting}
              onClick={(event) => {
                event.preventDefault();
                void deleteCategory();
              }}
            >
              {deleting && <Spinner data-icon="inline-start" />}
              {deleting ? t("ui.8f48d409458d660d") : t("common.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

function CreateTaskSheet({
  tasks,
  categories,
  onCreated,
  onCategoriesChanged,
}: {
  tasks: Task[];
  categories: TaskCategory[];
  onCreated: () => void;
  onCategoriesChanged: () => void;
}) {
  const [open, setOpen] = React.useState(false);
  const [name, setName] = React.useState("");
  const [categoryID, setCategoryID] = React.useState<number | undefined>(undefined);
  const [description, setDescription] = React.useState("");
  const [goal, setGoal] = React.useState("");
  const [selectedTemplateID, setSelectedTemplateID] = React.useState<number | null>(null);
  const [profiles, setProfiles] = React.useState<LLMProfile[]>([]);
  const [companies, setCompanies] = React.useState<Company[]>([]);
  const [sourceTaskIDs, setSourceTaskIDs] = React.useState<string[]>([]);
  const [companyIDs, setCompanyIDs] = React.useState<number[]>([]);
  const [llmProfileIDs, setLLMProfileIDs] = React.useState<string[]>([]);
  const [creating, setCreating] = React.useState(false);
  const [timeoutMin, setTimeoutMin] = React.useState(""); // tiempo de espera de nivel de tarea(minutos);vacío/0 = Sin límite de tiempo
  const [heartbeatMin, setHeartbeatMin] = React.useState("10"); // planner latido del corazón(minutos);predeterminado10,límite inferior10(consistente con el backend)
  const [seedFirstIntent, setSeedFirstIntent] = React.useState(false); // Enviar intención inicial al crear,worker No es necesario esperar a la primera ronda. planner Comience a ejecutar directamente;Desactivado por defecto,Seguir estándares, planificar primero y luego ejecutar
  const [coverageEnabled, setCoverageEnabled] = React.useState(true); // Función de cobertura de activos;Activado de forma predeterminada. cerrar=No calculado/herramientas de cobertura de visualización, rango no acumulativo y rango oculto(company La asociación no se ve afectada)
  const [interceptRules, setInterceptRules] = React.useState<AssetInterceptRuleInput[]>([]); // reglas de interceptación de activos a nivel de tarea(Sólo esta tarea tiene efecto,No entrar a la tabla global)
  // método1 carga de archivos:guardar archivos temporalmente en drafts/<draftId>/uploads/,Obtenga la ruta absoluta y agréguela a la descripción.。
  const [uploading, setUploading] = React.useState(false);
  const [uploadCount, setUploadCount] = React.useState(0);
  const draftIdRef = React.useRef<string>("");
  const fileInputRef = React.useRef<HTMLInputElement>(null);
  const sheetContentRef = React.useRef<HTMLDivElement>(null);

  // load LLM profiles once for the create-task profile picker.
  React.useEffect(() => {
    api
      .llmProfiles()
      .then(setProfiles)
      .catch(() => setProfiles([]));
    api
      .companies()
      .then(setCompanies)
      .catch(() => setCompanies([]));
  }, []);

  // pickFiles uploads the chosen files into this draft's staging dir and appends their
  // absolute paths to the description; the task's agents open them by path via Read/Bash.
  async function pickFiles(files: FileList | null) {
    if (!files || files.length === 0) return;
    // crypto.randomUUID Sólo disponible en contexto de seguridad(https/localhost);Sutra IP+http Bajar de categoría al acceder。
    if (!draftIdRef.current) {
      draftIdRef.current =
        globalThis.crypto?.randomUUID?.() ?? `d${Date.now().toString(36)}${Math.random().toString(36).slice(2, 10)}`;
    }
    setUploading(true);
    try {
      const r = await api.chatUpload("staging", draftIdRef.current, Array.from(files));
      setDescription((prev) => appendUploads(prev, r.attachments));
      setUploadCount((n) => n + r.attachments.length);
    } catch (e) {
      toast.error(t("ui.3b7cd8178200c204") + (e as Error).message);
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = ""; // allow re-picking the same file
    }
  }

  async function createTask() {
    if (!description.trim() || !goal.trim()) {
      toast.error(t("ui.4583b56b2b24a926"));
      return;
    }
    if (sourceTaskIDs.length > MAX_SOURCE_TASKS) {
      toast.error(`Más asociado ${MAX_SOURCE_TASKS}tareas fuente`);
      return;
    }
    setCreating(true);
    try {
      const timeoutSec = Math.max(0, Math.floor(Number(timeoutMin) || 0)) * 60;
      const heartbeatSec = Math.max(10, Math.floor(Number(heartbeatMin) || 10)) * 60; // límite inferior 10min，consistente con el backend
      await api.createTask({
        name: name.trim(),
        categoryId: categoryID,
        description: description.trim(),
        goal: goal.trim(),
        llmProfileIds: llmProfileIDs.map(Number),
        sourceTaskIds: sourceTaskIDs,
        companyIds: companyIDs,
        timeoutSeconds: timeoutSec,
        seedFirstIntent,
        planHeartbeatSeconds: heartbeatSec,
        coverageEnabled,
        interceptRules: interceptRules
          .map((r) => ({ ...r, pattern: r.pattern.trim() }))
          .filter((r) => r.pattern !== ""),
      });
      toast.success(t("ui.72be08c7876a76da"));
      setName("");
      setCategoryID(undefined);
      setDescription("");
      setGoal("");
      setSelectedTemplateID(null);
      setSourceTaskIDs([]);
      setCompanyIDs([]);
      setLLMProfileIDs([]);
      setTimeoutMin("");
      setHeartbeatMin("10");
      setSeedFirstIntent(false);
      setCoverageEnabled(true);
      setInterceptRules([]);
      setUploadCount(0);
      draftIdRef.current = "";
      setOpen(false);
      onCreated();
    } catch (e) {
      toast.error("Error en la creación:" + (e as Error).message);
    } finally {
      setCreating(false);
    }
  }

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button size="sm">
          <PlusIcon />  {t("ui.528ef71f86882c58")} </Button>
      </SheetTrigger>
      {/* 45vw Cajón ancho derecho:Se puede desplazar toda la altura de la pantalla,Los formularios largos ya no están sujetos a restricciones de altura de ventanas emergentes. La pantalla estrecha se degrada al ancho completo。
            El contenido es flex columna:cabeza/Fijación del pie,zona del medio campo flex-1 desplazamiento independiente。 */}
      <SheetContent
        ref={sheetContentRef}
        side="right"
        className="w-full! max-w-none! gap-0 p-0 sm:w-[45vw]! sm:max-w-[45vw]!"
      >
        <SheetHeader className="border-b p-6">
          <SheetTitle>{t("task.new")}</SheetTitle>
          <SheetDescription>{t("ui.f0156847dc979d50")}</SheetDescription>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto p-6">
          <div className="grid gap-5">
            <TaskTemplateControls
              description={description}
              goal={goal}
              categoryID={categoryID}
              interceptRules={interceptRules}
              selectedTemplateID={selectedTemplateID}
              onSelectedTemplateIDChange={setSelectedTemplateID}
              onApply={(template) => {
                setDescription(template.description);
                setGoal(template.goal);
                setCategoryID(template.category_id ?? undefined);
                setInterceptRules(template.intercept_rules ?? []);
                setUploadCount(0);
              }}
              portalContainer={sheetContentRef}
            />
            <div className="grid gap-2">
              <Label htmlFor="name">{t("ui.42bb5727b2fd10b3")}</Label>
              <Input
                id="name"
                placeholder={t("ui.5123a48db0a67991")}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <Field>
              <FieldLabel htmlFor="task-category">{t("ui.62d216c0d8b232d5")}</FieldLabel>
              <CategoryPicker
                categories={categories}
                value={categoryID}
                onValueChange={setCategoryID}
                onCategoryCreated={onCategoriesChanged}
                portalContainer={sheetContentRef}
              />
              <FieldDescription>{t("ui.1c177672f709c81d")}</FieldDescription>
            </Field>
            <div className="grid gap-2">
              <Label htmlFor="description">{t("common.description")}</Label>
              <Textarea
                id="description"
                className="min-h-32"
                placeholder={t("ui.022874fc2d4796ae")}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
              {/* cargar archivo(Múltiples selecciones posibles):guardar temporalmente en drafts/,agregue la ruta absoluta a la descripción anterior,worker En consecuencia Read/Bash abierto。 */}
              <div className="flex flex-wrap items-center gap-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  multiple
                  className="hidden"
                  onChange={(e) => void pickFiles(e.target.files)}
                />
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={uploading}
                >
                  {uploading ? <Loader2Icon className="animate-spin" /> : <PaperclipIcon />}
                  cargar archivo </Button>
                <span className="text-muted-foreground text-xs">
                  {uploadCount > 0
                    ? `subido ${uploadCount}archivos, la ruta absoluta se ha agregado al final de la descripción (editable)`
                    : t("ui.8880863a644581a3")}
                </span>
              </div>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="goal">{t("ui.dfe62c1beb52d3b3")}</Label>
              <Textarea
                id="goal"
                className="min-h-32"
                placeholder={t("ui.1de4a884451c5bee")}
                value={goal}
                onChange={(e) => setGoal(e.target.value)}
              />
            </div>
            <Field>
              <FieldLabel htmlFor="source-tasks">{t("ui.5a796a3edca61856")}</FieldLabel>
              <SourceTaskPicker
                tasks={tasks}
                value={sourceTaskIDs}
                onValueChange={setSourceTaskIDs}
                portalContainer={sheetContentRef}
              />
              <FieldDescription>
                 {t("ui.bc8565c78b83a623")} {MAX_SOURCE_TASKS}{" "}
                 {t("ui.44b4677ae3467691")} </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="task-companies">{t("ui.71ec29de22fcd222")}</FieldLabel>
              <CompanyPicker
                companies={companies}
                value={companyIDs}
                onValueChange={setCompanyIDs}
                portalContainer={sheetContentRef}
              />
              <FieldDescription>
                 {t("ui.717cd5c22be63f2f")} </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="task-intercept-rules">{t("ui.7f46f2b4b58bf095")}</FieldLabel>
              <AssetInterceptRulesEditor value={interceptRules} onChange={setInterceptRules} />
              <FieldDescription>
                 {t("ui.1ff91b579aabe94f")} </FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="llm-profiles">{t("ui.22e0b0c125059b18")}</FieldLabel>
              <TaskLLMProfileChain
                profiles={profiles}
                value={llmProfileIDs}
                onValueChange={setLLMProfileIDs}
                inputId="llm-profiles"
                portalContainer={sheetContentRef}
              />
              <FieldDescription>{t("ui.3cf111542c079057")}</FieldDescription>
            </Field>

            {/* Los parámetros avanzados están plegados de forma predeterminada:tiempo de espera/latido del corazón/Primera intención,Expandir para ocupar espacio,Mantener frescos los caminos comunes。 */}
            <Collapsible>
              <CollapsibleTrigger className="group flex w-full items-center gap-2 border-t pt-4 text-sm font-medium">
                <ChevronRightIcon className="text-muted-foreground size-4 transition-transform group-data-[state=open]:rotate-90" />
                 {t("ui.05d8dd68c7d19361")} <span className="text-muted-foreground ml-auto text-xs font-normal">{t("ui.f2f051d198baff5e")}</span>
              </CollapsibleTrigger>
              <CollapsibleContent className="grid gap-5 pt-5">
                <div className="grid gap-2">
                  <Label htmlFor="timeout-min">{t("ui.90a546a046d7834d")}</Label>
                  <Input
                    id="timeout-min"
                    type="number"
                    min={0}
                    className="w-40"
                    placeholder={t("ui.36f63f2523363657")}
                    value={timeoutMin}
                    onChange={(e) => setTimeoutMin(e.target.value)}
                  />
                  <p className="text-muted-foreground text-xs">
                     {t("ui.f0f5cb2ab35c078f")} </p>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="heartbeat-min">{t("ui.127065790492cb0c")}</Label>
                  <Input
                    id="heartbeat-min"
                    type="number"
                    min={10}
                    className="w-40"
                    placeholder={t("ui.62261153ec8a16a1")}
                    value={heartbeatMin}
                    onChange={(e) => setHeartbeatMin(e.target.value)}
                  />
                  <p className="text-muted-foreground text-xs">
                     {t("ui.d97f1cacec38c7e2")} </p>
                </div>
                <div className="grid gap-2">
                  <label htmlFor="seed-first-intent" className="flex items-center gap-2 text-sm">
                    <Checkbox
                      id="seed-first-intent"
                      checked={seedFirstIntent}
                      onCheckedChange={(v) => setSeedFirstIntent(!!v)}
                    />
                     {t("ui.35cfaf26528893ea")} </label>
                  <p className="text-muted-foreground text-xs">
                     {t("ui.95a889159f44ac84")} </p>
                </div>
                <div className="grid gap-2">
                  <label htmlFor="coverage-enabled" className="flex items-center gap-2 text-sm">
                    <Checkbox
                      id="coverage-enabled"
                      checked={coverageEnabled}
                      onCheckedChange={(v) => setCoverageEnabled(!!v)}
                    />
                     {t("ui.6ab6391f5ead96a5")} </label>
                  <p className="text-muted-foreground text-xs">
                     {t("ui.1c3f0b0e4ca0df5b")} </p>
                </div>
              </CollapsibleContent>
            </Collapsible>
          </div>
        </div>

        <SheetFooter className="flex-row justify-end gap-2 border-t p-4">
          <SheetClose asChild>
            <Button variant="outline">{t("common.cancel")}</Button>
          </SheetClose>
          <Button onClick={createTask} disabled={creating || uploading}>
            {creating && <Spinner data-icon="inline-start" />}
            {creating ? t("ui.fee24ff82897e738") : t("common.create")}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}
