package fair

import (
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

//go:embed static/*
var assets embed.FS

func (h *Harness) Handler(host string) (http.Handler, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(secret)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		if r.Host != host {
			http.Error(w, "Host no autorizado", 403)
			return
		}
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "Origen no autorizado", 403)
			return
		}
		if r.Method == "GET" && (r.URL.Path == "/" || r.URL.Path == "/app.js" || r.URL.Path == "/app.css") {
			file := "index.html"
			contentType := "text/html; charset=utf-8"
			if r.URL.Path != "/" {
				file = strings.TrimPrefix(r.URL.Path, "/")
				contentType = "text/javascript; charset=utf-8"
				if file == "app.css" {
					contentType = "text/css; charset=utf-8"
				}
			}
			body, err := assets.ReadFile("static/" + file)
			if err != nil {
				http.Error(w, "Recurso no disponible", 500)
				return
			}
			if r.URL.Path == "/" {
				http.SetCookie(w, &http.Cookie{Name: "nexus_fair_session", Value: token, HttpOnly: true, SameSite: http.SameSiteStrictMode, Path: "/", MaxAge: 86400})
			}
			w.Header().Set("Content-Type", contentType)
			w.Write(body)
			return
		}
		cookie, err := r.Cookie("nexus_fair_session")
		if err != nil || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(token)) != 1 {
			http.Error(w, "Abre primero la consola local de NEXUS", 401)
			return
		}
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/state":
			writeJSON(w, h.State())
		case r.Method == "POST" && r.URL.Path == "/api/run":
			if r.Header.Get("Origin") != "http://"+host || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				http.Error(w, "Origen o contenido no autorizado", 403)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 1024)
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			var command struct {
				Mode string `json:"mode"`
			}
			if err := decoder.Decode(&command); err != nil {
				http.Error(w, "Solicitud no válida", 400)
				return
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				http.Error(w, "Se requiere una sola solicitud", 400)
				return
			}
			var result any
			var runErr error
			switch command.Mode {
			case "baseline":
				result, runErr = h.Run(r.Context(), false)
			case "secured":
				result, runErr = h.Run(r.Context(), true)
			case "evaluate":
				result, runErr = h.Evaluate(r.Context())
			default:
				http.Error(w, "Modo no reconocido", 400)
				return
			}
			if runErr != nil {
				http.Error(w, "No se completó la comprobación: "+runErr.Error(), 500)
				return
			}
			writeJSON(w, result)
		case r.Method == "GET" && r.URL.Path == "/api/report.json":
			state := h.State()
			if len(state.Runs) == 0 {
				http.Error(w, "Ejecuta una prueba antes de exportar", 409)
				return
			}
			w.Header().Set("Content-Disposition", `attachment; filename="nexus-evidencias.json"`)
			writeJSON(w, state)
		case r.Method == "GET" && r.URL.Path == "/api/report.md":
			state := h.State()
			if len(state.Runs) == 0 {
				http.Error(w, "Ejecuta una prueba antes de exportar", 409)
				return
			}
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="nexus-informe.md"`)
			io.WriteString(w, Markdown(state))
		default:
			http.NotFound(w, r)
		}
	}), nil
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(value)
}

func Markdown(state State) string {
	var out strings.Builder
	out.WriteString("# NEXUS — Evaluación ofensiva y defensiva\n\nLaboratorio de demostración con datos ficticios.\n\n")
	fmt.Fprintf(&out, "Objetivo HTTP: `%s`. Identidad lógica de eventos: `%s` (no es un destino de red).\n\n", state.Target, asset)
	out.WriteString("## Método y alcance\n\nCuatro peticiones GET al objetivo local creado por NEXUS: dos controles normales y dos comprobaciones activas. Las respuestas se convierten en eventos para el motor Sigma/telemetría de NEXUS. No se aceptan direcciones externas.\n\n")
	for _, run := range state.Runs {
		mode := "antes de corregir"
		if run.Secured {
			mode = "después de corregir"
		}
		fmt.Fprintf(&out, "## Ejecución %d — %s\n\nFecha UTC: %s. Duración: %d ms. Hallazgos confirmados: %d. Alertas: %d.\n\n", run.ID, mode, run.At.Format(time.RFC3339), run.ElapsedMS, len(run.Findings), len(run.Detection.Alerts))
		out.WriteString("| Evidencia | Ruta | HTTP | Acceso indebido confirmado |\n|---|---|---:|---|\n")
		for _, observation := range run.Observations {
			fmt.Fprintf(&out, "| %s | `%s` | %d | %t |\n", observation.ID, observation.Path, observation.Status, observation.Succeeded)
		}
		out.WriteString("\n")
		for _, finding := range run.Findings {
			fmt.Fprintf(&out, "- **%s** (%s). Evidencia: %s. Corrección: %s\n", finding.Title, finding.Severity, finding.EvidenceID, finding.Remediation)
		}
		fmt.Fprintf(&out, "\nDetección de intentos en estos cuatro casos: TP=%d, FP=%d, TN=%d, FN=%d. Las alertas de intentos pueden persistir aunque la corrección impida el acceso.\n\nHuella del lote de eventos: `%s`.\n\n", run.Metrics.TruePositive, run.Metrics.FalsePositive, run.Metrics.TrueNegative, run.Metrics.FalseNegative, run.Detection.EvidenceDigest)
		for _, observation := range run.Observations {
			fmt.Fprintf(&out, "### %s\n\nSHA-256 de la respuesta: `%s`\n\n````text\n%s\n````\n\n", observation.ID, observation.SHA256, observation.Body)
		}
	}
	if e := state.Evaluation; e != nil {
		fmt.Fprintf(&out, "## Experimento separado de evaluación de reglas\n\nCasos reservados: %d. Aciertos iniciales: %d. Aciertos candidatos: %d. Decisión: %s. Candidato con falsas alarmas: %s (estrategia conservada: %s).\n\n%s\n\n", e.Improvement.Holdout.Cases, e.Improvement.Holdout.BaselinePassed, e.Improvement.Holdout.CandidatePassed, e.Improvement.Decision, e.Noisy.Decision, e.Noisy.Active, e.Limitation)
	}
	out.WriteString("## Límites y atribución\n\nEsta ejecución demuestra dos fallos deliberados y dos reglas específicas del escenario. No demuestra cobertura universal, detección de zero-days, precisión en producción ni autonomía de un modelo de IA. La corrección conmuta una configuración preimplementada del objetivo ficticio; no modifica aplicaciones de clientes. Los resultados se conservan en memoria durante la sesión y deben exportarse.\n\nNEXUS es un proyecto derivado: consultar CREDITS.md para ARTEX y norma, y LICENSE. Este módulo reutiliza Sigma/telemetría, detectbench y evolve de NEXUS; la consola, el objetivo y las pruebas de feria están en lab/fair.\n")
	return out.String()
}
