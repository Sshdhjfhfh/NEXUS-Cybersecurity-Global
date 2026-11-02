import type { MetadataRoute } from "next";

const base = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";

// Generated at build time (also for the static export embedded in the backend).
export const dynamic = "force-static";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        // Keep the console, the client portal and the APIs out of search indexes.
        disallow: ["/api/", "/function/", "/system/", "/portal/", "/login", "/setup"],
      },
    ],
    sitemap: `${base}/sitemap.xml`,
  };
}
