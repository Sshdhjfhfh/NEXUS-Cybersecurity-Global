"use client";

import { t } from '@/lib/i18n';

// LLM Reintentar el software compartido configurado: el número de reintentos para cada una de las cinco capas + Intervalo」。
//
// Cinco pisos de adentro hacia afuera: Jianlian(SDK) → respuesta vacía(SDK) → Mismo provider ventana de seguridad → disyuntor de sondeo → Intención de volver a correr。
// Las primeras tres capas siguen el punto final, por lo que cada configuración del modelo puede anular el valor predeterminado global; las dos últimas capas están a nivel de proceso y solo tienen una copia global.。
//
// Todas las entradas siguen el mismo conjunto de ”Dejar en blanco = Semántica y backend ”No configurar” db.RetryRule consistente：
//   veces   vacío/0 = Usar valor predeterminado incorporado | -1 = Cierra esta capa y vuelve a intentarlo. | >0 = use esta cantidad de veces
//   Intervalo   vacío/0 = utilice el índice original de esta capa para retirarse | >0 = utilice este intervalo fijo de milisegundos en su lugar

import * as React from "react";

import { Loader2Icon, SaveIcon } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api } from "@/lib/api";
import type { LLMRetryOverride, LLMRetryPolicy, LLMRetryRule } from "@/lib/types";

export const ZERO_RULE: LLMRetryRule = { attempts: 0, interval_ms: 0 };
export const ZERO_OVERRIDE: LLMRetryOverride = {
  connect: ZERO_RULE,
  empty: ZERO_RULE,
  stream: ZERO_RULE,
};
const ZERO_POLICY: LLMRetryPolicy = {
  ...ZERO_OVERRIDE,
  breaker: ZERO_RULE,
  intent: ZERO_RULE,
};

type LayerMeta = {
  title: string;
  /** ¿Dónde ocurre esta capa de reintentos y quién los realiza? */
  where: string;
  /** ¿Qué tipo de error alcanzará este nivel? Específico del código de estado, no dejes que nadie adivine */
  trigger: string;
  /** Es un error parecer similar pero [no] llegar a este nivel, para no llenar el espacio en blanco y pensar que es la respuesta. bug */
  skips?: string;
  desc: string;
  attemptsLabel: string;
  /** Valor predeterminado cuando el número de veces se deja en blanco, se utiliza como marcador de posición */
  defAttempts: number;
  /** estrategia predeterminada cuando el intervalo se deja vacío, se utiliza para marcadores de posición */
  defInterval: string;
  /** Complete el número de veces -1 significado */
  offHint: string;
};

export const RETRY_LAYERS = {
  connect: {
    title: t("ui.216cb6b5e008f84d"),
    where: t("ui.bcaf3634f94bc9ec"),
    trigger:
      t("ui.cad212608c85f15b"),
    skips: t("ui.092edc5b33d6bc54"),
    desc: t("ui.6b0f5526b5b0d570"),
    attemptsLabel: t("ui.32de6fc673128a16"),
    defAttempts: 3,
    defInterval: t("ui.210d780a2be0dff2"),
    offHint: t("ui.97348b72d24f7fd2"),
  },
  empty: {
    title: t("ui.e618fa78518004d4"),
    where: t("ui.1ad08a9babc4a195"),
    trigger:
      t("ui.1cf7203bbe1c5567"),
    skips: t("ui.80621b670c57052a"),
    desc: t("ui.d525e976e18266d5"),
    attemptsLabel: t("ui.32de6fc673128a16"),
    defAttempts: 2,
    defInterval: t("ui.210d780a2be0dff2"),
    offHint: t("ui.2c22fe5b444d3a59"),
  },
  stream: {
    title: t("ui.71f878af442f5853"),
    where: t("ui.15736a531f631c0c"),
    trigger:
      t("ui.1a09f547937b3e8e"),
    skips:
      t("ui.dc74657651693bf5"),
    desc: t("ui.b221860d47473e0f"),
    attemptsLabel: t("ui.32de6fc673128a16"),
    defAttempts: 2,
    defInterval: t("ui.6af5f31681f3f3da"),
    offHint: t("ui.3081fa86e20718d5"),
  },
  breaker: {
    title: t("ui.c227914a7a050089"),
    where: t("ui.3a86334b63434fc6"),
    trigger:
      t("ui.b8494f9c27891270"),
    skips: t("ui.b001fa8a60d33d91"),
    desc: t("ui.4def89ae2acac103"),
    attemptsLabel: t("ui.38f8270e98a6d7fe"),
    defAttempts: 3,
    defInterval: t("ui.93f43d77a853a73a"),
    offHint: t("ui.972a681c92412a6a"),
  },
  intent: {
    title: t("ui.864460a0a9158de9"),
    where: t("ui.3a86334b63434fc6"),
    trigger:
      t("ui.6689f01152d31184"),
    skips: t("ui.4be3494865a26c68"),
    desc: t("ui.bfb982596ce2cc35"),
    attemptsLabel: t("ui.1ea5362b9acb9ccb"),
    defAttempts: 2,
    defInterval: t("ui.e83efd7e7ed49e5b"),
    offHint: t("ui.c6dc0bc7f1aa2d45"),
  },
} satisfies Record<string, LayerMeta>;

