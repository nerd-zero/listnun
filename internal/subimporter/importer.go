// Package subimporter implements a bulk ZIP/CSV importer of subscribers.
// It implements a simple queue for buffering imports and committing records
// to DB along with ZIP and CSV handling utilities. It is meant to be used as
// a singleton as each Importer instance is stateful, where it keeps track of
// an import in progress. Only one import should happen on a single importer
// instance at a time.
package subimporter

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/gofrs/uuid/v5"
	"github.com/knadh/listmonk/internal/i18n"
	"github.com/knadh/listmonk/internal/utils"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	// commitBatchSize is the number of inserts to commit in a single SQL transaction.
	commitBatchSize = 10000
)

// Various import statuses.
const (
	StatusNone      = "none"
	StatusImporting = "importing"
	StatusStopping  = "stopping"
	StatusFinished  = "finished"
	StatusFailed    = "failed"

	ModeSubscribe = "subscribe"
	ModeBlocklist = "blocklist"
)

// Importer represents the bulk CSV subscriber import system.
type Importer struct {
	opt  Options
	db   *sql.DB
	i18n *i18n.I18n

	domainBlocklist       map[string]struct{}
	hasBlocklistWildcards bool
	hasBlocklist          bool

	domainAllowlist       map[string]struct{}
	hasAllowlistWildcards bool
	hasAllowlist          bool

	titler cases.Caser
	stop   chan bool
	status Status
	sync.RWMutex
}

// Options represents import options.
type Options struct {
	UpsertStmt         *sql.Stmt
	BlocklistStmt      *sql.Stmt
	UpdateListDateStmt *sql.Stmt
	PostCB             func(subject string, data any) error

	DomainBlocklist []string
	DomainAllowlist []string

	TenantID int

	// ScrubSubmitFunc, if set, is called once per commit-batch window
	// (see flushWindow) with that session's target list IDs and the
	// window's emails, submitting them to Scrub for asynchronous
	// validation -- baked in at Importer construction time
	// (tenantImporters.Get, cmd/tenant_importer.go) from tenant settings,
	// same "requires a process restart to pick up a settings change"
	// precedent DomainBlocklist/DomainAllowlist above already have in
	// this per-tenant-cached importer. listIDs travels through here
	// (rather than being baked into the closure) because it's a
	// per-session value (SessionOpt.ListIDs), not known at Importer
	// construction time, but is needed by the eventual callback to know
	// which lists to unsubscribe invalid emails from / pause campaigns
	// on -- see cmd/scrub_batch.go's scrub_validation_batches.list_ids.
	// Results aren't known synchronously: Scrub reports back later via
	// the /webhooks/scrub/batch callback, which sets scrub_status and
	// handles risky/invalid rows directly -- this package only fires the
	// submission, it never sees the outcome. Never called for
	// ModeBlocklist rows (flushWindow gates on Mode), since those rows
	// are never sent to and validating them burns quota for no benefit.
	ScrubSubmitFunc func(listIDs []int, emails []string) error
}

// Session represents a single import session.
type Session struct {
	im       *Importer
	subQueue chan SubReq
	log      *log.Logger

	opt SessionOpt
}

// SessionOpt represents the options for an importer session.
type SessionOpt struct {
	Filename           string `json:"filename"`
	Mode               string `json:"mode"`
	SubStatus          string `json:"subscription_status"`
	Overwrite          bool   `json:"overwrite"`
	OverwriteUserInfo  bool   `json:"overwrite_userinfo"`
	OverwriteSubStatus bool   `json:"overwrite_subscription_status"`
	Delim              string `json:"delim"`
	ListIDs            []int  `json:"lists"`

	// TenantID is set by the HTTP handler from the resolved request tenant
	// (never from client-supplied JSON, hence no json tag) after Bind, same
	// as models.Message.TenantID in cmd/tx.go - imported subscribers must
	// carry the importing tenant's ID since UpsertStmt/BlocklistStmt insert
	// directly via database/sql, bypassing Core.WithTenant entirely.
	TenantID int `json:"-"`
}

