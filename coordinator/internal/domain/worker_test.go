package domain_test

import (
	"coordinator/internal/domain"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

var workerConfigJSON = `{
		"name": "Worker Name",
		"host": "127.0.0.1",
		"ssh_port": 45234,
		"ssh_user": "SSH User",
		"ssh_key": "/keys/worker-153",
		"threads": 100,
		"storage_path": "/var/lib/storage",
		"min_free_space_gb": 100
	}`

// FuzzWorkerConfigClone fuzzy worker config cloning tests
func FuzzWorkerConfigClone(f *testing.F) {
	f.Add([]byte(workerConfigJSON))

	f.Fuzz(func(t *testing.T, data []byte) {
		var orig domain.WorkerConfig
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

// FuzzWorkerClone fuzzy worker cloning tests
func FuzzWorkerClone(f *testing.F) {
	f.Add(fmt.Appendf([]byte{}, `{
		"id": 153,
		"config": %s,
		"status": "idle",
		"public_key": "AjVG2rNy/KYH6vcq6Uuekw=="
	}`, workerConfigJSON))

	f.Fuzz(func(t *testing.T, data []byte) {
		var orig domain.Worker
		if err := json.Unmarshal(data, &orig); err != nil {
			return // Skip invalid JSON bytes
		}

		clone := orig.Clone()

		// Deep-equality check
		if !reflect.DeepEqual(orig, clone) {
			t.Fatalf("clone mismatch:\norig:  %+v\nclone: %+v", orig, clone)
		}

		// Verify isolation
		{
			before := orig.Config.MinFreeSpaceGB
			clone.Config.MinFreeSpaceGB ^= 0x5555
			if orig.Config.MinFreeSpaceGB != before {
				t.Fatalf("Worker config alias: orig changed from %d to %d", before, orig.Config.MinFreeSpaceGB)
			}
		}
	})
}
