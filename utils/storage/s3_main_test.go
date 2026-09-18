package storage

import (
	"fmt"
	"os"
	"path/filepath"
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
	garageRepository = "dxflrs/garage"
	garageVersion    = "v2.4.1"
	garageDigest     = "sha256:9c96caa2612d3411acc5b0e6701fb238dbfba33e533a6d7d3d811a4b12d0d020"
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

	configPath, err := filepath.Abs("../../dev/garage.toml")
	if err != nil {
		fmt.Println("Could not resolve Garage configuration:", err)
		return 1
	}

	// dockertest's automatic pull passes Tag verbatim to the Docker API, which
	// accepts a tag or digest but not "version@digest". Pull by digest first.
	imageTag := garageVersion + "@" + garageDigest
	if _, err := pool.Client.InspectImage(garageRepository + ":" + imageTag); err != nil {
		if err := pool.Client.PullImage(docker.PullImageOptions{
			Repository: garageRepository,
			Tag:        garageDigest,
		}, docker.AuthConfiguration{}); err != nil {
			fmt.Println("Could not pull Garage:", err)
			return 1
		}
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: garageRepository,
		Tag:        imageTag,
		Cmd:        []string{"/garage", "server", "--single-node", "--default-bucket"},
		Env: []string{
			"GARAGE_DEFAULT_ACCESS_KEY=" + s3AccessKey,
			"GARAGE_DEFAULT_SECRET_KEY=" + s3SecretKey,
			"GARAGE_DEFAULT_BUCKET=" + bucketName,
		},
		Mounts:       []string{configPath + ":/etc/garage.toml:ro"},
		ExposedPorts: []string{"9000/tcp"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"9000/tcp": {{HostIP: "127.0.0.1", HostPort: ""}},
		},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		fmt.Println("Could not start Garage:", err)
		return 1
	}
	defer func() {
		if err := pool.Purge(resource); err != nil {
			fmt.Println("Could not purge Garage:", err)
			code = 1
		}
	}()

	if err = resource.Expire(300); err != nil {
		fmt.Println("Could not set Garage expiration:", err)
		return 1
	}

	if err = pool.Retry(s3Main.setupFunc(resource)); err != nil {
		fmt.Println("Unable to initialize Garage:", err)
		return 1
	}

	return m.Run()
}
