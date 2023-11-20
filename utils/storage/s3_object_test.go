package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/testdata/images"
)

const testObjKey = "testObj"

func TestS3Object(t *testing.T) {
	setupObject(t)
	t.Run("Read", tests3ObjectRead)
	t.Run("Close", tests3ObjectClose)
	t.Run("Seek", tests3ObjectSeek)
}

func tests3ObjectRead(t *testing.T) {
	t.Parallel()
	type args struct {
		p []byte
	}
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) *s3Object
		args      args
		wantN     int
		wantErr   bool
	}{
		{
			name: "success",
			setupFunc: func(t *testing.T) *s3Object {
				input := &s3.GetObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(testObjKey),
				}

				obj, err := getTestObject(t, input)
				assert.NoError(t, err)
				return obj
			},
			args: args{
				p: make([]byte, 1024),
			},
			wantN: 1024,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o := tt.setupFunc(t)
			gotN, err := o.Read(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("s3Object.Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("s3Object.Read() = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}

func tests3ObjectClose(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) *s3Object
		wantErr   bool
	}{
		{
			name: "success",
			setupFunc: func(t *testing.T) *s3Object {
				input := &s3.GetObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(testObjKey),
				}

				obj, err := getTestObject(t, input)
				assert.NoError(t, err)
				return obj
			},
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o := tt.setupFunc(t)
			if err := o.Close(); (err != nil) != tt.wantErr {
				t.Errorf("s3Object.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func tests3ObjectSeek(t *testing.T) {
	t.Parallel()
	type args struct {
		offset int64
		whence int
	}
	tests := []struct {
		name       string
		setupFunc  func(t *testing.T) *s3Object
		args       args
		wantNewPos func(len, pos, offset int64) int64
		wantErr    bool
	}{
		{
			name: "success/whence=0",
			setupFunc: func(t *testing.T) *s3Object {
				input := &s3.GetObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(testObjKey),
				}

				obj, err := getTestObject(t, input)
				assert.NoError(t, err)
				return obj
			},
			args: args{
				offset: 1024,
				whence: 0,
			},
			wantNewPos: func(len, pos, offset int64) int64 {
				return offset
			},
		},
		{
			name: "success/whence=1",
			setupFunc: func(t *testing.T) *s3Object {
				input := &s3.GetObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(testObjKey),
				}

				obj, err := getTestObject(t, input)
				assert.NoError(t, err)
				obj.pos = 16
				return obj
			},
			args: args{
				offset: 1024,
				whence: 1,
			},
			wantNewPos: func(len, pos, offset int64) int64 {
				return pos + offset
			},
		},
		{
			name: "success/whence=2",
			setupFunc: func(t *testing.T) *s3Object {
				input := &s3.GetObjectInput{
					Bucket: aws.String(bucketName),
					Key:    aws.String(testObjKey),
				}

				obj, err := getTestObject(t, input)
				assert.NoError(t, err)
				return obj
			},
			args: args{
				offset: -1024,
				whence: 2,
			},
			wantNewPos: func(len, pos, offset int64) int64 {
				return len + offset
			},
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			o := tt.setupFunc(t)
			pos := o.pos
			gotNewPos, err := o.Seek(tt.args.offset, tt.args.whence)
			if (err != nil) != tt.wantErr {
				t.Errorf("s3Object.Seek() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotNewPos != tt.wantNewPos(o.length, pos, tt.args.offset) {
				t.Errorf("s3Object.Seek() = %v, want %v", gotNewPos, tt.wantNewPos(o.length, o.pos, tt.args.offset))
			}
		})
	}
}

func setupObject(t *testing.T) {
	cli := s3Main.getClient()

	assert.NotNil(t, cli)

	f, err := images.ImageFS.Open("test.png")

	assert.NoError(t, err)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(testObjKey),
		Body:        f,
		ContentType: aws.String("image/png"),
	}

	uploader := manager.NewUploader(cli)

	_, err = uploader.Upload(context.Background(), input)

	assert.NoError(t, err)
}

func getTestObject(t *testing.T, input *s3.GetObjectInput) (*s3Object, error) {
	if input == nil {
		return nil, fmt.Errorf("input is nil")
	}
	cli := s3Main.getClient()

	assert.NotNil(t, cli)

	attrIn := s3.HeadObjectInput{
		Bucket: input.Bucket,
		Key:    input.Key,
	}

	attrOut, err := cli.HeadObject(context.Background(), &attrIn)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	objOut, err := cli.GetObject(context.Background(), input)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	obj := s3Object{
		client:   cli,
		input:    *input,
		resp:     objOut,
		length:   *attrOut.ContentLength,
		lengthOk: true,
		body:     objOut.Body,
	}

	return &obj, nil
}
