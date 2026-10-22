package i18n

import "sync"

// LangCookie holds a visitor's chosen language code. It is set by the
// language picker on the sign-in pages and read by those pages and by the
// admin app, so the choice is per browser rather than per workspace.
const LangCookie = "ln_lang"

// Loader loads the language pack for a code.
type Loader func(code string) (*I18n, error)

// Cache loads language packs on first use and keeps them. It only serves
// codes it was given, so a cookie can't make it read arbitrary files.
type Cache struct {
	mu    sync.Mutex
	load  Loader
	codes map[string]struct{}
	packs map[string]*I18n
}

// NewCache returns a Cache for the given language codes.
func NewCache(codes []string, load Loader) *Cache {
	c := &Cache{
		load:  load,
		codes: make(map[string]struct{}, len(codes)),
		packs: make(map[string]*I18n, len(codes)),
	}
	for _, code := range codes {
		c.codes[code] = struct{}{}
	}
	return c
}

// Get returns the pack for code, loading it once. ok is false for unknown
// codes or when the pack can't be loaded.
func (c *Cache) Get(code string) (*I18n, bool) {
	if _, known := c.codes[code]; !known {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if p, ok := c.packs[code]; ok {
		return p, true
	}
	p, err := c.load(code)
	if err != nil || p == nil {
		return nil, false
	}
	c.packs[code] = p
	return p, true
}
