package source

import "github.com/pion/webrtc/v4"

type Source interface {
	Check() error
	Attach(cid string, pc *webrtc.PeerConnection) error
}
