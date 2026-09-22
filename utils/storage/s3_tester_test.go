package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/ory/dockertest/v3"
)

type s3Tester struct {
	client   *s3.Client
	endpoint string
}

func init() {
	s3Main = &s3Tester{}
}

func (t *s3Tester) setupFunc(resource *dockertest.Resource) func() error {
	return func() error {
		cfg, err := s3TestConfig(context.Background())
		t.endpoint = fmt.Sprintf("http://%s", resource.GetHostPort("9000/tcp"))

		if err != nil {
			return err
		}

		t.client = s3.NewFromConfig(cfg, func(opt *s3.Options) {
			opt.UsePathStyle = true // virtual host styleだと名前解決ができない(bucket.localhost~~になるため)
			opt.BaseEndpoint = aws.String(t.endpoint)
		})

		// Wait for the S3 API and create the test bucket once RustFS is ready.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
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
}

func (t *s3Tester) getClient() *s3.Client {
	return t.client
}

func s3TestConfig(ctx context.Context) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("ap-northeast-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s3AccessKey, s3SecretKey, "")),
	)

	return cfg, err
}
