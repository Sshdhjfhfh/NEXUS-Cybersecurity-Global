import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";

// Authenticated console route prefixes (route group (main)). Only these require a
// session; everything else is public, so unknown routes fall through to Next and
// render the branded 404 instead of being bounced to /login.
const PROTECTED_PREFIXES = ["/function", "/system", "/dashboard", "/chat"];
const AUTH_PAGES = ["/login", "/setup"];
const MAINTENANCE = process.env.NEXT_PUBLIC_MAINTENANCE === "1";
const IS_PROD = process.env.NODE_ENV === "production";

function underPrefix(pathname: string, prefixes: string[]) {
  return prefixes.some((p) => pathname === p || pathname.startsWith(`${p}/`));
}

// Per-request nonce for a strict, nonce-based CSP (anti-XSS). Next automatically
// applies this nonce to its own scripts; our inline boot script reads x-nonce.
function makeNonce(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin);
}

function buildCsp(nonce: string): string {
  const base = [
    "default-src 'self'",
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data: blob:",
    "media-src 'self'",
    "font-src 'self' data:",
    "frame-ancestors 'none'",
    "base-uri 'self'",
    "form-action 'self'",
    "object-src 'none'",
  ];
  if (IS_PROD) {
    return [
      ...base,
      `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'`,
      "connect-src 'self'",
      "upgrade-insecure-requests",
    ].join("; ");
  }
  // Dev: HMR needs inline/eval + websockets. Report-only so it never breaks dev.
  return [...base, "script-src 'self' 'unsafe-inline' 'unsafe-eval'", "connect-src 'self' ws: wss:"].join("; ");
}

export function proxy(request: NextRequest) {
  if (process.env.NEXT_PUBLIC_MOCK === "1") return NextResponse.next();

  const { pathname } = request.nextUrl;
  const nonce = makeNonce();
  const csp = buildCsp(nonce);
  const cspHeader = IS_PROD ? "Content-Security-Policy" : "Content-Security-Policy-Report-Only";

  const token = request.cookies.get("nexus_ia_token")?.value;
  const isAuthPage = underPrefix(pathname, AUTH_PAGES);
  const isProtected = underPrefix(pathname, PROTECTED_PREFIXES);

  const redirect = (url: string) => {
    const res = NextResponse.redirect(new URL(url, request.url));
    res.headers.set(cspHeader, csp);
    return res;
  };

  if (MAINTENANCE && !isProtected && !isAuthPage && pathname !== "/maintenance") {
    return redirect("/maintenance");
  }
  if (isAuthPage) {
    if (token) return redirect("/function/tasks");
  } else if (isProtected && !token) {
    return redirect("/login");
  }

  // Pass through: forward the nonce to the page (x-nonce) and set the CSP header.
  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("x-nonce", nonce);
  const res = NextResponse.next({ request: { headers: requestHeaders } });
  res.headers.set(cspHeader, csp);
  return res;
}

export const config = {
  matcher: [
    "/((?!_next/static|_next/image|favicon\\.ico|api/|.*\\.(?:png|jpg|jpeg|gif|webp|svg|ico|woff2?|ttf|otf|mp4|webm|mov|m4v)$).*)",
  ],
};