// Status represents statistics from an ongoing import session.
type Status struct {
	Name     string `json:"name"`
	Total    int    `json:"total"`
	Imported int    `json:"imported"`
	// Risky always reads 0 during the import itself: Scrub validation is
	// asynchronous (see flushWindow/ScrubSubmitFunc), so results -- risky
	// included -- aren't known until the /webhooks/scrub/batch callback
	// resolves each submitted batch, well after this Status stops being
	// polled. Kept for API/frontend type stability rather than removed.
	Risky  int    `json:"risky"`
	Status string `json:"status"`
	logBuf *bytes.Buffer
} // @name ImportStatus

// SubReq is a wrapper over the Subscriber model.
type SubReq struct {
	models.Subscriber
	Lists          []int    `json:"lists"`
	ListUUIDs      []string `json:"list_uuids"`
	PreconfirmSubs bool     `json:"preconfirm_subscriptions"`
} // @name CreateSubscriberReq

type importStatusTpl struct {
	Name     string
	Status   string
	Imported int
	Total    int
}

var (
	// ErrIsImporting is thrown when an import request is made while an
	// import is already running.
	ErrIsImporting = errors.New("import is already running")

	csvHeaders = map[string]bool{
		"email":      true,
		"name":       true,
		"attributes": true}

	regexCleanStr = regexp.MustCompile("[[:^ascii:]]")
)

// New returns a new instance of Importer.
func New(opt Options, db *sql.DB, i *i18n.I18n) *Importer {
	im := Importer{
		opt:             opt,
		db:              db,
		i18n:            i,
		domainBlocklist: make(map[string]struct{}, len(opt.DomainBlocklist)),
		domainAllowlist: make(map[string]struct{}, len(opt.DomainAllowlist)),
		status:          Status{Status: StatusNone, logBuf: bytes.NewBuffer(nil)},
		stop:            make(chan bool, 1),
		titler:          cases.Title(language.Und),
	}

	// Domain blocklist.
	mp, hasWildcards := makeDomainMap(opt.DomainBlocklist)
	im.domainBlocklist = mp
	im.hasBlocklistWildcards = hasWildcards
	im.hasBlocklist = len(mp) > 0

	// Domain allowlist.
	mp, hasWildcards = makeDomainMap(opt.DomainAllowlist)
	im.domainAllowlist = mp
	im.hasAllowlistWildcards = hasWildcards
	im.hasAllowlist = len(mp) > 0

	return &im
}

// NewSession returns an new instance of Session. It takes the name
// of the uploaded file, but doesn't do anything with it but retains it for stats.
func (im *Importer) NewSession(opt SessionOpt) (*Session, error) {
	if !im.isDone() {
		return nil, errors.New("an import is already running")
	}

	// For API backwards compatibility, if the old 'overwrite'
	// field is set, set both overwrite fields to true.
	if opt.Overwrite {
		opt.OverwriteUserInfo = true
		opt.OverwriteSubStatus = true
	}

	im.Lock()
	im.status = Status{Status: StatusImporting,
		Name:   opt.Filename,
		logBuf: bytes.NewBuffer(nil)}
	im.Unlock()

	s := &Session{
		im:       im,
		log:      log.New(im.status.logBuf, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile),
		subQueue: make(chan SubReq, commitBatchSize),
		opt:      opt,
	}

	s.log.Printf("processing '%s'", opt.Filename)
	return s, nil
}

// GetStats returns the global Stats of the importer.
func (im *Importer) GetStats() Status {
	im.RLock()
	defer im.RUnlock()

	return Status{
		Name:     im.status.Name,
		Status:   im.status.Status,
		Total:    im.status.Total,
		Imported: im.status.Imported,
		Risky:    im.status.Risky,
	}
}

// GetLogs returns the log entries of the last import session.
func (im *Importer) GetLogs() []byte {
	im.RLock()
	defer im.RUnlock()

	if im.status.logBuf == nil {
		return []byte{}
	}

	return im.status.logBuf.Bytes()
}

// setStatus sets the Importer's status.
func (im *Importer) setStatus(status string) {
	im.Lock()
	im.status.Status = status
	im.Unlock()
}

