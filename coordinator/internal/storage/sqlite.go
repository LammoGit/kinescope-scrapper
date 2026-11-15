//go:build !postgres && sqlite

package storage

import "fmt"

// NewStorage is not implemented yet. Build with `-tags postgres` and set
// DATABASE_URL to use a durable storage.
func NewStorage() (Storage, error) {
	return nil, fmt.Errorf("storage: sqlite backend is not implemented; build with -tags postgres")
}
