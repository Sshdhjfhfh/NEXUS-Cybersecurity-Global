package telemetry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Reject ambiguous envelopes before attribution. Fields retain case-sensitive
// names; envelope/source keys follow encoding/json's case-insensitive matching.
func checkJSONKeys(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var walk func(int, bool) error
	walk = func(depth int, fields bool) error {
		if depth > 4 {
			return fmt.Errorf("JSON nesting limit exceeded")
		}
		token, err := d.Token()
		if err != nil {
			return err
		}
		switch token {
		case json.Delim('{'):
			seen := map[string]bool{}
			for d.More() {
				keyToken, err := d.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return fmt.Errorf("invalid JSON key")
				}
				canonical := key
				if !fields {
					canonical = strings.ToLower(key)
				}
				if seen[canonical] {
					return fmt.Errorf("duplicate event JSON key")
				}
				seen[canonical] = true
				if err := walk(depth+1, depth == 0 && canonical == "fields"); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		case json.Delim('['):
			for d.More() {
				if err := walk(depth+1, fields); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	return walk(0, false)
}
