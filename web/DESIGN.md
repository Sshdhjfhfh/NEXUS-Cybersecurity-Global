# NEXUS — Sistema de diseño

Una sola fuente de verdad visual para el **Client Portal** (empresas, gobiernos,
organizaciones) y la **Admin Console** interna. Objetivo: que un ministerio o un
banco confíe sus datos más sensibles — sobrio, claro, rápido y auditable.

> Estado: **en construcción (incremento 1).** Establece los tokens base y el
> primer componente compartido. Las pantallas se migran de forma progresiva
> sobre esta base; no se reconstruye la app desde cero.

## 1. Tokens de color (`src/app/globals.css`)

Definidos como variables CSS en `:root` (claro) y `.dark` (oscuro), expuestos a
Tailwind v4 vía `@theme inline` (`bg-sev-critical`, `text-brand-cyan`, …).

**Marca** (del directivo): navy `#060B18` · cian `#22D3EE` · azul eléctrico `#2563EB`.
En claro el cian se oscurece a `--brand-cyan: #0E7490` para contraste AA de texto.

**Rampa de severidad — ÚNICA, consumida por portal y consola.** La severidad es
**independiente del tema** (una Crítica es roja en claro y en oscuro), renderizada
como *badge sólido* con su `*-foreground`:

| Severidad | Token | Color | Letra |
| --- | --- | --- | --- |
| Crítica | `--sev-critical` | `#B91C1C` | C |
| Alta | `--sev-high` | `#C2410C` (naranja, fijo) | A |
| Media | `--sev-medium` | `#A16207` | M |
| Baja | `--sev-low` | `#1D4ED8` | B |
| Info | `--sev-info` | `#475569` | I |

> Antes: la consola mostraba Alta=rojo y Baja=gris; el portal, Alta=naranja y
> Baja=azul. El color de severidad es *la* señal de una herramienta de seguridad;
> ahora hay una sola rampa.

**Gráficos.** `--chart-1..5` eran cinco grises; ahora es una paleta categórica de
marca (azul/cian/violeta/naranja/verde), con variantes más brillantes en oscuro.

## 2. Accesibilidad (WCAG 2.1 AA)

- La severidad se codifica por **color + letra + etiqueta**, nunca solo color
  (daltonismo, impresión en gris). `SeverityBadge` lleva `aria-label`.
- Badges sólidos con texto `*-foreground` blanco → contraste ≥ AA sobre tarjetas
  claras y oscuras (se evita el texto tintado al 15% que fallaba AA).

## 3. Componentes base

- **`SeverityBadge`** (`src/components/severity-badge.tsx`) — HECHO. Compartido
  por portal y consola. `normalizeSeverity()` mapea los strings del backend (es/en).
- Próximos: `DataState` (loading/error/empty con reintento — hoy "error" se ve
  igual que "sin datos"), `StatCard`, `DataTable`, `ClassificationTag`,
  `PageHeader`, selector de organización + búsqueda `Ctrl+K`.

## 4. Tipografía

Geist (ya en el repo). Escala: 12 / 14 / 16 / 20 / 24 / 30 / 36. Alto contraste,
jerarquía clara.

## 5. Plan de pantallas

- **Portal:** Resumen → Hallazgos (tabla + detalle) → Evaluaciones (con flujo de
  autorización / alcance firmado) → Informes.
- **Consola:** Resumen operativo → Solicitudes/Objetivos → Hallazgos/Tráfico →
  Auditoría (migración progresiva a los tokens y componentes de arriba).

## 6. Reglas

- Nada de colores Tailwind crudos (`bg-red-500`) para semántica: usar tokens.
- Datos de ejemplo siempre etiquetados "Datos de ejemplo". Sin inventar clientes,
  cifras ni certificaciones.
- Cada pantalla se revisa en claro/oscuro y móvil, y pasa a11y + crítica de diseño.
