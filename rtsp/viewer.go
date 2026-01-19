package rtsp

import (
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type Packet struct {
	codec  string
	packet *rtp.Packet
}

type Viewer struct {
	pc *webrtc.PeerConnection

	ssrc uint32

	videoQueue chan *rtp.Packet
	audioQueue chan *rtp.Packet

	videoTrack *webrtc.TrackLocalStaticRTP
	audioTrack *webrtc.TrackLocalStaticRTP
}

func newViewer(pc *webrtc.PeerConnection) *Viewer {
	s := &Viewer{
		pc:         pc,
		videoQueue: make(chan *rtp.Packet, 100),
		audioQueue: make(chan *rtp.Packet, 100),
	}

	go s.videoSender()
	go s.audioSender()

	return s
}

func (s *Viewer) PutVideo(data *rtp.Packet) {
	if len(s.videoQueue) < cap(s.videoQueue) {
		s.videoQueue <- data
	}
}

func (s *Viewer) PutAudio(data *rtp.Packet) {
	if len(s.audioQueue) < cap(s.audioQueue) {
		s.audioQueue <- data
	}
}

func (s *Viewer) videoSender() {
	for {
		//TODO select quit
		pkt := <-s.videoQueue
		if pkt == nil {
			break
		}

		//pkt.SSRC = s.ssrc
		err := s.videoTrack.WriteRTP(pkt)
		if err != nil {
			break
		}
	}
}

func (s *Viewer) audioSender() {
	for {
		//TODO select quit
		pkt := <-s.audioQueue
		if pkt == nil {
			break
		}
		err := s.audioTrack.WriteRTP(pkt)
		if err != nil {
			break
		}
	}
}

func consumeRtpSender(sender *webrtc.RTPSender) {
	buf := make([]byte, 1500)
	for {
		if _, _, err := sender.Read(buf); err != nil {
			return
		}
	}
}
