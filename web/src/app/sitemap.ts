import type { MetadataRoute } from "next";

const base = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";

// Public marketing routes only (the authenticated console is not indexed).
const routes = [
  "",
  "/platform",
  "/services",
  "/research",
  "/about",
  "/contact",
  "/pricing",
  "/solutions",
  "/request-demo",
  "/trust",
  "/legal",
  "/legal/terms",
  "/legal/privacy",
  "/legal/cookies",
  "/legal/aup",
  "/legal/security",
  "/legal/vdp",
];

export default function sitemap(): MetadataRoute.Sitemap {
  const now = new Date();
  return routes.map((r) => ({
    url: `${base}${r}`,
    lastModified: now,
    changeFrequency: "monthly",
    priority: r === "" ? 1 : 0.7,
  }));
}
