import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

const AUTH_PAGES = ["/login", "/setup"];
// Web pública de marketing (grupo de rutas (external)): accesible sin sesión.
const PUBLIC_PAGES = [
  "/",
  "/solutions",
  "/request-demo",
  "/platform",
  "/services",
  "/research",
  "/about",
  "/contact",
  "/pricing",
  "/trust",
  "/sitemap.xml",
  "/robots.txt",
];
// Prefijos públicos (p. ej. documentos legales en /legal/*).
const PUBLIC_PREFIXES = ["/legal"];

export function proxy(request: NextRequest) {
  // Mock demo：No hay inicio de sesión real, todas las páginas están permitidas (cliente auth El guardia también te dejará ir）。
  if (process.env.NEXT_PUBLIC_MOCK === "1") return NextResponse.next();

  const { pathname } = request.nextUrl;
  if (PUBLIC_PAGES.includes(pathname)) return NextResponse.next();
  if (PUBLIC_PREFIXES.some((p) => pathname === p || pathname.startsWith(`${p}/`))) {
    return NextResponse.next();
  }

  const token = request.cookies.get("nexus_ia_token")?.value;
  const isAuthPage = AUTH_PAGES.some((p) => pathname === p || pathname.startsWith(`${p}/`));

  // No iniciado sesión → Saltar a la página de inicio de sesión
  if (!token && !isAuthPage) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  // Acceder al inicio de sesión cuando ya haya iniciado sesión/página de inicialización → saltar a la interfaz principal
  if (token && isAuthPage) {
    return NextResponse.redirect(new URL("/function/tasks", request.url));
  }

  return NextResponse.next();
}

export const config = {
  // saltar Next.js enrutamiento interno、API Enrutamiento、favicon y public/ archivos estáticos (incluidas imágenes, fuentes, etc.)）
  matcher: [
    "/((?!_next/static|_next/image|favicon\\.ico|api/|.*\\.(?:png|jpg|jpeg|gif|webp|svg|ico|woff2?|ttf|otf|mp4|webm|mov|m4v)$).*)",
  ],
};
