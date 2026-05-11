package main

import (
	"os"
	"path/filepath"
)

type ReplicationJob struct {
	ObjectID  int
	Bucket    string
	ObjectKey string
	VersionID string
	Data      []byte
}

var replicationQueue chan ReplicationJob

func initReplicationQueue() {
	replicationQueue = make(chan ReplicationJob, 100)
}

func replicationWorker() {
	for job := range replicationQueue {

		for _, node := range storageNodes[1:] {

			bucketPath := filepath.Join(node, job.Bucket)

			err := os.MkdirAll(bucketPath, os.ModePerm)
			if err != nil {
				continue
			}

			objectDir := filepath.Join(
				bucketPath,
				filepath.Dir(job.ObjectKey),
			)

			err = os.MkdirAll(objectDir, os.ModePerm)
			if err != nil {
				continue
			}

			versionedObjectKey := job.VersionID +
				"_" +
				filepath.Base(job.ObjectKey)

			objectPath := filepath.Join(
				objectDir,
				versionedObjectKey,
			)

			err = os.WriteFile(objectPath, job.Data, 0644)
			if err != nil {
				continue
			}

			_, err = db.Exec(`
            INSERT INTO replicas (
                object_id,
                node_name,
                file_path,
                status
            )
            VALUES ($1, $2, $3, $4)
        `,
				job.ObjectID,
				node,
				objectPath,
				"completed",
			)

			if err != nil {
				continue
			}
		}
	}
}
