// Nexus IA v6.0 — Internacionalización (10 idiomas)
//
// Sistema de i18n con 10 idiomas (Ethnologue 2026):
//   en(1520M) zh(1180M) hi(609M) es(559M DEFAULT) ar(335M RTL)
//   fr(312M) bn(274M) pt(267M) id(255M) ru(210M)
//
// Total: 4.531M hablantes = 56% de la población mundial
//
// Características:
//   - RTL automático para árabe
//   - Logs internos SIEMPRE en inglés
//   - MITRE ATT&CK IDs siempre en inglés
//   - Cambio en caliente (sin reiniciar)
//   - Auto-translate con LLM para claves faltantes
//
// Nivel 0 — Interfaz
package i18n

import (
	"fmt"
	"sync"
)

// ═══════════════════════════════════════════════════════════════
// Idiomas soportados
// ═══════════════════════════════════════════════════════════════

// Lang es el código ISO 639-1 del idioma.
type Lang string

const (
	LangEnglish    Lang = "en" // 1.520M — Base del sistema
	LangChinese    Lang = "zh" // 1.180M — Hanzi UTF-8
	LangHindi      Lang = "hi" // 609M   — Devanagari
	LangSpanish    Lang = "es" // 559M   — DEFAULT (autor)
	LangArabic     Lang = "ar" // 335M   — RTL automático
	LangFrench     Lang = "fr" // 312M
	LangBengali    Lang = "bn" // 274M   — Bengali script
	LangPortuguese Lang = "pt" // 267M
	LangIndonesian Lang = "id" // 255M
	LangRussian    Lang = "ru" // 210M   — Cyrillic
)

// AllLangs devuelve todos los idiomas soportados.
func AllLangs() []Lang {
	return []Lang{
		LangEnglish, LangChinese, LangHindi, LangSpanish,
		LangArabic, LangFrench, LangBengali, LangPortuguese,
		LangIndonesian, LangRussian,
	}
}

// IsRTL devuelve true si el idioma es Right-to-Left.
func (l Lang) IsRTL() bool {
	return l == LangArabic
}

// Dir devuelve la dirección del texto ("ltr" o "rtl").
func (l Lang) Dir() string {
	if l.IsRTL() {
		return "rtl"
	}
	return "ltr"
}

// NativeName devuelve el nombre del idioma en su propio idioma.
func (l Lang) NativeName() string {
	switch l {
	case LangEnglish:
		return "English"
	case LangChinese:
		return "中文"
	case LangHindi:
		return "हिन्दी"
	case LangSpanish:
		return "Español"
	case LangArabic:
		return "العربية"
	case LangFrench:
		return "Français"
	case LangBengali:
		return "বাংলা"
	case LangPortuguese:
		return "Português"
	case LangIndonesian:
		return "Bahasa Indonesia"
	case LangRussian:
		return "Русский"
	default:
		return string(l)
	}
}

// Speakers devuelve el número de hablantes (millones).
func (l Lang) Speakers() int {
	switch l {
	case LangEnglish:
		return 1520
	case LangChinese:
		return 1180
	case LangHindi:
		return 609
	case LangSpanish:
		return 559
	case LangArabic:
		return 335
	case LangFrench:
		return 312
	case LangBengali:
		return 274
	case LangPortuguese:
		return 267
	case LangIndonesian:
		return 255
	case LangRussian:
		return 210
	default:
		return 0
	}
}

