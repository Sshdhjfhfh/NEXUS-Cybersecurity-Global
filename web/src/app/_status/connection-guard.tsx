"use client";

import { useEffect, useState } from "react";

import { StatusScreen } from "./status-screen";

// Full-screen overlay shown when the browser goes offline, so a dropped internet
// connection never leaves a blank page. It disappears automatically when the
// connection returns (the `online` event), and offers a manual retry.
export function ConnectionGuard() {
  const [offline, setOffline] = useState(false);

  useEffect(() => {
    const sync = () => setOffline(typeof navigator !== "undefined" && navigator.onLine === false);
    sync();
    window.addEventListener("online", sync);
    window.addEventListener("offline", sync);
    return () => {
      window.removeEventListener("online", sync);
      window.removeEventListener("offline", sync);
    };
  }, []);

  if (!offline) return null;

  return (
    <div className="fixed inset-0 z-[100]">
      <StatusScreen
        spinner
        es={{
          title: "Se perdió la conexión a Internet",
          message: "Estamos intentando reconectarte. La página volverá automáticamente en cuanto regrese la conexión.",
          cta: "Reintentar",
        }}
        en={{
          title: "Internet connection lost",
          message: "We're trying to reconnect you. The page will come back automatically once you're online again.",
          cta: "Retry",
        }}
        onAction={() => {
          if (typeof window !== "undefined") window.location.reload();
        }}
      />
    </div>
  );
}
