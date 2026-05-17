package service

import (
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/google/uuid"
)

type objectRepository interface {
	SaveObjectWithJob(
		bucket string,
		objectKey string,
		objectPath string,
		size int64,
		contentType string,
		checksum string,
		versionID string,
	) error
	GetLatestObject(bucket string, objectKey string) (string, string, error)
}

type objectStorage interface {
	StorePrimary(bucket string, objectKey string, data []byte, versionID string) (string, error)
}

type ObjectService struct {
	repo  objectRepository
	store objectStorage
}

func NewObjectService(repo objectRepository, store objectStorage) *ObjectService {
	return &ObjectService{repo: repo, store: store}
}

func (s *ObjectService) UploadObject(bucket string, objectKey string, body io.Reader, contentType string) error {
	versionID := uuid.New().String()

	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	hasher.Write(data)
	checksum := hex.EncodeToString(hasher.Sum(nil))

	objectPath, err := s.store.StorePrimary(bucket, objectKey, data, versionID)
	if err != nil {
		return err
	}

	size := int64(len(data))

	return s.repo.SaveObjectWithJob(
		bucket,
		objectKey,
		objectPath,
		size,
		contentType,
		checksum,
		versionID,
	)
}

func (s *ObjectService) GetLatestObject(bucket string, objectKey string) (string, string, error) {
	return s.repo.GetLatestObject(bucket, objectKey)
}
