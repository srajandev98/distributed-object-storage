package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	bucketPath := filepath.Join("storage", bucket)

	err := os.MkdirAll(bucketPath, os.ModePerm)
	if err != nil {
		http.Error(w, "failed to create bucket", http.StatusInternalServerError)
		return
	}

	objectPath := filepath.Join(bucketPath, objectKey)

	err = os.MkdirAll(filepath.Dir(objectPath), os.ModePerm)
	if err != nil {
		http.Error(w, "failed to create object directory", http.StatusInternalServerError)
		return
	}

	file, err := os.Create(objectPath)
	if err != nil {
		http.Error(w, "failed to create object", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, "failed to save object", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("object uploaded successfully"))
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

	objectPath := filepath.Join("storage", bucket, objectKey)

	file, err := os.Open(objectPath)
	if err != nil {
		http.Error(w, "object not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	filename := filepath.Base(objectKey)

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "failed to send object", http.StatusInternalServerError)
		return
	}
}

func main() {
	http.HandleFunc("/upload/", uploadHandler)

	http.HandleFunc("/download/", downloadHandler)

	http.ListenAndServe(":8080", nil)
}
