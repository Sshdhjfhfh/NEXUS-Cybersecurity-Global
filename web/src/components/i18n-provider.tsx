// Nexus IA v6.0 — I18n Provider
//
// Inicializa i18n al montar la app y aplica el idioma guardado.
// Es un Client Component (necesario para useEffect).
'use client';

import { useEffect, useState, type ReactNode } from 'react';

import { initI18n, useTranslation } from '@/lib/i18n';

export function I18nProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    initI18n()
      .then(() => setReady(true))
      .catch((err) => {
        console.warn('i18n init failed:', err);
        setReady(true);
      });
  }, []);

  if (!ready) {
    return <div style={{ visibility: 'hidden' }}>{children}</div>;
  }

  return <>{children}</>;
}

// Hook re-exportado para conveniencia
export { useTranslation };
