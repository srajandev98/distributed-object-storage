package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func generateSignature(data string) string {
	h := hmac.New(sha256.New, []byte(appSecret))

	h.Write([]byte(data))

	signature := h.Sum(nil)

	return base64.URLEncoding.EncodeToString(signature)
}

func verifySignature(data string, providedSignature string) bool {
	expectedSignature := generateSignature(data)

	return hmac.Equal(
		[]byte(expectedSignature),
		[]byte(providedSignature),
	)
}

func presignHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/presign/")

	parts := strings.SplitN(path, "/", 2)

	if len(parts) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	bucket := filepath.Base(parts[0])

	objectKey := filepath.Clean(parts[1])

	expires := time.Now().Add(15 * time.Minute).Unix()

	dataToSign := bucket + "/" + objectKey + ":" + strconv.FormatInt(expires, 10)

	signature := generateSignature(dataToSign)

	url := fmt.Sprintf(
		"/download/%s/%s?expires=%d&signature=%s",
		bucket,
		objectKey,
		expires,
		signature,
	)

	w.Header().Set("Content-Type", "application/json")

	response := fmt.Sprintf(`{"url":"%s"}`, url)

	w.Write([]byte(response))
}

func validatePresignedURL(
	r *http.Request,
	bucket string,
	objectKey string,
) error {

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

	valid := verifySignature(dataToVerify, signature)

	if !valid {
		return errors.New("invalid signature")
	}

	return nil
}
