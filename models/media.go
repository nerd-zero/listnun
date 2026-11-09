package models

// MediaUsage is the storage a workspace's media takes up.
type MediaUsage struct {
	// Bytes is the total size of the files whose size is known.
	Bytes int64 `db:"bytes" json:"bytes"`
	Files int   `db:"files" json:"files"`
	// Unknown files aren't in Bytes: uploaded before sizes were recorded
	// and not measured yet, or their size couldn't be read.
	Unknown int `db:"unknown" json:"unknown"`
} // @name MediaUsage

// UnsizedMedia is a file waiting to have its size measured.
type UnsizedMedia struct {
	ID       int    `db:"id"`
	TenantID int    `db:"tenant_id"`
	Filename string `db:"filename"`
	Provider string `db:"provider"`
}
