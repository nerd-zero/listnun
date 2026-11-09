package subimporter

import (
	"archive/zip"
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nerd-zero/listnun/models"
)

// What a CSV column is saved as (Column.As).
const (
	AsEmail = "email"
	AsName  = "name"
	// AsAttribute saves the column's text as one attribute, Column.Key.
	AsAttribute = "attribute"
	// AsAttributes is a column of JSON objects merged into the attributes,
	// the format of listnun's own exports.
	AsAttributes = "attributes"
	AsSkip       = "skip"
)

// Problem reasons, for a row that wasn't imported.
const (
	ReasonNoEmail       = "no_email"
	ReasonInvalidEmail  = "invalid_email"
	ReasonBlockedDomain = "blocked_domain"
	ReasonColumns       = "columns"
)

const (
	// numSamples is how many values of each column Inspect returns.
	numSamples = 3

	// maxProblems is how many problem rows an import or a check keeps. The
	// counts stay exact past it; only the listed rows stop.
	maxProblems = 1000

	// existingBatch is how many addresses a check looks up at a time.
	existingBatch = 5000
)

// Column says what one CSV column (by position) is saved as.
type Column struct {
	Index int    `json:"index"`
	As    string `json:"as"`
	// Key is the attribute name when As is AsAttribute.
	Key string `json:"key,omitempty"`
}

// InspectedColumn is a CSV column, a few of its values and a guess at what
// it holds.
type InspectedColumn struct {
	Header  string   `json:"header"`
	Samples []string `json:"samples"`
	Column
}

// Inspection describes a CSV before it's imported.
type Inspection struct {
	Delim   string            `json:"delim"`
	Rows    int               `json:"rows"`
	Columns []InspectedColumn `json:"columns"`
}

// Problem is a row that wasn't imported, and why.
type Problem struct {
	// Row is the row's line in the file as a spreadsheet numbers it: the
	// header is row 1.
	Row    int    `json:"row"`
	Value  string `json:"value"`
	Reason string `json:"reason"`
}

// CheckResult is what importing a file would do, before anything is
// written.
type CheckResult struct {
	Rows int `json:"rows"`
	// Valid is the number of different, valid addresses in the file.
	Valid int `json:"valid"`
	// Existing is how many of them are already subscribers.
	Existing int `json:"existing"`
	// NumProblems counts the rows that wouldn't be imported; Problems lists
	// up to maxProblems of them.
	NumProblems int       `json:"num_problems"`
	Problems    []Problem `json:"problems"`
}

// ErrNoEmailColumn is returned when no column is saved as the email.
var ErrNoEmailColumn = errors.New("no email column")

// delims are the delimiters DetectDelim recognises, most likely first.
var delims = []rune{',', ';', '\t', '|'}

// DetectDelim returns the delimiter the CSV at path's header uses: the one
// it has most of, or a comma.
func DetectDelim(path string) (rune, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	line, err := bufio.NewReader(f).ReadString('\n')
	if err != nil && err != io.EOF {
		return 0, err
	}

	best, most := ',', 0
	for _, d := range delims {
		if n := strings.Count(line, string(d)); n > most {
			best, most = d, n
		}
	}
	return best, nil
}

// Inspect reads the CSV at path's header and first rows, counts its rows,
// and guesses what each column holds.
func Inspect(path string) (Inspection, error) {
	delim, err := DetectDelim(path)
	if err != nil {
		return Inspection{}, err
	}

	f, err := os.Open(path)
	if err != nil {
		return Inspection{}, err
	}
	defer f.Close()

	rd := csv.NewReader(f)
	rd.Comma = delim
	rd.FieldsPerRecord = -1

	hdr, err := rd.Read()
	if err != nil {
		return Inspection{}, err
	}

	out := Inspection{Delim: string(delim), Columns: make([]InspectedColumn, len(hdr))}
	for i, h := range hdr {
		h = cleanHeader(h)
		out.Columns[i] = InspectedColumn{Header: h, Samples: []string{}, Column: GuessColumn(i, h)}
	}

	for {
		row, err := rd.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// A malformed row is skipped by the import too.
			var pe *csv.ParseError
			if errors.As(err, &pe) {
				out.Rows++
				continue
			}
			return out, err
		}
		out.Rows++
		for i := range out.Columns {
			if i < len(row) && len(out.Columns[i].Samples) < numSamples && strings.TrimSpace(row[i]) != "" {
				out.Columns[i].Samples = append(out.Columns[i].Samples, row[i])
			}
		}
	}

	// Only one column can be the email or the name: keep the first guess.
	seen := map[string]bool{}
	for i, c := range out.Columns {
		if c.As != AsEmail && c.As != AsName {
			continue
		}
		if seen[c.As] {
			out.Columns[i].Column = Column{Index: i, As: AsAttribute, Key: AttributeKey(c.Header)}
		}
		seen[c.As] = true
	}

	return out, nil
}

