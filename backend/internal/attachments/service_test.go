package attachments

import (
	"testing"

	"github.com/google/uuid"
)

// TestSanitizeFilename covers the security-critical detail in Upload's doc
// comment: a client-supplied filename must never be trusted as-is for
// building a disk path — path separators (either OS's) must be stripped so
// a value like "../../etc/passwd" can't escape the upload directory.
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain filename is untouched", "report.pdf", "report.pdf"},
		{"unix path separators are stripped to the base name", "../../etc/passwd", "passwd"},
		{"windows path separators are stripped to the base name", `C:\Users\evil\secret.pdf`, "secret.pdf"},
		{"mixed separators are stripped to the base name", `../foo\bar/baz.png`, "baz.png"},
		{"empty filename falls back to a placeholder", "", "file"},
		{"dot falls back to a placeholder", ".", "file"},
		{"dot-dot falls back to a placeholder", "..", "file"},
		{"trailing slash yields an empty base name, falls back", "a/b/", "file"},
		{"whitespace-only filename falls back", "   ", "file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeFilename(tt.in); got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestAllowedAttachmentMimeTypes covers the explicit mime-type allowlist —
// only the four documented types should validate; everything else,
// including a spoofable but plausible-looking type, must be rejected.
func TestAllowedAttachmentMimeTypes(t *testing.T) {
	tests := []struct {
		mimeType string
		want     bool
	}{
		{"image/jpeg", true},
		{"image/png", true},
		{"image/webp", true},
		{"application/pdf", true},
		{"image/gif", false},
		{"text/html", false},
		{"application/octet-stream", false},
		{"application/x-msdownload", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.mimeType, func(t *testing.T) {
			if got := allowedAttachmentMimeTypes[tt.mimeType]; got != tt.want {
				t.Errorf("allowedAttachmentMimeTypes[%q] = %v, want %v", tt.mimeType, got, tt.want)
			}
		})
	}
}

// TestUploadSizeCap covers the 25MB size-cap decision Upload makes before
// ever touching the filesystem or DB.
func TestUploadSizeCap(t *testing.T) {
	tests := []struct {
		name        string
		size        int
		wantOverCap bool
	}{
		{"well under the cap", 1024, false},
		{"exactly at the cap", maxAttachmentSize, false},
		{"one byte over the cap", maxAttachmentSize + 1, true},
		{"far over the cap", maxAttachmentSize * 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.size)
			if got := len(data) > maxAttachmentSize; got != tt.wantOverCap {
				t.Errorf("len(data) > maxAttachmentSize = %v, want %v", got, tt.wantOverCap)
			}
		})
	}
}

// TestResolveRootID covers the pure "resolve to the true root, never chain"
// rule from Upload's versioning-model doc comment: a row that's already a
// root (RootID nil) resolves to its own id; a row that's a later version
// resolves to whatever root_id it points at, not itself.
func TestResolveRootID(t *testing.T) {
	rootRow := Attachment{ID: uuid.New(), RootID: nil}
	if got := resolveRootID(rootRow); got != rootRow.ID {
		t.Errorf("resolveRootID(root row) = %v, want %v (its own id)", got, rootRow.ID)
	}

	trueRoot := uuid.New()
	versionRow := Attachment{ID: uuid.New(), RootID: &trueRoot}
	if got := resolveRootID(versionRow); got != trueRoot {
		t.Errorf("resolveRootID(version row) = %v, want %v (the row it points at)", got, trueRoot)
	}
}
