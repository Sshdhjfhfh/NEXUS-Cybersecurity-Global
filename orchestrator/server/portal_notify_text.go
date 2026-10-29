package server

import (
	"fmt"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/nexus-ia/nexus/db"
)

// notifyText holds every string of the portal emails in one language. Emails
// never name a finding or include technical detail: a reference number, the
// severity and the evaluation title, with a link to the portal.
type notifyText struct {
	Lang string

	FooterAuto, FooterSecurity, FooterStaff, FooterMinimal, FooterOptOut string

	VerifySubject, VerifyTitle, VerifyIntro, VerifyCTA, VerifyNote string
	ResetSubject, ResetTitle, ResetIntro, ResetCTA, ResetNote      string

	FindingRef func(fid int64) string

	ReleasedSubject func(title string) string
	ReleasedTitle   string
	ReleasedIntro   func(title string) string
	ReleasedCTA     string
	ReleasedNote    string

	AssignedSubject func(fid int64) string
	AssignedTitle   string
	AssignedIntro   func(actor, ref, sev, title string) string
	DueLine         func(date string) string
	OpenCTA         string

	FixedSubject func(fid int64) string
	FixedTitle   string
	FixedIntro   func(ref, sev, title string) string
	ReproSubject func(fid int64) string
	ReproTitle   string
	ReproIntro   func(ref, sev, title string) string

	DigestSubject   func(overdue, expiring int) string
	DigestTitle     string
	DigestIntro     string
	OverdueSection  string
	ExpiringSection string
	ItemText        func(ref, title string) string
	OverdueMeta     func(date string, days int) string
	ExpiresMeta     func(date string) string
	DigestCTA       string

	OpsSubject func(org string, fid int64) string
	OpsTitle   string
	OpsIntro   func(org, ref, sev, title string) string
	OpsCTA     string
}

