package storage

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type s3Tester struct {
	client   *s3.Client
	endpoint string
}

func (t *s3Tester) setup(ctx context.Context, endpoint string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("ap-northeast-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s3AccessKey, s3SecretKey, "")),
	)
	if err != nil {
		return err
	}

	t.endpoint = endpoint
	t.client = s3.NewFromConfig(cfg, func(opt *s3.Options) {
		opt.UsePathStyle = true // virtual host styleだと名前解決ができない(bucket.localhost~~になるため)
		opt.BaseEndpoint = aws.String(endpoint)
	})

	_, err = t.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucketName)})
	if err == nil {
		return nil
	}

	var notFound *types.NotFound
	if !errors.As(err, &notFound) {
		return err
	}

	_, err = t.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucketName)})
	return err
}

func (t *s3Tester) getClient() *s3.Client {
	return t.client
}
