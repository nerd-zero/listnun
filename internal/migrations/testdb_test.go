package migrations

import (
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// testDB connects to the test database (LISTNUN_db__* env, else the
// local dev defaults), or skips the test.
func testDB(t *testing.T) *sqlx.DB {
	t.Helper()
	get := func(env, def string) string {
		if v := os.Getenv(env); v != "" {
			return v
		}
		return def
	}
	db, err := sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		get("LISTNUN_db__host", "127.0.0.1"), get("LISTNUN_db__port", "5435"),
		get("LISTNUN_db__user", "listnun-dev"), get("LISTNUN_db__password", "listnun-dev"),
		get("LISTNUN_db__database", "listnun-dev")))
	if err != nil {
		t.Skipf("skipping: no reachable test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
