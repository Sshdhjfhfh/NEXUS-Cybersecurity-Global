// Package recallfixture provides synthetic Spanish retrieval cases, without an
// LLM, network traffic, external targets, credentials, or production data.
package recallfixture

import (
	"fmt"
	"time"

	"github.com/nexus-ia/nexus/core/memory"
)

// Memories deliberately includes substring collisions: "red" occurs inside
// "redactor". Both notes are legitimate; the query must identify the right one.
func Memories() []memory.Memory {
	out := []memory.Memory{}
	for i := 0; i < 48; i++ {
		for j, title := range []string{"Red", "Redactor"} {
			out = append(out, memory.Memory{Name: fmt.Sprintf("%s %03d", title, i), Description: "Caso sintético de recuperación", Content: "Dato de laboratorio, sin información privada.", ModTime: time.Unix(int64(i*2+j), 0).UTC()})
		}
	}
	return out
}
