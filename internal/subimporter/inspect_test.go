package subimporter

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/nerd-zero/listnun/internal/i18n"
)

func testImporter(t *testing.T, opt Options) *Importer {
	t.Helper()
	lang, err := i18n.New([]byte(`{"_.code": "en", "_.name": "English",
		"subscribers.invalidEmail": "Invalid e-mail",
		"subscribers.domainBlocklisted": "Domain blocked"}`))
	if err != nil {
		t.Fatal(err)
	}
	return New(opt, nil, lang)
}

func writeCSV(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "subs.csv")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGuessColumn(t *testing.T) {
	cases := map[string]Column{
		"Email Address":   {Index: 0, As: AsEmail},
		"e-mail":          {Index: 0, As: AsEmail},
		"full_name":       {Index: 0, As: AsName},
		"attributes":      {Index: 0, As: AsAttributes},
		"created_at":      {Index: 0, As: AsSkip},
		"":                {Index: 0, As: AsSkip},
		"Favourite Fruit": {Index: 0, As: AsAttribute, Key: "favourite_fruit"},
	}
	for h, want := range cases {
		if got := GuessColumn(0, h); got != want {
			t.Errorf("GuessColumn(%q) = %+v, want %+v", h, got, want)
		}
	}
}

func TestInspect(t *testing.T) {
	// Semicolons, a BOM, two columns that look like the email, and a row
	// with a blank value that shouldn't show as a sample.
	path := writeCSV(t, "\xef\xbb\xbfemail_address;Name;city;mail\n"+
		"ana@example.com;Ana;Harare;x@example.com\n"+
		"tendai@example.com;Tendai;;y@example.com\n"+
		"rudo@example.com;Rudo;Bulawayo;z@example.com\n"+
		"kuda@example.com;Kuda;Mutare;w@example.com\n")

	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Delim != ";" || got.Rows != 4 {
		t.Fatalf("delim %q rows %d, want ; and 4", got.Delim, got.Rows)
	}
	want := []InspectedColumn{
		{Header: "email_address", Samples: []string{"ana@example.com", "tendai@example.com", "rudo@example.com"}, Column: Column{Index: 0, As: AsEmail}},
		{Header: "Name", Samples: []string{"Ana", "Tendai", "Rudo"}, Column: Column{Index: 1, As: AsName}},
		{Header: "city", Samples: []string{"Harare", "Bulawayo", "Mutare"}, Column: Column{Index: 2, As: AsAttribute, Key: "city"}},
		// Only one column can be the email.
		{Header: "mail", Samples: []string{"x@example.com", "y@example.com", "z@example.com"}, Column: Column{Index: 3, As: AsAttribute, Key: "mail"}},
	}
	if !reflect.DeepEqual(got.Columns, want) {
		t.Errorf("columns =\n%+v\nwant\n%+v", got.Columns, want)
	}
}

func TestCheck(t *testing.T) {
	im := testImporter(t, Options{DomainBlocklist: []string{"blocked.com"}})
	path := writeCSV(t, "who,mail,plan\n"+
		"Ana,ana@example.com,pro\n"+
		"Ana again,ANA@example.com,pro\n"+ // the same person twice
		"Tendai,tendai@example.com,free\n"+
		"Nobody,,free\n"+
		"Bad,kuda@@example.com,free\n"+
		"Spam,x@blocked.com,free\n"+
		"Short\n")

	var looked []string
	existing := func(emails []string) (int, error) {
		looked = append(looked, emails...)
		return 1, nil
	}
	cols := []Column{{Index: 0, As: AsName}, {Index: 1, As: AsEmail}, {Index: 2, As: AsAttribute, Key: "plan"}}

	got, err := im.Check(path, ',', cols, existing)
	if err != nil {
		t.Fatal(err)
	}
	want := CheckResult{
		Rows:        7,
		Valid:       2,
		Existing:    1,
		NumProblems: 4,
		Problems: []Problem{
			{Row: 5, Value: "", Reason: ReasonNoEmail},
			{Row: 6, Value: "kuda@@example.com", Reason: ReasonInvalidEmail},
			{Row: 7, Value: "x@blocked.com", Reason: ReasonBlockedDomain},
			{Row: 8, Value: "", Reason: ReasonColumns},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Check =\n%+v\nwant\n%+v", got, want)
	}
	if !reflect.DeepEqual(looked, []string{"ana@example.com", "tendai@example.com"}) {
		t.Errorf("looked up %v, want each valid address once", looked)
	}

	if _, err := im.Check(path, ',', []Column{{Index: 0, As: AsName}}, existing); !errors.Is(err, ErrNoEmailColumn) {
		t.Errorf("no email column: err = %v, want ErrNoEmailColumn", err)
	}
}

// With Columns set, LoadCSV saves matched columns as attributes, ignores
// skipped ones, and keeps each row it leaves out.
func TestLoadCSVWithColumns(t *testing.T) {
	im := testImporter(t, Options{})
	path := writeCSV(t, "full name,email,city,created\n"+
		"Ana Dube,ana@example.com,Harare,2025-03-14\n"+
		"Bad,not-an-email,Harare,2025-03-14\n"+
		"Tendai,tendai@example.com,,2025-06-02\n")

	s, err := im.NewSession(SessionOpt{Filename: "subs.csv", Mode: ModeSubscribe, Columns: []Column{
		{Index: 0, As: AsName},
		{Index: 1, As: AsEmail},
		{Index: 2, As: AsAttribute, Key: "City"},
		{Index: 3, As: AsSkip},
	}})
	if err != nil {
		t.Fatal(err)
	}

	var got []SubReq
	done := make(chan struct{})
	go func() {
		for sub := range s.subQueue {
			got = append(got, sub)
		}
		close(done)
	}()
	if err := s.LoadCSV(path, ','); err != nil {
		t.Fatal(err)
	}
	<-done

	if len(got) != 2 {
		t.Fatalf("queued %d, want 2", len(got))
	}
	if got[0].Name != "Ana Dube" || got[0].Attribs["city"] != "Harare" || len(got[0].Attribs) != 1 {
		t.Errorf("first = %q %v, want Ana Dube with only city", got[0].Name, got[0].Attribs)
	}
	// An empty cell sets no attribute.
	if got[1].Attribs != nil {
		t.Errorf("second attribs = %v, want none", got[1].Attribs)
	}

	want := []Problem{{Row: 3, Value: "not-an-email", Reason: ReasonInvalidEmail}}
	if p := im.GetProblems(); !reflect.DeepEqual(p, want) {
		t.Errorf("problems = %+v, want %+v", p, want)
	}
	if st := im.GetStats(); st.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", st.Skipped)
	}
}
