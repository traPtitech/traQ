package logging

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	"testing"
)

func TestHTTPRequest(t *testing.T) {
	t.Parallel()

	p := &HTTPPayload{}
	assert.Equal(t, p, HTTPRequest(p).Interface.(*HTTPPayload))
}

func TestHTTPPayload_MarshalLogObject(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	p := &HTTPPayload{
		RequestMethod:                  "GET",
		RequestURL:                     "/test",
		RequestSize:                    "123",
		Status:                         200,
		ResponseSize:                   "2222",
		UserAgent:                      "Test",
		RemoteIP:                       "127.0.0.1",
		ServerIP:                       "127.0.0.1",
		Referer:                        "/",
		Latency:                        "0.012s",
		CacheLookup:                    true,
		CacheHit:                       true,
		CacheValidatedWithOriginServer: false,
		CacheFillBytes:                 "123",
		Protocol:                       "HTTP/1.1",
	}

	enc := zapcore.NewMapObjectEncoder()

	if assert.NoError(p.MarshalLogObject(enc)) {
		assert.EqualValues(p.RequestMethod, enc.Fields["requestMethod"])
		assert.EqualValues(p.RequestURL, enc.Fields["requestUrl"])
		assert.EqualValues(p.RequestSize, enc.Fields["requestSize"])
		assert.EqualValues(p.Status, enc.Fields["status"])
		assert.EqualValues(p.ResponseSize, enc.Fields["responseSize"])
		assert.EqualValues(p.UserAgent, enc.Fields["userAgent"])
		assert.EqualValues(p.RemoteIP, enc.Fields["remoteIp"])
		assert.EqualValues(p.ServerIP, enc.Fields["serverIp"])
		assert.EqualValues(p.Referer, enc.Fields["referer"])
		assert.EqualValues(p.Latency, enc.Fields["latency"])
		assert.EqualValues(p.CacheLookup, enc.Fields["cacheLookup"])
		assert.EqualValues(p.CacheHit, enc.Fields["cacheHit"])
		assert.EqualValues(p.CacheValidatedWithOriginServer, enc.Fields["cacheValidatedWithOriginServer"])
		assert.EqualValues(p.CacheFillBytes, enc.Fields["cacheFillBytes"])
		assert.EqualValues(p.Protocol, enc.Fields["protocol"])
	}
}
