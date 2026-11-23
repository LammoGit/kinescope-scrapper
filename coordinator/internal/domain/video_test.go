package domain_test

import (
	"coordinator/internal/domain"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

var (
	videoMetadataJSON = `{
		"title": "Title",
		"source_url": "https://kinescope.io/1",
		"download_url": "https://kinescope.io/2",
		"found_at": "2026-11-22T20:00:00Z",
		"downloaded_at": "2026-11-22T20:15:00Z",
		"file_size": 1024,
		"mime_type": "video/mp4",
		"status": "Uploaded"
	}`
	videoStorageMetadataJSON = `{
		"bucket_name": "Bucket",
		"object_key": "Object Key",
		"checksum": "Gk1KIRvMVHBTotewhwX1nyRLdPX+W4J+zIVxOew7Z2L4OpqKZDjkvSO4dlhuYXsiL00Ygn20OkHyh4kSUcnsfvxO7id52S0cbSIzLV8wyeuhbinCTQ5masWO7Vrc8DWciDZ8be7TkAdwhuhw13PHcVWASC1GYaV/lFN8sZZP5EiQtSQE0aycdPE+D7a446nvBzyp9Y9oOuAyn9YU1rv5rPIIq5e69OLGOybG682RNgmCReGfB+Zlw/NEM5/QIN1GM9t0ujdwpZeKQxZg3v5222uexU2jSyUCtdKmmfgVskVrK4oJYXzT48qBcs/ddGlqLWTw8Frbp24lyBo2VoDAhQ==",
		"server_name": "Server Name",
		"local_path": "/var/lib/storage"
	}`
)

// FuzzVideoMetadataClone fuzzy video metadata cloning tests
func FuzzVideoMetadataClone(f *testing.F) {
	f.Add([]byte(videoMetadataJSON))

	f.Fuzz(func(t *testing.T, data []byte) {
		var orig domain.VideoMetadata
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

// FuzzVideoStorageMetadataClone fuzzy video storage metadata cloning tests
func FuzzVideoStorageMetadataClone(f *testing.F) {
	f.Add([]byte(videoStorageMetadataJSON))

	f.Fuzz(func(t *testing.T, data []byte) {
		var orig domain.VideoStorageMetadata
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

// FuzzVideoClone fuzzy video cloning tests
func FuzzVideoClone(f *testing.F) {
	f.Add(fmt.Appendf([]byte{}, `{
		"id": 153,
		"job_id": 1000,
		"metadata": %s,
		"storage_metadata": %s
	}`, videoMetadataJSON, videoStorageMetadataJSON))

	f.Fuzz(func(t *testing.T, data []byte) {
		var orig domain.Video
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
			before := orig.Metadata.FileSize
			clone.Metadata.FileSize ^= 0x5555
			if orig.Metadata.FileSize != before {
				t.Fatalf("Video metadata alias: orig changed from %d to %d", before, orig.Metadata.FileSize)
			}
		}

		{
			before := orig.StorageMetadata.Checksum
			clone.StorageMetadata.Checksum += "123"
			if orig.StorageMetadata.Checksum != before {
				t.Fatalf("Video storage metadata alias: orig changed from %s to %s", before, orig.StorageMetadata.Checksum)
			}
		}
	})
}
