// Centralized corporate / legal configuration for NEXUS Cybersecurity Global.
//
// Every value marked PENDING must be supplied by the business BEFORE any
// contractual document, legal page, or footer legal notice is published.
// Do NOT invent legal names, registration numbers, addresses, emails or
// jurisdictions — leaving PENDING is correct until the real data exists.
export const PENDING = "PENDIENTE" as const;

export const company = {
  tradeName: "NEXUS Cybersecurity Global",
  legalName: PENDING, // razón social registrada
  countryOfIncorporation: PENDING,
  registrationNumber: PENDING, // RUC / company registration number
  legalAddress: PENDING,
  contractJurisdiction: PENDING,
  email: {
    general: PENDING,
    privacy: PENDING, // DPO / privacy contact
    security: PENDING, // security / vulnerability disclosure
    support: PENDING,
  },
  // Commercial modality decides whether Perú's Libro de Reclamaciones applies.
  // Set once the business model and selling entity are confirmed.
  peruConsumerFacing: PENDING as typeof PENDING | boolean,
} as const;

// Per-document metadata: a legal page is only publishable once its draft has
// been reviewed and these are filled in.
export type DocStatus = "pending" | "draft" | "legal-review" | "approved";
export const legalDocsStatus: Record<string, { version: string; effectiveDate: string; status: DocStatus }> = {
  terms: { version: PENDING, effectiveDate: PENDING, status: "pending" },
  privacy: { version: PENDING, effectiveDate: PENDING, status: "pending" },
  cookies: { version: PENDING, effectiveDate: PENDING, status: "pending" },
  aup: { version: PENDING, effectiveDate: PENDING, status: "pending" },
  security: { version: PENDING, effectiveDate: PENDING, status: "pending" },
  vdp: { version: PENDING, effectiveDate: PENDING, status: "pending" },
  dpa: { version: PENDING, effectiveDate: PENDING, status: "pending" },
  sla: { version: PENDING, effectiveDate: PENDING, status: "pending" },
};

export function isConfigured(v: string | boolean): boolean {
  return v !== PENDING && (typeof v !== "string" || v.trim() !== "");
}
