package subimporter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nerd-zero/listnun/internal/i18n"
)

func TestLoadCSVCountsSkippedRows(t *testing.T) {
	lang, err := i18n.New([]byte(`{"_.code": "en", "_.name": "English", "subscribers.invalidEmail": "Invalid e-mail"}`))
	if err != nil {
		t.Fatal(err)
	}
	im := New(Options{}, nil, lang)

	csv := "email,name\n" +
		"ok1@example.com,One\n" +
		"not-an-email,Bad\n" +
		"ok2@example.com\n" + // too few columns
		"ok3@example.com,Three\n"
	path := filepath.Join(t.TempDir(), "subs.csv")
	if err := os.WriteFile(path, []byte(csv), 0o600); err != nil {
		t.Fatal(err)
	}

	s, err := im.NewSession(SessionOpt{Filename: "subs.csv", Mode: ModeSubscribe})
	if err != nil {
		t.Fatal(err)
	}

	// Drain the queue the way Start would, without a database.
	var queued int
	done := make(chan struct{})
	go func() {
		for range s.subQueue {
			queued++
		}
		close(done)
	}()

	if err := s.LoadCSV(path, ','); err != nil {
		t.Fatal(err)
	}
	<-done

	st := im.GetStats()
	if queued != 2 {
		t.Errorf("queued = %d, want 2", queued)
	}
	if st.Skipped != 2 {
		t.Errorf("skipped = %d, want 2", st.Skipped)
	}
	if st.Total != 4 {
		t.Errorf("total = %d, want 4 data rows", st.Total)
	}
	if st.Failed != 0 {
		t.Errorf("failed = %d, want 0", st.Failed)
	}
}
