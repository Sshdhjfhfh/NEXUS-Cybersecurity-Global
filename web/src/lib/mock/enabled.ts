// Mock interruptor. Variables públicas inyectadas durante el tiempo de construcción.（NEXT_PUBLIC_ el prefijo solo es legible por el navegador）。
// Vercel conjunto superior NEXT_PUBLIC_MOCK=1 Recorre toda la estación. mock、No se requiere backend。
export const MOCK = process.env.NEXT_PUBLIC_MOCK === "1";
