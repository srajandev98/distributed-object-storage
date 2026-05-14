package service

import (
	"bytes"
	"errors"
	"testing"
)

type fakeRepo struct {
	called bool
	err    error

	bucket      string
	objectKey   string
	objectPath  string
	size        int64
	contentType string
	checksum    string
	versionID   string
}

func (f *fakeRepo) SaveObjectWithJob(
	bucket string,
	objectKey string,
	objectPath string,
	size int64,
	contentType string,
	checksum string,
	versionID string,
) error {
	f.called = true
	f.bucket = bucket
	f.objectKey = objectKey
	f.objectPath = objectPath
	f.size = size
	f.contentType = contentType
	f.checksum = checksum
	f.versionID = versionID
	return f.err
}

func (f *fakeRepo) GetLatestObject(bucket string, objectKey string) (string, string, error) {
	return "", "", nil
}

type fakeStore struct {
	called bool
	err    error

	bucket    string
	objectKey string
	data      []byte
	versionID string

	returnPath string
}

func (f *fakeStore) StorePrimary(bucket string, objectKey string, data []byte, versionID string) (string, error) {
	f.called = true
	f.bucket = bucket
	f.objectKey = objectKey
	f.data = append([]byte{}, data...)
	f.versionID = versionID
	return f.returnPath, f.err
}

type errReader struct{}

func (e errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestUploadObjectSavesObjectAndEnqueuesReplicationJob(t *testing.T) {
	repo := &fakeRepo{}
	store := &fakeStore{returnPath: "storage/node1/my-bucket/v1_file.txt"}
	svc := NewObjectService(repo, store)

	body := []byte("hello-distributed-storage")
	err := svc.UploadObject("my-bucket", "docs/file.txt", bytes.NewReader(body), "text/plain")
	if err != nil {
		t.Fatalf("UploadObject returned error: %v", err)
	}

	if !store.called {
		t.Fatalf("expected StorePrimary to be called")
	}
	if !repo.called {
		t.Fatalf("expected SaveObjectWithJob to be called")
	}
	if repo.bucket != "my-bucket" {
		t.Fatalf("unexpected bucket: %s", repo.bucket)
	}
	if repo.objectKey != "docs/file.txt" {
		t.Fatalf("unexpected object key: %s", repo.objectKey)
	}
	if repo.objectPath != store.returnPath {
		t.Fatalf("expected object path %s, got %s", store.returnPath, repo.objectPath)
	}
	if repo.size != int64(len(body)) {
		t.Fatalf("expected size %d, got %d", len(body), repo.size)
	}
	if repo.contentType != "text/plain" {
		t.Fatalf("unexpected content type: %s", repo.contentType)
	}
	if repo.checksum == "" {
		t.Fatalf("expected checksum to be set")
	}
	if repo.versionID == "" {
		t.Fatalf("expected version id to be set")
	}
	if store.versionID != repo.versionID {
		t.Fatalf("expected same version id between storage and repository")
	}
}

func TestUploadObjectReturnsReadError(t *testing.T) {
	repo := &fakeRepo{}
	store := &fakeStore{returnPath: "storage/node1/my-bucket/v1_file.txt"}
	svc := NewObjectService(repo, store)

	err := svc.UploadObject("my-bucket", "docs/file.txt", errReader{}, "text/plain")
	if err == nil {
		t.Fatalf("expected error from read failure")
	}
	if store.called {
		t.Fatalf("store should not be called when body read fails")
	}
	if repo.called {
		t.Fatalf("repo should not be called when body read fails")
	}
}

func TestUploadObjectReturnsStoreError(t *testing.T) {
	repo := &fakeRepo{}
	store := &fakeStore{
		returnPath: "storage/node1/my-bucket/v1_file.txt",
		err:        errors.New("store failed"),
	}
	svc := NewObjectService(repo, store)

	err := svc.UploadObject("my-bucket", "docs/file.txt", bytes.NewReader([]byte("hello")), "text/plain")
	if err == nil {
		t.Fatalf("expected error from storage failure")
	}
	if !store.called {
		t.Fatalf("expected store to be called")
	}
	if repo.called {
		t.Fatalf("repo should not be called when storage fails")
	}
}

func TestUploadObjectReturnsRepositoryError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("repo failed")}
	store := &fakeStore{returnPath: "storage/node1/my-bucket/v1_file.txt"}
	svc := NewObjectService(repo, store)

	err := svc.UploadObject("my-bucket", "docs/file.txt", bytes.NewReader([]byte("hello")), "text/plain")
	if err == nil {
		t.Fatalf("expected error from repository failure")
	}
	if !store.called {
		t.Fatalf("expected store to be called")
	}
	if !repo.called {
		t.Fatalf("expected repo to be called")
	}
}