// getStatus get's the Importer's status.
func (im *Importer) getStatus() string {
	im.RLock()
	status := im.status.Status
	im.RUnlock()
	return status
}

// isDone returns true if the importer is working (importing|stopping).
func (im *Importer) isDone() bool {
	im.RLock()
	defer im.RUnlock()
	s := im.status.Status
	return s != StatusImporting && s != StatusStopping
}

// incrementImportCount sets the Importer's "imported" counter.
func (im *Importer) incrementImportCount(n int) {
	im.Lock()
	im.status.Imported += n
	im.Unlock()
}

// sendNotif sends admin notifications for import completions.
func (im *Importer) sendNotif(status string) error {
	var (
		s   = im.GetStats()
		out = importStatusTpl{
			Name:     s.Name,
			Status:   status,
			Imported: s.Imported,
			Total:    s.Total,
		}
		subject = fmt.Sprintf("%s: %s import", im.titler.String(status), s.Name)
	)
	return im.opt.PostCB(subject, out)
}

// Start is a blocking function that selects on a channel queue until all
// subscriber entries in the import session are imported. It should be
// invoked as a goroutine.
func (s *Session) Start() {
	var (
		tx    *sql.Tx
		stmt  *sql.Stmt
		err   error
		total = 0
		cur   = 0
	)

	listIDs := make([]int, len(s.opt.ListIDs))
	copy(listIDs, s.opt.ListIDs)
	listIDsArr := pq.Array(listIDs)

	for sub := range s.subQueue {
		if cur == 0 {
			// New transaction batch.
			tx, err = s.im.db.Begin()
			if err != nil {
				s.log.Printf("error creating DB transaction: %v", err)
				continue
			}

			if s.opt.Mode == ModeSubscribe {
				stmt = tx.Stmt(s.im.opt.UpsertStmt)
			} else {
				stmt = tx.Stmt(s.im.opt.BlocklistStmt)
			}
		}

		uu, err := uuid.NewV4()
		if err != nil {
			s.log.Printf("error generating UUID: %v", err)
			tx.Rollback()
			break
		}

		if s.opt.Mode == ModeSubscribe {
			_, err = stmt.Exec(uu, sub.Email, sub.Name, sub.Attribs, listIDsArr, s.opt.SubStatus, s.opt.OverwriteUserInfo, s.opt.OverwriteSubStatus, s.opt.TenantID)
		} else if s.opt.Mode == ModeBlocklist {
			_, err = stmt.Exec(uu, sub.Email, sub.Name, sub.Attribs, s.opt.TenantID)
		}
		if err != nil {
			s.log.Printf("error executing insert: %v", err)
			tx.Rollback()
			break
		}
		cur++
		total++

		// Batch size is met. Commit.
		if cur%commitBatchSize == 0 {
			if err := tx.Commit(); err != nil {
				tx.Rollback()
				s.log.Printf("error committing to DB: %v", err)
			} else {
				s.im.incrementImportCount(cur)
				s.log.Printf("imported %d", total)
			}

			cur = 0
		}
	}

	// Queue's closed and there's nothing left to commit.
	if cur == 0 {
		s.im.setStatus(StatusFinished)
		s.log.Printf("imported finished")
		if _, err := s.im.opt.UpdateListDateStmt.Exec(listIDsArr); err != nil {
			s.log.Printf("error updating lists date: %v", err)
		}
		s.im.sendNotif(StatusFinished)
		return
	}

	// Queue's closed and there are records left to commit.
	if err := tx.Commit(); err != nil {
		tx.Rollback()
		s.im.setStatus(StatusFailed)
		s.log.Printf("error committing to DB: %v", err)
		s.im.sendNotif(StatusFailed)
		return
	}

	s.im.incrementImportCount(cur)
	s.im.setStatus(StatusFinished)
	s.log.Printf("imported finished")
	if _, err := s.im.opt.UpdateListDateStmt.Exec(listIDsArr); err != nil {
		s.log.Printf("error updating lists date: %v", err)
	}

	s.im.sendNotif(StatusFinished)
}

