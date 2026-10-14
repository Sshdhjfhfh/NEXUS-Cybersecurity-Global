import { t } from '@/lib/i18n';

// Herramienta de análisis para tablas de campos de canales y valores de configuración.。
//
// separado de la página porque ésta es**datos**En lugar de una vista: describe qué campos están disponibles para cada canal、
// Qué controles se deben utilizar y cómo convertir el texto del formulario en valores de configuración（JSON）Conversión bidireccional de。
// Después de colocar un archivo separado, solo necesita moverse aquí para agregar un nuevo canal y no es necesario cambiar la página en sí.。
// El nombre para mostrar y la introducción del tipo de canal. Se coloca en el front-end porque solo afecta la redacción y el back-end no necesita saberlo.。
export const KIND_LABEL: Record<string, string> = {
  dingtalk: t("ui.02bd6f0992ef54f9"),
  feishu: t("ui.292225302513e50a"),
  wecom: t("ui.81b0c96591b7e605"),
  webhook: t("ui.5e181d7268b22539"),
  telegram: "Telegram",
  email: t("ui.5db1fd52d3ea38d0"),
};

// Definiciones de campos de configuración para cada canal。
//
// Aquí se guarda deliberadamente una tabla de campo de front-end en lugar de enviarla al back-end. schema：El backend solo es responsable de
// Validate（requerido/Formato），UI Lo que se necesita es diseño y tipo de control. No les preocupa lo mismo.。
// El único punto de acoplamiento es secret_keys —— El backend proporciona qué campos deben representarse en cuadros de contraseña，
// Porque solo la implementación del canal sabe qué valores se cuentan como credenciales (toda la empresa WeChat Webhook Sólo las credenciales，
// Y Dingding es solo uno de ellos. secret）。Al agregar un nuevo canal, una entrada menos aquí solo dejará el formulario en blanco.，
// No se producirá un error en silencio (a continuación hasFields avisará）。
export type FieldKind = "text" | "password" | "number" | "select" | "textarea" | "switch" | "kv" | "list";
export interface FieldDef {
  key: string;
  label: string;
  kind: FieldKind;
  placeholder?: string;
  help?: string;
  options?: { value: string; label: string }[];
}
export const CHANNEL_FIELDS: Record<string, FieldDef[]> = {
  dingtalk: [
    {
      key: "webhook",
      label: "dirección de webhook",
      kind: "text",
      placeholder: "https://oapi.dingtalk.com/robot/send?access_token=...",
    },
    {
      key: "secret",
      label: t("ui.e6dfce19384904c2"),
      kind: "password",
      help: t("ui.1521331e63f17506"),
    },
  ],
  feishu: [
    {
      key: "webhook",
      label: "dirección de webhook",
      kind: "text",
      placeholder: "https://open.feishu.cn/open-apis/bot/v2/hook/...",
    },
    { key: "secret", label: t("ui.5812f57b4b200bf9"), kind: "password", help: t("ui.37654da6986a6ece") },
  ],
  wecom: [
    {
      key: "webhook",
      label: "dirección de webhook",
      kind: "text",
      placeholder: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...",
    },
  ],
  webhook: [
    { key: "url", label: t("ui.cb0108f07f84c5de"), kind: "text", placeholder: "https://your-endpoint.example.com/hook" },
    {
      key: "method",
      label: t("ui.233ce7f3cc0138b8"),
      kind: "select",
      options: [
        { value: "POST", label: t("ui.8402a31f7eb488f0") },
        { value: "PUT", label: t("ui.177da97ed0a391cf") },
        { value: "PATCH", label: t("ui.98464381dfc7e27f") },
        { value: "GET", label: t("ui.dde57ae28ade685e") },
      ],
    },
    { key: "headers", label: t("ui.129b9483c00931e8"), kind: "kv", help: t("ui.fa8c5f971877161e") },
    {
      key: "body_template",
      label: t("ui.c372bfa41dbbe8fc"),
      kind: "textarea",
      help:
        t("ui.f820a2ac1e1ed02a") +
        t("ui.7201ba1012c00b2a") +
        t("ui.e271bd02d5cfed1e"),
    },
  ],
  telegram: [
    { key: "bot_token", label: "Bot Token", kind: "password", placeholder: "123456:ABC-DEF..." },
    { key: "chat_id", label: "Chat ID", kind: "text", placeholder: "-1001234567890" },
    {
      key: "base_url",
      label: t("ui.2360d692b10b9445"),
      kind: "text",
      placeholder: "https://api.telegram.org",
      help: t("ui.83aa7c5224f8e589"),
    },
  ],
  email: [
    { key: "host", label: t("ui.edcab84d9a20bfd2"), kind: "text", placeholder: "smtp.example.com" },
    {
      key: "port",
      label: "puerto",
      kind: "number",
      placeholder: "587",
      help: t("ui.33c73751b85bdbbb"),
    },
    { key: "username", label: t("ui.e7470f6ac54b7a27"), kind: "text" },
    { key: "password", label: t("ui.a18fee9ce186b72e"), kind: "password" },
    { key: "from", label: t("ui.1fe512ab94b68a80"), kind: "text", placeholder: "nexus@example.com" },
    { key: "to", label: t("ui.449c73e689f81102"), kind: "list", help: t("ui.cbbdbfd42c6bc133") },
    { key: "tls", label: t("ui.ab07f4782f62ed75"), kind: "switch", help: t("ui.4b13ff42b7db8f18") },
  ],
};

export const SEVERITY_OPTIONS = [
  { value: "", label: t("ui.553f5b2ff5d004f1") },
  { value: "low", label: t("ui.905eeb687353b1bc") },
  { value: "medium", label: t("ui.88d47cfd78f3b7ab") },
  { value: "high", label: t("ui.6b3383cbba9ca986") },
  { value: "critical", label: t("ui.cca2201ba28a4c37") },
];

export type ChannelForm = {
  name: string;
  kind: string;
  mode: "realtime" | "digest";
  enabled: boolean;
  ratePerMin: string;
  config: Record<string, unknown>;
  minSeverity: string;
  includeText: string;
  excludeText: string;
  taskIDsText: string;
  assetIDsText: string;
  onStatusChange: boolean;
};

export const emptyForm = (kind: string): ChannelForm => ({
  name: "",
  kind,
  mode: "realtime",
  enabled: true,
  ratePerMin: "",
  config: {},
  minSeverity: "",
  includeText: "",
  excludeText: "",
  taskIDsText: "",
  assetIDsText: "",
  onStatusChange: false,
});

// parseKV Analizar ”Cada línea KEY=VALUE」Campo de texto de。
export function parseKV(text: string): Record<string, string> {
  const out: Record<string, string> = {};
  for (const line of text.split("\n")) {
    const t = line.trim();
    if (!t) continue;
    const i = t.indexOf("=");
    if (i > 0) out[t.slice(0, i).trim()] = t.slice(i + 1).trim();
  }
  return out;
}
// parseIDs analizar comas/espacios en blanco separados id Lista。
export function parseIDs(text: string): number[] {
  return text
    .split(/[\s,，]+/)
    .map((s) => s.trim())
    .filter(Boolean)
    .map((s) => Number(s))
    .filter((n) => Number.isFinite(n) && n > 0);
}
// parseKeywords línea de análisis/Lista de palabras clave separadas por comas (el nombre del tipo de vulnerabilidad puede contener espacios, por lo tanto, córtelo por línea o coma）。
export function parseKeywords(text: string): string[] {
  return text
    .split(/[\n,，]+/)
    .map((s) => s.trim())
    .filter(Boolean);
}
