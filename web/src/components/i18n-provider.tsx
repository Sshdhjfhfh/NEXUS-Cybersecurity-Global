// Nexus IA v6.0 — I18n Provider
//
// PROBLEMA que resuelve:
//   El proyecto es static export. El HTML se genera en build time, donde
//   no hay window ni fetch disponible. Si renderizamos children antes de
//   cargar las traducciones, quedan los [key] fijos en el HTML.
//
// SOLUCIÓN:
//   No renderizar children hasta que initI18n() haya terminado.
//   En SSR: devuelve un placeholder.
//   En cliente: carga los JSON, luego renderiza children con t() funcionando.
'use client';

import { useEffect, useState, type ReactNode } from 'react';

import { initI18n, useTranslation } from '@/lib/i18n';

export function I18nProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    initI18n()
      .then(() => setReady(true))
      .catch((err) => {
        console.warn('i18n init failed:', err);
        setError(String(err));
        setReady(true); // Mostrar de todas formas
      });
  }, []);

  if (!ready) {
    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: '100vh',
          color: '#888',
          fontSize: '14px',
          fontFamily: 'system-ui, sans-serif',
        }}
      >
        Cargando...
      </div>
    );
  }

  if (error) {
    console.warn('i18n error:', error);
  }

  return <>{children}</>;
}

// Hook re-exportado para conveniencia
export { useTranslation };
