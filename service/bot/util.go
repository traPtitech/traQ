package bot

import jsoniter "github.com/json-iterator/go"

func makePayloadJSON(payload interface{}) (b []byte, releaseFunc func(), err error) {
	cfg := jsoniter.ConfigFastest
	stream := cfg.BorrowStream(nil)
	releaseFunc = func() { cfg.ReturnStream(stream) }
	stream.WriteVal(payload)
	stream.WriteRaw("\n")
	if err = stream.Error; err != nil {
		releaseFunc()
		return nil, nil, err
	}
	return stream.Buffer(), releaseFunc, nil
}