type LayerKey = keyof typeof RETRY_LAYERS;

/** palabras humanas en milisegundos, solo se usan para hacer eco junto al cuadro de entrada para evitar contar ceros。 */
function humanMs(ms: number) {
  if (!Number.isFinite(ms) || ms <= 0) return "";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60_000) return `${Number((ms / 1000).toFixed(2))}s`;
  return `${Number((ms / 60_000).toFixed(2))}min`;
}

/** Entrada digital controlada: cadena vacía ↔ 0，estado intermedio（"-"、"1e"）Déjelo local como está, sin molestar al padre.。 */
function NumField({
  id,
  value,
  onChange,
  placeholder,
  min,
}: {
  id: string;
  value: number;
  onChange: (n: number) => void;
  placeholder: string;
  min: number;
}) {
  const [text, setText] = React.useState(value === 0 ? "" : String(value));
  // mantenerse al día cuando el padre cambia un conjunto completo de valores (leer políticas, cambiar configuraciones); no llegarás aquí cuando escribas tú mismo，
  // porque entonces value Ya es igual al texto local parse El resultado después。
  React.useEffect(() => {
    const incoming = value === 0 ? "" : String(value);
    setText((cur) => (Number(cur || 0) === value ? cur : incoming));
  }, [value]);
  return (
    <Input
      id={id}
      type="number"
      min={min}
      className="w-28 shrink-0"
      value={text}
      placeholder={placeholder}
      onChange={(e) => {
        setText(e.target.value);
        const n = Number(e.target.value);
        onChange(e.target.value.trim() === "" || !Number.isFinite(n) ? 0 : Math.trunc(n));
      }}
    />
  );
}

/** dos botones para una capa de reintentos。idPrefix Se utiliza para guardar cuando la misma página aparece varias veces. label de htmlFor。 */
export function RetryRuleFields({
  layer,
  idPrefix,
  value,
  onChange,
  compact,
}: {
  layer: LayerKey;
  idPrefix: string;
  value: LLMRetryRule;
  onChange: (r: LLMRetryRule) => void;
  /** true = Versión compacta del cajón de configuración: omitir la descripción de la expansión, dejando solo la frase ”¿Qué error llegará a este nivel?” */
  compact?: boolean;
}) {
  const meta = RETRY_LAYERS[layer];
  const human = humanMs(value.interval_ms);
  return (
    <div className={compact ? "grid gap-2" : "grid gap-3 rounded-lg border p-3"}>
      <div className="grid gap-0.5">
        <div className="flex flex-wrap items-baseline gap-2">
          <Label className="text-sm">{meta.title}</Label>
          <span className="text-muted-foreground text-xs">{meta.where}</span>
        </div>
        {/* Qué errores alcanzarán este nivel, específicamente el código de estado; si completa la perilla pero no puede ver el efecto, probablemente se deba a que el error no cae en este nivel en absoluto.。 */}
        <p className="text-muted-foreground text-xs">
          <span className="font-medium text-foreground">{t("ui.b8734b28075b3344")}</span>：{meta.trigger}
        </p>
        {!compact && meta.skips && (
          <p className="text-muted-foreground text-xs">
            <span className="font-medium text-foreground">{t("ui.e95f0d25667b7113")}</span>：{meta.skips}
          </p>
        )}
        {!compact && <p className="text-muted-foreground text-xs">{meta.desc}</p>}
      </div>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <div className="flex items-center gap-2">
          <Label htmlFor={`${idPrefix}-${layer}-n`} className="text-muted-foreground text-xs">
            {meta.attemptsLabel}
          </Label>
          <NumField
            id={`${idPrefix}-${layer}-n`}
            min={-1}
            value={value.attempts}
            placeholder={`predeterminado ${meta.defAttempts}`}
            onChange={(n) => onChange({ ...value, attempts: n })}
          />
        </div>
        <div className="flex items-center gap-2">
          <Label htmlFor={`${idPrefix}-${layer}-ms`} className="text-muted-foreground text-xs">
             {t("ui.a06d982b9db13ff7")} </Label>
          <NumField
            id={`${idPrefix}-${layer}-ms`}
            min={0}
            value={value.interval_ms}
            placeholder={t("ui.25c1273300b50f1d")}
            onChange={(n) => onChange({ ...value, interval_ms: n })}
          />
          <span className="text-muted-foreground text-xs">{human ? `Fijo ${human}` : meta.defInterval}</span>
        </div>
      </div>
      {!compact && <p className="text-muted-foreground text-xs">{t("ui.2ca8a46354dc49d5")}{meta.offHint}。</p>}
    </div>
  );
}

