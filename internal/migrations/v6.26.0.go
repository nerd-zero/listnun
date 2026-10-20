package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_26_0 changes users.list_role_id from ON DELETE CASCADE to ON DELETE
// SET NULL. With CASCADE, deleting a list role deleted every user who held
// it; now those users simply lose their list role. The constraint is
// dropped by name and re-added, so running this twice is harmless.
func V6_26_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
		ALTER TABLE users DROP CONSTRAINT IF EXISTS users_list_role_id_fkey;
		ALTER TABLE users ADD CONSTRAINT users_list_role_id_fkey
			FOREIGN KEY (list_role_id) REFERENCES roles(id) ON DELETE SET NULL;
	`); err != nil {
		return err
	}

	return nil
}
