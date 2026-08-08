package s3dest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeS3 is a minimal in-memory stand-in for S3-compatible object storage,
// just enough of the PUT/GET/DELETE/ListObjectsV2 surface for Write/Read/
// List/Delete to round-trip against, addressed path-style
// (/<bucket>/<key>) the same way Dest.client configures UsePathStyle.
type fakeS3 struct {
	mu      sync.Mutex
	objects map[string][]byte // "<bucket>/<key>" -> body
}

func newFakeS3(t *testing.T) (*httptest.Server, *fakeS3) {
	t.Helper()
	f := &fakeS3{objects: map[string][]byte{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		parts := strings.SplitN(path, "/", 2)
		bucket := parts[0]

		if r.Method == http.MethodGet && r.URL.Query().Get("list-type") == "2" {
			f.list(w, bucket, r.URL.Query().Get("prefix"))
			return
		}

		var key string
		if len(parts) > 1 {
			key = parts[1]
		}
		full := bucket + "/" + key
		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			f.mu.Lock()
			f.objects[full] = body
			f.mu.Unlock()
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			f.mu.Lock()
			data, ok := f.objects[full]
			f.mu.Unlock()
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write(data)
		case http.MethodDelete:
			f.mu.Lock()
			delete(f.objects, full)
			f.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, f
}

func (f *fakeS3) list(w http.ResponseWriter, bucket, prefix string) {
	f.mu.Lock()
	var keys []string
	for full := range f.objects {
		b, k, ok := strings.Cut(full, "/")
		if !ok || b != bucket || !strings.HasPrefix(k, prefix) {
			continue
		}
		keys = append(keys, k)
	}
	f.mu.Unlock()
	sort.Strings(keys)

	var buf strings.Builder
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`)
	for _, k := range keys {
		buf.WriteString("<Contents><Key>" + k + "</Key></Contents>")
	}
	buf.WriteString(`<IsTruncated>false</IsTruncated></ListBucketResult>`)
	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(buf.String()))
}

// testDest builds a Dest with static test credentials, bypassing New's
// ambient AWS credential-chain load so the test never depends on the
// environment it runs in.
func testDest() *Dest {
	return &Dest{
		awsCfg: aws.Config{
			Region:      "us-east-1",
			Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
		},
	}
}

func testCfg(endpoint, bucket, prefix string) types.BackupConfig {
	return types.BackupConfig{
		UserID:     "u1",
		S3Bucket:   bucket,
		S3Prefix:   prefix,
		S3Endpoint: endpoint,
	}
}

func TestDelete_RemovesAllObjectsUnderPrefix(t *testing.T) {
	srv, _ := newFakeS3(t)
	d := testDest()
	cfg := testCfg(srv.URL, "bucket", "u1")

	for _, name := range []string{"meals.csv", "rollups.csv", "photo_p1.jpg"} {
		if err := d.Write(context.Background(), cfg, name, []byte("data")); err != nil {
			t.Fatalf("Write(%s): %v", name, err)
		}
	}

	if err := d.Delete(context.Background(), cfg); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	got, err := d.List(context.Background(), cfg)
	if err != nil {
		t.Fatalf("List after Delete: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no objects left after Delete, got %v", got)
	}
}

func TestDelete_OtherUsersPrefixUntouched(t *testing.T) {
	srv, _ := newFakeS3(t)
	d := testDest()
	u1 := testCfg(srv.URL, "bucket", "u1")
	u2 := testCfg(srv.URL, "bucket", "u2")

	if err := d.Write(context.Background(), u1, "meals.csv", []byte("u1 data")); err != nil {
		t.Fatalf("Write u1: %v", err)
	}
	if err := d.Write(context.Background(), u2, "meals.csv", []byte("u2 data")); err != nil {
		t.Fatalf("Write u2: %v", err)
	}

	if err := d.Delete(context.Background(), u1); err != nil {
		t.Fatalf("Delete u1: %v", err)
	}

	got, err := d.Read(context.Background(), u2, "meals.csv")
	if err != nil {
		t.Fatalf("expected u2's object to survive u1's Delete: %v", err)
	}
	if string(got) != "u2 data" {
		t.Fatalf("u2 content mismatch: %q", got)
	}
}

func TestDelete_EmptyPrefixIsNoop(t *testing.T) {
	srv, _ := newFakeS3(t)
	d := testDest()
	cfg := testCfg(srv.URL, "bucket", "never-written")

	if err := d.Delete(context.Background(), cfg); err != nil {
		t.Fatalf("Delete on empty prefix: %v", err)
	}
}

func TestDelete_MissingBucketErrors(t *testing.T) {
	d := testDest()
	cfg := types.BackupConfig{UserID: "u1"} // no S3Bucket

	if err := d.Delete(context.Background(), cfg); err == nil {
		t.Fatal("expected error when s3_bucket is not configured")
	}
}
