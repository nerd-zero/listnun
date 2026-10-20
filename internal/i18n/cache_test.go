package i18n

import (
	"errors"
	"testing"
)

func TestCache(t *testing.T) {
	calls := map[string]int{}
	c := NewCache([]string{"en", "de", "pt"}, func(code string) (*I18n, error) {
		calls[code]++
		if code == "pt" {
			return nil, errors.New("broken pack")
		}
		return New([]byte(`{"_.code": "` + code + `", "_.name": "x"}`))
	})

	if _, ok := c.Get("../../etc/passwd"); ok {
		t.Fatal("unknown code was served")
	}
	if len(calls) != 0 {
		t.Fatal("unknown code reached the loader")
	}

	de, ok := c.Get("de")
	if !ok || de.T("_.code") != "de" {
		t.Fatalf("de: ok=%v", ok)
	}
	c.Get("de")
	if calls["de"] != 1 {
		t.Errorf("de loaded %d times, want 1", calls["de"])
	}

	if _, ok := c.Get("pt"); ok {
		t.Error("a pack that failed to load was served")
	}
}
