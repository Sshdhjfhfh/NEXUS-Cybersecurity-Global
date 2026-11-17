import * as React from "react";
import { cn } from "@/lib/utils";

// DataState renders one consistent UI for the four states of any async data
// region — loading, error, empty, or content — so that an ERROR is never shown
// the same as EMPTY. A failed findings fetch that looks like "no findings" gives
// a security team false assurance; the error state here is visually distinct
// (destructive border + a retry) and announced to assistive tech.
//
//   <DataState loading={l} error={e} empty={!items.length} onRetry={reload}>
//     …content…
//   </DataState>

function errorText(error: unknown): string {
  if (!error) return "";
  if (typeof error === "string") return error;
  if (error instanceof Error) return error.message;
  return "Error";
}

export function DataState({
  loading,
  error,
  empty,
  onRetry,
  children,
  loadingLabel = "Cargando…",
  errorTitle = "No se pudieron cargar los datos",
  emptyTitle = "Sin datos",
  emptyHint,
  retryLabel = "Reintentar",
  className,
}: {
  loading?: boolean;
  error?: unknown;
  empty?: boolean;
  onRetry?: () => void;
  children: React.ReactNode;
  loadingLabel?: string;
  errorTitle?: string;
  emptyTitle?: string;
  emptyHint?: string;
  retryLabel?: string;
  className?: string;
}) {
  if (loading) {
    return (
      <div
        role="status"
        aria-live="polite"
        aria-busy="true"
        className={cn("flex items-center justify-center gap-2 py-10 text-sm text-muted-foreground", className)}
      >
        <span
          aria-hidden
          className="size-4 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-muted-foreground"
        />
        {loadingLabel}
      </div>
    );
  }
  if (error) {
    return (
      <div
        role="alert"
        className={cn(
          "flex flex-col items-center gap-3 rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-8 text-center",
          className,
        )}
      >
        <p className="text-sm font-medium text-destructive">{errorTitle}</p>
        <p className="max-w-md break-words text-xs text-muted-foreground">{errorText(error)}</p>
        {onRetry && (
          <button
            type="button"
            onClick={onRetry}
            className="rounded-md border border-border px-3 py-1.5 text-xs font-medium hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {retryLabel}
          </button>
        )}
      </div>
    );
  }
  if (empty) {
    return (
      <div className={cn("flex flex-col items-center gap-1 px-4 py-12 text-center", className)}>
        <p className="text-sm font-medium">{emptyTitle}</p>
        {emptyHint && <p className="max-w-md text-xs text-muted-foreground">{emptyHint}</p>}
      </div>
    );
  }
  return <>{children}</>;
}
