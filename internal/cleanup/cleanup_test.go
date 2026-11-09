package cleanup

import (
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	ok := Rules{Days: 180, Activity: ActivityEither, Action: ActionUnsubscribe, Schedule: "0 2 * * 0"}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid rules rejected: %v", err)
	}
	for name, r := range map[string]Rules{
		"too short":   {Days: 7, Activity: ActivityEither, Action: ActionUnsubscribe, Schedule: "0 2 * * 0"},
		"activity":    {Days: 180, Activity: "visited", Action: ActionUnsubscribe, Schedule: "0 2 * * 0"},
		"action":      {Days: 180, Activity: ActivityOpened, Action: "archive", Schedule: "0 2 * * 0"},
		"bad cron":    {Days: 180, Activity: ActivityOpened, Action: ActionDelete, Schedule: "every sunday"},
		"empty rules": {},
	} {
		if r.Validate() == nil {
			t.Errorf("%s: invalid rules accepted", name)
		}
	}
}

func TestEffective(t *testing.T) {
	custom := Rules{Days: 90, Activity: ActivityClicked, Action: ActionBlocklist, Schedule: "0 3 * * 1"}
	if got := (Settings{Mode: ModeCustom, Rules: custom}).Effective(DefaultManaged); got.Days != 90 {
		t.Errorf("custom mode used %+v", got)
	}
	if got := (Settings{Mode: ModeManaged, Rules: custom}).Effective(DefaultManaged); got.Days != 180 {
		t.Errorf("managed mode used %+v", got)
	}
}

func TestDue(t *testing.T) {
	// Sundays at 02:00; 26 Oct 2026 is a Monday.
	sunday2am := time.Date(2026, 10, 25, 2, 0, 0, 0, time.UTC)

	due, next, err := Due("0 2 * * 0", sunday2am.Add(10*time.Minute), time.Time{})
	if err != nil || !due {
		t.Fatalf("10 min after the tick with no previous run: due=%v err=%v", due, err)
	}
	if want := sunday2am.Add(7 * 24 * time.Hour); !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}

	if due, _, _ := Due("0 2 * * 0", sunday2am.Add(10*time.Minute), sunday2am.Add(time.Minute)); due {
		t.Error("ran again after this tick's run")
	}
	if due, _, _ := Due("0 2 * * 0", sunday2am.Add(2*time.Hour), time.Time{}); due {
		t.Error("ran a tick missed by more than the run window")
	}
	if due, _, _ := Due("0 2 * * 0", sunday2am.Add(-time.Minute), time.Time{}); due {
		t.Error("ran before the scheduled time")
	}
}
