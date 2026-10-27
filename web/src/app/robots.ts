import type { MetadataRoute } from "next";

const base = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        // Keep the authenticated console and APIs out of search indexes.
        disallow: ["/api/", "/function/", "/system/", "/login", "/setup"],
      },
    ],
    sitemap: `${base}/sitemap.xml`,
  };
}
