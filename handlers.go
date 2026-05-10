package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	_ "github.com/lib/pq"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/upload/")

	parts := strings.SplitN(path, "/", 2)

	if len(parts) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	bucket := filepath.Base(parts[0])

	objectKey := filepath.Clean(parts[1])

	versionID := uuid.New().String()

	versionedObjectKey := versionID + "_" + filepath.Base(objectKey)

	bucketPath := filepath.Join("storage", bucket)

	err := os.MkdirAll(bucketPath, os.ModePerm)
	if err != nil {
		http.Error(w, "failed to create bucket", http.StatusInternalServerError)
		return
	}

	objectDir := filepath.Join(bucketPath, filepath.Dir(objectKey))

	err = os.MkdirAll(objectDir, os.ModePerm)
	if err != nil {
		http.Error(w, "failed to create object directory", http.StatusInternalServerError)
		return
	}

	objectPath := filepath.Join(objectDir, versionedObjectKey)

	file, err := os.Create(objectPath)
	if err != nil {
		http.Error(w, "failed to create object", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	hasher := sha256.New()

	writer := io.MultiWriter(file, hasher)

	size, err := io.Copy(writer, r.Body)
	if err != nil {
		http.Error(w, "failed to save object", http.StatusInternalServerError)
		return
	}

	checksum := hex.EncodeToString(hasher.Sum(nil))

	contentType := r.Header.Get("Content-Type")

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "failed to start transaction", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`
        UPDATE objects
        SET is_latest = FALSE
        WHERE bucket = $1
        AND object_key = $2
    `,
		bucket,
		objectKey,
	)

	if err != nil {
		tx.Rollback()

		http.Error(w, "failed to update old versions", http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec(`
        INSERT INTO objects (
            bucket,
            object_key,
            file_path,
            size,
            content_type,
            checksum,
            version_id,
            is_latest
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
    `,
		bucket,
		objectKey,
		objectPath,
		size,
		contentType,
		checksum,
		versionID,
	)

	if err != nil {
		tx.Rollback()

		http.Error(w, "failed to save metadata", http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("versioned object uploaded successfully"))
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/download/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	bucket := filepath.Base(parts[0])
	objectKey := filepath.Clean(parts[1])

	err := validatePresignedURL(r, bucket, objectKey)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var filePath string
	var contentType string

	err = db.QueryRow(`
      SELECT file_path, content_type
      FROM objects
      WHERE bucket = $1
      AND object_key = $2
      AND is_latest = TRUE
      LIMIT 1
    `,
		bucket,
		objectKey,
	).Scan(&filePath, &contentType)

	if err != nil {
		http.Error(w, "object not found", http.StatusNotFound)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "failed to open object", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	filename := filepath.Base(objectKey)

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "failed to send object", http.StatusInternalServerError)
		return
	}
}
