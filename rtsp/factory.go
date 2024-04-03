package rtsp

import (
	"github.com/zgwit/webrtc-streamer/source"
)

func init() {
	source.Register("rtsp", factory)
}

func factory(url string) (source.Source, error) {
	return &Camera{Url: url}, nil
}
