package main

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// updateCheckURL is the GitHub API endpoint for this fork's latest published
// release. /releases/latest skips drafts and pre-releases, so only real
// (prod) CalVer releases are ever announced.
const updateCheckURL = "https://api.github.com/repos/nerd-zero/listmonk/releases/latest"

type AppUpdate struct {
	Update struct {
		ReleaseVersion string `json:"release_version"`
		ReleaseDate    string `json:"release_date"`
		URL            string `json:"url"`
		Description    string `json:"description"`

		// This is computed and set locally based on the local version.
		IsNew bool `json:"is_new"`
	} `json:"update"`
	Messages []struct {
		Date        string `json:"date"`
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Priority    string `json:"priority"`
	} `json:"messages"`
} // @name AppUpdate

// ghRelease is the subset of the GitHub release payload that's used.
type ghRelease struct {
	TagName     string `json:"tag_name"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
}

// reCalVer matches the fork's CalVer release tags (YYYY.MM.NNN, with an
// optional -suffix such as -alpha), as produced by .github/workflows/build.yml.
var reCalVer = regexp.MustCompile(`^v?(\d{4})\.(\d{2})\.(\d+)(?:-.*)?$`)

// parseCalVer returns the numeric parts of a CalVer tag, or false if the tag
// isn't CalVer.
func parseCalVer(v string) ([3]int, bool) {
	var out [3]int
	m := reCalVer.FindStringSubmatch(v)
	if m == nil {
		return out, false
	}
	for i := range out {
		n, _ := strconv.Atoi(m[i+1])
		out[i] = n
	}
	return out, true
}

// isNewerCalVer reports whether remote is a newer CalVer release than cur.
func isNewerCalVer(remote, cur string) bool {
	r, ok := parseCalVer(remote)
	if !ok {
		return false
	}
	c, ok := parseCalVer(cur)
	if !ok {
		return false
	}
	for i := range r {
		if r[i] != c[i] {
			return r[i] > c[i]
		}
	}
	return false
}

// checkUpdates is a blocking function that checks for updates to the app
// at the given intervals. On detecting a newer release, it sets the global
// update status that renders a prompt on the UI.
func (a *App) checkUpdates(curVersion string, interval time.Duration) {
	fnCheck := func() {
		req, err := http.NewRequest(http.MethodGet, updateCheckURL, nil)
		if err != nil {
			a.log.Printf("error creating remote update request: %v", err)
			return
		}
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			a.log.Printf("error checking for remote update: %v", err)
			return
		}
		defer resp.Body.Close()

		// 404 means no release has been published yet.
		if resp.StatusCode == http.StatusNotFound {
			return
		}
		if resp.StatusCode != http.StatusOK {
			a.log.Printf("non 200 response on remote update check: %d", resp.StatusCode)
			return
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			a.log.Printf("error reading remote update payload: %v", err)
			return
		}

		var rel ghRelease
		if err := json.Unmarshal(b, &rel); err != nil {
			a.log.Printf("error unmarshalling remote update payload: %v", err)
			return
		}

		var out AppUpdate
		out.Update.ReleaseVersion = rel.TagName
		out.Update.ReleaseDate = rel.PublishedAt
		out.Update.URL = rel.HTMLURL
		out.Update.Description = rel.Body

		// There is an update. Set it on the global app state.
		if isNewerCalVer(rel.TagName, curVersion) {
			out.Update.IsNew = true
			a.log.Printf("new update %s found", rel.TagName)
		}

		a.Lock()
		a.update = &out
		a.Unlock()
	}

	// Give a 15 minute buffer after app start in case the admin wants to disable
	// update checks entirely and not make a request to upstream.
	time.Sleep(time.Minute * 15)
	fnCheck()

	// Thereafter, check every $interval.
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		fnCheck()
	}
}
