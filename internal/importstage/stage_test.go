package importstage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportStage(t *testing.T) {
	st := New()
	now := time.Now()

	dir := t.TempDir()
	oldDir := filepath.Join(dir, "old")
	if err := os.Mkdir(oldDir, 0o700); err != nil {
		t.Fatal(err)
	}

	old, err := st.Put(File{TenantID: 1, Dir: oldDir, At: now.Add(-3 * time.Hour)}, now.Add(-3*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	id, err := st.Put(File{TenantID: 1, Name: "a.csv", At: now}, now)
	if err != nil {
		t.Fatal(err)
	}

	if f, ok := st.Get(1, id, now); !ok || f.Name != "a.csv" {
		t.Errorf("get own file = %+v, %v", f, ok)
	}
	// Another workspace can't use the file, even with its ID.
	if _, ok := st.Get(2, id, now); ok {
		t.Error("another tenant got the file")
	}
	if _, ok := st.Get(1, id, now.Add(TTL+time.Minute)); ok {
		t.Error("got a file past its expiry")
	}

	// The second put dropped the expired file and removed its directory.
	if _, ok := st.Get(1, old, now.Add(-3*time.Hour)); ok {
		t.Error("expired file still kept")
	}
	if _, err := os.Stat(oldDir); !os.IsNotExist(err) {
		t.Errorf("expired file's directory still there: %v", err)
	}
}
