import type { ReactNode } from "react";

import { PortalI18nProvider } from "./_lib/i18n";

export default function PortalLayout({ children }: { children: ReactNode }) {
  return <PortalI18nProvider>{children}</PortalI18nProvider>;
}
