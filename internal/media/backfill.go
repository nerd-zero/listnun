package media

import "github.com/nerd-zero/listnun/models"

// StoreFor resolves a tenant's current media store and its provider name.
type StoreFor func(tenantID int) (Store, string, error)

// MeasureSizes records the size of each file in rows. A file gets
// SizeUnknown when its tenant now uses a different provider than it was
// uploaded to, when the store can't report sizes, or when the lookup
// fails, so it isn't picked up again. It returns how many rows it updated.
func MeasureSizes(rows []models.UnsizedMedia, storeFor StoreFor, set func(id int, size int64) error) (int, error) {
	n := 0
	for _, r := range rows {
		size := int64(SizeUnknown)
		if st, provider, err := storeFor(r.TenantID); err == nil && provider == r.Provider {
			if sz, ok := st.(Sizer); ok {
				if v, err := sz.Size(r.Filename); err == nil && v >= 0 {
					size = v
				}
			}
		}
		if err := set(r.ID, size); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
