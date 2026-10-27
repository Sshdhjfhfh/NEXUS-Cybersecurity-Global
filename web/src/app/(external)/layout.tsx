import type { ReactNode } from "react";

import { SiteFooter, SiteHeader } from "./_site/chrome";
import { SiteI18nProvider } from "./_site/i18n";

// Public marketing website layout (NEXUS Global Website). Self-contained chrome
// and i18n, independent of the authenticated console.
export default function SiteLayout({ children }: { children: ReactNode }) {
  return (
    <SiteI18nProvider>
      <div className="flex min-h-screen flex-col bg-[#060b18] text-white">
        <SiteHeader />
        <main className="flex-1">{children}</main>
        <SiteFooter />
      </div>
    </SiteI18nProvider>
  );
}
