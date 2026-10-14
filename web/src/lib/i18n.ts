// Nexus IA v6.0 — Helper i18n para el frontend (Next.js)
//
// Uso:
//   import { t, setLang, useTranslation } from '@/lib/i18n';
//
//   const { t } = useTranslation();
//   <h1>{t('nav.dashboard')}</h1>
//
// Idiomas soportados (10):
//   en, zh, hi, es (default), ar (RTL), fr, bn, pt, id, ru

'use client';

import { useEffect, useState, useCallback } from 'react';

// ═══════════════════════════════════════════════════════════════
// Configuración
// ═══════════════════════════════════════════════════════════════

export const DEFAULT_LANG = 'es';
export const FALLBACK_LANG = 'en';

export const ALL_LANGS = [
  'en', 'zh', 'hi', 'es', 'ar', 'fr', 'bn', 'pt', 'id', 'ru',
] as const;

export type Lang = typeof ALL_LANGS[number];

export const RTL_LANGS: Lang[] = ['ar'];

export const LANG_NAMES: Record<Lang, string> = {
  en: 'English',
  zh: '中文',
  hi: 'हिन्दी',
  es: 'Español',
  ar: 'العربية',
  fr: 'Français',
  bn: 'বাংলা',
  pt: 'Português',
  id: 'Bahasa Indonesia',
  ru: 'Русский',
};

// ═══════════════════════════════════════════════════════════════
// Estado en memoria + localStorage
// ═══════════════════════════════════════════════════════════════

const STORAGE_KEY = 'nexus_lang';

let currentLang: Lang = DEFAULT_LANG;
let translations: Record<string, string> = {};
let fallbackTranslations: Record<string, string> = {};

function isRTL(lang: Lang): boolean {
  return RTL_LANGS.includes(lang);
}

function isValidLang(s: string): s is Lang {
  return (ALL_LANGS as readonly string[]).includes(s);
}

// ═══════════════════════════════════════════════════════════════
// Carga de traducciones
// ═══════════════════════════════════════════════════════════════

/**
 * Carga las traducciones de un idioma.
 * Las sirve desde /locales/<lang>.json (público en Next.js).
 */
async function loadTranslations(lang: Lang): Promise<Record<string, string>> {
  try {
    const resp = await fetch(`/locales/${lang}.json`);
    if (!resp.ok) return {};
    const raw = await resp.json();
    return flatten(raw);
  } catch {
    return {};
  }
}

/**
 * Convierte JSON anidado en claves planas: {a:{b:"x"}} → {"a.b":"x"}
 */
function flatten(obj: Record<string, unknown>, prefix = ''): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (typeof v === 'string') {
      out[key] = v;
    } else if (v && typeof v === 'object') {
      Object.assign(out, flatten(v as Record<string, unknown>, key));
    }
  }
  return out;
}

/**
 * Inicializa i18n. Llama esto una vez al arrancar la app.
 */
export async function initI18n(): Promise<void> {
  // Determinar idioma
  let lang: Lang = DEFAULT_LANG;
  if (typeof window !== 'undefined') {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored && isValidLang(stored)) {
      lang = stored;
    } else if (navigator.language) {
      const nav = navigator.language.split('-')[0];
      if (isValidLang(nav)) lang = nav;
    }
  }

  // Cargar traducciones en paralelo
  const [current, fallback] = await Promise.all([
    loadTranslations(lang),
    lang !== FALLBACK_LANG ? loadTranslations(FALLBACK_LANG) : Promise.resolve({}),
  ]);

  currentLang = lang;
  translations = current;
  fallbackTranslations = fallback;

  applyDir(lang);
}

/**
 * Aplica la dirección (ltr/rtl) al document.
 */
function applyDir(lang: Lang): void {
  if (typeof document === 'undefined') return;
  const dir = isRTL(lang) ? 'rtl' : 'ltr';
  document.documentElement.dir = dir;
  document.documentElement.lang = lang;
}

// ═══════════════════════════════════════════════════════════════
// API pública
// ═══════════════════════════════════════════════════════════════

/**
 * Traduce una clave al idioma actual.
 * Cadena de fallback: current → fallback → [key]
 */
export function t(key: string): string {
  if (translations[key]) return translations[key];
  if (fallbackTranslations[key]) return fallbackTranslations[key];
  return `[${key}]`;
}

/**
 * Cambia el idioma en caliente.
 */
export async function setLang(lang: Lang): Promise<void> {
  if (!isValidLang(lang)) return;
  const current = await loadTranslations(lang);
  const fallback = lang !== FALLBACK_LANG
    ? await loadTranslations(FALLBACK_LANG)
    : {};

  currentLang = lang;
  translations = current;
  fallbackTranslations = fallback;

  if (typeof window !== 'undefined') {
    localStorage.setItem(STORAGE_KEY, lang);
  }
  applyDir(lang);

  // Notificar a los suscriptores
  subscribers.forEach((fn) => fn(lang));
}

/**
 * Devuelve el idioma actual.
 */
export function getLang(): Lang {
  return currentLang;
}

// ═══════════════════════════════════════════════════════════════
// React hook
// ═══════════════════════════════════════════════════════════════

const subscribers = new Set<(lang: Lang) => void>();

/**
 * Hook de React para usar i18n.
 *
 *   const { t, lang, setLang, dir } = useTranslation();
 */
export function useTranslation() {
  const [lang, setLangState] = useState<Lang>(currentLang);

  useEffect(() => {
    const fn = (newLang: Lang) => setLangState(newLang);
    subscribers.add(fn);
    return () => {
      subscribers.delete(fn);
    };
  }, []);

  const changeLang = useCallback(async (newLang: Lang) => {
    await setLang(newLang);
  }, []);

  return {
    t,
    lang,
    setLang: changeLang,
    dir: isRTL(lang) ? 'rtl' : 'ltr',
    isRTL: isRTL(lang),
  };
}
