// Package s3dest implements backup.Destination against S3-compatible object
// storage. Credentials always come from the ambient AWS credential chain
// (env vars, shared config, instance/task role) — this package never stores
// or accepts per-user credentials. Only bucket/prefix/region/endpoint vary
// per user, taken from types.BackupConfig.
package s3dest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Dest writes backup files to S3-compatible buckets.
type Dest struct {
	awsCfg aws.Config
}

// New loads the default AWS credential chain once at startup.
func New(ctx context.Context) (*Dest, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("s3dest: load AWS config: %w", err)
	}
	return &Dest{awsCfg: cfg}, nil
}

// client builds an S3 client for this call only, applying a per-user
// S3Region/S3Endpoint override (e.g. a self-hosted MinIO) on top of the
// shared base credential chain.
func (d *Dest) client(cfg types.BackupConfig) *s3.Client {
	return s3.NewFromConfig(d.awsCfg, func(o *s3.Options) {
		if cfg.S3Region != "" {
			o.Region = cfg.S3Region
		}
		if cfg.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
			o.UsePathStyle = true // required by most non-AWS S3-compatible endpoints
		}
	})
}

// Write uploads data to cfg.S3Bucket at "<S3Prefix>/<filename>".
func (d *Dest) Write(ctx context.Context, cfg types.BackupConfig, filename string, data []byte) error {
	if cfg.S3Bucket == "" {
		return fmt.Errorf("s3dest: s3_bucket not configured")
	}
	client := d.client(cfg)

	key := filename
	if cfg.S3Prefix != "" {
		key = strings.TrimSuffix(cfg.S3Prefix, "/") + "/" + filename
	}

	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(cfg.S3Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("s3dest: put object %s/%s: %w", cfg.S3Bucket, key, err)
	}
	return nil
}

// List returns filenames under cfg.S3Bucket/cfg.S3Prefix, prefix-stripped so
// results are directly usable as Write/Read filenames.
func (d *Dest) List(ctx context.Context, cfg types.BackupConfig) ([]string, error) {
	if cfg.S3Bucket == "" {
		return nil, fmt.Errorf("s3dest: s3_bucket not configured")
	}
	client := d.client(cfg)
	prefix := ""
	if cfg.S3Prefix != "" {
		prefix = strings.TrimSuffix(cfg.S3Prefix, "/") + "/"
	}
	var out []string
	var token *string
	for {
		resp, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
			Bucket:            aws.String(cfg.S3Bucket),
			Prefix:            aws.String(prefix),
			ContinuationToken: token,
		})
		if err != nil {
			return nil, fmt.Errorf("s3dest: list objects %s/%s: %w", cfg.S3Bucket, prefix, err)
		}
		for _, obj := range resp.Contents {
			out = append(out, strings.TrimPrefix(aws.ToString(obj.Key), prefix))
		}
		if !aws.ToBool(resp.IsTruncated) {
			break
		}
		token = resp.NextContinuationToken
	}
	return out, nil
}

// Read fetches cfg.S3Bucket/cfg.S3Prefix/filename.
func (d *Dest) Read(ctx context.Context, cfg types.BackupConfig, filename string) ([]byte, error) {
	if cfg.S3Bucket == "" {
		return nil, fmt.Errorf("s3dest: s3_bucket not configured")
	}
	client := d.client(cfg)
	key := filename
	if cfg.S3Prefix != "" {
		key = strings.TrimSuffix(cfg.S3Prefix, "/") + "/" + filename
	}
	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfg.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("s3dest: get object %s/%s: %w", cfg.S3Bucket, key, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("s3dest: read object body %s/%s: %w", cfg.S3Bucket, key, err)
	}
	return data, nil
}
