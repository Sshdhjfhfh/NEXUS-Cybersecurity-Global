import type { ReactNode } from "react";

import { headers } from "next/headers";

import type { Metadata } from "next";

import { I18nProvider } from "@/components/i18n-provider";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { APP_CONFIG } from "@/config/app-config";
import { fontVars } from "@/lib/fonts/registry";
import { PREFERENCE_DEFAULTS } from "@/lib/preferences/preferences-config";
import { ThemeBootScript } from "@/scripts/theme-boot";
import { PreferencesStoreProvider } from "@/stores/preferences/preferences-provider";

import { ConnectionGuard } from "./_status/connection-guard";

import "./globals.css";

export const metadata: Metadata = {
  title: APP_CONFIG.meta.title,
  description: APP_CONFIG.meta.description,
};

export default async function RootLayout({ children }: Readonly<{ children: ReactNode }>) {
  const { theme_mode, theme_preset, content_layout, navbar_style, sidebar_variant, sidebar_collapsible, font } =
    PREFERENCE_DEFAULTS;
  // Nonce for the pre-hydration inline script under the strict CSP (set in proxy.ts).
  // The static export (production build embedded in the backend) has no request:
  // there the backend allows each page's inline scripts by hash instead.
  const nonce = process.env.NEXT_EXPORT === "1" ? undefined : ((await headers()).get("x-nonce") ?? undefined);
  return (
    <html
      lang="es"
      translate="no"
      className="notranslate"
      data-theme-mode={theme_mode}
      data-theme-preset={theme_preset}
      data-content-layout={content_layout}
      data-navbar-style={navbar_style}
      data-sidebar-variant={sidebar_variant}
      data-sidebar-collapsible={sidebar_collapsible}
      data-font={font}
      suppressHydrationWarning
    >
      <head>
        {/* The app handles translation; browser translators can replace nodes React owns. */}
        <meta name="google" content="notranslate" />
        {/* Applies theme and layout preferences on load to avoid flicker and unnecessary server rerenders. */}
        <ThemeBootScript nonce={nonce} />
      </head>
      <body className={`${fontVars} min-h-screen antialiased`}>
        <TooltipProvider>
          <PreferencesStoreProvider
            themeMode={theme_mode}
            themePreset={theme_preset}
            contentLayout={content_layout}
            navbarStyle={navbar_style}
            font={font}
          >
            <I18nProvider>{children}</I18nProvider>
            <Toaster />
            <ConnectionGuard />
          </PreferencesStoreProvider>
        </TooltipProvider>
      </body>
    </html>
  );
}
