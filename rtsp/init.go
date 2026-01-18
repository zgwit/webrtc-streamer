package rtsp

import (
	"webrtc-streamer/source"
)

func init() {
	source.Register("rtsp", factory)
}

func factory(url string, options source.Options) (source.Source, error) {
	return &Client{Url: url}, nil
}