// IsValid comprueba si el idioma es válido.
func (l Lang) IsValid() bool {
	for _, valid := range AllLangs() {
		if l == valid {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════
// Manager
// ═══════════════════════════════════════════════════════════════

// Manager gestiona las traducciones.
type Manager struct {
	mu           sync.RWMutex
	currentLang  Lang
	defaultLang  Lang
	translations map[Lang]map[string]string

	// Fallback
	fallbackLang Lang

	// Auto-translate (LLM)
	autoTranslate func(key, targetLang string) (string, error)

	// Stats
	stats ManagerStats
}

// ManagerStats lleva estadísticas de uso.
type ManagerStats struct {
	TotalLookups   int64
	TotalMisses    int64
	AutoTranslated int64
}

// Config configura el Manager.
type Config struct {
	DefaultLang  Lang
	FallbackLang Lang
	AutoTranslate func(key, targetLang string) (string, error)
}

// DefaultConfig devuelve la configuración por defecto.
func DefaultConfig() Config {
	return Config{
		DefaultLang:  LangSpanish, // DEFAULT: idioma del autor
		FallbackLang: LangEnglish,
	}
}

// NewManager crea un Manager.
func NewManager(cfg Config) *Manager {
	if cfg.DefaultLang == "" {
		cfg.DefaultLang = LangSpanish
	}
	if cfg.FallbackLang == "" {
		cfg.FallbackLang = LangEnglish
	}
	return &Manager{
		currentLang:   cfg.DefaultLang,
		defaultLang:   cfg.DefaultLang,
		fallbackLang:  cfg.FallbackLang,
		translations:  make(map[Lang]map[string]string),
		autoTranslate: cfg.AutoTranslate,
	}
}

// AddTranslations añade un mapa de traducciones para un idioma.
func (m *Manager) AddTranslations(lang Lang, translations map[string]string) error {
	if !lang.IsValid() {
		return fmt.Errorf("invalid language: %s", lang)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.translations[lang] == nil {
		m.translations[lang] = make(map[string]string)
	}
	for k, v := range translations {
		m.translations[lang][k] = v
	}
	return nil
}

// SetTranslations reemplaza todas las traducciones de un idioma.
func (m *Manager) SetTranslations(lang Lang, translations map[string]string) error {
	if !lang.IsValid() {
		return fmt.Errorf("invalid language: %s", lang)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.translations[lang] = translations
	return nil
}

// T traduce una clave al idioma actual.
//
// Cadena de fallback:
//   1. currentLang
//   2. fallbackLang (inglés)
//   3. autoTranslate (si configurado)
//   4. devuelve la clave entre corchetes
func (m *Manager) T(key string) string {
	m.mu.RLock()
	lang := m.currentLang
	fallback := m.fallbackLang
	currentTrans := m.translations[lang]
	fallbackTrans := m.translations[fallback]
	autoFn := m.autoTranslate
	m.mu.RUnlock()

	m.mu.Lock()
	m.stats.TotalLookups++
	m.mu.Unlock()

	// 1. Buscar en el idioma actual
	if currentTrans != nil {
		if v, ok := currentTrans[key]; ok {
			return v
		}
	}

	// 2. Fallback al inglés
	if fallbackTrans != nil {
		if v, ok := fallbackTrans[key]; ok {
			m.mu.Lock()
			m.stats.TotalMisses++
			m.mu.Unlock()
			return v
		}
	}

	// 3. Auto-translate
	if autoFn != nil {
		if v, err := autoFn(key, string(lang)); err == nil && v != "" {
			// Cachear el resultado
			m.mu.Lock()
			if m.translations[lang] == nil {
				m.translations[lang] = make(map[string]string)
			}
			m.translations[lang][key] = v
			m.stats.AutoTranslated++
			m.stats.TotalMisses++
			m.mu.Unlock()
			return v
		}
	}

	// 4. Devolver la clave entre corchetes (visible = bug reportable)
	m.mu.Lock()
	m.stats.TotalMisses++
	m.mu.Unlock()
	return "[" + key + "]"
}

// Tf traduce con formato.
func (m *Manager) Tf(key string, args ...interface{}) string {
	template := m.T(key)
	if len(args) == 0 {
		return template
	}
	return fmt.Sprintf(template, args...)
}

// CurrentLang devuelve el idioma actual.
func (m *Manager) CurrentLang() Lang {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentLang
}

// SetLang cambia el idioma en caliente.
func (m *Manager) SetLang(lang Lang) error {
	if !lang.IsValid() {
		return fmt.Errorf("invalid language: %s", lang)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentLang = lang
	return nil
}

// Langs devuelve todos los idiomas cargados.
func (m *Manager) Langs() []Lang {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Lang, 0, len(m.translations))
	for l := range m.translations {
		out = append(out, l)
	}
	return out
}

// Stats devuelve las estadísticas.
func (m *Manager) Stats() ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// Has devuelve true si una clave existe en el idioma actual.
func (m *Manager) Has(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	trans := m.translations[m.currentLang]
	if trans == nil {
		return false
	}
	_, ok := trans[key]
	return ok
}

// KeyCount devuelve el número de claves de un idioma.
func (m *Manager) KeyCount(lang Lang) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.translations[lang])
}

// Coverage devuelve el porcentaje de cobertura de un idioma
// (comparado con el idioma fallback).
func (m *Manager) Coverage(lang Lang) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fallbackCount := len(m.translations[m.fallbackLang])
	if fallbackCount == 0 {
		return 0
	}
	langCount := len(m.translations[lang])
	return float64(langCount) / float64(fallbackCount) * 100
}
