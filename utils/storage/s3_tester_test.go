package storage

import (
	"context"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ory/dockertest/v3"
)

type s3Tester struct {
	client *s3.Client
}

func init() {
	s3Main = &s3Tester{}
}

func (t *s3Tester) setupFunc(resource *dockertest.Resource) func() error {
	return func() error {
		cfg, err := s3TestConfig(context.Background(), resource.GetPort("9000/tcp"))
		if err != nil {
			return err
		}

		t.client = s3.NewFromConfig(cfg, func(opt *s3.Options) {
			opt.UsePathStyle = true // virtual host styleだと名前解決ができない(bucket.localhost~~になるため)
		})

		return minioHealthCheck(resource.GetPort("9000/tcp"))
	}
}

func (t *s3Tester) setupBucket() error {
	input := s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := t.getClient().CreateBucket(context.Background(), &input)
	return err
}

func (t *s3Tester) teardown() error {
	return nil
}

func (t *s3Tester) getClient() *s3.Client {
	return t.client
}

func s3TestConfig(ctx context.Context, port string) (aws.Config, error) {
	ep := fmt.Sprintf("http://localhost:%s", port)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("ap-northeast-1"),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(_ string, region string, _ ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:           ep,
						SigningRegion: region,
					}, nil
				},
			),
		),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("ROOT", "PASSWORD", "")),
	)

	return cfg, err
}

func minioHealthCheck(host string) error {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/minio/health/live", host))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed: %s", resp.Status)
	}

	return nil
}
