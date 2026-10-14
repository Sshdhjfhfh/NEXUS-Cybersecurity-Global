import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// Cajón/cuadro de diálogo(Sheet/Dialog)de onInteractOutside Desactivar la asistencia de juicio。
//
// Antecedentes:En el cajón Radix capa elástica(Select tirar hacia abajo、DropdownMenu、Popover espera)Sí portal al cajón
// Afuera. Haga clic en la máscara cuando la capa elástica esté activada/quiero guardarlo fuera del cajón,esta vez pointerdown será Select y Sheet dos
// DismissableLayer Procesar simultáneamente;Select Cierra primero y si discrete Evento、React se sincronizará flush,Entonces
// girar Sheet El procesador es una capa elástica. data-state Ya traducido closed —— en"ahora"Detectar si la capa elástica está abierta
// Naturalmente poco confiable(Medición real verificada)。
//
// enfoque correcto:Radix de pointerdown Escuchando en la etapa burbujeante;somos capture escenario(antes que eso)primero
// "¿Hay alguna capa elástica abierta en este momento?"grábalo,onInteractOutside lea este valor de registro nuevamente para decidir si liberar el apagado。
function isRadixOverlayOpenNow(): boolean {
  if (typeof document === "undefined") return false;
  return !!document.querySelector(
    [
      "[data-slot='select-trigger'][data-state='open']",
      "[data-slot='select-content'][data-state='open']",
      "[role='listbox'][data-state='open']",
      "[data-radix-popper-content-wrapper]",
      "[aria-expanded='true'][data-state='open']",
    ].join(","),
  );
}

let overlayOpenAtLastPointerDown = false;
if (typeof document !== "undefined") {
  document.addEventListener(
    "pointerdown",
    () => {
      overlayOpenAtLastPointerDown = isRadixOverlayOpenNow();
    },
    true, // capture:Consíguelo primero Radix En etapa de burbujeo pointerdown Grabado ante el procesador
  );
}

// radixOverlayWasOpenAtPointerDown Regreso"más reciente pointerdown Si hay Radix capa elástica
// abierto"。Cajón/cuadro de diálogo correspondiente:Máscara de puntos cuando la capa elástica está encendida → Sólo se recoge la capa elástica, no ella misma。
export function radixOverlayWasOpenAtPointerDown(): boolean {
  return overlayOpenAtLastPointerDown;
}

// copyText escribir texto en el portapapeles,Devolver si tuvo éxito。
// Antecedentes:navigator.clipboard Sólo en contexto de seguridad(HTTPS / localhost)Disponible;Aprobado IP + HTTP
// Cuando se accede es undefined,Bajar de categoría a execCommand("copy")。
export async function copyText(text: string): Promise<boolean> {
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      // Continuar con el plan de degradación
    }
  }
  try {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.style.position = "fixed";
    textarea.style.left = "-9999px";
    textarea.style.top = "0";
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    const ok = document.execCommand("copy");
    document.body.removeChild(textarea);
    return ok;
  } catch {
    return false;
  }
}

export const getInitials = (str: string): string => {
  if (typeof str !== "string" || !str.trim()) return "?";

  return (
    str
      .trim()
      .split(/\s+/)
      .filter(Boolean)
      .map((word) => word[0])
      .join("")
      .toUpperCase() || "?"
  );
};

export function formatCurrency(
  amount: number,
  opts?: {
    currency?: string;
    locale?: string;
    minimumFractionDigits?: number;
    maximumFractionDigits?: number;
    noDecimals?: boolean;
  },
) {
  const { currency = "USD", locale = "en-US", minimumFractionDigits, maximumFractionDigits, noDecimals } = opts ?? {};

  const formatOptions: Intl.NumberFormatOptions = {
    style: "currency",
    currency,
    minimumFractionDigits: noDecimals ? 0 : minimumFractionDigits,
    maximumFractionDigits: noDecimals ? 0 : maximumFractionDigits,
  };

  return new Intl.NumberFormat(locale, formatOptions).format(amount);
}