// Stop stops an active import session.
func (s *Session) Stop() {
	close(s.subQueue)
}

// ExtractZIP takes a ZIP file's path and extracts all .csv files in it to
// a temporary directory, and returns the name of the temp directory and the
// list of extracted .csv files.
func (s *Session) ExtractZIP(srcPath string, maxCSVs int) (string, []string, error) {
	if s.im.isDone() {
		return "", nil, ErrIsImporting
	}

	failed := true
	defer func() {
		if failed {
			s.im.setStatus(StatusFailed)
		}
	}()

	z, err := zip.OpenReader(srcPath)
	if err != nil {
		return "", nil, err
	}
	defer z.Close()

	// Create a temporary directory to extract the files.
	dir, err := os.MkdirTemp("", "listmonk")
	if err != nil {
		s.log.Printf("error creating temporary directory for extracting ZIP: %v", err)
		return "", nil, err
	}

	files := make([]string, 0, len(z.File))
	for _, f := range z.File {
		fName := f.FileInfo().Name()

		// Skip directories.
		if f.FileInfo().IsDir() {
			s.log.Printf("skipping directory '%s'", fName)
			continue
		}

		// Skip files without the .csv extension.
		if !strings.HasSuffix(strings.ToLower(fName), ".csv") {
			s.log.Printf("skipping non .csv file '%s'", fName)
			continue
		}

		// Sanitize the file name to prevent ZIP slip path traversal.
		fName = filepath.Base(fName)

		s.log.Printf("extracting '%s'", fName)
		src, err := f.Open()
		if err != nil {
			s.log.Printf("error opening '%s' from ZIP: '%v'", fName, err)
			return "", nil, err
		}
		defer src.Close()

		out, err := os.OpenFile(dir+"/"+fName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			s.log.Printf("error creating '%s/%s': '%v'", dir, fName, err)
			return "", nil, err
		}
		defer out.Close()

		if _, err := io.Copy(out, src); err != nil {
			s.log.Printf("error extracting to '%s/%s': '%v'", dir, fName, err)
			return "", nil, err
		}
		s.log.Printf("extracted '%s'", fName)

		files = append(files, fName)
		if len(files) > maxCSVs {
			s.log.Printf("won't extract any more files. Maximum is %d", maxCSVs)
			break
		}
	}

	if len(files) == 0 {
		s.log.Println("no CSV files found in the ZIP")
		return "", nil, errors.New("no CSV files found in the ZIP")
	}

	failed = false
	return dir, files, nil
}

