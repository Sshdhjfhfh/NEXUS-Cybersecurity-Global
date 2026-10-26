// Nexus IA v6.0 — Absorbido de norma/memory
// Upstream: github.com/Autumn-27/norma/memory
// Licencia: AGPL-3.0

// Package memory gives the agent persistent, file-backed long-term memory.
// Each memory is a markdown file with YAML frontmatter
// (name / description / type) in a memory directory; a MEMORY.md index lists
// them. The agent recalls memories relevant to a query and records new ones via
// tools, and relevant memories can be auto-injected into the conversation.
package memory

import (
	"context"
	"fmt"
	"golang.org/x/text/unicode/norm"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Type classifies a memory.
type Type string

const (
	TypeUser      Type = "user"      // facts about the user (role, preferences)
	TypeFeedback  Type = "feedback"  // guidance on how the agent should work
	TypeProject   Type = "project"   // ongoing work, goals, constraints
	TypeReference Type = "reference" // pointers to external resources
)

// Memory is a single stored fact.
type Memory struct {
	Name        string
	Description string
	Type        Type
	Content     string
	Path        string
	ModTime     time.Time
}

// Store is a memory directory.
type Store struct {
	Dir string
	// Ranker is an optional host-owned retrieval policy. It does not grant tools
	// access to the evaluator or to another task's memory.
	Ranker Ranker
}

type Ranker interface {
	Rank(context.Context, []Memory, string, int) []Memory
}

// NewStore roots a store at dir (created on first write).
func NewStore(dir string) *Store { return &Store{Dir: dir} }

const indexFile = "MEMORY.md"

func (s *Store) ensure() error {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	return s.validateDir()
}

// validateDir is also used on reads: an existing store must not redirect a
// task's memory to another directory. Reading an absent store never creates it.
func (s *Store) validateDir() error {
	info, err := os.Lstat(filepath.Clean(s.Dir))
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("memory: store must be a directory, not a symlink")
	}
	return nil
}

// Striped locks serialize writers across Store instances without retaining one
// mutex per historical task. Atomic renames keep readers free of partial files.
var storeLocks [64]sync.Mutex

func (s *Store) lock() func() {
	path, _ := filepath.Abs(s.Dir)
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(path))
	mu := &storeLocks[hash.Sum32()%uint32(len(storeLocks))]
	mu.Lock()
	return mu.Unlock
}

func atomicWrite(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".memory-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

var slugRe = regexp.MustCompile(`[^\p{L}\p{N}]+`)

func slug(name string) string {
	s := slugRe.ReplaceAllString(strings.ToLower(norm.NFC.String(name)), "-")
	return strings.Trim(s, "-")
}

// Scan reads every memory file (excluding the index), newest first.
func (s *Store) Scan() ([]Memory, error) {
	if err := s.validateDir(); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Memory
	for _, e := range entries {
		if e.IsDir() || e.Type()&os.ModeSymlink != 0 || e.Name() == indexFile || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		m, err := s.readFile(filepath.Join(s.Dir, e.Name()))
		if err == nil {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModTime.After(out[j].ModTime) })
	return out, nil
}

// Read returns the memory with the given name (or its slug).
func (s *Store) Read(name string) (Memory, error) {
	if slug(name) == "" {
		return Memory{}, fmt.Errorf("memory: invalid name")
	}
	p := filepath.Join(s.Dir, slug(name)+".md")
	return s.readFile(p)
}

func (s *Store) readFile(path string) (Memory, error) {
	if err := s.validateDir(); err != nil {
		return Memory{}, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return Memory{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > 65536 {
		return Memory{}, fmt.Errorf("memory: invalid file")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Memory{}, err
	}
	m := parseFrontmatter(string(data))
	m.Path = path
	if m.Name == "" {
		m.Name = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	if info, e := os.Stat(path); e == nil {
		m.ModTime = info.ModTime()
	}
	return m, nil
}

// Save writes a memory file and rebuilds the MEMORY.md index.
func (s *Store) Save(m Memory) error {
	unlock := s.lock()
	defer unlock()
	if slug(m.Name) == "" || strings.ContainsAny(m.Name+m.Description, "\r\n") {
		return fmt.Errorf("memory: invalid metadata")
	}
	if !utf8.ValidString(m.Name+m.Description+m.Content) || len(renderFrontmatter(m)) > 65536 {
		return fmt.Errorf("memory: invalid or oversized content")
	}
	switch m.Type {
	case "", TypeUser, TypeFeedback, TypeProject, TypeReference:
	default:
		return fmt.Errorf("memory: invalid type")
	}
	if err := s.ensure(); err != nil {
		return err
	}
	if m.Name == "" {
		return fmt.Errorf("memory: name is required")
	}
	path := filepath.Join(s.Dir, slug(m.Name)+".md")
	if err := atomicWrite(path, []byte(renderFrontmatter(m))); err != nil {
		return err
	}
	return s.rebuildIndex()
}

// Index returns the MEMORY.md content (an index of memories), building it from a
// scan if the file is absent or is not a regular file.
func (s *Store) Index() string {
	if err := s.validateDir(); err == nil {
		path := filepath.Join(s.Dir, indexFile)
		if info, err := os.Lstat(path); err == nil && info.Mode().IsRegular() {
			if data, err := os.ReadFile(path); err == nil {
				return string(data)
			}
		}
	}
	return s.buildIndex()
}

func (s *Store) rebuildIndex() error {
	return atomicWrite(filepath.Join(s.Dir, indexFile), []byte(s.buildIndex()))
}

func (s *Store) buildIndex() string {
	mems, _ := s.Scan()
	var b strings.Builder
	b.WriteString("# Memory\n\n")
	if len(mems) == 0 {
		b.WriteString("(no memories yet)\n")
		return b.String()
	}
	for _, m := range mems {
		fmt.Fprintf(&b, "- [%s](%s.md) — %s\n", m.Name, slug(m.Name), m.Description)
	}
	return b.String()
}

// parseFrontmatter extracts name/description/type and the body from a markdown
// file with optional YAML frontmatter.
func parseFrontmatter(s string) Memory {
	var m Memory
	if strings.HasPrefix(s, "---") {
		rest := strings.TrimPrefix(s, "---")
		rest = strings.TrimLeft(rest, "\r\n")
		if i := strings.Index(rest, "\n---"); i >= 0 {
			head := rest[:i]
			body := rest[i+len("\n---"):]
			body = strings.TrimLeft(body, "-")
			m.Content = strings.TrimLeft(body, "\r\n")
			for _, line := range strings.Split(head, "\n") {
				k, v, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				k = strings.TrimSpace(k)
				v = strings.TrimSpace(v)
				switch k {
				case "name":
					m.Name = v
				case "description":
					m.Description = v
				case "type":
					m.Type = Type(v)
				}
			}
			return m
		}
	}
	m.Content = s
	return m
}

func renderFrontmatter(m Memory) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "name: %s\n", m.Name)
	fmt.Fprintf(&b, "description: %s\n", m.Description)
	if m.Type != "" {
		fmt.Fprintf(&b, "type: %s\n", m.Type)
	}
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimRight(m.Content, "\n"))
	b.WriteString("\n")
	return b.String()
}
