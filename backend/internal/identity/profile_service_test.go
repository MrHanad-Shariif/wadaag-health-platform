package identity

import (
	"reflect"
	"testing"
)

// strPtr is a small test helper since Go has no address-of-literal syntax.
func strPtr(s string) *string { return &s }

// TestMergeProfileUpdate covers the "upsert-if-missing" partial-update
// logic: fields omitted from the PATCH body (nil) must fall back to
// whatever is already on the row, not be wiped to empty — including when
// there's no existing row at all (the zero-value UserProfile{} case
// Service.UpdateProfile passes in when GetUserProfile returns
// ErrProfileNotFound).
func TestMergeProfileUpdate(t *testing.T) {
	existing := UserProfile{
		Bio:                strPtr("old bio"),
		Languages:          []string{"en", "so"},
		AvailabilityStatus: strPtr("available"),
	}

	tests := []struct {
		name             string
		current          UserProfile
		in               UpdateProfileInput
		wantBio          *string
		wantLanguages    []string
		wantAvailability *string
	}{
		{
			name:             "empty input leaves every existing field unchanged",
			current:          existing,
			in:               UpdateProfileInput{},
			wantBio:          strPtr("old bio"),
			wantLanguages:    []string{"en", "so"},
			wantAvailability: strPtr("available"),
		},
		{
			name:             "supplying only bio leaves languages and availability untouched",
			current:          existing,
			in:               UpdateProfileInput{Bio: strPtr("new bio")},
			wantBio:          strPtr("new bio"),
			wantLanguages:    []string{"en", "so"},
			wantAvailability: strPtr("available"),
		},
		{
			name:             "supplying only languages leaves bio and availability untouched",
			current:          existing,
			in:               UpdateProfileInput{Languages: &[]string{"ar"}},
			wantBio:          strPtr("old bio"),
			wantLanguages:    []string{"ar"},
			wantAvailability: strPtr("available"),
		},
		{
			name:             "all fields supplied overwrite all fields",
			current:          existing,
			in:               UpdateProfileInput{Bio: strPtr("new"), Languages: &[]string{"fr"}, AvailabilityStatus: strPtr("busy")},
			wantBio:          strPtr("new"),
			wantLanguages:    []string{"fr"},
			wantAvailability: strPtr("busy"),
		},
		{
			name:             "no existing row (zero-value current) with a partial update only sets the supplied field",
			current:          UserProfile{},
			in:               UpdateProfileInput{AvailabilityStatus: strPtr("available")},
			wantBio:          nil,
			wantLanguages:    nil,
			wantAvailability: strPtr("available"),
		},
		{
			name:             "explicit empty-slice languages is a real update, not treated as absent",
			current:          existing,
			in:               UpdateProfileInput{Languages: &[]string{}},
			wantBio:          strPtr("old bio"),
			wantLanguages:    []string{},
			wantAvailability: strPtr("available"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bio, languages, availability := mergeProfileUpdate(tt.current, tt.in)

			if !reflect.DeepEqual(bio, tt.wantBio) {
				t.Errorf("bio = %v, want %v", derefOrNil(bio), derefOrNil(tt.wantBio))
			}
			if !reflect.DeepEqual(languages, tt.wantLanguages) {
				t.Errorf("languages = %v, want %v", languages, tt.wantLanguages)
			}
			if !reflect.DeepEqual(availability, tt.wantAvailability) {
				t.Errorf("availabilityStatus = %v, want %v", derefOrNil(availability), derefOrNil(tt.wantAvailability))
			}
		})
	}
}

func derefOrNil(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// TestDetectImageExt covers the magic-byte sniffing UploadProfilePhoto
// relies on instead of trusting a client-declared Content-Type: it must
// accept genuine JPEG/PNG/WebP signatures and reject anything else,
// including short/empty input.
func TestDetectImageExt(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantExt string
		wantOK  bool
	}{
		{
			name:    "jpeg magic bytes",
			data:    []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
			wantExt: ".jpg",
			wantOK:  true,
		},
		{
			name:    "png magic bytes",
			data:    []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00},
			wantExt: ".png",
			wantOK:  true,
		},
		{
			name:    "webp magic bytes (RIFF....WEBP)",
			data:    []byte("RIFF\x00\x00\x00\x00WEBP"),
			wantExt: ".webp",
			wantOK:  true,
		},
		{
			name:   "plain text is rejected",
			data:   []byte("not an image, just text padded out"),
			wantOK: false,
		},
		{
			name:   "empty input is rejected",
			data:   []byte{},
			wantOK: false,
		},
		{
			name:   "riff container that isn't webp is rejected",
			data:   []byte("RIFF\x00\x00\x00\x00WAVE"),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, ok := detectImageExt(tt.data)
			if ok != tt.wantOK {
				t.Fatalf("detectImageExt() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && ext != tt.wantExt {
				t.Errorf("detectImageExt() ext = %q, want %q", ext, tt.wantExt)
			}
		})
	}
}
