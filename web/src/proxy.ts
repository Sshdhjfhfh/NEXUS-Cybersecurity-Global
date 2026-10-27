import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

// Authenticated console route prefixes (route group (main)). Only these require a
// session; everything else is public, so unknown routes fall through to Next and
// render the branded 404 instead of being bounced to /login.
const PROTECTED_PREFIXES = ["/function", "/system", "/dashboard", "/chat"];
const AUTH_PAGES = ["/login", "/setup"];
const MAINTENANCE = process.env.NEXT_PUBLIC_MAINTENANCE === "1";

function underPrefix(pathname: string, prefixes: string[]) {
  return prefixes.some((p) => pathname === p || pathname.startsWith(`${p}/`));
}

export function proxy(request: NextRequest) {
  // Mock demo: no real auth, allow everything (client-side guard still applies).
  if (process.env.NEXT_PUBLIC_MOCK === "1") return NextResponse.next();

  const { pathname } = request.nextUrl;
  const token = request.cookies.get("nexus_ia_token")?.value;
  const isAuthPage = underPrefix(pathname, AUTH_PAGES);
  const isProtected = underPrefix(pathname, PROTECTED_PREFIXES);

  // Maintenance mode takes the public site offline (not the console or login).
  if (MAINTENANCE && !isProtected && !isAuthPage && pathname !== "/maintenance") {
    return NextResponse.redirect(new URL("/maintenance", request.url));
  }

  // Visiting login/setup while already signed in → go to the console.
  if (isAuthPage) {
    if (token) return NextResponse.redirect(new URL("/function/tasks", request.url));
    return NextResponse.next();
  }

  // Console routes require a session.
  if (isProtected && !token) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  // Public marketing, legal, maintenance and unknown routes are served by Next.
  return NextResponse.next();
}

export const config = {
  // Skip Next internals, the API, favicon and static assets (images, video, fonts).
  matcher: [
    "/((?!_next/static|_next/image|favicon\\.ico|api/|.*\\.(?:png|jpg|jpeg|gif|webp|svg|ico|woff2?|ttf|otf|mp4|webm|mov|m4v)$).*)",
  ],
};
