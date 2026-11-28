package domain_test

import (
	"bytes"
	"coordinator/internal/domain"
	"encoding/json"
	"reflect"
	"testing"
)

// FuzzPasswordCredentialsClone fuzzy password credentials cloning tests
func FuzzPasswordCredentialsClone(f *testing.F) {
	f.Add([]byte(`{
		"hash": "4hyuI2346MID28hFLzVJow==",
		"salt": "X86R2BRP0iQAofuMrJlrWw=="
	}`))

	f.Add([]byte(`{"id": 1, "job_id": 1}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var orig domain.PasswordCredentials
		if err := json.Unmarshal(data, &orig); err != nil {
			return // Skip invalid JSON bytes
		}

		clone := orig.Clone()

		// Deep-equality check
		if !reflect.DeepEqual(orig, clone) {
			t.Fatalf("clone mismatch:\norig:  %+v\nclone: %+v", orig, clone)
		}
	})
}

// FuzzUserClone fuzzy user cloning tests
func FuzzUserClone(f *testing.F) {
	f.Add([]byte(`{
		"user_id": 231,
		"username": "username",
		"email": "example@email.com",
		"passowrd": {
			"hash": "73mDqB8DfL6GNMGrpT/Z/w==",
			"salt": "+gMf/i+UboO4wqQZgaLvGw=="
		},
		"user_role": "viewer",
		"avatar": "Pvzw6ZtT+tCRaksZw/lFmw==",
		"created_at": "2026-11-22T20:00:00Z"
	}`))

	f.Add([]byte(`{
		"username": "jane doe",
		"email": "janedoe@gmail.com",
		"passowrd": {
			"hash": "H/iX/xdE7X2mIPszt2PXcQ==",
			"salt": "Duod1ELHIsy8E7ribxi9tw=="
		},
		"user_role": "operator",
		"created_at": "2026-11-22T20:30:00Z"
	}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var orig domain.User
		if err := json.Unmarshal(data, &orig); err != nil {
			return // Skip invalid JSON bytes
		}

		clone := orig.Clone()

		// Deep-equality check
		if !reflect.DeepEqual(orig, clone) {
			t.Fatalf("clone mismatch:\norig:  %+v\nclone: %+v", orig, clone)
		}

		// Verify isolation
		if len(clone.Password.Hash) != 0 {
			before := orig.Password.Hash
			clone.Password.Hash[0] = 1
			if !bytes.Equal(orig.Password.Hash, before) {
				t.Fatalf("Password credentials alias: orig changed from %d to %d", before, orig.Password.Hash)
			}
		}
	})
}
