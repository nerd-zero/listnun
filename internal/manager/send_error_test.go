package manager

import (
	"errors"
	"io"
	"log"
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
