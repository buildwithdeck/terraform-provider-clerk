package provider

import (
	"encoding/json"
	"time"
)

// normalizeJSON re-marshals a JSON string to produce canonical key ordering,
// preventing spurious Terraform diffs caused by key reordering from the API.
func normalizeJSON(s string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return s
	}
	return string(b)
}

// millisToRFC3339 converts a Unix timestamp in milliseconds to an RFC3339 string.
func millisToRFC3339(ms int64) string {
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}
