package server

import "fmt"

// reportText holds every string printed in the PDF reports, per language.
type reportText struct {
	ExecTitle, TechTitle, Confidential, PageOf, Draft, DraftNote string
	DocControl, NoScope, Organization, Evaluation, Scope         string
	Requested, ReportDate, ReportID, Classification, PreparedBy  string
	Notice                                                       string

	ExecSummary, BySeverity, Priorities, NoPriorities, Recommendations string
	Unassigned, Overdue, Total, And                                    string
	ColSeverity, ColFinding, ColState, ColOwner, ColDue, ColClass      string
	BucketOpen, BucketInProgress, BucketFixed, BucketRisk              string

	ExposureTitle, ExposureNone, NoFindings, AllClosed string
	Exposure                                           map[string]string
	CountSev                                           func(n int, sev string) string
	ExposureLine                                       func(parts string) string

	KpiTotal, KpiTotalHint, KpiOpen, KpiFixed, KpiOverdue, KpiOverdueHint string
	InProgressN                                                           func(n int) string
	FixedOf                                                               func(done, total int) string

	RecSev                                        func(n int, sev string, days int) string
	RecOverdue, RecUnassigned, RecVerify, RecRisk func(n int) string
	RecPlan                                       func(n int) string
	RecRetest                                     string

	TechSummary, FindingIndex, FindingDetails, NoRetest string
	TechIntro                                           func(n int) string
	Detected, LastRetest, RiskAccepted                  string
	RiskAcceptedBy                                      func(who, date string) string
	Description, TechnicalDetail                        string
	Verdict, RetestStatus                               map[string]string

	MethodTitle, HowWeWork, MethodBody, SeverityScale, Meaning, SLA string
	StatesTitle, Limitations, LimitationsBody                       string
	Days                                                            func(n int) string

	Sev, State, SevDef, StateDef map[string]string
	DateFmt                      string
	Months                       [12]string
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

var reportTexts = map[string]*reportText{"es": reportES(), "en": reportEN()}

func reportES() *reportText {
	sevLower := map[string]string{"critical": "crítica", "high": "alta", "medium": "media", "low": "baja", "unrated": "sin clasificar"}
	return &reportText{
		ExecTitle:      "Informe ejecutivo de seguridad",
		TechTitle:      "Informe técnico de seguridad",
		Confidential:   "CONFIDENCIAL",
		PageOf:         "Página %d de %s",
		Draft:          "BORRADOR",
		DraftNote:      "Borrador interno: los resultados aún no se han publicado al cliente.",
		DocControl:     "Control del documento",
		NoScope:        "Alcance no detallado en la solicitud.",
		Organization:   "Organización",
		Evaluation:     "Evaluación",
		Scope:          "Alcance",
		Requested:      "Solicitada",
		ReportDate:     "Fecha del informe",
		ReportID:       "Identificador",
		Classification: "Clasificación",
		PreparedBy:     "Elaborado por",
		Notice: "Este documento contiene información confidencial sobre la seguridad de la organización indicada. " +
			"Su distribución debe limitarse al personal autorizado. Refleja el estado en la fecha de emisión; " +
			"la versión actualizada está siempre disponible en el portal de clientes de NEXUS.",

		ExecSummary:      "Resumen ejecutivo",
		BySeverity:       "Hallazgos por severidad y estado",
		Priorities:       "Prioridades de corrección",
		NoPriorities:     "No hay hallazgos abiertos en esta evaluación.",
		Recommendations:  "Recomendaciones",
		Unassigned:       "Sin asignar",
		Overdue:          "vencido",
		Total:            "Total",
		And:              "y",
		ColSeverity:      "Severidad",
		ColFinding:       "Hallazgo",
		ColState:         "Estado",
		ColOwner:         "Responsable",
		ColDue:           "Fecha objetivo",
		ColClass:         "Tipo",
		BucketOpen:       "Abiertos",
		BucketInProgress: "En curso",
		BucketFixed:      "Corregidos",
		BucketRisk:       "Riesgo aceptado",

		ExposureTitle: "Nivel de exposición",
		ExposureNone:  "Sin riesgos abiertos",
		NoFindings:    "No se identificaron hallazgos en el alcance evaluado.",
		AllClosed:     "Todos los hallazgos están corregidos y verificados o con el riesgo aceptado.",
		Exposure:      map[string]string{"critical": "Crítico", "high": "Alto", "medium": "Moderado", "low": "Bajo", "unrated": "Sin clasificar"},
		CountSev: func(n int, sev string) string {
			if sev == "unrated" {
				return fmt.Sprintf("%d %s", n, plural(n, "hallazgo sin clasificar", "hallazgos sin clasificar"))
			}
			return fmt.Sprintf("%d %s de severidad %s", n, plural(n, "hallazgo", "hallazgos"), sevLower[sev])
		},
		ExposureLine: func(parts string) string {
			return "Pendiente de corregir: " + parts + ". El nivel corresponde a la severidad más alta que sigue abierta."
		},

		KpiTotal:       "Hallazgos",
		KpiTotalHint:   "verificados y publicados",
		KpiOpen:        "Abiertos",
		KpiFixed:       "Remediación",
		KpiOverdue:     "Vencidos",
		KpiOverdueHint: "fuera de la fecha objetivo",
		InProgressN:    func(n int) string { return fmt.Sprintf("%d en corrección", n) },
		FixedOf:        func(d, t int) string { return fmt.Sprintf("%d de %d corregidos", d, t) },

		RecSev: func(n int, sev string, days int) string {
			return fmt.Sprintf("Corregir %d %s de severidad %s en un máximo de %d días desde su detección.",
				n, plural(n, "hallazgo abierto", "hallazgos abiertos"), sevLower[sev], days)
		},
		RecOverdue: func(n int) string {
			return fmt.Sprintf("%d %s superado la fecha objetivo: revisar su responsable y acordar una nueva fecha.",
				n, plural(n, "hallazgo ha", "hallazgos han"))
		},
		RecUnassigned: func(n int) string {
			return fmt.Sprintf("Asignar un responsable a %d %s que aún no lo %s.",
				n, plural(n, "hallazgo abierto", "hallazgos abiertos"), plural(n, "tiene", "tienen"))
		},
		RecVerify: func(n int) string {
			return fmt.Sprintf("Cuando terminen las correcciones en curso (%d), solicitar la verificación de NEXUS desde el portal.", n)
		},
		RecRisk: func(n int) string {
			return fmt.Sprintf("Revisar antes de su caducidad %d %s.", n, plural(n, "riesgo aceptado", "riesgos aceptados"))
		},
		RecPlan: func(n int) string {
			return fmt.Sprintf("Planificar la corrección de %d %s de severidad media o baja dentro de los plazos de referencia.",
				n, plural(n, "hallazgo", "hallazgos"))
		},
		RecRetest: "Programar una nueva evaluación tras cambios relevantes en los sistemas o, como mínimo, una vez al año.",

		TechSummary:    "Resumen técnico",
		FindingIndex:   "Índice de hallazgos",
		FindingDetails: "Detalle de los hallazgos",
		NoRetest:       "Sin verificación todavía",
		TechIntro: func(n int) string {
			return fmt.Sprintf("Esta evaluación reúne %d %s verificados por NEXUS. El estado de cada uno refleja el seguimiento de la remediación en la fecha del informe.",
				n, plural(n, "hallazgo", "hallazgos"))
		},
		Detected:     "Detectado",
		LastRetest:   "Última verificación",
		RiskAccepted: "Aceptación de riesgo",
		RiskAcceptedBy: func(who, date string) string {
			return fmt.Sprintf("Aceptado por %s, válido hasta el %s. Justificación:", who, date)
		},
		Description:     "Descripción",
		TechnicalDetail: "Detalle técnico",
		Verdict:         map[string]string{"reproduced": "El problema se reprodujo", "fixed": "Corregido", "inconclusive": "No concluyente", "": "Sin veredicto"},
		RetestStatus:    map[string]string{"pending": "Pendiente", "running": "En curso", "completed": "Completada", "failed": "No se pudo completar", "stopped": "Detenida"},

		MethodTitle: "Alcance y metodología",
		HowWeWork:   "Cómo se realizó la evaluación",
		MethodBody: "Las pruebas se realizaron exclusivamente sobre el alcance autorizado por su organización, combinando el análisis " +
			"automatizado de la plataforma NEXUS con la revisión de nuestro equipo. Este informe incluye solo hallazgos revisados " +
			"y publicados por NEXUS; los resultados descartados en el triaje y los falsos positivos no aparecen. Un hallazgo solo " +
			"se considera corregido cuando NEXUS vuelve a probarlo y confirma que el problema ya no se reproduce.",
		SeverityScale: "Escala de severidad y plazos de referencia",
		Meaning:       "Significado",
		SLA:           "Plazo de corrección",
		StatesTitle:   "Estados de remediación",
		Limitations:   "Limitaciones",
		LimitationsBody: "La evaluación refleja el estado de los sistemas durante el periodo de pruebas y dentro del alcance acordado. " +
			"La ausencia de hallazgos en un área no garantiza la ausencia de vulnerabilidades. Los estados y fechas de este documento " +
			"corresponden al momento de su emisión.",
		Days: func(n int) string { return fmt.Sprintf("%d días", n) },

		Sev: map[string]string{"critical": "Crítica", "high": "Alta", "medium": "Media", "low": "Baja", "unrated": "Sin clasificar"},
		State: map[string]string{
			"open": "Abierta", "in_progress": "En corrección", "retest_requested": "Verificación solicitada",
			"retest_running": "Verificando", "still_vulnerable": "Sigue vulnerable",
			"verified_fixed": "Corregida y verificada", "risk_accepted": "Riesgo aceptado",
		},
		SevDef: map[string]string{
			"critical": "Compromiso directo de sistemas o datos sensibles con poca o ninguna dificultad.",
			"high":     "Impacto significativo en confidencialidad, integridad o disponibilidad; explotación factible.",
			"medium":   "Impacto limitado o que requiere condiciones previas para explotarse.",
			"low":      "Impacto menor; mejora de la postura de seguridad recomendada.",
		},
		StateDef: map[string]string{
			"open":             "pendiente de que la organización inicie la corrección.",
			"in_progress":      "la organización está aplicando la corrección.",
			"retest_requested": "la organización pidió a NEXUS verificar la corrección.",
			"still_vulnerable": "la verificación de NEXUS reprodujo el problema.",
			"verified_fixed":   "NEXUS confirmó que el problema ya no se reproduce.",
			"risk_accepted":    "la organización asumió el riesgo con justificación y fecha de caducidad.",
		},
		DateFmt: "%d de %s de %d",
		Months:  [12]string{"enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre"},
	}
}

func reportEN() *reportText {
	sevLower := map[string]string{"critical": "critical", "high": "high", "medium": "medium", "low": "low", "unrated": "unrated"}
	return &reportText{
		ExecTitle:      "Executive security report",
		TechTitle:      "Technical security report",
		Confidential:   "CONFIDENTIAL",
		PageOf:         "Page %d of %s",
		Draft:          "DRAFT",
		DraftNote:      "Internal draft: these results have not been published to the client yet.",
		DocControl:     "Document control",
		NoScope:        "No scope details were given in the request.",
		Organization:   "Organization",
		Evaluation:     "Assessment",
		Scope:          "Scope",
		Requested:      "Requested",
		ReportDate:     "Report date",
		ReportID:       "Identifier",
		Classification: "Classification",
		PreparedBy:     "Prepared by",
		Notice: "This document contains confidential information about the security of the organization named above. " +
			"Distribution must be limited to authorized personnel. It reflects the situation on its issue date; " +
			"the up-to-date version is always available in the NEXUS client portal.",

		ExecSummary:      "Executive summary",
		BySeverity:       "Findings by severity and status",
		Priorities:       "Remediation priorities",
		NoPriorities:     "There are no open findings in this assessment.",
		Recommendations:  "Recommendations",
		Unassigned:       "Unassigned",
		Overdue:          "overdue",
		Total:            "Total",
		And:              "and",
		ColSeverity:      "Severity",
		ColFinding:       "Finding",
		ColState:         "Status",
		ColOwner:         "Owner",
		ColDue:           "Target date",
		ColClass:         "Type",
		BucketOpen:       "Open",
		BucketInProgress: "In progress",
		BucketFixed:      "Fixed",
		BucketRisk:       "Risk accepted",

		ExposureTitle: "Exposure level",
		ExposureNone:  "No open risks",
		NoFindings:    "No findings were identified within the assessed scope.",
		AllClosed:     "Every finding is fixed and verified, or its risk has been accepted.",
		Exposure:      map[string]string{"critical": "Critical", "high": "High", "medium": "Moderate", "low": "Low", "unrated": "Unrated"},
		CountSev: func(n int, sev string) string {
			if sev == "unrated" {
				return fmt.Sprintf("%d unrated %s", n, plural(n, "finding", "findings"))
			}
			return fmt.Sprintf("%d %s-severity %s", n, sevLower[sev], plural(n, "finding", "findings"))
		},
		ExposureLine: func(parts string) string {
			return "Still to fix: " + parts + ". The level is the highest severity that remains open."
		},

		KpiTotal:       "Findings",
		KpiTotalHint:   "verified and published",
		KpiOpen:        "Open",
		KpiFixed:       "Remediation",
		KpiOverdue:     "Overdue",
		KpiOverdueHint: "past their target date",
		InProgressN:    func(n int) string { return fmt.Sprintf("%d being fixed", n) },
		FixedOf:        func(d, t int) string { return fmt.Sprintf("%d of %d fixed", d, t) },

		RecSev: func(n int, sev string, days int) string {
			return fmt.Sprintf("Fix the %d open %s-severity %s within %d days of detection.",
				n, sevLower[sev], plural(n, "finding", "findings"), days)
		},
		RecOverdue: func(n int) string {
			return fmt.Sprintf("%d %s past the target date: review the owner and agree on a new date.",
				n, plural(n, "finding is", "findings are"))
		},
		RecUnassigned: func(n int) string {
			return fmt.Sprintf("Assign an owner to the %d open %s without one.", n, plural(n, "finding", "findings"))
		},
		RecVerify: func(n int) string {
			return fmt.Sprintf("Once the fixes in progress (%d) are done, request a NEXUS verification from the portal.", n)
		},
		RecRisk: func(n int) string {
			return fmt.Sprintf("Review the %d accepted %s before %s.", n, plural(n, "risk", "risks"), plural(n, "it expires", "they expire"))
		},
		RecPlan: func(n int) string {
			return fmt.Sprintf("Plan the remediation of %d medium or low-severity %s within the reference deadlines.",
				n, plural(n, "finding", "findings"))
		},
		RecRetest: "Schedule a new assessment after significant changes to your systems, or at least once a year.",

		TechSummary:    "Technical summary",
		FindingIndex:   "Finding index",
		FindingDetails: "Finding details",
		NoRetest:       "Not verified yet",
		TechIntro: func(n int) string {
			return fmt.Sprintf("This assessment contains %d %s verified by NEXUS. Each status reflects remediation tracking on the report date.",
				n, plural(n, "finding", "findings"))
		},
		Detected:     "Detected",
		LastRetest:   "Latest verification",
		RiskAccepted: "Risk acceptance",
		RiskAcceptedBy: func(who, date string) string {
			return fmt.Sprintf("Accepted by %s, valid until %s. Justification:", who, date)
		},
		Description:     "Description",
		TechnicalDetail: "Technical detail",
		Verdict:         map[string]string{"reproduced": "The issue reproduced", "fixed": "Fixed", "inconclusive": "Inconclusive", "": "No verdict"},
		RetestStatus:    map[string]string{"pending": "Pending", "running": "Running", "completed": "Completed", "failed": "Could not complete", "stopped": "Stopped"},

		MethodTitle: "Scope and methodology",
		HowWeWork:   "How the assessment was performed",
		MethodBody: "Testing was performed only within the scope authorized by your organization, combining automated analysis " +
			"by the NEXUS platform with review by our team. This report only includes findings reviewed and published by NEXUS; " +
			"results discarded during triage and false positives are excluded. A finding is only considered fixed once NEXUS " +
			"tests it again and confirms the issue no longer reproduces.",
		SeverityScale: "Severity scale and reference deadlines",
		Meaning:       "Meaning",
		SLA:           "Remediation deadline",
		StatesTitle:   "Remediation statuses",
		Limitations:   "Limitations",
		LimitationsBody: "The assessment reflects the state of the systems during the testing period and within the agreed scope. " +
			"The absence of findings in an area does not guarantee the absence of vulnerabilities. Statuses and dates in this " +
			"document reflect the time it was issued.",
		Days: func(n int) string { return fmt.Sprintf("%d days", n) },

		Sev: map[string]string{"critical": "Critical", "high": "High", "medium": "Medium", "low": "Low", "unrated": "Unrated"},
		State: map[string]string{
			"open": "Open", "in_progress": "Being fixed", "retest_requested": "Verification requested",
			"retest_running": "Verifying", "still_vulnerable": "Still vulnerable",
			"verified_fixed": "Fixed and verified", "risk_accepted": "Risk accepted",
		},
		SevDef: map[string]string{
			"critical": "Direct compromise of systems or sensitive data with little or no difficulty.",
			"high":     "Significant impact on confidentiality, integrity or availability; exploitation is feasible.",
			"medium":   "Limited impact, or exploitation requires preconditions.",
			"low":      "Minor impact; recommended improvement to the security posture.",
		},
		StateDef: map[string]string{
			"open":             "waiting for the organization to start the fix.",
			"in_progress":      "the organization is applying the fix.",
			"retest_requested": "the organization asked NEXUS to verify the fix.",
			"still_vulnerable": "the NEXUS verification reproduced the issue.",
			"verified_fixed":   "NEXUS confirmed the issue no longer reproduces.",
			"risk_accepted":    "the organization accepted the risk with a justification and an expiry date.",
		},
		DateFmt: "%[2]s %[1]d, %[3]d",
		Months:  [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
	}
}
