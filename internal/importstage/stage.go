// Package importstage keeps files uploaded for a subscriber import until
// the import starts, so columns can be matched and options chosen first.
// Like the importer's own status, files live in this process's memory.
package importstage

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"sync"
	"time"
)

// TTL is how long an uploaded file waits for its import to start.
const TTL = 2 * time.Hour

// File is an uploaded file waiting to be imported.
type File struct {
	TenantID int
	Name     string
	Size     int64
	// Path is the CSV (taken out of the ZIP, for a ZIP); Dir holds it and is
	// removed when the file expires.
	Path  string
	Dir   string
	Delim rune
	At    time.Time
}

// Stage holds uploaded files by ID.
type Stage struct {
	mu    sync.Mutex
	files map[string]File
}

// New returns an empty Stage.
func New() *Stage {
	return &Stage{files: map[string]File{}}
}

// Put keeps f and returns its ID, dropping (and deleting) expired files.
func (st *Stage) Put(f File, now time.Time) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b)

	st.mu.Lock()
	defer st.mu.Unlock()
	for k, old := range st.files {
		if now.Sub(old.At) > TTL {
			os.RemoveAll(old.Dir)
			delete(st.files, k)
		}
	}
	st.files[id] = f
	return id, nil
}

// Get returns the tenant's file with this ID, if it hasn't expired.
func (st *Stage) Get(tenantID int, id string, now time.Time) (File, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	f, ok := st.files[id]
	if !ok || f.TenantID != tenantID || now.Sub(f.At) > TTL {
		return File{}, false
	}
	return f, true
}
