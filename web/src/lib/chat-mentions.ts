export const mentionKinds = [
  { kind: "finding", label: "Hallazgos", token: "\u6f0f\u6d1e", alias: "finding" },
  { kind: "asset", label: "Activos", token: "\u8d44\u4ea7", alias: "asset" },
  { kind: "company", label: "Empresa", token: "\u4f01\u4e1a", alias: "company" },
  { kind: "endpoint", label: "Interfaz", token: "\u63a5\u53e3", alias: "api" },
  { kind: "ip", label: "IP", token: "IP", alias: "ip" },
  { kind: "app", label: "Aplicación", token: "\u5e94\u7528", alias: "app" },
  { kind: "root_domain", label: "Dominio", token: "\u57df\u540d", alias: "domain" },
  { kind: "subdomain", label: "Subdominio", token: "\u5b50\u57df\u540d", alias: "subdomain" },
  { kind: "service", label: "Servicio", token: "\u670d\u52a1", alias: "service" },
] as const;

export type MentionKind = (typeof mentionKinds)[number]["kind"];
export interface ChatMention {
  kind: MentionKind;
  id: number;
  label: string;
  description: string;
}

export function activeMention(value: string, caret: number) {
  const before = value.slice(0, caret);
  const start = before.lastIndexOf("@");
  if (start < 0 || (start > 0 && /[\w.+/-]/.test(before[start - 1]))) return null;
  const query = before.slice(start + 1);
  if (/[[\]\r\n@]/.test(query) || query.length > 220) return null;
  return { start, end: caret, query };
}

export function mentionSearch(query: string) {
  const text = query.trimStart().toLowerCase();
  for (const item of mentionKinds) {
    for (const alias of [item.label.toLowerCase(), item.alias, item.token.toLowerCase()]) {
      if (text === alias || text.startsWith(`${alias} `) || (/[^a-z]/.test(alias) && text.startsWith(alias))) {
        return { kind: item.kind, query: query.trimStart().slice(alias.length).trim(), categories: [] };
      }
    }
  }
  const categories = mentionKinds.filter(
    (item) => item.label.toLowerCase().startsWith(text) || item.alias.startsWith(text) || item.token.toLowerCase().startsWith(text),
  );
  return { kind: "" as const, query: query.trim(), categories };
}

export function mentionToken(item: ChatMention) {
  const kind = mentionKinds.find((entry) => entry.kind === item.kind)?.token ?? "\u8d44\u4ea7";
  const label = item.label
    .replace(/[[\]]/g, (char) => (char === "[" ? "（" : "）"))
    .replace(/\s+/g, " ")
    .slice(0, 100);
  return `@[${kind}#${item.id} ${label}]`;
}

export function selectedMentions(value: string) {
  return [...value.matchAll(/@\[(\u6f0f\u6d1e|\u8d44\u4ea7|\u4f01\u4e1a|\u63a5\u53e3|IP|\u5e94\u7528|\u57df\u540d|\u5b50\u57df\u540d|\u670d\u52a1)#([0-9]+)(?: ([^\]\r\n]*))?\]/g)].map(
    (match) => ({
      token: match[0],
      label: `${mentionKinds.find((item) => item.token === match[1])?.label ?? match[1]} #${match[2]}${match[3] ? ` · ${match[3]}` : ""}`,
      start: match.index,
    }),
  );
}
