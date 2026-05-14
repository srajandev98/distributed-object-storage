package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"distributed-object-storage/internal/repository"
	"distributed-object-storage/internal/service"
)

type Handler struct {
	objects   *service.ObjectService
	appSecret string
}

func NewHandler(objects *service.ObjectService, appSecret string) *Handler {
	return &Handler{objects: objects, appSecret: appSecret}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/upload/", h.uploadHandler)
	mux.HandleFunc("/download/", h.downloadHandler)
	mux.HandleFunc("/presign/", h.presignHandler)
}

func (h *Handler) uploadHandler(w http.ResponseWriter, r *http.Request) {
	bucket, objectKey, ok := parseBucketObjectPath(r.URL.Path, "/upload/")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	err := h.objects.UploadObject(bucket, objectKey, r.Body, r.Header.Get("Content-Type"))
	if err != nil {
		http.Error(w, "failed to upload object", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write([]byte("versioned object uploaded successfully"))
}

func (h *Handler) downloadHandler(w http.ResponseWriter, r *http.Request) {
	bucket, objectKey, ok := parseBucketObjectPath(r.URL.Path, "/download/")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	if err := h.validatePresignedURL(r, bucket, objectKey); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	filePath, contentType, err := h.objects.GetLatestObject(bucket, objectKey)
	if err != nil {
		if repository.IsNotFound(err) {
			http.Error(w, "object not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to fetch metadata", http.StatusInternalServerError)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "failed to open object", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(objectKey))
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	if _, err = io.Copy(w, file); err != nil {
		http.Error(w, "failed to send object", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) presignHandler(w http.ResponseWriter, r *http.Request) {
	bucket, objectKey, ok := parseBucketObjectPath(r.URL.Path, "/presign/")
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	expires := time.Now().Add(15 * time.Minute).Unix()
	dataToSign := bucket + "/" + objectKey + ":" + strconv.FormatInt(expires, 10)
	signature := h.generateSignature(dataToSign)
	url := fmt.Sprintf("/download/%s/%s?expires=%d&signature=%s", bucket, objectKey, expires, signature)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"url":"%s"}`, url)))
}

func parseBucketObjectPath(path string, prefix string) (string, string, bool) {
	trimmed := strings.TrimPrefix(path, prefix)
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	bucket := filepath.Base(parts[0])
	objectKey := filepath.Clean(parts[1])
	return bucket, objectKey, true
}

func (h *Handler) generateSignature(data string) string {
	mac := hmac.New(sha256.New, []byte(h.appSecret))
	mac.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

func (h *Handler) verifySignature(data string, providedSignature string) bool {
	expectedSignature := h.generateSignature(data)
	return hmac.Equal([]byte(expectedSignature), []byte(providedSignature))
}

func (h *Handler) validatePresignedURL(r *http.Request, bucket string, objectKey string) error {
	expires := r.URL.Query().Get("expires")
	signature := r.URL.Query().Get("signature")

	if expires == "" || signature == "" {
		return errors.New("missing signature")
	}

	expiresUnix, err := strconv.ParseInt(expires, 10, 64)
	if err != nil {
		return errors.New("invalid expiration")
	}
	if time.Now().Unix() > expiresUnix {
		return errors.New("url expired")
	}

	dataToVerify := bucket + "/" + objectKey + ":" + expires
	if !h.verifySignature(dataToVerify, signature) {
		return errors.New("invalid signature")
	}

	return nil
}
