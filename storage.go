package main

import (
	"os"
	"path/filepath"
)

var storageNodes = []string{
	"storage/node1",
	"storage/node2",
	"storage/node3",
}

func initStorage() error {
	for _, node := range storageNodes {
		err := os.MkdirAll(node, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func storeObject(
	bucket string,
	objectKey string,
	data []byte,
	versionID string,
) (string, error) {

	primaryNode := storageNodes[0]

	bucketPath := filepath.Join(primaryNode, bucket)

	err := os.MkdirAll(bucketPath, os.ModePerm)
	if err != nil {
		return "", err
	}

	objectDir := filepath.Join(
		bucketPath,
		filepath.Dir(objectKey),
	)

	err = os.MkdirAll(objectDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	versionedObjectKey := versionID +
		"_" +
		filepath.Base(objectKey)

	objectPath := filepath.Join(
		objectDir,
		versionedObjectKey,
	)

	err = os.WriteFile(objectPath, data, 0644)
	if err != nil {
		return "", err
	}

	return objectPath, nil
}
