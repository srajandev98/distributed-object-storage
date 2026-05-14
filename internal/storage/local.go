package storage

import (
	"os"
	"path/filepath"
)

type Local struct {
	nodes []string
}

func NewLocal(nodes []string) *Local {
	return &Local{nodes: nodes}
}

func (l *Local) Init() error {
	for _, node := range l.nodes {
		if err := os.MkdirAll(node, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func (l *Local) StorePrimary(bucket string, objectKey string, data []byte, versionID string) (string, error) {
	primaryNode := l.nodes[0]
	bucketPath := filepath.Join(primaryNode, bucket)
	if err := os.MkdirAll(bucketPath, os.ModePerm); err != nil {
		return "", err
	}

	objectDir := filepath.Join(bucketPath, filepath.Dir(objectKey))
	if err := os.MkdirAll(objectDir, os.ModePerm); err != nil {
		return "", err
	}

	versionedObjectKey := versionID + "_" + filepath.Base(objectKey)
	objectPath := filepath.Join(objectDir, versionedObjectKey)
	if err := os.WriteFile(objectPath, data, 0644); err != nil {
		return "", err
	}

	return objectPath, nil
}

func (l *Local) Read(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (l *Local) ReplicateToSecondaries(bucket string, objectKey string, versionID string, data []byte) ([]string, error) {
	paths := make([]string, 0, len(l.nodes)-1)

	for _, node := range l.nodes[1:] {
		bucketPath := filepath.Join(node, bucket)
		if err := os.MkdirAll(bucketPath, os.ModePerm); err != nil {
			return nil, err
		}

		objectDir := filepath.Join(bucketPath, filepath.Dir(objectKey))
		if err := os.MkdirAll(objectDir, os.ModePerm); err != nil {
			return nil, err
		}

		versionedObjectKey := versionID + "_" + filepath.Base(objectKey)
		objectPath := filepath.Join(objectDir, versionedObjectKey)
		if err := os.WriteFile(objectPath, data, 0644); err != nil {
			return nil, err
		}

		paths = append(paths, objectPath)
	}

	return paths, nil
}

func (l *Local) SecondaryNodes() []string {
	return l.nodes[1:]
}