// LoadCSV loads a CSV file and validates and imports the subscriber entries in it.
func (s *Session) LoadCSV(srcPath string, delim rune) error {
	if s.im.isDone() {
		return ErrIsImporting
	}

	// Default status is "failed" in case the function
	// returns at one of the many possible errors.
	failed := true
	defer func() {
		if failed {
			s.im.setStatus(StatusFailed)
		}
	}()

	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}

	rd := csv.NewReader(f)
	rd.Comma = delim
	rd.ReuseRecord = true

	// Read the header.
	csvHdr, err := rd.Read()
	if err != nil {
		s.log.Printf("error reading header from '%s': '%v'", srcPath, err)
		return err
	}

	hdrKeys := s.mapCSVHeaders(csvHdr, csvHeaders)
	// email is a required header.
	if _, ok := hdrKeys["email"]; !ok {
		s.log.Printf("'email' column not found in '%s'", srcPath)
		return errors.New("'email' column not found")
	}

	var (
		lnHdr = len(hdrKeys)
		i     = 0

		// window buffers up to commitBatchSize syntactically-valid rows
		// so Scrub can be called once per window (bulk) rather than once
		// per row -- keeps memory bounded and progress reporting
		// (status.Total below) at the same granularity as before, and
		// limits a transient Scrub error's blast radius to one window
		// rather than the whole file. Rows only reach the window after
		// the existing per-row syntax/domain check (ValidateFields)
		// already passed, same as before this change.
		window []windowRow
	)
	for {
		i++

		// Check for the stop signal.
		select {
		case <-s.im.stop:
			s.flushWindow(window)
			failed = false
			close(s.subQueue)
			s.log.Println("stop request received")
			return nil
		default:
		}

		cols, err := rd.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			if err, ok := err.(*csv.ParseError); ok && err.Err == csv.ErrFieldCount {
				s.log.Printf("skipping line %d. %v", i, err)
				continue
			} else {
				s.log.Printf("error reading CSV '%s'", err)
				return err
			}
		}

		if len(cols) < lnHdr {
			s.log.Printf("skipping line %d. column count (%d) does not match minimum header count (%d)", i, len(cols), lnHdr)
			continue
		}

		sub := SubReq{}
		if idx, ok := hdrKeys["email"]; ok {
			sub.Email = cols[idx]
		}
		if idx, ok := hdrKeys["name"]; ok {
			sub.Name = cols[idx]
		}

		sub, err = s.im.ValidateFields(sub)
		if err != nil {
			s.log.Printf("skipping line %d: %v: %v", i, err, cols)
			continue
		}

		// JSON attributes.
		if idx, ok := hdrKeys["attributes"]; ok && len(cols[idx]) > 0 {
			var (
				attribs models.JSON
				b       = []byte(cols[idx])
			)
			if err := json.Unmarshal(b, &attribs); err != nil {
				s.log.Printf("skipping invalid attributes JSON on line %d for '%s': %v", i, sub.Email, err)
			} else {
				sub.Attribs = attribs
			}
		}

		window = append(window, windowRow{sub: sub, line: i})
		if len(window) >= commitBatchSize {
			s.flushWindow(window)
			window = window[:0]
		}

		if i%commitBatchSize == 0 {
			s.im.Lock()
			s.im.status.Total = i
			s.im.Unlock()
		}
	}

	s.flushWindow(window)

	s.im.Lock()
	s.im.status.Total = i
	s.im.Unlock()

	close(s.subQueue)
	failed = false

	return nil
}

// windowRow pairs a syntactically-valid SubReq with its source line
// number, for logging when Scrub subsequently flags/rejects it.
type windowRow struct {
	sub  SubReq
	line int
}

// flushWindow pushes every row in the window onto subQueue for Start() to
// insert, then -- if this session has Scrub validation wired (see
// NewSession) -- asynchronously submits the window's emails to Scrub for
// validation. Unlike the old synchronous ValidateBulk, results are no
// longer known at flush time: Scrub validates out-of-band and reports
// back later via the /webhooks/scrub/batch callback (see
// cmd/scrub_batch.go), which is what actually sets scrub_status and
// unsubscribes/tags rows Scrub flags invalid. So every syntactically-valid
// row reaches the DB immediately with scrub_status left unset (pending),
// rather than invalid_syntax/undeliverable rows being skipped
// pre-insert as they were when validation was synchronous.
func (s *Session) flushWindow(window []windowRow) {
	if len(window) == 0 {
		return
	}

	for _, w := range window {
		s.subQueue <- w.sub
	}

	// Blocklist-mode rows are never sent to, so validating them against
	// Scrub burns quota for no benefit.
	if s.im.opt.ScrubSubmitFunc == nil || s.opt.Mode != ModeSubscribe {
		return
	}

	emails := make([]string, len(window))
	for i, w := range window {
		emails[i] = w.sub.Email
	}
	if err := s.im.opt.ScrubSubmitFunc(s.opt.ListIDs, emails); err != nil {
		s.log.Printf("error submitting batch to scrub, subscribers will remain unchecked: %v", err)
	}
}

// Stop sends a signal to stop the existing import.
func (im *Importer) Stop() {
	if im.getStatus() != StatusImporting {
		im.Lock()
		im.status = Status{Status: StatusNone}
		im.Unlock()

		return
	}

	select {
	case im.stop <- true:
		im.setStatus(StatusStopping)
	default:
	}
}

