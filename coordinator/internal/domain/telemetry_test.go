package domain_test

import (
	"coordinator/internal/domain"
	"encoding/json"
	"reflect"
	"testing"
)

// TestMetrics tests for the Metrics type
func TestMetrics(t *testing.T) {
	tests := []struct {
		name  string
		input []domain.Metrics
		want  domain.Metrics
	}{
		{
			name:  "no arguments return zero metrics",
			input: nil,
			want:  domain.Metrics{},
		},
		{
			name: "single metric returns exact copy",
			input: []domain.Metrics{
				{
					RangeID: 5, JobID: 10, WorkerID: 20,
					CPULoad: 42.5, FreeSpaceBytes: 1024, OldestInQueue: 30,
				},
			},
			want: domain.Metrics{
				RangeID: 5, JobID: 10, WorkerID: 20,
				CPULoad: 42.5, FreeSpaceBytes: 1024, OldestInQueue: 30,
			},
		},
		{
			name: "averages ratios, sums throughput/counters, takes max OldestInQueue, and keeps matching IDs",
			input: []domain.Metrics{
				{
					RangeID:              1,
					JobID:                10,
					WorkerID:             101,
					CPULoad:              20.0,
					MemoryLoad:           40.0,
					DiskLoad:             10.0,
					NetworkLoad:          30.0,
					FreeSpaceBytes:       1000,
					OpenedFiles:          12,
					ActiveConnections:    4,
					WorkerUptime:         100.0,
					IdentifiersPerSecond: 250.0,
					Requests:             50,
					FoundMaterials:       10,
					ActiveDownloads:      3,
					BytesPerSecond:       5000.0,
					AvgCheckSpeed:        100.0,
					AvgDownloadSpeed:     200.0,
					DownloadQueueSize:    5,
					RepeatedAttempts:     1,
					FinalErrors:          0,
					OldestInQueue:        120,
				},
				{
					RangeID:              2,
					JobID:                10,
					WorkerID:             102,
					CPULoad:              60.0,
					MemoryLoad:           80.0,
					DiskLoad:             50.0,
					NetworkLoad:          70.0,
					FreeSpaceBytes:       3000,
					OpenedFiles:          18,
					ActiveConnections:    6,
					WorkerUptime:         300.0,
					IdentifiersPerSecond: 350.0,
					Requests:             70,
					FoundMaterials:       15,
					ActiveDownloads:      7,
					BytesPerSecond:       7000.0,
					AvgCheckSpeed:        200.0,
					AvgDownloadSpeed:     400.0,
					DownloadQueueSize:    9,
					RepeatedAttempts:     3,
					FinalErrors:          2,
					OldestInQueue:        45,
				},
			},
			want: domain.Metrics{
				RangeID:              0,
				JobID:                10,
				WorkerID:             0,
				CPULoad:              40.0,    // averaged
				MemoryLoad:           60.0,    // averaged
				DiskLoad:             30.0,    // averaged
				NetworkLoad:          50.0,    // averaged
				FreeSpaceBytes:       4000,    // summed
				OpenedFiles:          30,      // summed
				ActiveConnections:    10,      // summed
				WorkerUptime:         200.0,   // averaged
				IdentifiersPerSecond: 600.0,   // summed
				Requests:             120,     // summed
				FoundMaterials:       25,      // summed
				ActiveDownloads:      10,      // summed
				BytesPerSecond:       12000.0, // summed
				AvgCheckSpeed:        150.0,   // averaged
				AvgDownloadSpeed:     300.0,   // averaged
				DownloadQueueSize:    14,      // summed
				RepeatedAttempts:     4,       // summed
				FinalErrors:          2,       // summed
				OldestInQueue:        120,     // max
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.AverageMetrics(tt.input...)
			if got != tt.want {
				t.Fatalf("AverageMetrics() mismatch:\ngot:  %+v\nwant: %+v", got, tt.want)
			}
		})
	}
}

// FuzzMetricsClone fuzzy metrics cloning tests
func FuzzMetricsClone(f *testing.F) {
	f.Add([]byte(`{
		"range_id": 123,
		"job_id": 747,
		"worker_id": 345,
		"cpu_load": 52.6,
		"memory_load": 93.71,
		"disk_load": 63.63,
		"network_load": 49,
		"free_space_bytes": 1024,
		"opened_files": 123,
		"active_connections": 12,
		"worker_uptime": 123573,
		"identifiers_per_second": 583,
		"requests": 152,
		"found_materials": 123,
		"active_downloads": 678,
		"bytes_per_second": 12412,
		"avg_check_speed": 126.4321,
		"avg_download_speed": 1251.12,
		"download_queue_size": 151.12,
		"repeated_attempts": 621,
		"final_errors": 125,
		"oldest_in_queue": 966
	}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var orig domain.Metrics
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
