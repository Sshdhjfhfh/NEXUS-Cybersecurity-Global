"use client";

import { t } from '@/lib/i18n';

import * as React from "react";

import { getLocalStorageValue, setLocalStorageValue } from "@/lib/local-storage.client";

// Envío de cuadro de entrada de sesión/Tecla de avance de línea. Preferencia pura de front-end: solo soltar localStorage，No almacenado en la base de datos, no sincronizado con la cuenta，
// Entonces, cambiar de navegador requiere un reinicio. ver issue #39——0.3.2 poner Ctrl+Enter Enviar cambiado a Enter enviar，
// Devolver las llaves antiguas aquí es opcional。
export type ChatSendMode = "enter" | "ctrl-enter";

export const CHAT_SEND_MODE_KEY = "nexus_ia_chat_send_mode";
export const DEFAULT_CHAT_SEND_MODE: ChatSendMode = "enter";

export const CHAT_SEND_MODE_OPTIONS: { value: ChatSendMode; label: string }[] = [
  { value: "enter", label: t("ui.e50e785989837f64") },
  { value: "ctrl-enter", label: t("ui.7f84272834c50ad1") },
];

function parseMode(raw: string | null): ChatSendMode {
  return raw === "ctrl-enter" || raw === "enter" ? raw : DEFAULT_CHAT_SEND_MODE;
}

// una colección de suscriptores dentro de la misma pestaña。localStorage de storage El evento sólo se activa en la pestaña ”Otros”，
// Esta página debe cambiarse después de cambiar la configuración. emit Notifique el cuadro de entrada en la misma página; de lo contrario, debe actualizarse para que surta efecto.。
const listeners = new Set<() => void>();

function subscribe(listener: () => void) {
  listeners.add(listener);
  window.addEventListener("storage", listener);
  return () => {
    listeners.delete(listener);
    window.removeEventListener("storage", listener);
  };
}

// Devuelve una cadena literal，Object.is Comparar por valor, no permite useSyncExternalStore atrapado en un bucle。
function getSnapshot(): ChatSendMode {
  return parseMode(getLocalStorageValue(CHAT_SEND_MODE_KEY));
}

// No disponible en el servidor localStorage，renderizar primero el valor predeterminado，hydrate Después getSnapshot Corrección nuevamente。
function getServerSnapshot(): ChatSendMode {
  return DEFAULT_CHAT_SEND_MODE;
}

export function useChatSendMode(): ChatSendMode {
  return React.useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}

export function setChatSendMode(mode: ChatSendMode) {
  setLocalStorageValue(CHAT_SEND_MODE_KEY, mode);
  for (const listener of listeners) listener();
}

// shouldSubmitOnKey Determinar si se debe enviar una pulsación de tecla。
// isComposing / keyCode 229 El método de entrada, como el chino, selecciona palabras y debe liberarse; de lo contrario, la selección de palabras se enviará por error al presionar Enter.。
// enter Patrón excluido únicamente Shift，con 0.3.2 El comportamiento sigue siendo consistente palabra por palabra: los usuarios que no cambien la configuración sentirán lo mismo.。
// ctrl-enter Modos aceptados simultáneamente Ctrl con Cmd（macOS）。
export function shouldSubmitOnKey(e: React.KeyboardEvent, mode: ChatSendMode): boolean {
  if (e.key !== "Enter") return false;
  if (e.nativeEvent.isComposing || e.nativeEvent.keyCode === 229) return false;
  if (mode === "ctrl-enter") return e.ctrlKey || e.metaKey;
  return !e.shiftKey;
}
