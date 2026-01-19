package rtsp

import (
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

type Packet struct {
	codec  string
	packet *rtp.Packet
}

type Viewer struct {
	pc *webrtc.PeerConnection

	ssrc uint32

	videoQueue chan []byte
	audioQueue chan []byte

	videoTrack *webrtc.TrackLocalStaticSample
	audioTrack *webrtc.TrackLocalStaticSample
}

func newViewer(pc *webrtc.PeerConnection) *Viewer {
	s := &Viewer{
		pc:         pc,
		videoQueue: make(chan []byte, 100),
		audioQueue: make(chan []byte, 100),
	}

	go s.videoSender()
	go s.audioSender()

	return s
}

func (s *Viewer) PutVideo(data []byte) {
	if len(s.videoQueue) < cap(s.videoQueue) {
		s.videoQueue <- data
	}
}

func (s *Viewer) PutAudio(data []byte) {
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
		err := s.videoTrack.WriteSample(media.Sample{
			Data:     pkt,
			Duration: time.Second / 30, //帧率
		})
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
		err := s.audioTrack.WriteSample(media.Sample{
			Data:     pkt,
			Duration: time.Second / 30, //帧率
		})
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
