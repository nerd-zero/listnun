package events

import "testing"

// A subscriber whose queue was full stopped the event reaching every
// subscriber after it.
func TestPublishSkipsAFullSubscriber(t *testing.T) {
	ev := New()
	full, err := ev.Subscribe("full")
	if err != nil {
		t.Fatal(err)
	}
	for len(full) < cap(full) {
		full <- Event{}
	}
	ok, err := ev.Subscribe("ok")
	if err != nil {
		t.Fatal(err)
	}

	if err := ev.Publish(Event{Type: TypeError, Message: "x"}); err == nil {
		t.Error("Publish: want an error for the full subscriber")
	}
	select {
	case e := <-ok:
		if e.Message != "x" {
			t.Errorf("got %+v, want the published event", e)
		}
	default:
		t.Error("the subscriber with room didn't get the event")
	}
}
