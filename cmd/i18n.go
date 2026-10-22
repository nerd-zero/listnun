package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"

	"github.com/knadh/stuffbin"
	"github.com/labstack/echo/v4"
	"github.com/nerd-zero/listnun/internal/i18n"
)

type i18nLang struct {
	Code string `json:"code"`
	Name string `json:"name"`
} // @name I18nLang

type i18nLangRaw struct {
	Code string `json:"_.code"`
	Name string `json:"_.name"`
}

var reLangCode = regexp.MustCompile(`[^a-zA-Z_0-9\\-]`)

// GetI18nLang returns the JSON language pack given the language code.
//
//	@ID				getI18nLang
//	@Summary		Get i18n language pack
//	@Tags			misc
//	@Produce		json
//	@Param			lang	path		string	true	"Language code (e.g. en)"
//	@Success		200		{object}	object
//	@Failure		400		{object}	echo.HTTPError
//	@Router			/api/lang/{lang} [get]
func (a *App) GetI18nLang(c echo.Context) error {
	lang := c.Param("lang")
	if len(lang) > 6 || reLangCode.MatchString(lang) {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid language code.")
	}

	i, ok, err := getI18nLang(lang, a.fs)
	if err != nil && !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Unknown language.")
	}

	return c.JSON(http.StatusOK, okResp{json.RawMessage(i.JSON())})
}

// getI18nLangList returns the list of available i18n languages.
func getI18nLangList(fs stuffbin.FileSystem) ([]i18nLang, error) {
	list, err := fs.Glob("/i18n/*.json")
	if err != nil {
		return nil, err
	}

	// Read language JSON files from the fs.
	var out []i18nLang
	for _, l := range list {
		b, err := fs.Get(l)
		if err != nil {
			return out, fmt.Errorf("error reading lang file: %s: %v", l, err)
		}

		var r i18nLangRaw
		if err := json.Unmarshal(b.ReadBytes(), &r); err != nil {
			return out, fmt.Errorf("error parsing lang file: %s: %v", l, err)
		}

		out = append(out, i18nLang(r))
	}

	// Sort by language code.
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Code < out[j].Code
	})

	return out, nil
}

// The bool indicates whether the specified language could be loaded. If it couldn't
// be, the app shouldn't halt but throw a warning.
func getI18nLang(lang string, fs stuffbin.FileSystem) (*i18n.I18n, bool, error) {
	const def = "en"

	b, err := fs.Read(fmt.Sprintf("/i18n/%s.json", def))
	if err != nil {
		return nil, false, fmt.Errorf("error reading default i18n language file: %s: %v", def, err)
	}

	// Initialize with the default language.
	i, err := i18n.New(b)
	if err != nil {
		return nil, false, fmt.Errorf("error unmarshalling i18n language: %s: %v", lang, err)
	}

	// Load the selected language on top of it.
	b, err = fs.Read(fmt.Sprintf("/i18n/%s.json", lang))
	if err != nil {
		return i, true, fmt.Errorf("error reading i18n language file: %s: %v", lang, err)
	}
	if err := i.Load(b); err != nil {
		return i, true, fmt.Errorf("error loading i18n language file: %s: %v", lang, err)
	}

	return i, true, nil
}

// initLangCache lists the bundled languages and returns a cache that loads
// each one (on top of English) the first time it is asked for.
func initLangCache(fs stuffbin.FileSystem) ([]i18nLang, *i18n.Cache) {
	list, err := getI18nLangList(fs)
	if err != nil {
		lo.Printf("error listing languages; the sign-in language picker is off: %v", err)
	}

	codes := make([]string, len(list))
	for n, l := range list {
		codes[n] = l.Code
	}

	return list, i18n.NewCache(codes, func(code string) (*i18n.I18n, error) {
		i, ok, err := getI18nLang(code, fs)
		if !ok {
			return nil, err
		}
		return i, nil
	})
}

// visitorI18n is the language for the sign-in pages: the one the visitor
// picked (the ln_lang cookie) if it's bundled, else the workspace's.
func (a *App) visitorI18n(c echo.Context) *i18n.I18n {
	if ck, err := c.Cookie(i18n.LangCookie); err == nil && a.langs != nil {
		if l, ok := a.langs.Get(ck.Value); ok {
			return l
		}
	}
	return a.i18n
}
