package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Reader reads knowledge files from disk and concatenates their content.
// All file paths are resolved relative to baseDir; paths that escape
// baseDir are rejected to prevent directory traversal.
type Reader struct {
	baseDir string
	mu      sync.RWMutex
	cache   map[string]cacheEntry
}

type cacheEntry struct {
	modTime time.Time
	size    int64
	text    string
}

// NewReader creates a reader with the base directory for knowledge files.
func NewReader(knowledgeDir string) *Reader {
	abs, err := filepath.Abs(knowledgeDir)
	if err != nil {
		abs = knowledgeDir
	}
	return &Reader{
		baseDir: abs,
		cache:   map[string]cacheEntry{},
	}
}

// ReadByPaths reads files at the given paths and concatenates their
// contents into a single knowledge context string for the Planner LLM.
// Paths are resolved relative to baseDir; absolute paths or paths
// that escape baseDir via ".." are rejected.
func (r *Reader) ReadByPaths(paths []string) (string, error) {
	if len(paths) == 0 {
		return "", nil
	}
	var b strings.Builder
	for i, path := range paths {
		safePath, err := r.resolveSafe(path)
		if err != nil {
			b.WriteString(fmt.Sprintf("--- [rejected: %s — %v] ---\n", path, err))
			continue
		}
		data, err := r.readFileCached(safePath)
		if err != nil {
			b.WriteString("--- [file not found: " + path + "] ---\n")
			continue
		}
		if i > 0 {
			b.WriteString("\n---\n\n")
		}
		b.WriteString("# Knowledge: " + path + "\n\n")
		b.WriteString(strings.TrimSpace(string(data)))
		b.WriteString("\n")
	}
	return b.String(), nil
}

// Invalidate drops cached entries for specified paths.
// Relative paths are resolved under knowledge base directory.
func (r *Reader) Invalidate(paths ...string) {
	if len(paths) == 0 {
		r.mu.Lock()
		r.cache = map[string]cacheEntry{}
		r.mu.Unlock()
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, p := range paths {
		safe, err := r.resolveSafe(p)
		if err != nil {
			continue
		}
		delete(r.cache, safe)
	}
}

func (r *Reader) readFileCached(safePath string) ([]byte, error) {
	info, err := os.Stat(safePath)
	if err != nil {
		return nil, err
	}

	r.mu.RLock()
	cached, ok := r.cache[safePath]
	r.mu.RUnlock()
	if ok && cached.modTime.Equal(info.ModTime()) && cached.size == info.Size() {
		return []byte(cached.text), nil
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.cache[safePath] = cacheEntry{
		modTime: info.ModTime(),
		size:    info.Size(),
		text:    string(data),
	}
	r.mu.Unlock()
	return data, nil
}

// resolveSafe ensures the path stays inside baseDir.
func (r *Reader) resolveSafe(path string) (string, error) {
	candidate := path
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(r.baseDir, candidate)
	}
	abs, err := filepath.Abs(candidate)
	if err != nil {
		return "", fmt.Errorf("cannot resolve: %w", err)
	}
	abs = filepath.Clean(abs)
	base := filepath.Clean(r.baseDir)
	if !strings.HasPrefix(abs, base+string(filepath.Separator)) && abs != base {
		return "", fmt.Errorf("path escapes knowledge directory")
	}
	return abs, nil
}
