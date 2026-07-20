package services

import (
	"mime/multipart"
	"net/textproto"
	"testing"
)

// makeDocHeader builds a minimal *multipart.FileHeader for ValidateDocumentFile tests.
func makeDocHeader(filename, ct string, size int64) *multipart.FileHeader {
	h := make(textproto.MIMEHeader)
	if ct != "" {
		h.Set("Content-Type", ct)
	}
	return &multipart.FileHeader{
		Filename: filename,
		Header:   h,
		Size:     size,
	}
}

const smallFile = int64(1 << 20) // 1 MB — well within the 25 MB limit

func TestValidateDocumentFile_DWG(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		ct       string
		size     int64
		wantErr  bool
	}{
		// ── DWG accepted ──────────────────────────────────────────────────────────
		{
			name:     "dwg lowercase + application/dwg",
			filename: "plan.dwg", ct: "application/dwg", size: smallFile,
			wantErr: false,
		},
		{
			name:     "DWG uppercase + application/dwg",
			filename: "plan.DWG", ct: "application/dwg", size: smallFile,
			wantErr: false,
		},
		{
			name:     "Dwg mixed-case + application/dwg",
			filename: "plan.Dwg", ct: "application/dwg", size: smallFile,
			wantErr: false,
		},
		{
			name:     "dwg + application/octet-stream (browser generic)",
			filename: "plan.dwg", ct: "application/octet-stream", size: smallFile,
			wantErr: false,
		},
		{
			name:     "dwg + application/acad",
			filename: "plan.dwg", ct: "application/acad", size: smallFile,
			wantErr: false,
		},
		{
			name:     "dwg + application/x-acad",
			filename: "plan.dwg", ct: "application/x-acad", size: smallFile,
			wantErr: false,
		},
		{
			name:     "dwg + application/autocad",
			filename: "plan.dwg", ct: "application/autocad", size: smallFile,
			wantErr: false,
		},
		{
			name:     "dwg + application/x-autocad",
			filename: "plan.dwg", ct: "application/x-autocad", size: smallFile,
			wantErr: false,
		},
		{
			name:     "dwg + application/x-dwg",
			filename: "plan.dwg", ct: "application/x-dwg", size: smallFile,
			wantErr: false,
		},
		{
			name:     "dwg + image/vnd.dwg",
			filename: "plan.dwg", ct: "image/vnd.dwg", size: smallFile,
			wantErr: false,
		},
		// ── DWG rejected ─────────────────────────────────────────────────────────
		{
			// .exe with octet-stream must never be accepted
			name:     "exe + application/octet-stream rejected",
			filename: "virus.exe", ct: "application/octet-stream", size: smallFile,
			wantErr: true,
		},
		{
			// Only the LAST extension matters; double extension is a mismatch
			name:     "fake double extension drawing.dwg.exe rejected",
			filename: "drawing.dwg.exe", ct: "application/octet-stream", size: smallFile,
			wantErr: true,
		},
		{
			// Path traversal prefix must not affect the extension check
			name:     "path traversal ../plan.dwg.exe rejected",
			filename: "../plan.dwg.exe", ct: "application/octet-stream", size: smallFile,
			wantErr: true,
		},
		{
			// .dwg with an unrecognised MIME (not in dwgMIMEValues) is rejected
			name:     "dwg + text/plain rejected",
			filename: "plan.dwg", ct: "text/plain", size: smallFile,
			wantErr: true,
		},
		// ── Oversized ────────────────────────────────────────────────────────────
		{
			name:     "dwg oversized rejected",
			filename: "huge.dwg", ct: "application/dwg", size: MaxDocumentFileSize + 1,
			wantErr: true,
		},
		// ── Existing types still accepted ────────────────────────────────────────
		{
			name:     "pdf still accepted",
			filename: "report.pdf", ct: "application/pdf", size: smallFile,
			wantErr: false,
		},
		{
			name:     "docx still accepted",
			filename: "doc.docx",
			ct:       "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			size:     smallFile, wantErr: false,
		},
		{
			name:     "xlsx still accepted",
			filename: "data.xlsx",
			ct:       "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			size:     smallFile, wantErr: false,
		},
		{
			name:     "png still accepted",
			filename: "photo.png", ct: "image/png", size: smallFile,
			wantErr: false,
		},
		{
			name:     "zip still accepted",
			filename: "archive.zip", ct: "application/zip", size: smallFile,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hdr := makeDocHeader(tt.filename, tt.ct, tt.size)
			err := ValidateDocumentFile(hdr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDocumentFile(%q, %q, %d) error = %v, wantErr = %v",
					tt.filename, tt.ct, tt.size, err, tt.wantErr)
			}
		})
	}
}

// TestValidateDocumentFile_Roles documents that upload authorization is enforced
// by route-level middleware (requireRoles("direktor", "inzenjer")) and is not
// affected by this change. No modification was made to routes or middleware.

func TestNormalizeDocumentMIME(t *testing.T) {
	tests := []struct {
		ct       string
		filename string
		want     string
	}{
		// DWG: any MIME → canonical
		{"application/octet-stream", "plan.dwg", canonicalDWGMIME},
		{"application/acad", "plan.dwg", canonicalDWGMIME},
		{"application/dwg", "plan.DWG", canonicalDWGMIME},
		{"image/vnd.dwg", "plan.Dwg", canonicalDWGMIME},
		{"application/x-autocad", "plan.dwg", canonicalDWGMIME},
		// Non-DWG: unchanged
		{"application/pdf", "report.pdf", "application/pdf"},
		{"image/png", "photo.png", "image/png"},
		// octet-stream for non-DWG must not be normalized
		{"application/octet-stream", "virus.exe", "application/octet-stream"},
		// path traversal doesn't confuse the ext check
		{"application/octet-stream", "../safe.dwg", canonicalDWGMIME},
	}
	for _, tt := range tests {
		got := NormalizeDocumentMIME(tt.ct, tt.filename)
		if got != tt.want {
			t.Errorf("NormalizeDocumentMIME(%q, %q) = %q, want %q",
				tt.ct, tt.filename, got, tt.want)
		}
	}
}
