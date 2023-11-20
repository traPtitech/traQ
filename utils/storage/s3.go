package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils"
)

// S3FileStorage OpenStack Swiftストレージ
type S3FileStorage struct {
	bucket   string
	client   *s3.Client
	cacheDir string
	mutexes  *utils.KeyMutex
}

// NewS3FileStorage 引数の情報でS3ストレージを生成します
func NewS3FileStorage(bucket, region, endpoint, apiKey, apiSecret string, forcePathStyle bool, cacheDir string) (*S3FileStorage, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithEndpointResolverWithOptions(
			aws.EndpointResolverWithOptionsFunc(
				func(service string, region string, options ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:           endpoint,
						SigningRegion: region,
					}, nil
				},
			),
		),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(apiKey, apiSecret, "")),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(opt *s3.Options) {
		opt.UsePathStyle = forcePathStyle
	})

	m := &S3FileStorage{
		bucket:   bucket,
		client:   client,
		cacheDir: cacheDir,
		mutexes:  utils.NewKeyMutex(256),
	}

	return m, nil
}

// OpenFileByKey ファイルを取得します
func (fs *S3FileStorage) OpenFileByKey(key string, fileType model.FileType) (reader io.ReadSeekCloser, err error) {
	cacheName := fs.getCacheFilePath(key)

	input := &s3.GetObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	}

	if !fs.cacheable(fileType) {

		file, err := fs.getObject(context.Background(), input)
		if err != nil {
			var nsk *types.NoSuchKey
			if errors.As(err, &nsk) {
				return nil, ErrFileNotFound
			}
			return nil, err
		}
		return file, nil
	}

	fs.mutexes.Lock(key)
	if _, err := os.Stat(cacheName); os.IsNotExist(err) {
		defer fs.mutexes.Unlock(key)
		remote, err := fs.getObject(context.Background(), input)
		if err != nil {
			var nsk *types.NoSuchKey
			if errors.As(err, &nsk) {
				return nil, ErrFileNotFound
			}
			return nil, err
		}

		// save cache
		file, err := os.OpenFile(cacheName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666) // ファイルが存在していた場合はエラーにしてremoteを返す
		if err != nil {
			return remote, nil
		}

		if _, err := io.Copy(file, remote); err != nil {
			file.Close()
			_ = os.Remove(cacheName)
			return nil, err
		}

		_, _ = file.Seek(0, 0)
		return file, nil
	}
	fs.mutexes.Unlock(key)

	// from cache
	reader, err = os.Open(cacheName)
	if err != nil {
		return nil, ErrFileNotFound
	}
	return reader, nil
}

// SaveByKey srcの内容をkeyで指定されたファイルに書き込みます
func (fs *S3FileStorage) SaveByKey(src io.Reader, key, name, contentType string, fileType model.FileType) (err error) {
	if fs.cacheable(fileType) {
		cacheName := fs.getCacheFilePath(key)

		file, fe := os.Create(cacheName)
		if fe == nil {
			defer func() {
				file.Close()
				if err != nil {
					_ = os.Remove(cacheName)
				}
			}()
			src = io.TeeReader(src, file)
		}
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(fs.bucket),
		Key:         aws.String(key),
		Body:        src,
		ContentType: aws.String(contentType),
		Metadata: map[string]string{
			"Content-Disposition": fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(name)),
		},
	}

	uploader := manager.NewUploader(fs.client)

	_, err = uploader.Upload(context.Background(), input)
	return
}

// DeleteByKey ファイルを削除します
func (fs *S3FileStorage) DeleteByKey(key string, _ model.FileType) (err error) {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(key),
	}

	_, err = fs.client.DeleteObject(context.Background(), input)
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return ErrFileNotFound
		}
		return err
	}

	// delete cache
	cacheName := fs.getCacheFilePath(key)
	if _, err := os.Stat(cacheName); err == nil {
		_ = os.Remove(cacheName)
	}
	return nil
}

// GenerateAccessURL keyで指定されたファイルの直接アクセスURLを発行する。
func (fs *S3FileStorage) GenerateAccessURL(key string, fileType model.FileType) (string, error) {
	if !fs.cacheable(fileType) {
		if _, err := os.Stat(fs.getCacheFilePath(key)); os.IsNotExist(err) {

			pc := s3.NewPresignClient(fs.client)

			req, _ := pc.PresignGetObject(context.Background(), &s3.GetObjectInput{
				Bucket: aws.String(fs.bucket),
				Key:    aws.String(key),
			}, func(options *s3.PresignOptions) {
				options.Expires = 5 * time.Minute
			})

			return req.URL, nil
		}
	}
	return "", nil
}

func (fs *S3FileStorage) getCacheFilePath(key string) string {
	return fs.cacheDir + "/" + key
}

func (fs *S3FileStorage) cacheable(fileType model.FileType) bool {
	return fileType == model.FileTypeIcon || fileType == model.FileTypeStamp || fileType == model.FileTypeThumbnail
}

func (fs *S3FileStorage) getObject(ctx context.Context, input *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3Object, error) {
	if input == nil {
		return nil, fmt.Errorf("input is nil")
	}

	attrIn := s3.HeadObjectInput{
		Bucket: input.Bucket,
		Key:    input.Key,
	}

	attrOut, err := fs.client.HeadObject(ctx, &attrIn)
	if err != nil {
		return nil, err
	}

	objOut, err := fs.client.GetObject(ctx, input, optFns...)
	if err != nil {
		return nil, err
	}

	obj := s3Object{
		client:   fs.client,
		input:    *input,
		resp:     objOut,
		length:   *attrOut.ContentLength,
		lengthOk: true,
		body:     objOut.Body,
	}

	return &obj, nil
}
