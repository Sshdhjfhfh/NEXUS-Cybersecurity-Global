// Nexus IA v6.0 — Cargador de traducciones desde JSON
package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LoadDir carga todos los archivos .json de un directorio.
func (m *Manager) LoadDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Nombre del archivo sin extensión = código de idioma
		langCode := entry.Name()[:len(entry.Name())-5]
		lang := Lang(langCode)
		if !lang.IsValid() {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		translations, err := LoadJSON(path)
		if err != nil {
			return fmt.Errorf("load %s: %w", path, err)
		}

		if err := m.SetTranslations(lang, translations); err != nil {
			return fmt.Errorf("set %s: %w", lang, err)
		}
	}

	return nil
}

// LoadJSON carga un archivo JSON de traducciones.
// Soporta JSON plano {"key": "value"} y anidado {"section": {"key": "value"}}.
func LoadJSON(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	out := make(map[string]string)
	flatten("", raw, out)
	return out, nil
}

// flatten convierte JSON anidado en claves planas tipo "section.key".
func flatten(prefix string, raw map[string]interface{}, out map[string]string) {
	for k, v := range raw {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case string:
			out[key] = val
		case map[string]interface{}:
			flatten(key, val, out)
		default:
			// Ignorar otros tipos
		}
	}
}

// SaveJSON guarda un mapa de traducciones como JSON (para export).
func SaveJSON(path string, translations map[string]string) error {
	data, err := json.MarshalIndent(translations, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
