// Draft legal content for NEXUS. These are PRELIMINARY drafts pending review by a
// qualified lawyer — never presented as final/approved. Company-specific data
// (legal name, address, jurisdiction) comes from web/src/config/company.ts and is
// shown as PENDING until provided. Do not invent corporate facts here.

export type LegalSection = { h: string; p: string[] };
export type LegalDocContent = {
  title: { en: string; es: string };
  intro: { en: string; es: string };
  sections: { en: LegalSection[]; es: LegalSection[] };
};

export const LEGAL_DOCS: Record<string, LegalDocContent> = {
  terms: {
    title: { en: "Terms of Service", es: "Términos del Servicio" },
    intro: {
      en: "These terms govern access to and use of the NEXUS platform (SaaS) and NEXUS professional security services. By using them you agree to these terms.",
      es: "Estos términos rigen el acceso y uso de la plataforma NEXUS (SaaS) y de los servicios profesionales de seguridad de NEXUS. Al usarlos, aceptas estos términos.",
    },
    sections: {
      en: [
        {
          h: "1. Provider",
          p: ["The services are provided by NEXUS Cybersecurity Global (legal entity details pending confirmation)."],
        },
        {
          h: "2. Services",
          p: [
            "A security platform offered as SaaS, and expert-led offensive/defensive security engagements, each described in the applicable order or statement of work.",
          ],
        },
        {
          h: "3. Eligibility & accounts",
          p: [
            "Services are offered to organizations. You are responsible for the accuracy of registration data and for safeguarding account credentials.",
          ],
        },
        {
          h: "4. Authorized use",
          p: [
            "You may use the services only for your own lawful security activities and within the scope you are authorized for. Security testing requires a separate, documented authorization.",
          ],
        },
        {
          h: "5. Prohibited activities",
          p: [
            "No testing of systems you do not own or are not authorized to assess; no attempts to break the platform's authorization controls; no unlawful, infringing or abusive use.",
          ],
        },
        { h: "6. Fees", p: ["Fees are set in the applicable order or SOW. Taxes may apply."] },
        {
          h: "7. Intellectual property",
          p: ["NEXUS retains all rights to the platform and its software. You retain rights to your data and content."],
        },
        {
          h: "8. Confidentiality",
          p: [
            "Each party protects the other's confidential information and uses it only to perform under these terms.",
          ],
        },
        {
          h: "9. Data protection",
          p: [
            "Processing of personal data is governed by the Privacy Policy and, where applicable, a Data Processing Agreement.",
          ],
        },
        {
          h: "10. Warranty & liability",
          p: [
            "Services are provided on a commercially reasonable basis. Warranty disclaimers and liability limits are subject to legal review and the applicable order.",
          ],
        },
        {
          h: "11. Term & termination",
          p: [
            "Either party may terminate as set out in the order. Sections meant to survive (IP, confidentiality, liability) survive termination.",
          ],
        },
        {
          h: "12. Governing law",
          p: ["Governing law and venue are PENDING and will be set once the contracting entity is confirmed."],
        },
      ],
      es: [
        {
          h: "1. Proveedor",
          p: [
            "Los servicios los presta NEXUS Cybersecurity Global (datos de la entidad legal pendientes de confirmación).",
          ],
        },
        {
          h: "2. Servicios",
          p: [
            "Una plataforma de seguridad ofrecida como SaaS y evaluaciones de seguridad ofensivas/defensivas dirigidas por expertos, descritas en el pedido u orden de trabajo aplicable.",
          ],
        },
        {
          h: "3. Elegibilidad y cuentas",
          p: [
            "Los servicios se ofrecen a organizaciones. Eres responsable de la exactitud de los datos de registro y de proteger las credenciales.",
          ],
        },
        {
          h: "4. Uso autorizado",
          p: [
            "Solo puedes usar los servicios para tus actividades de seguridad lícitas y dentro del alcance para el que estás autorizado. Las pruebas de seguridad requieren una autorización documentada aparte.",
          ],
        },
        {
          h: "5. Actividades prohibidas",
          p: [
            "No probar sistemas que no posees o para los que no tienes autorización; no intentar vulnerar los controles de autorización de la plataforma; ningún uso ilícito, infractor o abusivo.",
          ],
        },
        {
          h: "6. Tarifas",
          p: ["Las tarifas se fijan en el pedido u orden de trabajo aplicable. Pueden aplicarse impuestos."],
        },
        {
          h: "7. Propiedad intelectual",
          p: [
            "NEXUS conserva todos los derechos sobre la plataforma y su software. Tú conservas los derechos sobre tus datos y contenidos.",
          ],
        },
        {
          h: "8. Confidencialidad",
          p: ["Cada parte protege la información confidencial de la otra y la usa solo para cumplir estos términos."],
        },
        {
          h: "9. Protección de datos",
          p: [
            "El tratamiento de datos personales se rige por la Política de Privacidad y, cuando aplique, por un Acuerdo de Tratamiento de Datos.",
          ],
        },
        {
          h: "10. Garantía y responsabilidad",
          p: [
            "Los servicios se prestan con una diligencia comercialmente razonable. Las exclusiones de garantía y los límites de responsabilidad están sujetos a revisión legal y al pedido aplicable.",
          ],
        },
        {
          h: "11. Vigencia y terminación",
          p: [
            "Cualquiera de las partes puede terminar según el pedido. Las cláusulas destinadas a sobrevivir (PI, confidencialidad, responsabilidad) sobreviven a la terminación.",
          ],
        },
        {
          h: "12. Ley aplicable",
          p: [
            "La ley aplicable y la jurisdicción están PENDIENTES y se fijarán cuando se confirme la entidad contratante.",
          ],
        },
      ],
    },
  },
  privacy: {
    title: { en: "Privacy Policy", es: "Política de Privacidad" },
    intro: {
      en: "How NEXUS collects, uses and protects personal data, and the rights you can exercise.",
      es: "Cómo NEXUS recopila, usa y protege los datos personales, y los derechos que puedes ejercer.",
    },
    sections: {
      en: [
        {
          h: "Controller",
          p: [
            "NEXUS Cybersecurity Global (legal entity and contact details pending). Privacy contact: see the Contact page.",
          ],
        },
        {
          h: "Data we collect",
          p: [
            "Contact and demo-request data you submit (name, work email, organization, country, message), and basic technical data needed to run the site securely.",
          ],
        },
        {
          h: "Why we use it",
          p: [
            "To respond to your request, provide and secure the services, and comply with legal obligations. We do not sell personal data.",
          ],
        },
        {
          h: "Legal bases",
          p: [
            "Consent (e.g. marketing), performance of a contract, legitimate interests and legal obligations, as applicable to your jurisdiction.",
          ],
        },
        {
          h: "Your rights",
          p: [
            "Access, rectification, erasure, restriction, portability and objection, and withdrawal of consent, subject to applicable law (e.g. GDPR, Perú Ley 29733).",
          ],
        },
        {
          h: "Retention",
          p: [
            "We keep personal data only as long as necessary for the purposes above or as required by law, then delete it.",
          ],
        },
        {
          h: "Processors & transfers",
          p: [
            "Any subprocessors and international transfers will be listed and safeguarded; the subprocessor list is pending.",
          ],
        },
        {
          h: "Contact & complaints",
          p: [
            "Exercise your rights via the privacy contact. You may also contact your local data protection authority.",
          ],
        },
      ],
      es: [
        {
          h: "Responsable",
          p: [
            "NEXUS Cybersecurity Global (entidad legal y datos de contacto pendientes). Contacto de privacidad: ver la página de Contacto.",
          ],
        },
        {
          h: "Datos que recopilamos",
          p: [
            "Datos de contacto y de solicitud de demo que envías (nombre, email corporativo, organización, país, mensaje) y datos técnicos básicos para operar el sitio de forma segura.",
          ],
        },
        {
          h: "Para qué los usamos",
          p: [
            "Para responder a tu solicitud, prestar y proteger los servicios y cumplir obligaciones legales. No vendemos datos personales.",
          ],
        },
        {
          h: "Bases jurídicas",
          p: [
            "Consentimiento (p. ej. marketing), ejecución de un contrato, interés legítimo y obligaciones legales, según tu jurisdicción.",
          ],
        },
        {
          h: "Tus derechos",
          p: [
            "Acceso, rectificación, supresión, limitación, portabilidad y oposición, y revocación del consentimiento, conforme a la ley aplicable (p. ej. RGPD, Ley 29733 de Perú).",
          ],
        },
        {
          h: "Conservación",
          p: [
            "Conservamos los datos personales solo el tiempo necesario para los fines anteriores o lo exigido por ley, y luego los eliminamos.",
          ],
        },
        {
          h: "Encargados y transferencias",
          p: [
            "Cualquier subencargado y transferencia internacional se listará y protegerá; la lista de subencargados está pendiente.",
          ],
        },
        {
          h: "Contacto y reclamaciones",
          p: [
            "Ejerce tus derechos a través del contacto de privacidad. También puedes acudir a tu autoridad de protección de datos.",
          ],
        },
      ],
    },
  },
  cookies: {
    title: { en: "Cookie Policy", es: "Política de Cookies" },
    intro: {
      en: "How we use cookies and similar technologies, and how to control them.",
      es: "Cómo usamos cookies y tecnologías similares, y cómo controlarlas.",
    },
    sections: {
      en: [
        {
          h: "What we use",
          p: [
            "Necessary cookies required for the site to function, and — only with your consent — optional analytics and marketing cookies.",
          ],
        },
        {
          h: "Your control",
          p: [
            "Use the cookie banner or the 'Cookie preferences' link in the footer to accept, reject or configure optional cookies at any time. Necessary cookies cannot be disabled.",
          ],
        },
        {
          h: "No pre-consent tracking",
          p: ["Optional cookies default to OFF and no non-essential tracking runs before you consent."],
        },
        {
          h: "Current cookies",
          p: [
            "A detailed, verified list of specific cookies is pending and will be published here before launch — we won't claim cookies we don't actually set.",
          ],
        },
      ],
      es: [
        {
          h: "Qué usamos",
          p: [
            "Cookies necesarias para que el sitio funcione y —solo con tu consentimiento— cookies opcionales de analítica y marketing.",
          ],
        },
        {
          h: "Tu control",
          p: [
            "Usa el banner de cookies o el enlace 'Preferencias de cookies' del pie de página para aceptar, rechazar o configurar las opcionales cuando quieras. Las necesarias no se pueden desactivar.",
          ],
        },
        {
          h: "Sin rastreo previo al consentimiento",
          p: [
            "Las cookies opcionales están desactivadas por defecto y no se ejecuta rastreo no esencial antes de tu consentimiento.",
          ],
        },
        {
          h: "Cookies actuales",
          p: [
            "La lista detallada y verificada de cookies concretas está pendiente y se publicará aquí antes del lanzamiento — no declararemos cookies que no usemos realmente.",
          ],
        },
      ],
    },
  },
  aup: {
    title: { en: "Acceptable Use & Authorized Testing", es: "Uso Aceptable y Pruebas Autorizadas" },
    intro: {
      en: "Contracting NEXUS does not by itself authorize testing of any system. Offensive activities require verified authorization.",
      es: "Contratar NEXUS no autoriza por sí mismo probar ningún sistema. Las actividades ofensivas requieren autorización verificada.",
    },
    sections: {
      en: [
        {
          h: "Authorization is required",
          p: [
            "Before any assessment, you must register the target assets and provide verifiable authority over them. Testing outside a signed scope is prohibited and blocked.",
          ],
        },
        {
          h: "Scope & restrictions",
          p: [
            "Each engagement defines allowed targets, test types, time windows, restrictions, emergency contacts and stop conditions. These are enforced in the backend and the tool-execution layer, not only in the UI.",
          ],
        },
        {
          h: "Detect, don't exploit beyond scope",
          p: ["Our tooling detects and reports; it does not exploit beyond the agreed scope."],
        },
        { h: "Revocation & audit", p: ["Authorizations can be revoked, are versioned, and every action is audited."] },
        {
          h: "Prohibited use",
          p: [
            "No use of the platform to attack third parties, to break its own authorization controls, or for any unlawful purpose.",
          ],
        },
      ],
      es: [
        {
          h: "La autorización es obligatoria",
          p: [
            "Antes de cualquier evaluación debes registrar los activos objetivo y acreditar autoridad verificable sobre ellos. Probar fuera de un alcance firmado está prohibido y se bloquea.",
          ],
        },
        {
          h: "Alcance y restricciones",
          p: [
            "Cada evaluación define objetivos permitidos, tipos de prueba, ventanas horarias, restricciones, contactos de emergencia y condiciones de interrupción. Se aplican en el backend y en la capa de ejecución de herramientas, no solo en la interfaz.",
          ],
        },
        {
          h: "Detectar, no explotar fuera de alcance",
          p: ["Nuestras herramientas detectan y reportan; no explotan más allá del alcance acordado."],
        },
        {
          h: "Revocación y auditoría",
          p: ["Las autorizaciones pueden revocarse, están versionadas y cada acción queda auditada."],
        },
        {
          h: "Uso prohibido",
          p: [
            "No usar la plataforma para atacar a terceros, para vulnerar sus propios controles de autorización, ni para ningún fin ilícito.",
          ],
        },
      ],
    },
  },
  security: {
    title: { en: "Security Overview", es: "Resumen de Seguridad" },
    intro: {
      en: "A transparent, verifiable summary of our security approach. We state only what we actually implement.",
      es: "Un resumen transparente y verificable de nuestro enfoque de seguridad. Declaramos solo lo que realmente implementamos.",
    },
    sections: {
      en: [
        {
          h: "Authorization before capability",
          p: [
            "Assessment scope is enforced outside the model; a target or path outside the signed scope aborts execution. No agent widens its own permission.",
          ],
        },
        {
          h: "Authentication",
          p: [
            "Console access requires authentication; credentials are stored hashed. Public lead capture is rate-limited and bot-filtered.",
          ],
        },
        { h: "Auditing", p: ["Security-relevant actions are logged for audit."] },
        {
          h: "In progress",
          p: [
            "Additional hardening (security headers/CSP, MFA, external audit) is in progress and tracked in our launch checklist. We do not claim certifications (e.g. SOC 2, ISO 27001) we have not obtained.",
          ],
        },
      ],
      es: [
        {
          h: "Autorización antes que capacidad",
          p: [
            "El alcance de la evaluación se aplica fuera del modelo; un objetivo o ruta fuera del alcance firmado aborta la ejecución. Ningún agente amplía su propio permiso.",
          ],
        },
        {
          h: "Autenticación",
          p: [
            "El acceso a la consola requiere autenticación; las credenciales se almacenan con hash. La captación pública de leads tiene límite de tasa y filtrado de bots.",
          ],
        },
        { h: "Auditoría", p: ["Las acciones relevantes para la seguridad se registran para auditoría."] },
        {
          h: "En desarrollo",
          p: [
            "El endurecimiento adicional (cabeceras de seguridad/CSP, MFA, auditoría externa) está en curso y se rastrea en nuestro checklist de lanzamiento. No afirmamos certificaciones (p. ej. SOC 2, ISO 27001) que no hayamos obtenido.",
          ],
        },
      ],
    },
  },
  vdp: {
    title: { en: "Vulnerability Disclosure Policy", es: "Política de Divulgación de Vulnerabilidades" },
    intro: {
      en: "Found a security issue in NEXUS? We welcome responsible reports.",
      es: "¿Encontraste un problema de seguridad en NEXUS? Agradecemos los reportes responsables.",
    },
    sections: {
      en: [
        {
          h: "How to report",
          p: ["Email the security contact listed on the Contact page with steps to reproduce and impact."],
        },
        {
          h: "Our commitment",
          p: [
            "We will acknowledge your report, investigate, and keep you informed. Please give us reasonable time to remediate before public disclosure.",
          ],
        },
        {
          h: "Safe harbor",
          p: [
            "Good-faith research consistent with this policy will not be pursued. Do not access data beyond what's needed to demonstrate the issue, and do not disrupt services.",
          ],
        },
        { h: "Scope", p: ["The scope of in-scope assets is pending and will be published here."] },
      ],
      es: [
        {
          h: "Cómo reportar",
          p: [
            "Escribe al contacto de seguridad indicado en la página de Contacto con los pasos para reproducir y el impacto.",
          ],
        },
        {
          h: "Nuestro compromiso",
          p: [
            "Acusaremos recibo de tu reporte, investigaremos y te mantendremos informado. Danos un tiempo razonable para remediar antes de una divulgación pública.",
          ],
        },
        {
          h: "Puerto seguro",
          p: [
            "La investigación de buena fe conforme a esta política no será perseguida. No accedas a más datos de los necesarios para demostrar el problema ni interrumpas los servicios.",
          ],
        },
        { h: "Alcance", p: ["El alcance de los activos incluidos está pendiente y se publicará aquí."] },
      ],
    },
  },
};

export const LEGAL_ORDER = ["terms", "privacy", "cookies", "aup", "security", "vdp"] as const;
