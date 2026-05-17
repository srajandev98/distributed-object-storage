package replication

import (
	"errors"
	"fmt"
	"time"

	"distributed-object-storage/internal/repository"
)

type workerRepository interface {
	ClaimReplicationJob() (repository.ReplicationJob, error)
	MarkReplicationComplete(jobID int) error
	MarkReplicationFailure(job repository.ReplicationJob, nextStatus string, nextAttempt int, nextDelaySeconds int, lastErr string) error
	RecordReplica(objectID int, nodeName string, filePath string) error
}

type workerStorage interface {
	Read(path string) ([]byte, error)
	ReplicateToSecondaries(bucket string, objectKey string, versionID string, data []byte) ([]string, error)
	SecondaryNodes() []string
}

type Worker struct {
	repo  workerRepository
	store workerStorage
}

func NewWorker(repo workerRepository, store workerStorage) *Worker {
	return &Worker{repo: repo, store: store}
}

func (w *Worker) Run() {
	for {
		job, err := w.repo.ClaimReplicationJob()
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		if err = w.handleJob(job); err != nil {
			_ = w.markFailure(job, err)
			continue
		}

		_ = w.repo.MarkReplicationComplete(job.ID)
	}
}

func (w *Worker) handleJob(job repository.ReplicationJob) error {
	data, err := w.store.Read(job.SourcePath)
	if err != nil {
		return err
	}

	paths, err := w.store.ReplicateToSecondaries(job.Bucket, job.ObjectKey, job.VersionID, data)
	if err != nil {
		return err
	}

	nodes := w.store.SecondaryNodes()
	if len(paths) != len(nodes) {
		return errors.New("replication mismatch")
	}

	for i, p := range paths {
		if err = w.repo.RecordReplica(job.ObjectID, nodes[i], p); err != nil {
			return err
		}
	}

	return nil
}

func (w *Worker) markFailure(job repository.ReplicationJob, cause error) error {
	nextAttempt := job.AttemptCount + 1
	status := "pending"
	if nextAttempt >= job.MaxAttempts {
		status = "failed"
	}

	backoffSeconds := nextAttempt * nextAttempt
	err := w.repo.MarkReplicationFailure(job, status, nextAttempt, backoffSeconds, cause.Error())
	if err != nil {
		return err
	}

	if status == "failed" {
		return fmt.Errorf("replication job %d exhausted retries: %w", job.ID, cause)
	}
	return nil
}
