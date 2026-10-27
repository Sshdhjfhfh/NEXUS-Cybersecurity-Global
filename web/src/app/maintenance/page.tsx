"use client";

import { StatusScreen } from "../_status/status-screen";

export default function Maintenance() {
  return (
    <StatusScreen
      tone="text-amber-400"
      es={{
        title: "En mantenimiento",
        message: "Estamos realizando mejoras en la plataforma. Volveremos en breve. Gracias por tu paciencia.",
      }}
      en={{
        title: "Under maintenance",
        message: "We're making improvements to the platform and will be back shortly. Thanks for your patience.",
      }}
    />
  );
}