var notifyTexts = map[string]*notifyText{
	"es": {
		Lang:           "es",
		FooterAuto:     "Este es un mensaje automático de NEXUS Cybersecurity Global. No respondas a este correo.",
		FooterSecurity: "Te lo enviamos por la seguridad de tu cuenta; este tipo de aviso no se puede desactivar.",
		FooterStaff:    "Aviso interno para el equipo de NEXUS.",
		FooterMinimal:  "Por seguridad, nuestros avisos no incluyen detalles técnicos de los hallazgos: consúltalos en el portal.",
		FooterOptOut:   "Puedes desactivar estos avisos en el portal, en Seguridad › Avisos por email.",

		VerifySubject: "Verifica tu email · NEXUS",
		VerifyTitle:   "Confirma tu dirección de email",
		VerifyIntro:   "Confirma esta dirección para tu cuenta del portal de clientes de NEXUS.",
		VerifyCTA:     "Verificar email",
		VerifyNote:    "El enlace caduca en 48 horas. Si no creaste esta cuenta, ignora este mensaje.",
		ResetSubject:  "Restablece tu contraseña · NEXUS",
		ResetTitle:    "Restablece tu contraseña",
		ResetIntro:    "Hemos recibido una solicitud para restablecer la contraseña de tu cuenta de NEXUS.",
		ResetCTA:      "Elegir una contraseña nueva",
		ResetNote:     "El enlace caduca en 1 hora. Si no fuiste tú, ignora este mensaje: tu contraseña no cambiará.",

		FindingRef: func(fid int64) string { return fmt.Sprintf("Hallazgo #%d", fid) },

		ReleasedSubject: func(title string) string { return "Resultados disponibles: " + title + " · NEXUS" },
		ReleasedTitle:   "Los resultados de tu evaluación están disponibles",
		ReleasedIntro: func(title string) string {
			return "NEXUS ha publicado los resultados de «" + title + "». Ya puedes revisar los hallazgos, " +
				"asignar responsables y seguir su corrección en el portal."
		},
		ReleasedCTA:  "Ver resultados",
		ReleasedNote: "Desde la evaluación puedes descargar el informe ejecutivo y el informe técnico en PDF.",

		AssignedSubject: func(fid int64) string { return fmt.Sprintf("Se te ha asignado el hallazgo #%d · NEXUS", fid) },
		AssignedTitle:   "Tienes un hallazgo asignado",
		AssignedIntro: func(actor, ref, sev, title string) string {
			return actor + " te ha asignado el " + lowerFirst(ref) + " (severidad " + sev + ") de la evaluación «" + title + "»."
		},
		DueLine: func(date string) string { return "Fecha objetivo de corrección: " + date + "." },
		OpenCTA: "Abrir en el portal",

		FixedSubject: func(fid int64) string { return fmt.Sprintf("Corrección verificada: hallazgo #%d · NEXUS", fid) },
		FixedTitle:   "NEXUS ha verificado la corrección",
		FixedIntro: func(ref, sev, title string) string {
			return "La verificación de NEXUS confirma que el " + lowerFirst(ref) + " (severidad " + sev +
				") de «" + title + "» ya no es explotable. Queda marcado como corregido."
		},
		ReproSubject: func(fid int64) string { return fmt.Sprintf("El hallazgo #%d sigue presente · NEXUS", fid) },
		ReproTitle:   "La verificación sigue detectando el problema",
		ReproIntro: func(ref, sev, title string) string {
			return "NEXUS ha vuelto a probar el " + lowerFirst(ref) + " (severidad " + sev + ") de «" + title +
				"» y sigue siendo explotable. Revisa la corrección y solicita una nueva verificación cuando esté lista."
		},

		DigestSubject: func(overdue, expiring int) string {
			switch {
			case overdue > 0 && expiring > 0:
				return fmt.Sprintf("Remediación: %d %s y %d %s · NEXUS", overdue, plural(overdue, "vencido", "vencidos"),
					expiring, plural(expiring, "riesgo por caducar", "riesgos por caducar"))
			case overdue > 0:
				return fmt.Sprintf("Remediación: %d %s fuera de plazo · NEXUS", overdue, plural(overdue, "hallazgo", "hallazgos"))
			default:
				return fmt.Sprintf("Remediación: %d %s · NEXUS", expiring,
					plural(expiring, "aceptación de riesgo por caducar", "aceptaciones de riesgo por caducar"))
			}
		},
		DigestTitle:     "Resumen de remediación",
		DigestIntro:     "Estos elementos necesitan atención. Te avisamos una sola vez por cada uno.",
		OverdueSection:  "Hallazgos fuera de la fecha objetivo",
		ExpiringSection: "Aceptaciones de riesgo que caducan en los próximos 7 días",
		ItemText:        func(ref, title string) string { return ref + " · " + title },
		OverdueMeta: func(date string, days int) string {
			return fmt.Sprintf("Fecha objetivo: %s (hace %d %s)", date, days, plural(days, "día", "días"))
		},
		ExpiresMeta: func(date string) string { return "Caduca el " + date + "; después volverá a contar como abierto." },
		DigestCTA:   "Ir a remediación",

		OpsSubject: func(org string, fid int64) string {
			return fmt.Sprintf("Verificación solicitada: %s · hallazgo #%d", org, fid)
		},
		OpsTitle: "Un cliente solicita una verificación",
		OpsIntro: func(org, ref, sev, title string) string {
			return org + " indica que ha corregido el " + lowerFirst(ref) + " (severidad " + sev + ") de «" + title +
				"» y solicita que NEXUS lo verifique."
		},
		OpsCTA: "Abrir verificaciones pendientes",
	},
	"en": {
		Lang:           "en",
		FooterAuto:     "This is an automated message from NEXUS Cybersecurity Global. Please do not reply.",
		FooterSecurity: "We sent this for the security of your account; this kind of notice cannot be turned off.",
		FooterStaff:    "Internal notice for the NEXUS team.",
		FooterMinimal:  "For security, our notices never include technical details of findings: see them in the portal.",
		FooterOptOut:   "You can turn these notices off in the portal under Security › Email notifications.",

		VerifySubject: "Verify your email · NEXUS",
		VerifyTitle:   "Confirm your email address",
		VerifyIntro:   "Confirm this address for your NEXUS client portal account.",
		VerifyCTA:     "Verify email",
		VerifyNote:    "The link expires in 48 hours. If you did not create this account, ignore this message.",
		ResetSubject:  "Reset your password · NEXUS",
		ResetTitle:    "Reset your password",
		ResetIntro:    "We received a request to reset the password of your NEXUS account.",
		ResetCTA:      "Choose a new password",
		ResetNote:     "The link expires in 1 hour. If this was not you, ignore this message: your password will not change.",

		FindingRef: func(fid int64) string { return fmt.Sprintf("Finding #%d", fid) },

		ReleasedSubject: func(title string) string { return "Results available: " + title + " · NEXUS" },
		ReleasedTitle:   "Your assessment results are available",
		ReleasedIntro: func(title string) string {
			return "NEXUS has published the results of “" + title + "”. You can now review the findings, " +
				"assign owners and track their remediation in the portal."
		},
		ReleasedCTA:  "View results",
		ReleasedNote: "From the assessment you can download the executive and technical PDF reports.",

		AssignedSubject: func(fid int64) string { return fmt.Sprintf("Finding #%d was assigned to you · NEXUS", fid) },
		AssignedTitle:   "A finding was assigned to you",
		AssignedIntro: func(actor, ref, sev, title string) string {
			return actor + " assigned you " + lowerFirst(ref) + " (" + sev + " severity) of the assessment “" + title + "”."
		},
		DueLine: func(date string) string { return "Remediation target date: " + date + "." },
		OpenCTA: "Open in the portal",

		FixedSubject: func(fid int64) string { return fmt.Sprintf("Fix verified: finding #%d · NEXUS", fid) },
		FixedTitle:   "NEXUS verified the fix",
		FixedIntro: func(ref, sev, title string) string {
			return "The NEXUS verification confirms that " + lowerFirst(ref) + " (" + sev + " severity) of “" + title +
				"” is no longer exploitable. It is now marked as fixed."
		},
		ReproSubject: func(fid int64) string { return fmt.Sprintf("Finding #%d is still present · NEXUS", fid) },
		ReproTitle:   "The verification still detects the issue",
		ReproIntro: func(ref, sev, title string) string {
			return "NEXUS tested " + lowerFirst(ref) + " (" + sev + " severity) of “" + title + "” again and it is still exploitable. " +
				"Review the fix and request a new verification when it is ready."
		},

		DigestSubject: func(overdue, expiring int) string {
			switch {
			case overdue > 0 && expiring > 0:
				return fmt.Sprintf("Remediation: %d overdue and %d expiring %s · NEXUS", overdue, expiring,
					plural(expiring, "risk", "risks"))
			case overdue > 0:
				return fmt.Sprintf("Remediation: %d overdue %s · NEXUS", overdue, plural(overdue, "finding", "findings"))
			default:
				return fmt.Sprintf("Remediation: %d risk %s expiring · NEXUS", expiring,
					plural(expiring, "acceptance", "acceptances"))
			}
		},
		DigestTitle:     "Remediation summary",
		DigestIntro:     "These items need attention. We notify you once about each of them.",
		OverdueSection:  "Findings past their target date",
		ExpiringSection: "Risk acceptances expiring in the next 7 days",
		ItemText:        func(ref, title string) string { return ref + " · " + title },
		OverdueMeta: func(date string, days int) string {
			return fmt.Sprintf("Target date: %s (%d %s ago)", date, days, plural(days, "day", "days"))
		},
		ExpiresMeta: func(date string) string { return "Expires on " + date + "; after that it counts as open again." },
		DigestCTA:   "Go to remediation",

		OpsSubject: func(org string, fid int64) string {
			return fmt.Sprintf("Verification requested: %s · finding #%d", org, fid)
		},
		OpsTitle: "A client requests a verification",
		OpsIntro: func(org, ref, sev, title string) string {
			return org + " reports that " + lowerFirst(ref) + " (" + sev + " severity) of “" + title +
				"” is fixed and asks NEXUS to verify it."
		},
		OpsCTA: "Open pending verifications",
	},
}

