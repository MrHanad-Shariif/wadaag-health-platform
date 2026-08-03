package notifications

import "testing"

// TestMarshalPayload covers the pure "map -> JSON bytes" step Publish
// delegates to — in particular that a nil map encodes as "{}" (matching the
// notifications.payload column's NOT NULL DEFAULT '{}'::jsonb) rather than
// the literal string "null", which a bare json.Marshal(nil) would produce.
func TestMarshalPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		want    string
	}{
		{name: "nil map encodes as empty object", payload: nil, want: "{}"},
		{name: "empty map encodes as empty object", payload: map[string]any{}, want: "{}"},
		{
			name:    "populated map encodes its fields",
			payload: map[string]any{"referral_id": "abc", "status": "accepted"},
			want:    `{"referral_id":"abc","status":"accepted"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := marshalPayload(tt.payload)
			if err != nil {
				t.Fatalf("marshalPayload returned error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("marshalPayload(%v) = %q, want %q", tt.payload, string(got), tt.want)
			}
		})
	}
}
