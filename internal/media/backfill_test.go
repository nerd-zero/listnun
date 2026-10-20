package media

import (
	"errors"
	"io"
	"testing"

	"github.com/nerd-zero/listnun/models"
)

type fakeStore struct{ sizes map[string]int64 }

func (fakeStore) Put(string, string, io.ReadSeeker) (string, error) { return "", nil }
func (fakeStore) Delete(string) error                                { return nil }
func (fakeStore) GetURL(string) string                               { return "" }
func (fakeStore) GetBlob(string) ([]byte, error)                     { return nil, nil }
func (f fakeStore) Size(name string) (int64, error) {
	if v, ok := f.sizes[name]; ok {
		return v, nil
	}
	return 0, errors.New("not found")
}


func TestMeasureSizes(t *testing.T) {
	stores := map[int]Store{
		1: fakeStore{sizes: map[string]int64{"a.png": 1200}},
		2: struct{ Store }{fakeStore{}}, // hides Size: not a Sizer
	}
	storeFor := func(tid int) (Store, string, error) {
		if tid == 3 {
			return nil, "", errors.New("no settings")
		}
		return stores[tid], "filesystem", nil
	}

	rows := []models.UnsizedMedia{
		{ID: 1, TenantID: 1, Filename: "a.png", Provider: "filesystem"},
		{ID: 2, TenantID: 1, Filename: "gone.png", Provider: "filesystem"},
		{ID: 3, TenantID: 1, Filename: "a.png", Provider: "s3"},
		{ID: 4, TenantID: 2, Filename: "b.png", Provider: "filesystem"},
		{ID: 5, TenantID: 3, Filename: "c.png", Provider: "filesystem"},
	}
	got := map[int]int64{}
	n, err := MeasureSizes(rows, storeFor, func(id int, size int64) error { got[id] = size; return nil })
	if err != nil || n != 5 {
		t.Fatalf("n=%d err=%v", n, err)
	}

	want := map[int]int64{1: 1200, 2: SizeUnknown, 3: SizeUnknown, 4: SizeUnknown, 5: SizeUnknown}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("row %d: size %d, want %d", id, got[id], w)
		}
	}
}
