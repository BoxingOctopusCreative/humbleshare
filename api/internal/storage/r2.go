package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ObjectStore uploads, downloads, and deletes opaque objects (avatars/banners).
type ObjectStore interface {
	Put(ctx context.Context, key string, body io.Reader, contentType string, size int64) error
	Get(ctx context.Context, key string) (body io.ReadCloser, contentType string, size int64, err error)
	Delete(ctx context.Context, key string) error
}

// R2 is an S3-compatible Cloudflare R2 client bound to one bucket.
type R2 struct {
	client *s3.Client
	bucket string
}

// R2Credentials are shared across static and user-assets buckets.
type R2Credentials struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
}

// NewR2Client builds a shared R2/S3 client, or nil when credentials are incomplete.
func NewR2Client(creds R2Credentials) *s3.Client {
	accountID := strings.TrimSpace(creds.AccountID)
	accessKey := strings.TrimSpace(creds.AccessKeyID)
	secret := strings.TrimSpace(creds.SecretAccessKey)
	if accountID == "" || accessKey == "" || secret == "" {
		return nil
	}
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	return s3.New(s3.Options{
		BaseEndpoint: aws.String(endpoint),
		Region:       "auto",
		Credentials:  credentials.NewStaticCredentialsProvider(accessKey, secret, ""),
		UsePathStyle: true,
	})
}

// NewR2 binds a client to a bucket. Returns nil when client or bucket is missing.
func NewR2(client *s3.Client, bucket string) *R2 {
	bucket = strings.TrimSpace(bucket)
	if client == nil || bucket == "" {
		return nil
	}
	return &R2{client: client, bucket: bucket}
}

// NewR2FromConfig is a convenience for a single-bucket client (tests / simple callers).
func NewR2FromConfig(accountID, accessKeyID, secretAccessKey, bucket string) *R2 {
	return NewR2(NewR2Client(R2Credentials{
		AccountID:       accountID,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}), bucket)
}

func (r *R2) Bucket() string { return r.bucket }

func (r *R2) Put(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return fmt.Errorf("r2 put %s: %w", key, err)
	}
	return nil
}

func (r *R2) Get(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	out, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", 0, fmt.Errorf("r2 get %s: %w", key, err)
	}
	ct := "application/octet-stream"
	if out.ContentType != nil && *out.ContentType != "" {
		ct = *out.ContentType
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	return out.Body, ct, size, nil
}

func (r *R2) Delete(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("r2 delete %s: %w", key, err)
	}
	return nil
}
