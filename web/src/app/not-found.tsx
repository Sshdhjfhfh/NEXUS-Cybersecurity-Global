"use client";

import { StatusScreen } from "./_status/status-screen";

export default function NotFound() {
  return (
    <StatusScreen
      code="404"
      es={{
        title: "Página no encontrada",
        message: "La página que buscas no existe o fue movida.",
        cta: "Volver al inicio",
      }}
      en={{
        title: "Page not found",
        message: "The page you're looking for doesn't exist or has moved.",
        cta: "Back to home",
      }}
      href="/"
    />
  );
}
