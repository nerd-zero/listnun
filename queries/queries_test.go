package queries

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/goyesql/v2"
	_ "github.com/lib/pq"
)

// testDSN uses the same LISTNUN_db__* env vars as CI
// (.github/workflows/tests.yml), falling back to the local dev database.
func testDSN() string {
	get := func(env, def string) string {
		if v := os.Getenv(env); v != "" {
			return v
		}
		return def
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		get("LISTNUN_db__host", "127.0.0.1"),
		get("LISTNUN_db__port", "5435"),
		get("LISTNUN_db__user", "listnun-dev"),
		get("LISTNUN_db__password", "listnun-dev"),
		get("LISTNUN_db__database", "listnun-dev"),
	)
}

// templated matches the placeholders the app fills in before preparing
// (%s at boot; %query%, %order% and the like per request).
var templated = regexp.MustCompile(`%(s|[a-z_]+%)`)

// TestQueriesPrepare prepares every query in queries/*.sql against a real
// schema, as boot does. A query Postgres can't prepare is fatal at startup
// (e.g. an untyped $1 in a UNION resolving to TEXT for an integer column),
// so this catches it before a release crash-loops.
func TestQueriesPrepare(t *testing.T) {
	db, err := sqlx.Connect("postgres", testDSN())
	if err != nil {
		t.Skipf("skipping: no reachable test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	files, err := filepath.Glob("*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("no query files found: %v", err)
	}

	for _, file := range files {
		b, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		qMap, err := goyesql.ParseBytes(b)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}

		names := make([]string, 0, len(qMap))
		for name := range qMap {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			q := qMap[name].Query
			t.Run(filepath.Base(file)+"/"+name, func(t *testing.T) {
				// Templated queries are filled in at boot or per request
				// and can't be prepared from their raw text.
				if templated.MatchString(q) {
					t.Skip("templated query")
				}
				stmt, err := db.Preparex(q)
				if err != nil {
					t.Fatalf("prepare: %v", err)
				}
				stmt.Close()
			})
		}
	}
}
