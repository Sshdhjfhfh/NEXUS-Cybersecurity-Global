package report

import "strings"

// Lang selects the language of a rendered export.
type Lang string

const (
	ES Lang = "es" // product default
	EN Lang = "en"
	ZH Lang = "zh" // the upstream (ARTEX) wording
)

// ParseLang picks the first supported language from a lang query value or an
// Accept-Language header ("en-US,en;q=0.9"); anything else is Spanish.
func ParseLang(v string) Lang {
	for _, part := range strings.Split(v, ",") {
		tag, _, _ := strings.Cut(part, ";")
		base, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
		switch l := Lang(base); l {
		case ES, EN, ZH:
			return l
		}
	}
	return ES
}

type texts struct {
	colon, listSep string

	reportTitle, untitledTask, goal, unspecified, generated string
	summary, confirmedFmt, assets, findings, noConfirmed    string
	uncategorized, pocEvidence                              string

	exportTitle, total, countFmt, severity, count, details, noMatches string
	class, status, task, foundAt, evidence, detailReport, overview    string

	traffic, trafficMetaFmt, trafficStale, trafficItemFmt, request, response string

	csvHeader  []string
	severities map[string]string
	statuses   map[string]string
}

var catalog = map[Lang]*texts{
	ES: {
		colon: ": ", listSep: ", ",
		reportTitle: "Informe de pruebas de penetración", untitledTask: "Tarea sin nombre",
		goal: "Objetivo", unspecified: "(sin especificar)", generated: "Generado",
		summary: "Resumen", confirmedFmt: "Hallazgos confirmados: **%d**", assets: "Activos",
		findings: "Hallazgos", noConfirmed: "_No se confirmaron vulnerabilidades._",
		uncategorized: "Sin clasificar", pocEvidence: "PoC / evidencia",
		exportTitle: "Informe consolidado de hallazgos", total: "Total de hallazgos", countFmt: "%d",
		severity: "Severidad", count: "Cantidad", details: "Detalle de hallazgos",
		noMatches: "_No hay hallazgos que coincidan._",
		class:     "Categoría", status: "Estado", task: "Tarea", foundAt: "Fecha de detección",
		evidence: "Evidencia", detailReport: "Informe detallado", overview: "Descripción",
		traffic:        "Evidencia de tráfico asociada",
		trafficMetaFmt: "Versión de la evidencia: %d; vínculos: %d.",
		trafficStale:   "La evidencia cambió; el informe detallado está pendiente de actualizar.",
		trafficItemFmt: "%d. **Evidencia #%d · %s** — `%s %s`, código %d\n",
		request:        "Solicitud", response: "Respuesta",
		csvHeader:  []string{"ID", "Nombre", "Categoría", "Severidad", "Estado", "Tarea", "Fecha de detección", "Resumen", "Evidencias de tráfico", "IDs de evidencia"},
		severities: map[string]string{"critical": "Crítica", "high": "Alta", "medium": "Media", "low": "Baja", "info": "Informativa"},
		statuses: map[string]string{
			"pending": "Pendiente", "in_progress": "En curso", "confirmed": "Confirmado", "resolved": "Resuelto",
			"fixed": "Corregido", "false_positive": "Falso positivo", "ignored": "Ignorado",
			"duplicate": "Duplicado", "risk_accepted": "Riesgo aceptado",
		},
	},
	EN: {
		colon: ": ", listSep: ", ",
		reportTitle: "Penetration test report", untitledTask: "Untitled task",
		goal: "Goal", unspecified: "(not specified)", generated: "Generated",
		summary: "Summary", confirmedFmt: "Confirmed findings: **%d**", assets: "Assets",
		findings: "Findings", noConfirmed: "_No vulnerabilities were confirmed._",
		uncategorized: "Uncategorized", pocEvidence: "PoC / evidence",
		exportTitle: "Findings summary report", total: "Total findings", countFmt: "%d",
		severity: "Severity", count: "Count", details: "Finding details",
		noMatches: "_No matching findings._",
		class:     "Category", status: "Status", task: "Task", foundAt: "Found at",
		evidence: "Evidence", detailReport: "Detailed report", overview: "Overview",
		traffic:        "Linked traffic evidence",
		trafficMetaFmt: "Evidence version: %d; bindings: %d.",
		trafficStale:   "The evidence changed; the detailed report needs updating.",
		trafficItemFmt: "%d. **Evidence #%d · %s** — `%s %s`, status %d\n",
		request:        "Request", response: "Response",
		csvHeader:  []string{"ID", "Name", "Category", "Severity", "Status", "Task", "Found at", "Summary", "Traffic evidence", "Evidence IDs"},
		severities: map[string]string{"critical": "Critical", "high": "High", "medium": "Medium", "low": "Low", "info": "Info"},
		statuses: map[string]string{
			"pending": "Pending", "in_progress": "In progress", "confirmed": "Confirmed", "resolved": "Resolved",
			"fixed": "Fixed", "false_positive": "False positive", "ignored": "Ignored",
			"duplicate": "Duplicate", "risk_accepted": "Risk accepted",
		},
	},
	ZH: {
		colon: "：", listSep: "、",
		reportTitle: "渗透测试报告", untitledTask: "未命名任务",
		goal: "任务目标", unspecified: "（未指定）", generated: "生成时间",
		summary: "摘要", confirmedFmt: "确认发现：**%d** 个", assets: "资产",
		findings: "发现", noConfirmed: "_本次未确认漏洞。_",
		uncategorized: "未分类", pocEvidence: "PoC / 证据",
		exportTitle: "漏洞发现汇总报告", total: "发现总数", countFmt: "%d 个",
		severity: "严重等级", count: "数量", details: "漏洞明细",
		noMatches: "_无匹配的漏洞。_",
		class:     "类别", status: "状态", task: "所属任务", foundAt: "发现时间",
		evidence: "证据", detailReport: "详细报告", overview: "概述",
		traffic:        "关联流量证据",
		trafficMetaFmt: "证据版本：%d；绑定数量：%d。",
		trafficStale:   "证据已变更，详细报告待更新。",
		trafficItemFmt: "%d. **证据 #%d · %s** — `%s %s`，状态码 %d\n",
		request:        "请求报文", response: "响应报文",
		csvHeader:  []string{"ID", "名称", "类别", "严重等级", "状态", "所属任务", "发现时间", "概述", "流量证据数量", "流量证据ID"},
		severities: map[string]string{"critical": "严重", "high": "高危", "medium": "中危", "low": "低危", "info": "信息"},
		statuses: map[string]string{
			"pending": "待处理", "in_progress": "处理中", "confirmed": "已确认", "resolved": "已处理",
			"fixed": "已修复", "false_positive": "误报", "ignored": "忽略",
			"duplicate": "重复", "risk_accepted": "风险接受",
		},
	},
}

func (l Lang) t() *texts {
	if t, ok := catalog[l]; ok {
		return t
	}
	return catalog[ES]
}

// sev returns the severity label; an empty severity reads as info and an
// unknown one is shown as stored.
func (t *texts) sev(s string) string {
	s = nz(s, "info")
	if label, ok := t.severities[s]; ok {
		return label
	}
	return s
}

func (t *texts) stat(s string) string {
	s = nz(s, "pending")
	if label, ok := t.statuses[s]; ok {
		return label
	}
	return s
}

// tag is the bracketed severity in a finding heading, e.g. [ALTA].
func (t *texts) tag(s string) string { return "[" + strings.ToUpper(t.sev(s)) + "]" }

// field renders one "- **label**: value" line.
func (t *texts) field(label, value string) string {
	return "- **" + label + "**" + t.colon + value + "\n"
}
