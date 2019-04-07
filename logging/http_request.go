package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// HTTPRequest Stackdriver logging httpRequest Field
func HTTPRequest(req *HTTPPayload) zap.Field {
	return zap.Object("httpRequest", req)
}

// HTTPPayload Stackdriver logging httpRequest Payload
type HTTPPayload struct {
	RequestMethod                  string `json:"requestMethod"`
	RequestURL                     string `json:"requestUrl"`
	RequestSize                    string `json:"requestSize"`
	Status                         int    `json:"status"`
	ResponseSize                   string `json:"responseSize"`
	UserAgent                      string `json:"userAgent"`
	RemoteIP                       string `json:"remoteIp"`
	ServerIP                       string `json:"serverIp"`
	Referer                        string `json:"referer"`
	Latency                        string `json:"latency"`
	CacheLookup                    bool   `json:"cacheLookup"`
	CacheHit                       bool   `json:"cacheHit"`
	CacheValidatedWithOriginServer bool   `json:"cacheValidatedWithOriginServer"`
	CacheFillBytes                 string `json:"cacheFillBytes"`
	Protocol                       string `json:"protocol"`
}

// MarshalLogObject implements zapcore.ObjectMarshaller interface.
func (p HTTPPayload) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("requestMethod", p.RequestMethod)
	enc.AddString("requestUrl", p.RequestURL)
	enc.AddString("requestSize", p.RequestSize)
	enc.AddInt("status", p.Status)
	enc.AddString("responseSize", p.ResponseSize)
	enc.AddString("userAgent", p.UserAgent)
	enc.AddString("remoteIp", p.RemoteIP)
	enc.AddString("serverIp", p.ServerIP)
	enc.AddString("referer", p.Referer)
	enc.AddString("latency", p.Latency)
	enc.AddBool("cacheLookup", p.CacheLookup)
	enc.AddBool("cacheHit", p.CacheHit)
	enc.AddBool("cacheValidatedWithOriginServer", p.CacheValidatedWithOriginServer)
	enc.AddString("cacheFillBytes", p.CacheFillBytes)
	enc.AddString("protocol", p.Protocol)
	return nil
}
