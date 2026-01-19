package rtsp

import (
	"errors"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/description"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtpav1"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtph264"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtph265"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtpvp8"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtpvp9"
	"github.com/busy-cloud/boat/lib"
	"github.com/busy-cloud/boat/log"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type Encode func(au []byte) ([]*rtp.Packet, error)
type Decode func(pkt *rtp.Packet) ([][]byte, error)

type Client struct {
	Url   string
	Audio bool

	client *gortsplib.Client
	//medias []*description.Media

	videoFormat string
	videoCodec  string
	videoDecode Decode

	audioFormat string
	audioCodec  string

	viewers lib.Map[Viewer]
}

func (c *Client) Check() error {
	if c.client != nil {
		return nil
	}

	u, err := base.ParseURL(c.Url)
	if err != nil {
		return err
	}

	c.client = &gortsplib.Client{
		Scheme: u.Scheme,
		Host:   u.Host,
	}

	//连接
	err = c.client.Start()
	if err != nil {
		c.client.Close()
		c.client = nil
		return err
	}

	//获取信息
	desc, _, err := c.client.Describe(u)
	if err != nil {
		c.client.Close()
		c.client = nil
		return err
	}
	//c.medias = desc.Medias

	for _, media := range desc.Medias {
		if media.Type == description.MediaTypeVideo {
			for _, format := range media.Formats {
				codec := format.Codec()
				switch codec {
				case "H264":
					c.videoFormat = webrtc.MimeTypeH264
					c.videoCodec = codec
					decoder := &rtph264.Decoder{}
					_ = decoder.Init()
					c.videoDecode = decoder.Decode
				case "H265":
					c.videoFormat = webrtc.MimeTypeH265
					c.videoCodec = codec
					decoder := &rtph265.Decoder{}
					_ = decoder.Init()
					c.videoDecode = decoder.Decode
				case "VP8":
					c.videoFormat = webrtc.MimeTypeVP8
					c.videoCodec = codec
					decoder := &rtpvp8.Decoder{}
					_ = decoder.Init()
					c.videoDecode = func(pkt *rtp.Packet) ([][]byte, error) {
						nalu, err := decoder.Decode(pkt)
						if err != nil {
							return nil, err
						}
						return [][]byte{nalu}, nil
					}
				case "VP9":
					c.videoFormat = webrtc.MimeTypeVP9
					c.videoCodec = codec
					decoder := &rtpvp9.Decoder{}
					_ = decoder.Init()
					c.videoDecode = func(pkt *rtp.Packet) ([][]byte, error) {
						nalu, err := decoder.Decode(pkt)
						if err != nil {
							return nil, err
						}
						return [][]byte{nalu}, nil
					}
				case "AV1":
					c.videoFormat = webrtc.MimeTypeAV1
					c.videoCodec = codec
					decoder := &rtpav1.Decoder{}
					_ = decoder.Init()
					c.videoDecode = decoder.Decode
				default:
					continue
				}
				//只用第一个视频
				break
			}
		} else if media.Type == description.MediaTypeAudio {
			for _, format := range media.Formats {
				codec := format.Codec()
				switch codec {
				case "G722":
					c.audioFormat = webrtc.MimeTypeG722
					c.audioCodec = codec
				case "OPUS":
					c.audioFormat = webrtc.MimeTypeOpus
					c.audioCodec = codec
				case "PCMA":
					c.audioFormat = webrtc.MimeTypePCMA
					c.audioCodec = codec
				case "PCMU":
					c.audioFormat = webrtc.MimeTypePCMU
					c.audioCodec = codec
				default:
					continue
				}
				//只用第一个音频
				break
			}
		}
	}

	err = c.client.SetupAll(desc.BaseURL, desc.Medias)
	if err != nil {
		c.client.Close()
		c.client = nil
		return err
	}

	//播放
	_, err = c.client.Play(nil)
	if err != nil {
		c.client.Close()
		c.client = nil
		return err
	}

	//控制包
	c.client.OnPacketRTCPAny(func(media *description.Media, packet rtcp.Packet) {
		c.viewers.Range(func(name string, viewer *Viewer) bool {
			err := viewer.pc.WriteRTCP([]rtcp.Packet{packet})
			if err != nil {
				log.Error(err)
			}
			return true
		})
	})

	//监听数据包，并转发
	c.client.OnPacketRTPAny(func(media *description.Media, format format.Format, packet *rtp.Packet) {

		if media.Type == description.MediaTypeVideo {
			nalus, err := c.videoDecode(packet)
			if err != nil {

				if errors.Is(err, rtph264.ErrNonStartingPacketAndNoPrevious) ||
					errors.Is(err, rtph264.ErrMorePacketsNeeded) {
					return
				}

				log.Error(err)
				return
			}

			for _, nalu := range nalus {
				//H264
				nalu = append([]byte{0, 0, 0, 1}, nalu...)
				c.viewers.Range(func(name string, viewer *Viewer) bool {
					viewer.PutVideo(nalu)
					//for _, pkt := range nalus {
					//	//pkt.SSRC = viewer.ssrc //修改webrtc的SSRC
					//	viewer.PutVideo(pkt)
					//}
					return true
				})
			}

		} else if media.Type == description.MediaTypeAudio {
			c.viewers.Range(func(name string, viewer *Viewer) bool {
				//viewer.PutAudio(packet)
				return true
			})
		}
	})

	return nil
}

func (c *Client) Attach(cid string, pc *webrtc.PeerConnection) (err error) {
	if c.client == nil {
		return errors.New("未连接")
	}

	s := newViewer(pc)

	if c.videoFormat != "" {
		s.videoTrack, err = webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: c.videoFormat}, "pion-video", "pion-video")
		if err != nil {
			return err
		}
		if rtpSender, err := s.pc.AddTrack(s.videoTrack); err != nil {
			return err
		} else {
			go consumeRtpSender(rtpSender)
			s.ssrc = uint32(rtpSender.GetParameters().Encodings[0].SSRC)
		}
	}

	if c.audioFormat != "" {
		s.audioTrack, err = webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: c.audioFormat}, "pion-audio", "pion-audio")
		if err != nil {
			return err
		}
		if rtpSender, err := s.pc.AddTrack(s.audioTrack); err != nil {
			return err
		} else {
			go consumeRtpSender(rtpSender)
		}
	}

	c.viewers.Store(cid, s)
	return nil
}
