package manager

import (
	"errors"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/nerd-zero/listnun/models"
)

// sendErrStore is a fake Store that records reported campaign send errors.
type sendErrStore struct {
	Store    // unimplemented methods panic, which the test would catch
	reported []string
}

func (s *sendErrStore) RecordCampaignSendError(c *models.Campaign, err error) {
	s.reported = append(s.reported, c.Name+": "+err.Error())
}

// A failing campaign run reports its first send error once, not once per
// subscriber, and a new run reports again. Test sends report themselves
// (cmd's sendTestMessage), so a message without a run isn't reported here.
func TestReportSendErrorOncePerRun(t *testing.T) {
	st := &sendErrStore{}
	m := &Manager{store: st, log: log.New(io.Discard, "", 0)}
	camp := &models.Campaign{Name: "Tawanda"}
	loginErr := errors.New("535 5.7.8 Error: authentication failed")

	run := &pipe{camp: camp, m: m}
	for range 3 {
		m.reportSendError(CampaignMessage{Campaign: camp, pipe: run}, loginErr)
	}
	m.reportSendError(CampaignMessage{Campaign: camp}, loginErr)
	if len(st.reported) != 1 {
		t.Fatalf("reported %v, want the run's first error only", st.reported)
	}

	m.reportSendError(CampaignMessage{Campaign: camp, pipe: &pipe{camp: camp, m: m}}, loginErr)
	if len(st.reported) != 2 {
		t.Errorf("reported %v, want a new run to report again", st.reported)
	}
}

// A panic while sending (an empty SMTP pool did this) killed the worker
// goroutine and so the process; it's now an ordinary send error.
func TestSendSafelyTurnsPanicIntoError(t *testing.T) {
	err := sendSafely(func() error { panic("boom") })
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("err = %v, want the panic as an error", err)
	}
	if err := sendSafely(func() error { return nil }); err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}
