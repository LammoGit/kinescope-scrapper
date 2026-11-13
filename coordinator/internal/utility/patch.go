package utility

import (
	"bytes"
	"encoding/json"
)

// ApplyMergePatch merges an RFC 7386 JSON Merge Patch into target
// Fields present in patch overwrite target fields, null deletes,
// nested objects merge recursively, arrays are replaced wholesale
func ApplyMergePatch(target any, patch map[string]any) error {
	existing, err := json.Marshal(target)
	if err != nil {
		return err
	}

	var base map[string]any
	if err := json.Unmarshal(existing, &base); err != nil {
		return err
	}

	mergeInto(base, patch)

	merged, err := json.Marshal(base)
	if err != nil {
		return err
	}

	dec := json.NewDecoder(bytes.NewReader(merged))
	dec.DisallowUnknownFields()
	return dec.Decode(target)
}

// mergeInto applies src onto dst using RFC 7386 semantics
func mergeInto(dst, src map[string]any) {
	for k, v := range src {
		if v == nil {
			delete(dst, k)
			continue
		}
		if sv, ok := v.(map[string]any); ok {
			if dv, ok := dst[k].(map[string]any); ok {
				mergeInto(dv, sv)
				continue
			}
		}
		dst[k] = v
	}
}
