package camera

import (
	"webrtc-streamer/source"
)

func init() {
	source.Register("webcam", factory)
}

func factory(url string, options source.Options) (source.Source, error) {
	return &Camera{}, nil
}
