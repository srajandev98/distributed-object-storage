package repository

import (
	"database/sql"
	"errors"
	"fmt"
)

type Repository struct {
	db *sql.DB
}

type ReplicationJob struct {
	ID           int
	ObjectID     int
	Bucket       string
	ObjectKey    string
	VersionID    string
	SourcePath   string
	AttemptCount int
	MaxAttempts  int
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) EnsureReplicationJobsTable() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS replication_jobs (
			id SERIAL PRIMARY KEY,
			object_id INTEGER NOT NULL,
			bucket TEXT NOT NULL,
			object_key TEXT NOT NULL,
			version_id TEXT NOT NULL,
			source_file_path TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempt_count INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 5,
			next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_error TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_replication_jobs_fetch
		ON replication_jobs(status, next_run_at)
	`)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_replicas_object_node_unique
		ON replicas(object_id, node_name)
	`)
	return err
}

func (r *Repository) SaveObjectWithJob(
	bucket string,
	objectKey string,
	objectPath string,
	size int64,
	contentType string,
	checksum string,
	versionID string,
) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE objects
		SET is_latest = FALSE
		WHERE bucket = $1
		  AND object_key = $2
	`, bucket, objectKey)
	if err != nil {
		return err
	}

	var objectID int
	err = tx.QueryRow(`
		INSERT INTO objects (
			bucket, object_key, file_path, size, content_type, checksum, version_id, is_latest
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
		RETURNING id
	`, bucket, objectKey, objectPath, size, contentType, checksum, versionID).Scan(&objectID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO replication_jobs (
			object_id, bucket, object_key, version_id, source_file_path
		)
		VALUES ($1, $2, $3, $4, $5)
	`, objectID, bucket, objectKey, versionID, objectPath)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) GetLatestObject(bucket string, objectKey string) (string, string, error) {
	var filePath string
	var contentType string

	err := r.db.QueryRow(`
		SELECT file_path, content_type
		FROM objects
		WHERE bucket = $1
		  AND object_key = $2
		  AND is_latest = TRUE
		LIMIT 1
	`, bucket, objectKey).Scan(&filePath, &contentType)
	if err != nil {
		return "", "", err
	}

	return filePath, contentType, nil
}

func (r *Repository) ClaimReplicationJob() (ReplicationJob, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return ReplicationJob{}, err
	}
	defer tx.Rollback()

	var job ReplicationJob
	err = tx.QueryRow(`
		WITH next_job AS (
			SELECT id
			FROM replication_jobs
			WHERE status = 'pending'
			  AND next_run_at <= NOW()
			ORDER BY id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE replication_jobs j
		SET status = 'running',
		    updated_at = NOW()
		FROM next_job
		WHERE j.id = next_job.id
		RETURNING
			j.id, j.object_id, j.bucket, j.object_key, j.version_id,
			j.source_file_path, j.attempt_count, j.max_attempts
	`).Scan(
		&job.ID, &job.ObjectID, &job.Bucket, &job.ObjectKey, &job.VersionID,
		&job.SourcePath, &job.AttemptCount, &job.MaxAttempts,
	)
	if err != nil {
		return ReplicationJob{}, err
	}

	if err = tx.Commit(); err != nil {
		return ReplicationJob{}, err
	}

	return job, nil
}

func (r *Repository) MarkReplicationComplete(jobID int) error {
	result, err := r.db.Exec(`
		UPDATE replication_jobs
		SET status = 'completed', updated_at = NOW()
		WHERE id = $1
		  AND status = 'running'
	`, jobID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("replication job %d transition to completed rejected", jobID)
	}

	return nil
}

func (r *Repository) MarkReplicationFailure(job ReplicationJob, nextStatus string, nextAttempt int, nextDelaySeconds int, lastErr string) error {
	result, err := r.db.Exec(`
		UPDATE replication_jobs
		SET status = $2,
		    attempt_count = $3,
		    last_error = $4,
		    next_run_at = NOW() + ($5 * INTERVAL '1 second'),
		    updated_at = NOW()
		WHERE id = $1
		  AND status = 'running'
	`, job.ID, nextStatus, nextAttempt, lastErr, nextDelaySeconds)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("replication job %d transition to %s rejected", job.ID, nextStatus)
	}

	return nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func (r *Repository) RecordReplica(objectID int, nodeName string, filePath string) error {
	_, err := r.db.Exec(`
		INSERT INTO replicas (object_id, node_name, file_path, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, objectID, nodeName, filePath, "completed")
	return err
}