var (
	reNonWord = regexp.MustCompile(`[^a-z0-9]+`)

	emailHeaders = map[string]bool{
		"email": true, "e_mail": true, "email_address": true, "emailaddress": true,
		"mail": true, "e_mail_address": true, "subscriber_email": true, "contact_email": true,
	}
	nameHeaders = map[string]bool{
		"name": true, "full_name": true, "fullname": true, "display_name": true,
		"contact_name": true, "subscriber_name": true,
	}
	// skipHeaders are other tools' bookkeeping columns, which mean nothing
	// once imported.
	skipHeaders = map[string]bool{
		"id": true, "uuid": true, "created": true, "created_at": true, "updated": true,
		"updated_at": true, "date": true, "timestamp": true, "status": true,
		"optin_time": true, "confirm_time": true, "last_changed": true, "leid": true, "euid": true,
	}
)

// AttributeKey turns a column header into an attribute name: lower case,
// words joined by underscores.
func AttributeKey(h string) string {
	return strings.Trim(reNonWord.ReplaceAllString(strings.ToLower(h), "_"), "_")
}

// GuessColumn guesses what the column at index with header h holds.
func GuessColumn(index int, h string) Column {
	k := AttributeKey(h)
	switch {
	case emailHeaders[k]:
		return Column{Index: index, As: AsEmail}
	case nameHeaders[k]:
		return Column{Index: index, As: AsName}
	case k == "attributes" || k == "attribs":
		return Column{Index: index, As: AsAttributes}
	case k == "" || skipHeaders[k]:
		return Column{Index: index, As: AsSkip}
	}
	return Column{Index: index, As: AsAttribute, Key: k}
}

// cleanHeader trims a header and drops non-ASCII bytes such as a BOM.
func cleanHeader(h string) string {
	return regexCleanStr.ReplaceAllString(strings.TrimSpace(h), "")
}

// columnMap is a session's columns resolved for reading rows.
type columnMap struct {
	email, name, attribs int
	attrs                []Column
	// minCols is the fewest columns a row needs to hold every mapped one.
	minCols int
}

// newColumnMap resolves cols. -1 means a field isn't mapped.
func newColumnMap(cols []Column) (columnMap, error) {
	m := columnMap{email: -1, name: -1, attribs: -1}
	for _, c := range cols {
		if c.Index < 0 {
			continue
		}
		switch c.As {
		case AsEmail:
			m.email = c.Index
		case AsName:
			m.name = c.Index
		case AsAttributes:
			m.attribs = c.Index
		case AsAttribute:
			if c.Key = AttributeKey(c.Key); c.Key == "" {
				continue
			}
			m.attrs = append(m.attrs, c)
		default:
			continue
		}
		m.minCols = max(m.minCols, c.Index+1)
	}
	if m.email < 0 {
		return m, ErrNoEmailColumn
	}
	return m, nil
}

// legacyColumnMap maps a header the way imports always have: columns named
// email, name and attributes, anything else ignored.
func legacyColumnMap(hdr []string) columnMap {
	cols := []Column{}
	for i, h := range hdr {
		switch h := cleanHeader(h); h {
		case AsEmail, AsName, AsAttributes:
			cols = append(cols, Column{Index: i, As: h})
		}
	}
	m, _ := newColumnMap(cols)
	return m
}