// SanitizeEmail validates and sanitizes an e-mail string and returns the
// canonical (lowercased, trimmed) address. Domain allowlist/blocklist rules
// are enforced on top of the bare-address validation in utils.SanitizeEmail.
func (im *Importer) SanitizeEmail(email string) (string, error) {
	addr, err := utils.SanitizeEmail(email)
	if err != nil {
		return "", errors.New(im.i18n.T("subscribers.invalidEmail"))
	}

	// Check if the e-mail's domain is blocklisted. The e-mail domain and blocklist config
	// are always lowercase.
	if im.hasAllowlist || im.hasBlocklist {
		d := strings.Split(addr, "@")
		if len(d) != 2 {
			return addr, nil
		}

		domain := d[1]

		// If there's an allowlist, check if the domain is in it. Checking blocklist after that is moot.
		if im.hasAllowlist {
			if !im.checkInList(domain, im.hasAllowlistWildcards, im.domainAllowlist) {
				return "", errors.New(im.i18n.T("subscribers.domainBlocklisted"))
			}
		} else if im.hasBlocklist {
			if im.checkInList(domain, im.hasBlocklistWildcards, im.domainBlocklist) {
				return "", errors.New(im.i18n.T("subscribers.domainBlocklisted"))
			}
		}
	}

	return addr, nil
}

// ValidateFields validates incoming subscriber field values and returns sanitized fields.
func (im *Importer) ValidateFields(s SubReq) (SubReq, error) {
	if len(s.Email) > 1000 {
		return s, errors.New(im.i18n.T("subscribers.invalidEmail"))
	}

	em, err := im.SanitizeEmail(s.Email)
	if err != nil {
		return s, err
	}
	s.Email = strings.ToLower(em)

	// If there's no name, use the name part of the e-mail.
	s.Name = strings.TrimSpace(s.Name)
	if len(s.Name) == 0 {
		name := strings.ToLower(strings.Split(s.Email, "@")[0])

		parts := strings.Fields(strings.ReplaceAll(name, ".", " "))
		for n, p := range parts {
			parts[n] = im.titler.String(p)
		}

		s.Name = strings.Join(parts, " ")
	}

	return s, nil
}

// Check the domain against the given map of domains (block/allowlist).
func (im *Importer) checkInList(domain string, hasWildcards bool, mp map[string]struct{}) bool {
	// Check the domain as-is.
	if _, ok := mp[domain]; ok {
		return true
	}

	// If there are wildcards in the list and the email domain has a subdomain, check that.
	if hasWildcards && strings.Count(domain, ".") > 1 {
		parts := strings.Split(domain, ".")

		// Replace the first part of the subdomain with * and check if that exists in the list.
		// Eg: test.mail.example.com => *.mail.example.com
		parts[0] = "*"
		domain = strings.Join(parts, ".")

		if _, ok := mp[domain]; ok {
			return true
		}
	}

	return false
}

// mapCSVHeaders takes a list of headers obtained from a CSV file, a map of known headers,
// and returns a new map with each of the headers in the known map mapped by the position (0-n)
// in the given CSV list.
func (s *Session) mapCSVHeaders(csvHdrs []string, knownHdrs map[string]bool) map[string]int {
	// Map 0-n column index to the header keys, name: 0, email: 1 etc.
	// This is to allow dynamic ordering of columns in th CSV.
	hdrKeys := make(map[string]int)
	for i, h := range csvHdrs {
		// Clean the string of non-ASCII characters (BOM etc.).
		h := regexCleanStr.ReplaceAllString(strings.TrimSpace(h), "")
		if _, ok := knownHdrs[h]; !ok {
			s.log.Printf("ignoring unknown header '%s'", h)
			continue
		}
		hdrKeys[h] = i
	}

	return hdrKeys
}

func makeDomainMap(domains []string) (map[string]struct{}, bool) {
	var (
		out          = make(map[string]struct{}, len(domains))
		hasWildCards = false
	)
	for _, d := range domains {
		out[d] = struct{}{}

		// Domains with *. as the subdomain prefix, strip that
		// and add the full domain to the blocklist as well.
		// eg: *.example.com => example.com
		if strings.Contains(d, "*.") {
			hasWildCards = true
			out[strings.TrimPrefix(d, "*.")] = struct{}{}
		}
	}

	return out, hasWildCards
}
