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
  // PROVISIONAL placeholder addresses (owner approved inventing these for now).
  // Replace with real, monitored mailboxes before launch — see LAUNCH_CHECKLIST.
  email: {
    general: "contact@nexus-security.io",
    privacy: "privacy@nexus-security.io", // DPO / privacy contact
    security: "security@nexus-security.io", // security / vulnerability disclosure
    support: "support@nexus-security.io",
  },
  emailsAreProvisional: true,
  // Commercial modality: both a B2B SaaS platform and professional (pentesting)
  // security services. Confirmed by the owner.
  modality: "saas-and-services" as const,
  // B2B only (sold to organizations, not retail consumers), so Perú's Libro de
  // Reclamaciones (consumer protection) is treated as NOT applicable — subject to
  // legal confirmation once the selling entity and offering are finalized.
  peruConsumerFacing: false as typeof PENDING | boolean,
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
