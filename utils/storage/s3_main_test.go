package storage

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	bucketName  = "test-bucket"
	s3AccessKey = "GK0123456789abcdef0123456789abcdef"
	s3SecretKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	// Keep the version and digest in sync with compose.yaml.
	rustfsRepository = "rustfs/rustfs"
	rustfsVersion    = "1.0.0"
	rustfsDigest     = "sha256:8cc9801755448b71a786705ce76692c77e14936cccd87cf2fc31842e58f4d1ff"
)

var s3Main *s3Tester

func TestMain(m *testing.M) {
	os.Exit(runStorageTests(m))
}

func runStorageTests(m *testing.M) (code int) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		fmt.Println("Could not connect to docker:", err)
		return 1
	}
	pool.MaxWait = time.Minute

	// dockertest's automatic pull passes Tag verbatim to the Docker API, which
	// accepts a tag or digest but not "version@digest". Pull by digest first.
	imageTag := rustfsVersion + "@" + rustfsDigest
	if _, err := pool.Client.InspectImage(rustfsRepository + ":" + imageTag); err != nil {
		if err := pool.Client.PullImage(docker.PullImageOptions{
			Repository: rustfsRepository,
			Tag:        rustfsDigest,
		}, docker.AuthConfiguration{}); err != nil {
			fmt.Println("Could not pull RustFS:", err)
			return 1
		}
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: rustfsRepository,
		Tag:        imageTag,
		Env: []string{
			"RUSTFS_ACCESS_KEY=" + s3AccessKey,
			"RUSTFS_SECRET_KEY=" + s3SecretKey,
			"RUSTFS_REGION=ap-northeast-1",
			"RUSTFS_CONSOLE_ENABLE=false",
			"RUSTFS_OBS_LOGGER_LEVEL=error",
		},
		ExposedPorts: []string{"9000/tcp"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"9000/tcp": {{HostIP: "127.0.0.1", HostPort: "0"}},
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		// RustFS also exposes its console port, but the tests only need the S3 API.
		config.PublishAllPorts = false
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		fmt.Println("Could not start RustFS:", err)
		return 1
	}
	defer func() {
		if err := pool.Purge(resource); err != nil {
			fmt.Println("Could not purge RustFS:", err)
			code = 1
		}
	}()

	if err = resource.Expire(300); err != nil {
		fmt.Println("Could not set RustFS expiration:", err)
		return 1
	}

	if err = pool.Retry(s3Main.setupFunc(resource)); err != nil {
		fmt.Println("Unable to initialize RustFS:", err)
		return 1
	}

	return m.Run()
}
