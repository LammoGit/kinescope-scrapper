//go:build postgres && sqlite

package storage

import "fmt"

// NewStorage rejects the ambiguous build that enables both backends.
func NewStorage() (Storage, error) {
	return nil, fmt.Errorf("storage: postgres and sqlite build tags are mutually exclusive")
}
