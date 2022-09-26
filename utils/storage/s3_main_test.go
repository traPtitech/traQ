package storage

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"
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

	if err = s3Main.setup(); err != nil {
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

	os.Exit(code)
}
