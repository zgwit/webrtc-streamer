package rtsp

import (
	"github.com/deepch/vdk/av"
	"github.com/pion/webrtc/v3"
)

type Session struct {
	connection *webrtc.PeerConnection
	streams    map[int]*Stream
	queue      chan *av.Packet
}

func newClient(connection *webrtc.PeerConnection) *Session {
	s := &Session{
		connection: connection,
		streams:    make(map[int]*Stream),
		queue:      make(chan *av.Packet, 100),
	}
	go s.sender()
	return s
}

func (s *Session) Put(pkt *av.Packet) {
	if len(s.queue) < cap(s.queue) {
		s.queue <- pkt
	}
}

func (s *Session) sender() {
	for {
		//TODO select quit

		pkt := <-s.queue
		if pkt == nil {
			break
		}

		stream, ok := s.streams[int(pkt.Idx)]
		if !ok {
			continue
		}

		err := stream.Write(pkt)
		if err != nil {
			break
		}
	}
}
