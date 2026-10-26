package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestSpanishRecallAndReopen(t *testing.T) {
	dir := t.TempDir()
	err := NewStore(dir).Save(Memory{Name: "Autenticación", Type: TypeFeedback,
		Description: "Lección verificada", Content: "La sesión requiere autorización; evidencia: prueba local 42."})
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(dir)
	if got := store.Relevant("autenticacion autorizacion", 3); len(got) != 1 || got[0].Name != "Autenticación" {
		t.Fatalf("Spanish recall after reopening: %+v", got)
	}
	info, err := os.Stat(filepath.Join(dir, "autenticación.md"))
	if err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("memory file must be private: %v, %v", info, err)
	}
}

func TestConcurrentStoresKeepCompleteIndex(t *testing.T) {
	dir := t.TempDir()
	var wg sync.WaitGroup
	for i := range 24 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := NewStore(dir).Save(Memory{Name: fmt.Sprintf("Lección %d", i), Content: strings.Repeat("evidencia ", 100)})
			if err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	store := NewStore(dir)
	all, err := store.Scan()
	if err != nil || len(all) != 24 {
		t.Fatalf("memories=%d, err=%v", len(all), err)
	}
	index := store.Index()
	for i := range 24 {
		if !strings.Contains(index, fmt.Sprintf("[Lección %d]", i)) {
			t.Fatalf("index lost lesson %d", i)
		}
	}
}

func TestRejectsInvalidMetadataAndSymlinkReads(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	for _, m := range []Memory{
		{Name: "../", Content: "bad"},
		{Name: "titulo\ntype: user", Content: "bad"},
		{Name: "titulo", Type: "invalid", Content: "bad"},
		{Name: "titulo", Content: strings.Repeat("x", 65537)},
	} {
		if store.Save(m) == nil {
			t.Fatalf("accepted invalid memory: %q", m.Name)
		}
	}
	outside := filepath.Join(t.TempDir(), "private.txt")
	if err := os.WriteFile(outside, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "linked.md")); err != nil {
		t.Skip(err)
	}
	if _, err := store.Read("linked"); err == nil {
		t.Fatal("read followed a symlink")
	}
	if all, err := store.Scan(); err != nil || len(all) != 0 {
		t.Fatalf("scan followed a symlink: %+v, %v", all, err)
	}
}

func TestRejectsLinkedStoreDirectory(t *testing.T) {
	source := NewStore(t.TempDir())
	if err := source.Save(Memory{Name: "Otra tarea", Content: "evidencia privada"}); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(t.TempDir(), "memory")
	if err := os.Symlink(source.Dir, linked); err != nil {
		t.Skip(err)
	}
	store := NewStore(linked)
	if _, err := store.Read("Otra tarea"); err == nil {
		t.Error("read followed a linked store directory")
	}
	if notes, err := store.Scan(); err == nil || len(notes) != 0 {
		t.Errorf("scan followed a linked store directory: %v, %v", notes, err)
	}
	if notes := store.Relevant("evidencia", 3); len(notes) != 0 {
		t.Error("recall exposed another store through a directory link")
	}
	if strings.Contains(store.SystemSegment(), "Otra tarea") {
		t.Error("system prompt exposed another store's index")
	}
	if err := store.Save(Memory{Name: "Nueva nota", Content: "contenido"}); err == nil {
		t.Error("save followed a linked store directory")
	}
	if notes, err := source.Scan(); err != nil || len(notes) != 1 {
		t.Fatalf("linked store changed its target: %v, %v", notes, err)
	}
}

func TestLinkedIndexFallsBackToLocalNotes(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Save(Memory{Name: "Nota local", Content: "evidencia local"}); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "private.md")
	if err := os.WriteFile(outside, []byte("contenido de otra tarea"), 0o600); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(store.Dir, indexFile)
	if err := os.Remove(index); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, index); err != nil {
		t.Skip(err)
	}
	got := store.SystemSegment()
	if strings.Contains(got, "contenido de otra tarea") || !strings.Contains(got, "[Nota local]") {
		t.Fatalf("unsafe index fallback: %s", got)
	}
	info, err := os.Lstat(index)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("reading the index modified the store")
	}
}

func TestReadingAbsentStoreDoesNotCreateIt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "absent")
	store := NewStore(dir)
	if notes, err := store.Scan(); err != nil || len(notes) != 0 {
		t.Fatalf("absent store: %v, %v", notes, err)
	}
	_ = store.SystemSegment()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("read created the store: %v", err)
	}
}