func notifyTextFor(lang string) *notifyText {
	if t := notifyTexts[lang]; t != nil {
		return t
	}
	return notifyTexts["es"]
}

// userLang is the user's saved portal language (” -> default).
func userLang(pg *db.DB, userID int64) string {
	_, lang, err := pg.GetPortalUserPrefs(userID)
	if err != nil {
		return ""
	}
	return lang
}

// sevName is the lower-case severity name for running text.
func (t *notifyText) sevName(sev string) string {
	return lowerFirst(reportTextsFor(t.Lang).Sev[sevKey(sev)])
}

// sevLabel is the chip label of a severity.
func (t *notifyText) sevLabel(sev string) string {
	return reportTextsFor(t.Lang).Sev[sevKey(sev)]
}

// date formats a calendar day (YYYY-MM-DD or a time) like the reports do.
func (t *notifyText) date(day time.Time) string {
	rt := reportTextsFor(t.Lang)
	return fmt.Sprintf(rt.DateFmt, day.Day(), rt.Months[day.Month()-1], day.Year())
}

func (t *notifyText) dateStr(ymd string) string {
	d, err := time.Parse("2006-01-02", ymd)
	if err != nil {
		return ymd
	}
	return t.date(d)
}

func reportTextsFor(lang string) *reportText {
	if rt := reportTexts[lang]; rt != nil {
		return rt
	}
	return reportTexts["es"]
}

func lowerFirst(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError {
		return s
	}
	return string(unicode.ToLower(r)) + s[n:]
}
