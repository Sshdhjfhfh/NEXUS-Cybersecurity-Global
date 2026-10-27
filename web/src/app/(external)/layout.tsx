import type { ReactNode } from "react";

import type { Metadata } from "next";

import { SiteFooter, SiteHeader } from "./_site/chrome";
import { CookieConsent } from "./_site/cookie-consent";
import { SiteI18nProvider } from "./_site/i18n";

const base = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";
const description =
  "AI-powered offensive and defensive security, governed by humans and bounded by verifiable authorization. NEXUS Cybersecurity Global.";

export const metadata: Metadata = {
  metadataBase: new URL(base),
  title: { default: "NEXUS Cybersecurity Global", template: "%s · NEXUS Cybersecurity Global" },
  description,
  applicationName: "NEXUS Cybersecurity Global",
  robots: { index: true, follow: true },
  openGraph: {
    type: "website",
    siteName: "NEXUS Cybersecurity Global",
    title: "NEXUS Cybersecurity Global",
    description,
    images: [{ url: "/site/hero-soc.webp", width: 1600, height: 900, alt: "NEXUS Cybersecurity Global" }],
  },
  twitter: { card: "summary_large_image", title: "NEXUS Cybersecurity Global", description },
};

// Public marketing website layout (NEXUS Global Website). Self-contained chrome
// and i18n, independent of the authenticated console.
export default function SiteLayout({ children }: { children: ReactNode }) {
  return (
    <SiteI18nProvider>
      <div className="flex min-h-screen flex-col bg-[#060b18] text-white">
        <SiteHeader />
        <main className="flex-1">{children}</main>
        <SiteFooter />
        <CookieConsent />
      </div>
    </SiteI18nProvider>
  );
}
