// Package s3compat stores objects in S3-compatible backends and presigns reads.
package s3compat

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/watzon/cypra/internal/storage"
)

//revive:disable:exported

type Config struct {
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	PathStyle       bool
}

type Store struct {
	bucket    string
	client    *s3.Client
	presigner *s3.PresignClient
}

func New(config Config) Store {
	region := config.Region
	if region == "" {
		region = "auto"
	}
	awsConfig := aws.Config{
		Region:      region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(config.AccessKeyID, config.SecretAccessKey, "")),
	}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.UsePathStyle = config.PathStyle
		if config.Endpoint != "" {
			options.BaseEndpoint = aws.String(config.Endpoint)
		}
	})
	return Store{bucket: config.Bucket, client: client, presigner: s3.NewPresignClient(client)}
}

func (s Store) Put(ctx context.Context, object storage.Object) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(object.Key), Body: bytes.NewReader(object.Bytes), ContentType: aws.String(object.ContentType)})
	return err
}

func (s Store) Get(ctx context.Context, key string) (storage.Object, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return storage.Object{}, err
	}
	defer func() { _ = output.Body.Close() }()
	content, err := io.ReadAll(output.Body)
	if err != nil {
		return storage.Object{}, err
	}
	object := storage.Object{Key: key, Bytes: content}
	if output.ContentType != nil {
		object.ContentType = *output.ContentType
	}
	return object, nil
}

func (s Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	return err
}

func (s Store) SignedURL(key string, ttl time.Duration) (string, error) {
	if ttl > time.Hour {
		ttl = time.Hour
	}
	presigned, err := s.presigner.PresignGetObject(context.Background(), &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)}, func(options *s3.PresignOptions) {
		options.Expires = ttl
	})
	if err != nil {
		return "", err
	}
	return presigned.URL, nil
}