/** Las tres capas de cobertura en el cajón de configuración del modelo (las tres capas que siguen los puntos finales）。 */
export function ProfileRetryFields({
  value,
  onChange,
}: {
  value: LLMRetryOverride;
  onChange: (o: LLMRetryOverride) => void;
}) {
  return (
    <div className="grid gap-3 rounded-lg border p-3">
      <div className="grid gap-0.5">
        <Label className="text-sm">{t("ui.72eb690bd598626f")}</Label>
        <p className="text-muted-foreground text-xs">
           {t("ui.e493a79d6cd95fbe")} </p>
      </div>
      {(["connect", "empty", "stream"] as const).map((k) => (
        <div key={k} className="border-t pt-3 first:border-t-0 first:pt-0">
          <RetryRuleFields
            compact
            layer={k}
            idPrefix="pf"
            value={value[k]}
            onChange={(r) => onChange({ ...value, [k]: r })}
          />
        </div>
      ))}
    </div>
  );
}

/** 「Reintentar y retroceder」tab：valor predeterminado global para la capa cinco。 */
export function RetryPolicyPanel() {
  const [policy, setPolicy] = React.useState<LLMRetryPolicy>(ZERO_POLICY);
  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);

  const load = React.useCallback(async () => {
    setLoading(true);
    try {
      const p = await api.llmRetryPolicy();
      setPolicy({ ...ZERO_POLICY, ...p });
    } catch (e) {
      toast.error(`No se pudo leer la política de reintento:${(e as Error).message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    void load();
  }, [load]);

  async function save() {
    if (saving) return;
    setSaving(true);
    try {
      // El backend volverá a fijar el valor fuera de límites en el rango, lo devolverá y lo actualizará directamente con el valor devuelto. Lo que ves es lo que guardas.。
      const saved = await api.saveLLMRetryPolicy(policy);
      setPolicy({ ...ZERO_POLICY, ...saved });
      toast.success(t("ui.7412f0441f9cf715"));
    } catch (e) {
      toast.error(`Error al guardar:${(e as Error).message}`);
    } finally {
      setSaving(false);
    }
  }

  const set = (k: LayerKey) => (r: LLMRetryRule) => setPolicy((p) => ({ ...p, [k]: r }));

  if (loading) {
    return (
      <div className="flex items-center gap-2 rounded-lg border border-dashed p-10 text-muted-foreground text-sm">
        <Loader2Icon className="size-4 animate-spin" />  {t("ui.c5d3f9920304792f")} </div>
    );
  }

  return (
    <div className="grid gap-4">
      <div className="rounded-lg border bg-muted/30 p-3 text-muted-foreground text-xs leading-relaxed">
         {t("ui.b6c095db1bd4441c")} <span className="text-foreground">  {t("ui.c09f1b324cec7d91")}</span>
         {t("ui.0509d6d472fc7f46")} <span className="text-foreground">{t("ui.e03d1480beecff0b")}</span>
         {t("ui.f44837a29139ba64")} </div>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {(Object.keys(RETRY_LAYERS) as LayerKey[]).map((k) => (
          <RetryRuleFields key={k} layer={k} idPrefix="gl" value={policy[k]} onChange={set(k)} />
        ))}
      </div>

      <div className="flex gap-2">
        <Button onClick={save} disabled={saving}>
          {saving ? <Loader2Icon className="animate-spin" /> : <SaveIcon />}
          guardar </Button>
        <Button variant="outline" onClick={() => setPolicy(ZERO_POLICY)} disabled={saving}>
           {t("ui.5870a2ad3024704c")} </Button>
      </div>
    </div>
  );
}
