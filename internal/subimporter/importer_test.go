package subimporter

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
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

// Progress needs the real total before the first insert: it used to grow
// as rows were read, which runs a whole batch ahead of the inserts.
func TestLoadCSVKnowsTotalBeforeFirstRow(t *testing.T) {
	lang, err := i18n.New([]byte(`{"_.code": "en", "_.name": "English", "subscribers.invalidEmail": "Invalid e-mail"}`))
	if err != nil {
		t.Fatal(err)
	}
	im := New(Options{}, nil, lang)

	csv := "email,name\n"
	for i := 0; i < 50; i++ {
		csv += "user" + strconv.Itoa(i) + "@example.com,User\n"
	}
	path := filepath.Join(t.TempDir(), "subs.csv")
	if err := os.WriteFile(path, []byte(csv), 0o600); err != nil {
		t.Fatal(err)
	}

	s, err := im.NewSession(SessionOpt{Filename: "subs.csv", Mode: ModeSubscribe})
	if err != nil {
		t.Fatal(err)
	}

	totalAtFirst := make(chan int, 1)
	go func() {
		first := true
		for range s.subQueue {
			if first {
				totalAtFirst <- im.GetStats().Total
				first = false
			}
		}
	}()

	if err := s.LoadCSV(path, ','); err != nil {
		t.Fatal(err)
	}
	if got := <-totalAtFirst; got != 50 {
		t.Errorf("total when the first row was queued = %d, want 50", got)
	}
}

func TestCountCSVRows(t *testing.T) {
	cases := []struct {
		name string
		csv  string
		want int
	}{
		{"header only", "email,name\n", 0},
		{"rows", "email,name\na@example.com,A\nb@example.com,B\n", 2},
		// LoadCSV counts (and skips) rows with the wrong column count, so the
		// total includes them.
		{"short row", "email,name\na@example.com,A\nb@example.com\n", 2},
		{"no trailing newline", "email,name\na@example.com,A", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "subs.csv")
			if err := os.WriteFile(path, []byte(c.csv), 0o600); err != nil {
				t.Fatal(err)
			}
			if got := countCSVRows(path, ','); got != c.want {
				t.Errorf("countCSVRows = %d, want %d", got, c.want)
			}
		})
	}
	if got := countCSVRows(filepath.Join(t.TempDir(), "missing.csv"), ','); got != -1 {
		t.Errorf("missing file = %d, want -1", got)
	}
}

// GetStats copies the status field by field; a field it leaves out reads as
// zero to the API (pending once did, so progress never moved). Set every
// exported field and check they all come back.
func TestGetStatsReturnsEveryField(t *testing.T) {
	im := New(Options{}, nil, nil)

	want := Status{}
	v := reflect.ValueOf(&want).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		switch f.Kind() {
		case reflect.Int:
			f.SetInt(int64(i + 1))
		case reflect.String:
			f.SetString(v.Type().Field(i).Name)
		default:
			t.Fatalf("Status.%s has kind %s; extend this test", v.Type().Field(i).Name, f.Kind())
		}
	}

	im.Lock()
	im.status = want
	im.Unlock()

	if got := im.GetStats(); !reflect.DeepEqual(got, want) {
		t.Errorf("GetStats() = %+v, want %+v", got, want)
	}
}
