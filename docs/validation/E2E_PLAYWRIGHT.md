# Pruebas E2E (Playwright)

Suite para la web pública de NEXUS: `web/e2e/`.

## Requisito único (una sola vez)
El navegador de Playwright necesita librerías del sistema. En WSL/Ubuntu:

```bash
cd ~/nexus-real/web
sudo npx playwright install-deps chromium   # pide tu contraseña de sudo
```

(Equivalente manual: `sudo apt-get install -y libnspr4 libnss3 libasound2t64`.)

## Ejecutar
Con el stack en marcha (next dev :3000 + backend :8787 + Postgres):

```bash
cd ~/nexus-real/web
npm run test:e2e            # desktop (1440) + móvil (390)
npm run test:e2e -- --project=desktop
```

## Qué cubre
- **smoke**: todas las rutas públicas cargan sin error (home, subpáginas, /trust, /legal/*).
- **nav**: enlaces del header (desktop) y menú hamburguesa (móvil) llevan a la ruta correcta.
- **i18n**: el toggle ES↔EN cambia el texto y persiste al navegar.
- **demo-form**: envía el formulario (API interceptada, determinista), valida payload y éxito.
- **cookies**: banner aparece, "Aceptar todo" lo cierra y persiste, se reabre desde el footer.
- **notfound/responsive**: `/legal/*` desconocido muestra not-found; sin scroll horizontal.

La integración real del formulario contra PostgreSQL ya se verificó aparte (lead persistido).
