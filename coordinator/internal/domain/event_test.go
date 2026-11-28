package domain_test

import (
	"coordinator/internal/domain"
	"encoding/json"
	"reflect"
	"testing"
)

// FuzzEventClone fuzzy event cloning tests
func FuzzEventClone(f *testing.F) {
	f.Add([]byte(`{
		"time": "2026-11-22T20:00:00Z",
		"level": "Info",
		"component": "component",
		"job_id": 123,
		"worker_id": 934,
		"range_id": 592,
		"resource_id": 1256,
		"opperation_type": "operation",
		"result": "result",
		"error_code": 200,
		"message": "message"
	}`))

	f.Add([]byte(`{
		"id": 1,
		"level": ""
		"operation": "",
		"result": "",
		"error_code": 0,
		"message": ""
	}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var orig domain.Event
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
