package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Object struct {
	client     *s3.Client
	input      s3.GetObjectInput
	resp       *s3.GetObjectOutput
	length     int64
	lengthOk   bool
	body       io.ReadCloser
	pos        int64
	overSought bool
}

func (o *s3Object) Read(p []byte) (n int, err error) {
	if o.overSought {
		return 0, io.EOF
	}

	n, err = o.body.Read(p)
	o.pos += int64(n)
	return
}

func (o *s3Object) Close() (err error) {
	return o.body.Close()
}

// https://github.com/ncw/swift/blob/master/swift.go#L1668
func (o *s3Object) Seek(offset int64, whence int) (newPos int64, err error) {
	o.overSought = false

	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = o.pos + offset
	case io.SeekEnd:
		if !o.lengthOk {
			return o.pos, fmt.Errorf("length of file unknown")
		}
		newPos = o.length + offset
		if offset >= 0 {
			o.overSought = true
			return
		}
	default:
		panic("Unknown whence")
	}

	if newPos == o.pos {
		return
	}

	err = o.Close()
	if err != nil {
		return
	}

	if newPos > 0 {
		o.input.Range = aws.String(fmt.Sprintf("bytes=%d-", newPos))
	} else {
		o.input.Range = nil
	}

	output, err := o.client.GetObject(context.Background(), &o.input)

	if err != nil {
		return
	}

	o.body = output.Body
	o.resp = output
	o.pos = newPos
	return

}