// Check reads the CSV at path as an import with these columns would,
// without writing anything: it validates every row and asks existing how
// many of the valid addresses are already subscribers.
func (im *Importer) Check(path string, delim rune, cols []Column, existing func(emails []string) (int, error)) (CheckResult, error) {
	out := CheckResult{Problems: []Problem{}}

	f, err := os.Open(path)
	if err != nil {
		return out, err
	}
	defer f.Close()

	rd := csv.NewReader(f)
	rd.Comma = delim
	rd.FieldsPerRecord = -1
	rd.ReuseRecord = true

	hdr, err := rd.Read()
	if err != nil {
		return out, err
	}
	m, err := newColumnMap(cols)
	if len(cols) == 0 {
		m, err = legacyColumnMap(hdr), nil
		if m.email < 0 {
			err = ErrNoEmailColumn
		}
	}
	if err != nil {
		return out, err
	}

	var (
		seen  = map[string]struct{}{}
		batch = make([]string, 0, existingBatch)
	)
	lookup := func() error {
		if len(batch) == 0 {
			return nil
		}
		n, err := existing(batch)
		if err != nil {
			return err
		}
		out.Existing += n
		batch = batch[:0]
		return nil
	}

	addProblem := func(p Problem) {
		out.NumProblems++
		if len(out.Problems) < maxProblems {
			out.Problems = append(out.Problems, p)
		}
	}

	for line := 1; ; line++ {
		row, err := rd.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			var pe *csv.ParseError
			if !errors.As(err, &pe) {
				return out, err
			}
			out.Rows++
			addProblem(Problem{Row: line + 1, Reason: ReasonColumns})
			continue
		}
		out.Rows++

		sub, p, ok := im.readRow(row, m, line)
		if !ok {
			addProblem(p)
			continue
		}
		if _, dup := seen[sub.Email]; dup {
			continue
		}
		seen[sub.Email] = struct{}{}
		out.Valid++

		batch = append(batch, sub.Email)
		if len(batch) == existingBatch {
			if err := lookup(); err != nil {
				return out, err
			}
		}
	}
	if err := lookup(); err != nil {
		return out, err
	}

	return out, nil
}

// readRow turns a CSV row into a subscriber, or the problem that keeps it
// out. line is the data row's number from 1.
func (im *Importer) readRow(row []string, m columnMap, line int) (SubReq, Problem, bool) {
	p := Problem{Row: line + 1}
	if len(row) < m.minCols {
		if m.email < len(row) {
			p.Value = row[m.email]
		}
		p.Reason = ReasonColumns
		return SubReq{}, p, false
	}

	sub := SubReq{}
	sub.Email = row[m.email]
	p.Value = sub.Email
	if m.name >= 0 {
		sub.Name = row[m.name]
	}

	if strings.TrimSpace(sub.Email) == "" {
		p.Reason = ReasonNoEmail
		return sub, p, false
	}

	sub, err := im.ValidateFields(sub)
	if err != nil {
		p.Reason = ReasonInvalidEmail
		if errors.Is(err, errBlockedDomain) {
			p.Reason = ReasonBlockedDomain
		}
		return sub, p, false
	}

	// Attributes are the subscriber's own open-ended data, so a map.
	attribs := models.JSON{}
	if m.attribs >= 0 && strings.TrimSpace(row[m.attribs]) != "" {
		// A bad JSON cell doesn't keep the subscriber out; its attributes
		// are just left empty, as imports always have.
		_ = json.Unmarshal([]byte(row[m.attribs]), &attribs)
	}
	for _, c := range m.attrs {
		if v := strings.TrimSpace(row[c.Index]); v != "" {
			attribs[c.Key] = v
		}
	}
	if len(attribs) > 0 {
		sub.Attribs = attribs
	}

	return sub, p, true
}

// maxExtractSize caps the CSV taken out of a ZIP, so a small archive
// can't fill the disk.
const maxExtractSize = 1 << 30

// ErrNoCSVInZIP is returned when a ZIP holds no .csv file.
var ErrNoCSVInZIP = errors.New("no CSV files found in the ZIP")

// ExtractFirstCSV copies the first .csv file in the ZIP at zipPath into a
// new temporary directory, and returns its path and the directory (for the
// caller to remove).
func ExtractFirstCSV(zipPath string) (string, string, error) {
	z, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", "", err
	}
	defer z.Close()

	for _, f := range z.File {
		if f.FileInfo().IsDir() || !strings.HasSuffix(strings.ToLower(f.Name), ".csv") {
			continue
		}

		dir, err := os.MkdirTemp("", "listnun")
		if err != nil {
			return "", "", err
		}
		// Base drops any directories in the name (ZIP slip).
		path := filepath.Join(dir, filepath.Base(f.Name))
		if err := copyZipFile(f, path); err != nil {
			os.RemoveAll(dir)
			return "", "", err
		}
		return path, dir, nil
	}
	return "", "", ErrNoCSVInZIP
}

func copyZipFile(f *zip.File, path string) error {
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	n, err := io.Copy(out, io.LimitReader(src, maxExtractSize+1))
	if err != nil {
		return err
	}
	if n > maxExtractSize {
		return errors.New("the CSV in the ZIP is too large")
	}
	return out.Close()
}
