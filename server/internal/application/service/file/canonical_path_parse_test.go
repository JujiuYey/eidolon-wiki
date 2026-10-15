package file

import (
	"testing"
)

// Canonical storage://<backend-id>/... paths are produced by the resource
// catalog and consumed by backendScopedFileService. Providers reached bare
// (process-global env storage, legacy tenants) must tolerate the same form,
// otherwise storageurl outbound rewrites fail with "invalid ... file path" and
// internal handles leak to IM channels (#3151).
func TestStorageBackendInnerPath(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"canonical minio path", "storage://be-1/minio://weknora/data/a.jpg", "minio://weknora/data/a.jpg"},
		{"bare provider path unchanged", "minio://weknora/data/a.jpg", "minio://weknora/data/a.jpg"},
		{
			"legacy URL unchanged",
			"https://bucket.example.minio.local/a.jpg",
			"https://bucket.example.minio.local/a.jpg",
		},
		{"resource handle unchanged", "resource://abc123", "resource://abc123"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := storageBackendInnerPath(tc.in); got != tc.want {
				t.Fatalf("storageBackendInnerPath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseMinioFilePathAcceptsCanonicalBackendPath(t *testing.T) {
	svc := &minioFileService{bucketName: "weknora"}

	got, err := svc.parseMinioFilePath("storage://be-1/minio://weknora/data/a.jpg")
	if err != nil {
		t.Fatalf("canonical path rejected: %v", err)
	}
	if got != "data/a.jpg" {
		t.Fatalf("object key = %q, want %q", got, "data/a.jpg")
	}

	if _, err := svc.parseMinioFilePath("minio://weknora/data/a.jpg"); err != nil {
		t.Fatalf("bare path rejected: %v", err)
	}
}
