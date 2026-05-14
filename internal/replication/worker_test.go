package replication

import (
	"errors"
	"testing"

	"distributed-object-storage/internal/repository"
)

type fakeWorkerRepo struct {
	markFailureCalled bool

	lastJob          repository.ReplicationJob
	lastStatus       string
	lastAttempt      int
	lastDelaySeconds int
	lastErr          string

	markFailureErr error
}

func (f *fakeWorkerRepo) ClaimReplicationJob() (repository.ReplicationJob, error) {
	return repository.ReplicationJob{}, errors.New("not used in this test")
}

func (f *fakeWorkerRepo) MarkReplicationComplete(jobID int) error {
	return nil
}

func (f *fakeWorkerRepo) MarkReplicationFailure(job repository.ReplicationJob, nextStatus string, nextAttempt int, nextDelaySeconds int, lastErr string) error {
	f.markFailureCalled = true
	f.lastJob = job
	f.lastStatus = nextStatus
	f.lastAttempt = nextAttempt
	f.lastDelaySeconds = nextDelaySeconds
	f.lastErr = lastErr
	return f.markFailureErr
}

func (f *fakeWorkerRepo) RecordReplica(objectID int, nodeName string, filePath string) error {
	return nil
}

type fakeWorkerStore struct{}

func (f *fakeWorkerStore) Read(path string) ([]byte, error) {
	return nil, nil
}

func (f *fakeWorkerStore) ReplicateToSecondaries(bucket string, objectKey string, versionID string, data []byte) ([]string, error) {
	return nil, nil
}

func (f *fakeWorkerStore) SecondaryNodes() []string {
	return nil
}

func TestMarkFailureSchedulesRetry(t *testing.T) {
	repo := &fakeWorkerRepo{}
	store := &fakeWorkerStore{}
	worker := NewWorker(repo, store)

	job := repository.ReplicationJob{
		ID:           10,
		AttemptCount: 1,
		MaxAttempts:  5,
	}

	cause := errors.New("disk full")
	err := worker.markFailure(job, cause)
	if err != nil {
		t.Fatalf("expected no terminal error, got: %v", err)
	}

	if !repo.markFailureCalled {
		t.Fatalf("expected MarkReplicationFailure to be called")
	}
	if repo.lastStatus != "pending" {
		t.Fatalf("expected pending status, got %s", repo.lastStatus)
	}
	if repo.lastAttempt != 2 {
		t.Fatalf("expected next attempt 2, got %d", repo.lastAttempt)
	}
	if repo.lastDelaySeconds != 4 {
		t.Fatalf("expected quadratic backoff 4s, got %d", repo.lastDelaySeconds)
	}
	if repo.lastErr != "disk full" {
		t.Fatalf("expected error message to be persisted")
	}
}

func TestMarkFailureMarksTerminalFailure(t *testing.T) {
	repo := &fakeWorkerRepo{}
	store := &fakeWorkerStore{}
	worker := NewWorker(repo, store)

	job := repository.ReplicationJob{
		ID:           11,
		AttemptCount: 4,
		MaxAttempts:  5,
	}

	cause := errors.New("permission denied")
	err := worker.markFailure(job, cause)
	if err == nil {
		t.Fatalf("expected terminal error")
	}

	if repo.lastStatus != "failed" {
		t.Fatalf("expected failed status, got %s", repo.lastStatus)
	}
	if repo.lastAttempt != 5 {
		t.Fatalf("expected next attempt 5, got %d", repo.lastAttempt)
	}
	if repo.lastDelaySeconds != 25 {
		t.Fatalf("expected quadratic backoff 25s, got %d", repo.lastDelaySeconds)
	}
}

func TestMarkFailurePropagatesRepositoryError(t *testing.T) {
	repo := &fakeWorkerRepo{markFailureErr: errors.New("db write failed")}
	store := &fakeWorkerStore{}
	worker := NewWorker(repo, store)

	job := repository.ReplicationJob{
		ID:           12,
		AttemptCount: 0,
		MaxAttempts:  5,
	}

	err := worker.markFailure(job, errors.New("network timeout"))
	if err == nil {
		t.Fatalf("expected repository error")
	}
	if err.Error() != "db write failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMarkFailurePropagatesTransitionRejection(t *testing.T) {
	repo := &fakeWorkerRepo{markFailureErr: errors.New("replication job 12 transition to pending rejected")}
	store := &fakeWorkerStore{}
	worker := NewWorker(repo, store)

	job := repository.ReplicationJob{
		ID:           12,
		AttemptCount: 0,
		MaxAttempts:  5,
	}

	err := worker.markFailure(job, errors.New("network timeout"))
	if err == nil {
		t.Fatalf("expected transition rejection error")
	}
	if err.Error() != "replication job 12 transition to pending rejected" {
		t.Fatalf("unexpected error: %v", err)
	}
}
