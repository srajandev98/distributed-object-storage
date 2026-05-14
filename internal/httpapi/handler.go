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
	"path"
	"path/filepath"
	"regexp"
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

	bucket := parts[0]
	objectKey := parts[1]

	if !isValidBucketName(bucket) {
		return "", "", false
	}
	if !isValidObjectKey(objectKey) {
		return "", "", false
	}

	return bucket, objectKey, true
}

var bucketNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)

func isValidBucketName(name string) bool {
	if !bucketNamePattern.MatchString(name) {
		return false
	}

	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		return false
	}

	return true
}

func isValidObjectKey(key string) bool {
	if key == "" {
		return false
	}

	if strings.HasPrefix(key, "/") {
		return false
	}

	if strings.Contains(key, `\`) {
		return false
	}

	cleaned := path.Clean(key)
	if cleaned == "." || cleaned == ".." {
		return false
	}

	segments := strings.Split(cleaned, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}

	return cleaned == key
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
