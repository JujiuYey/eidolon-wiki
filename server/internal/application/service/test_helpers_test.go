package service

// Shared test doubles for the resource-catalog and file-service surfaces.
// They used to live in artifact_collector_test.go; that file went with the
// sandbox, but the knowledge binding/release tests still stub the same
// interfaces, so the doubles live here now.

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// stagingFileService serves fixed bytes by reference and counts reads; the
// attachment-staging tests use it to round-trip files without a backend.
type stagingFileService struct {
	files    map[string][]byte
	getCalls map[string]int
}

func (f *stagingFileService) CheckConnectivity(context.Context) error { return nil }
func (f *stagingFileService) SaveFile(context.Context, *multipart.FileHeader, uint64, string) (string, error) {
	panic("unexpected SaveFile")
}
func (f *stagingFileService) SaveBytes(context.Context, []byte, uint64, string, bool) (string, error) {
	panic("unexpected SaveBytes")
}
func (f *stagingFileService) GetFile(_ context.Context, filePath string) (io.ReadCloser, error) {
	if f.getCalls == nil {
		f.getCalls = make(map[string]int)
	}
	f.getCalls[filePath]++
	return io.NopCloser(bytes.NewReader(f.files[filePath])), nil
}
func (f *stagingFileService) GetFileURL(context.Context, string) (string, error) {
	panic("unexpected GetFileURL")
}
func (f *stagingFileService) DeleteFile(context.Context, string) error { return nil }
func (f *stagingFileService) CopyFile(context.Context, string, uint64, string) (string, error) {
	panic("unexpected CopyFile")
}

// fakeFileService captures uploads and returns a deterministic provider URL.
// Only SaveBytes is exercised by these tests; the other methods are
// implemented to satisfy interfaces.FileService but panic on use so a
// regression that starts calling them doesn't fail silently.
type fakeFileService struct {
	saved     map[string][]byte
	saveErr   error
	seq       int
	tenantIDs []uint64
}

func (f *fakeFileService) CheckConnectivity(_ context.Context) error { return nil }

func (f *fakeFileService) SaveFile(_ context.Context, _ *multipart.FileHeader, _ uint64, _ string) (string, error) {
	panic("SaveFile should not be called by these tests")
}

func (f *fakeFileService) SaveBytes(_ context.Context, data []byte, tenantID uint64, fileName string, _ bool) (string, error) {
	if f.saveErr != nil {
		return "", f.saveErr
	}
	if f.saved == nil {
		f.saved = map[string][]byte{}
	}
	f.seq++
	key := "fake://tenant-" + fileName
	f.saved[key] = append([]byte(nil), data...)
	f.tenantIDs = append(f.tenantIDs, tenantID)
	return key, nil
}

func (f *fakeFileService) GetFile(_ context.Context, filePath string) (io.ReadCloser, error) {
	if data, ok := f.saved[filePath]; ok {
		return io.NopCloser(strings.NewReader(string(data))), nil
	}
	return nil, io.ErrUnexpectedEOF
}

func (f *fakeFileService) GetFileURL(_ context.Context, _ string) (string, error) {
	panic("GetFileURL should not be called by these tests")
}

func (f *fakeFileService) DeleteFile(_ context.Context, _ string) error { return nil }

func (f *fakeFileService) CopyFile(_ context.Context, _ string, _ uint64, _ string) (string, error) {
	panic("CopyFile should not be called by these tests")
}

// bindCall records one Bind invocation for assertions.
type bindCall struct {
	ref       string
	ownerType string
	ownerID   string
	relation  string
}

// fakeCatalog captures Bind calls; every other ResourceCatalog method is a
// no-op stub because these tests only ever call Bind and Release.
type fakeCatalog struct {
	binds   []bindCall
	bindErr error
	// Release scripting, used by the deletion-guard tests. releaseRemaining
	// maps a reference to the binding count left after the release; a missing
	// entry answers -1 ("not a catalog handle").
	releaseRemaining map[string]int64
	releaseErr       error
	releases         []string
}

func (c *fakeCatalog) Register(context.Context, uint64, string, interfaces.ResourceRegistration) (string, error) {
	return "", nil
}

func (c *fakeCatalog) Resolve(context.Context, string) (*types.StoredResource, error) {
	return nil, nil
}

func (c *fakeCatalog) ResolvePath(_ context.Context, v string) (string, *types.StoredResource, error) {
	return v, nil, nil
}

func (c *fakeCatalog) Bind(_ context.Context, ref, ownerType, ownerID, relation string) error {
	c.binds = append(c.binds, bindCall{ref, ownerType, ownerID, relation})
	return c.bindErr
}
func (c *fakeCatalog) MarkDeleted(context.Context, string) error { return nil }

func (c *fakeCatalog) Release(_ context.Context, ref, ownerType, ownerID string) (int64, error) {
	c.releases = append(c.releases, ref+"|"+ownerType+"|"+ownerID)
	if c.releaseErr != nil {
		return -1, c.releaseErr
	}
	if remaining, ok := c.releaseRemaining[ref]; ok {
		return remaining, nil
	}
	return -1, nil
}

func (c *fakeCatalog) CreateAccessGrant(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func (c *fakeCatalog) ResolveAccessGrant(context.Context, string) (*types.StoredResource, error) {
	return nil, nil
}
