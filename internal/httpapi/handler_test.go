package httpapi

import "testing"

func TestParseBucketObjectPathValid(t *testing.T) {
	bucket, key, ok := parseBucketObjectPath("/upload/my-bucket/docs/file.txt", "/upload/")
	if !ok {
		t.Fatalf("expected valid path")
	}
	if bucket != "my-bucket" {
		t.Fatalf("unexpected bucket: %s", bucket)
	}
	if key != "docs/file.txt" {
		t.Fatalf("unexpected key: %s", key)
	}
}

func TestParseBucketObjectPathRejectsInvalidBucket(t *testing.T) {
	_, _, ok := parseBucketObjectPath("/upload/MyBucket/docs/file.txt", "/upload/")
	if ok {
		t.Fatalf("expected invalid bucket to be rejected")
	}
}

func TestParseBucketObjectPathRejectsTraversal(t *testing.T) {
	_, _, ok := parseBucketObjectPath("/upload/my-bucket/docs/../../secret.txt", "/upload/")
	if ok {
		t.Fatalf("expected traversal key to be rejected")
	}
}

func TestParseBucketObjectPathRejectsAbsoluteKey(t *testing.T) {
	_, _, ok := parseBucketObjectPath("/upload/my-bucket//etc/passwd", "/upload/")
	if ok {
		t.Fatalf("expected absolute key to be rejected")
	}
}

func TestIsValidObjectKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{key: "docs/file.txt", valid: true},
		{key: "a/b/c/data.json", valid: true},
		{key: "", valid: false},
		{key: "/root/file.txt", valid: false},
		{key: "../secret.txt", valid: false},
		{key: "docs/../secret.txt", valid: false},
		{key: "docs//file.txt", valid: false},
		{key: `docs\file.txt`, valid: false},
	}

	for _, tc := range tests {
		got := isValidObjectKey(tc.key)
		if got != tc.valid {
			t.Fatalf("key %q expected valid=%v, got=%v", tc.key, tc.valid, got)
		}
	}
}
