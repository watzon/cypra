package s3compat_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/watzon/cypra/internal/storage"
	"github.com/watzon/cypra/internal/storage/s3compat"
)

func TestSignedURLUsesS3CompatibleEndpoint(t *testing.T) {
	store := s3compat.New(s3compat.Config{Bucket: "cypra", Region: "auto", Endpoint: "https://s3.example.test", AccessKeyID: "key", SecretAccessKey: "secret", PathStyle: true})
	url, err := store.SignedURL("tenant/avatar.png", 2*time.Hour)
	if err != nil {
		t.Fatalf("signed url: %v", err)
	}
	if !strings.HasPrefix(url, "https://s3.example.test/cypra/tenant/avatar.png") {
		t.Fatalf("url = %q", url)
	}
	if !strings.Contains(url, "X-Amz-Expires=3600") {
		t.Fatalf("ttl was not capped in url = %q", url)
	}
}

func TestMinIORoundTrip(t *testing.T) {
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "minio/minio:RELEASE.2025-04-22T22-12-26Z",
			ExposedPorts: []string{"9000/tcp"},
			Env:          map[string]string{"MINIO_ROOT_USER": "minioadmin", "MINIO_ROOT_PASSWORD": "minioadmin"},
			Cmd:          []string{"server", "/data"},
			WaitingFor:   wait.ForHTTP("/minio/health/ready").WithPort("9000/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start minio: %v", err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Errorf("terminate minio: %v", err)
		}
	})
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("minio host: %v", err)
	}
	port, err := container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		t.Fatalf("minio port: %v", err)
	}
	endpoint := "http://" + host + ":" + port.Port()
	bucket := "cypra-test"
	client := s3.NewFromConfig(aws.Config{Region: "us-east-1", Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", ""))}, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})
	if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	store := s3compat.New(s3compat.Config{Bucket: bucket, Region: "us-east-1", Endpoint: endpoint, AccessKeyID: "minioadmin", SecretAccessKey: "minioadmin", PathStyle: true})
	want := []byte("avatar-bytes")
	if err := store.Put(ctx, storage.Object{Key: "tenant/avatar.txt", ContentType: "text/plain", Bytes: want}); err != nil {
		t.Fatalf("put: %v", err)
	}
	signedURL, err := store.SignedURL("tenant/avatar.txt", 5*time.Minute)
	if err != nil {
		t.Fatalf("signed url: %v", err)
	}
	resp, err := http.Get(signedURL) // #nosec G107 -- test-only presigned MinIO URL.
	if err != nil {
		t.Fatalf("fetch signed url: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	fetched, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read signed response: %v", err)
	}
	if string(fetched) != string(want) {
		t.Fatalf("signed response = %q", fetched)
	}
	got, err := store.Get(ctx, "tenant/avatar.txt")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got.Bytes) != string(want) || got.ContentType != "text/plain" {
		t.Fatalf("object = %+v", got)
	}
	if err := store.Delete(ctx, "tenant/avatar.txt"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
