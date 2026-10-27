"use client";

import { StatusScreen } from "./_status/status-screen";

// Root error boundary: a branded screen with retry instead of a blank page when
// a runtime error bubbles up.
export default function AppError({ reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return (
    <StatusScreen
      tone="text-rose-400"
      es={{
        title: "Algo salió mal",
        message: "Ocurrió un error inesperado. Puedes reintentar o volver más tarde.",
        cta: "Reintentar",
      }}
      en={{
        title: "Something went wrong",
        message: "An unexpected error occurred. You can retry or come back later.",
        cta: "Retry",
      }}
      onAction={() => reset()}
    />
  );
}
