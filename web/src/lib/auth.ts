const TOKEN_KEY = "nexus_ia_token";
const COOKIE_MAX_AGE = 7 * 24 * 60 * 60; // 7 días (segundos）

export interface CurrentUser {
  id: string;
  name: string;
  username: string;
  email: string;
  avatar: string;
  role: string;
}

export const auth = {
  getToken(): string | null {
    if (typeof window === "undefined") return null;
    // Mock demo：No hay inicio de sesión real, devuelve un valor falso token Deje pasar el guardia de enrutamiento e ingrese directamente a la interfaz principal.。
    return localStorage.getItem(TOKEN_KEY) ?? (process.env.NEXT_PUBLIC_MOCK === "1" ? "mock-demo" : null);
  },

  setToken(token: string): void {
    localStorage.setItem(TOKEN_KEY, token);
    // escritura sincrónica cookie，suministro Next.js middleware lectura del servidor
    document.cookie = `${TOKEN_KEY}=${encodeURIComponent(token)}; path=/; max-age=${COOKIE_MAX_AGE}; SameSite=Lax`;
  },

  clearToken(): void {
    localStorage.removeItem(TOKEN_KEY);
    document.cookie = `${TOKEN_KEY}=; path=/; max-age=0`;
  },

  // desde JWT payload de sub El campo analiza el usuario actual. Solo se utiliza para visualización y no realiza verificación de firma.。
  getCurrentUser(): CurrentUser | null {
    const token = this.getToken();
    if (!token) return null;
    try {
      const parts = token.split(".");
      if (parts.length !== 3) return null;
      // base64url → base64
      const payload = JSON.parse(atob(parts[1].replace(/-/g, "+").replace(/_/g, "/")));
      const subject: string = payload.sub ?? "nexus";
      const username = subject === "ARTEX" ? "nexus" : subject;
      return { id: "1", name: username, username, email: "", avatar: "", role: "operator" };
    } catch {
      return null;
    }
  },
};
