package storage

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const bucketName = "test-bucket"

var (
	s3Main *s3Tester
)

// 参考：https://github.com/volatiletech/sqlboiler/blob/master/templates/test/singleton
func TestMain(m *testing.M) {
	if s3Main == nil {
		fmt.Println("no ddbMain tester interface was ready")
		os.Exit(-1)
	}

	rand.Seed(time.Now().UnixNano())

	var err error

	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "minio/minio",
		Tag:        "latest",
		Cmd:        []string{"minio", "server", "/data"},
		Env: []string{
			"MINIO_ROOT_USER=AKID",
			"MINIO_ROOT_PASSWORD=SECRETPASSWORD",
			"MINIO_DOMAIN=s3",
		},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"9000/tcp": {{HostPort: "19000"}},
		},
	}, func(config *docker.HostConfig) {

		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	if err = resource.Expire(60); err != nil {
		log.Fatalf("Could not set resource expiration: %s", err)
	}

	if err = pool.Retry(s3Main.setupFunc(resource)); err != nil {
		fmt.Println("Unable to execute setup:", err)
		os.Exit(-4)
	}

	var code int

	err = s3Main.setupBucket()

	if err != nil {
		fmt.Println("failed to create table:", err)
	}

	code = m.Run()

	if err = s3Main.teardown(); err != nil {
		fmt.Println("Unable to execute teardown:", err)
		os.Exit(-5)
	}

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}
