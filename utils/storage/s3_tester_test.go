package storage

import (
	"context"
	"fmt"

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

		t.client = s3.NewFromConfig(cfg)

		return nil
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
	input := s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	}

	_, err := t.getClient().DeleteBucket(context.Background(), &input)
	return err
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
				func(service string, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:           ep,
						SigningRegion: region,
					}, nil
				},
			),
		),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("AKID", "SECRETPASSWORD", "")),
	)

	return cfg, err
}
