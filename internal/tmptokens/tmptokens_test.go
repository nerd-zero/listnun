package tmptokens

import (
	"testing"
	"time"
)

func TestPeekDoesNotConsumeOrCount(t *testing.T) {
	Set("peek-1", time.Hour, "data")
	t.Cleanup(func() { Get("peek-1") })

	// Far more peeks than maxTries: none may count as an attempt.
	for i := 0; i < maxTries*3; i++ {
		if d, ok := Peek("peek-1"); !ok || d != "data" {
			t.Fatalf("peek %d = %v, %v; want the token's data", i, d, ok)
		}
	}
	if _, err := Check("peek-1"); err != nil {
		t.Errorf("Check after peeks: %v; peeks must not use up the token's attempts", err)
	}
	if _, err := Get("peek-1"); err != nil {
		t.Errorf("Get after peeks: %v; peeks must not consume the token", err)
	}
	if _, ok := Peek("peek-1"); ok {
		t.Error("Peek found a token that Get consumed")
	}
}

func TestPeekExpiredOrMissing(t *testing.T) {
	Set("peek-expired", time.Nanosecond, "x")
	time.Sleep(time.Millisecond)
	if _, ok := Peek("peek-expired"); ok {
		t.Error("Peek returned an expired token")
	}
	if _, ok := Peek("never-set"); ok {
		t.Error("Peek returned a token that was never set")
	}
}
