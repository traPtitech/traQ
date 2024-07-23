package storage

import (
	"context"
	"fmt"
	"github.com/aws/smithy-go"
	smithyendpoints "github.com/aws/smithy-go/endpoints"
	"net/http"
	"net/url"

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

func (r *customResolver) ResolveEndpoint(_ context.Context, params s3.EndpointParameters) (smithyendpoints.Endpoint, error) {
	u, err := url.Parse(r.endpoint)
	if err != nil {
		return smithyendpoints.Endpoint{}, err
	}

	properties := smithy.Properties{}
	properties.Set("Region", params.Region)

	return smithyendpoints.Endpoint{
		URI:        *u,
		Properties: properties,
	}, nil
}

func s3TestConfig(ctx context.Context, port string) (aws.Config, error) {
	ep := fmt.Sprintf("http://localhost:%s", port)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("ap-northeast-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("ROOT", "PASSWORD", "")),
	)
	_ = s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.EndpointResolverV2 = &customResolver{endpoint: ep}
		o.BaseEndpoint = aws.String(ep)
	})
	if err != nil {
		return aws.Config{}, err
	}

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
