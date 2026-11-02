import {
  Activity,
  Ban,
  BellRing,
  Bot,
  Brain,
  Bug,
  Building,
  Building2,
  ClipboardCheck,
  ClipboardList,
  FolderOpen,
  FolderSync,
  Globe,
  HeartPulse,
  Inbox,
  LayoutDashboard,
  type LucideIcon,
  MessageSquare,
  Network,
  Plug,
  Radio,
  ScrollText,
  Settings2,
  ShieldAlert,
  ShieldCheck,
  Sparkles,
  Target,
  Terminal,
  Wrench,
} from "lucide-react";

import { t } from "@/lib/i18n";

export type NavBadge = "new" | "soon";

export interface NavSubItem {
  id: string;
  title: string;
  url: string;
  icon?: LucideIcon;
  badge?: NavBadge;
  disabled?: boolean;
  newTab?: boolean;
}

interface NavItemBase {
  id: string;
  title: string;
  icon?: LucideIcon;
  badge?: NavBadge;
  disabled?: boolean;
  newTab?: boolean;
}

export interface NavMainLinkItem extends NavItemBase {
  url: string;
  subItems?: never;
}

export interface NavMainParentItem extends NavItemBase {
  subItems: NavSubItem[];
}

export type NavMainItem = NavMainLinkItem | NavMainParentItem;

export interface NavGroup {
  id: number;
  label?: string;
  items: NavMainItem[];
}

export const sidebarItems: NavGroup[] = [
  {
    id: 1,
    label: t("nav.function"),
    items: [
      { id: "dashboard", title: t("nav.dashboard"), url: "/dashboard", icon: LayoutDashboard },
      { id: "chat", title: t("nav.chat"), url: "/chat", icon: MessageSquare },
      { id: "tasks", title: t("nav.tasks"), url: "/function/tasks", icon: Target },
      { id: "findings", title: t("ui.14311fa4669bfc01"), url: "/function/findings", icon: Bug },
      { id: "traffic", title: t("nav.traffic"), url: "/function/traffic", icon: Activity },
      { id: "commands", title: t("ui.4b2a3c8ce619fffc"), url: "/function/commands", icon: Terminal },
      { id: "llm-records", title: t("ui.c4340f9f49791ffe"), url: "/function/llm-records", icon: Radio },
      { id: "assets", title: t("nav.assets"), url: "/function/assets", icon: Network },
      { id: "lab", title: t("lab.title"), url: "/function/lab", icon: ShieldAlert },
      { id: "sync", title: t("ui.4b80250ca1fa8dad"), url: "/function/sync", icon: FolderSync },
      { id: "workspace", title: t("nav.workspace"), url: "/function/workspace", icon: FolderOpen },
    ],
  },
  {
    id: 2,
    label: t("nav.system"),
    items: [
      { id: "llm", title: "LLM", url: "/system/llm", icon: Brain },
      { id: "agents", title: "Agent", url: "/system/agents", icon: Bot },
      { id: "mcp", title: "MCP", url: "/system/mcp", icon: Plug },
      { id: "skills", title: "Skill", url: "/system/skills", icon: Sparkles },
      { id: "tools", title: t("nav.tools"), url: "/system/tools", icon: Wrench },
      { id: "notify", title: t("ui.a27efbf78b65937a"), url: "/system/notify", icon: BellRing },
      { id: "leads", title: "Leads", url: "/system/leads", icon: Inbox },
      { id: "portal-orgs", title: "Organizaciones", url: "/system/portal-orgs", icon: Building },
      { id: "portal-requests", title: "Portal", url: "/system/portal-requests", icon: Building2 },
      { id: "portal-targets", title: "Objetivos", url: "/system/portal-targets", icon: Globe },
      { id: "portal-retests", title: "Verificaciones", url: "/system/portal-retests", icon: ClipboardCheck },
      { id: "portal-audit", title: "Auditoría", url: "/system/portal-audit", icon: ShieldCheck },
      { id: "ops", title: "Estado del servicio", url: "/system/ops", icon: HeartPulse },
      { id: "intercept", title: t("ui.786df927eaa30c65"), url: "/system/intercept", icon: ShieldAlert },
      { id: "asset-intercept", title: t("ui.a6d0b2b98aeb2541"), url: "/system/intercept/assets", icon: Ban },
      { id: "approvals", title: t("ui.23fbeb68396df26c"), url: "/system/intercept/approvals", icon: ClipboardList },
      { id: "logs", title: t("nav.logs"), url: "/system/logs", icon: ScrollText },
      { id: "mfa", title: "2FA", url: "/system/mfa", icon: ShieldCheck },
      { id: "settings", title: t("ui.65a323f0c17c4950"), url: "/system/settings", icon: Settings2 },
    ],
  },
];
