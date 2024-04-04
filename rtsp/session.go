package rtsp

import (
	"github.com/deepch/vdk/av"
	"github.com/pion/webrtc/v3"
)

type Session struct {
	pc      *webrtc.PeerConnection
	streams map[int]*Stream
	queue   chan *av.Packet
}

func newSession(pc *webrtc.PeerConnection) *Session {
	s := &Session{
		pc:      pc,
		streams: make(map[int]*Stream),
		queue:   make(chan *av.Packet, 100),
	}

	go s.sender()

	return s
}

func (s *Session) attach(codecs []av.CodecData) (err error) {
	for i, stream := range codecs {
		var track *webrtc.TrackLocalStaticSample
		if stream.Type().IsVideo() {
			if stream.Type() == av.H264 {
				track, err = webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
					MimeType: webrtc.MimeTypeH264,
				}, "pion-rtsp-video", "pion-video")
				if err != nil {
					return err
				}
				if rtpSender, err := s.pc.AddTrack(track); err != nil {
					return err
				} else {
					go consumeRtpSender(rtpSender)
				}
			}
			if stream.Type() == av.H265 {
				track, err = webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
					MimeType: webrtc.MimeTypeH265,
				}, "pion-rtsp-video", "pion-video")
				if err != nil {
					return err
				}
				if rtpSender, err := s.pc.AddTrack(track); err != nil {
					return err
				} else {
					go consumeRtpSender(rtpSender)
				}
			}
		} else if stream.Type().IsAudio() {
			AudioCodecString := webrtc.MimeTypePCMA
			switch stream.Type() {
			case av.PCM_ALAW:
				AudioCodecString = webrtc.MimeTypePCMA
			case av.PCM_MULAW:
				AudioCodecString = webrtc.MimeTypePCMU
			case av.OPUS:
				AudioCodecString = webrtc.MimeTypeOpus
			default:
				//log.Println(ErrorIgnoreAudioTrack)
				continue
			}
			track, err = webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
				MimeType:  AudioCodecString,
				Channels:  uint16(stream.(av.AudioCodecData).ChannelLayout().Count()),
				ClockRate: uint32(stream.(av.AudioCodecData).SampleRate()),
			}, "pion-rtsp-audio", "pion-audio")
			if err != nil {
				return err
			}
			if rtpSender, err := s.pc.AddTrack(track); err != nil {
				return err
			} else {
				go consumeRtpSender(rtpSender)
			}
		}
		s.streams[i] = &Stream{track: track, codec: stream}
	}
	return nil
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

func consumeRtpSender(sender *webrtc.RTPSender) {
	buf := make([]byte, 1500)
	for {
		if _, _, err := sender.Read(buf); err != nil {
			return
		}
	}
}
